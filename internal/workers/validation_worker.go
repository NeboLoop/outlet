package workers

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/emailval"
	"github.com/outlet-sh/outlet/internal/services/validation"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// validationRetryDelays is the backoff after each transient failure (greylisting,
// temporary errors, timeouts). After the last one the result is final.
var validationRetryDelays = []time.Duration{15 * time.Minute, time.Hour, 4 * time.Hour}

// validationRetryDelay returns how long to wait before retrying an address that has
// been attempted `attempts` times, and false once retries are exhausted.
func validationRetryDelay(attempts int) (time.Duration, bool) {
	if attempts < 1 || attempts > len(validationRetryDelays) {
		return 0, false
	}
	return validationRetryDelays[attempts-1], true
}

// isTransient reports whether a result is worth retrying later.
func isTransient(res *emailval.Result) bool {
	if res.Status != emailval.StatusUnknown {
		return false
	}
	switch res.SubStatus {
	case emailval.SubGreylisted, emailval.SubMailServerTempError, emailval.SubTimeoutExceeded:
		return true
	}
	return false
}

// ValidationWorker processes bulk validation jobs. Jobs in 'processing' are picked up
// on every poll, so work resumes after a restart.
type ValidationWorker struct {
	svcCtx       *svc.ServiceContext
	pollInterval time.Duration
	concurrency  int // domains probed in parallel
	batchSize    int64

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// StartValidationWorker creates and starts the validation job worker.
func StartValidationWorker(svcCtx *svc.ServiceContext) *ValidationWorker {
	concurrency := svcCtx.Config.Validation.Concurrency
	if concurrency < 1 {
		concurrency = 20
	}
	ctx, cancel := context.WithCancel(context.Background())
	w := &ValidationWorker{
		svcCtx:       svcCtx,
		pollInterval: 5 * time.Second,
		concurrency:  concurrency,
		batchSize:    int64(concurrency * 10),
		ctx:          ctx,
		cancel:       cancel,
	}
	w.wg.Add(1)
	go w.run()
	return w
}

// Stop cancels in-flight probes and waits for the worker to exit.
func (w *ValidationWorker) Stop() {
	w.cancel()
	w.wg.Wait()
}

func (w *ValidationWorker) run() {
	defer w.wg.Done()
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	for {
		w.processJobs()
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// processJobs works through processing jobs one batch at a time, round-robin, until
// no job has due work.
func (w *ValidationWorker) processJobs() {
	for w.ctx.Err() == nil {
		jobs, err := w.svcCtx.DB.ListProcessingValidationJobs(w.ctx)
		if err != nil {
			if w.ctx.Err() == nil {
				logx.Errorf("validation worker: list jobs: %v", err)
			}
			return
		}
		worked := false
		for _, job := range jobs {
			if w.ctx.Err() != nil {
				return
			}
			if w.processBatch(job) {
				worked = true
			}
		}
		if !worked {
			return
		}
	}
}

// processBatch claims due rows of a job and checks them, domains in parallel and
// addresses within a domain one at a time. Returns whether any row was claimed.
func (w *ValidationWorker) processBatch(job db.ValidationJob) bool {
	jobID := sql.NullString{String: job.ID, Valid: true}
	rows, err := w.svcCtx.DB.ClaimPendingValidations(w.ctx, db.ClaimPendingValidationsParams{JobID: jobID, BatchSize: w.batchSize})
	if err != nil {
		if w.ctx.Err() == nil {
			logx.Errorf("validation worker: claim job %s: %v", job.ID, err)
		}
		return false
	}

	if len(rows) > 0 {
		byDomain := map[string][]db.EmailValidation{}
		for _, r := range rows {
			byDomain[r.Domain] = append(byDomain[r.Domain], r)
		}
		sem := make(chan struct{}, w.concurrency)
		var wg sync.WaitGroup
		for _, domainRows := range byDomain {
			wg.Add(1)
			sem <- struct{}{}
			go func(domainRows []db.EmailValidation) {
				defer wg.Done()
				defer func() { <-sem }()
				for _, r := range domainRows {
					if w.ctx.Err() != nil || w.cancelled(job) {
						return
					}
					w.check(r)
				}
			}(domainRows)
		}
		wg.Wait()
		if err := w.svcCtx.DB.RefreshValidationJobProgress(w.ctx, job.ID); err != nil && w.ctx.Err() == nil {
			logx.Errorf("validation worker: progress job %s: %v", job.ID, err)
		}
		return true
	}

	// Nothing due: the job is done once no pending or retry-scheduled rows remain.
	pending, err := w.svcCtx.DB.CountPendingValidations(w.ctx, jobID)
	if err == nil && pending == 0 {
		if err := w.svcCtx.DB.CompleteValidationJob(w.ctx, job.ID); err != nil {
			logx.Errorf("validation worker: complete job %s: %v", job.ID, err)
		} else {
			logx.Infof("validation worker: job %s completed", job.ID)
		}
	}
	return false
}

// cancelled reports whether the job was cancelled (or otherwise left processing).
func (w *ValidationWorker) cancelled(job db.ValidationJob) bool {
	cur, err := w.svcCtx.DB.GetValidationJob(w.ctx, db.GetValidationJobParams{ID: job.ID, OrgID: job.OrgID})
	return err != nil || cur.Status != "processing"
}

// check validates one queued row and either finalizes it or schedules a retry.
func (w *ValidationWorker) check(r db.EmailValidation) {
	res := w.svcCtx.Validation.Check(w.ctx, r.Email)
	if w.ctx.Err() != nil {
		return // shutting down: leave the row leased; it is retried after the lease expires
	}

	attempts := r.Attempts + 1
	row := validation.Row(res)
	params := db.UpdateEmailValidationResultParams{
		ID:            r.ID,
		Status:        row.Status,
		SubStatus:     row.SubStatus,
		FreeEmail:     row.FreeEmail,
		DidYouMean:    row.DidYouMean,
		Account:       row.Account,
		Domain:        r.Domain,
		DomainAgeDays: row.DomainAgeDays,
		SmtpProvider:  row.SmtpProvider,
		MxFound:       row.MxFound,
		MxRecord:      row.MxRecord,
		Attempts:      attempts,
	}
	if row.Domain != "" {
		params.Domain = row.Domain
	}

	delay, retry := validationRetryDelay(int(attempts))
	if retry && isTransient(res) {
		params.Status = "pending"
		params.RetryAt = sql.NullString{String: time.Now().UTC().Add(delay).Format(validation.TimeLayout), Valid: true}
	}
	if err := w.svcCtx.DB.UpdateEmailValidationResult(w.ctx, params); err != nil {
		logx.Errorf("validation worker: save %s: %v", r.ID, err)
		return
	}
	if params.Status != "pending" {
		w.svcCtx.Validation.UpdateContact(w.ctx, r.OrgID, r.Email, res)
	}
}
