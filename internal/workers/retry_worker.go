package workers

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// RetryWorkerConfig configures the retry worker
type RetryWorkerConfig struct {
	// How often to check for failed emails to retry
	PollInterval time.Duration

	// Maximum retry attempts before permanent failure
	MaxRetries int

	// Backoff intervals for retries (exponential backoff)
	RetryIntervals []time.Duration

	// Batch size for fetching failed emails
	BatchSize int
}

// DefaultRetryWorkerConfig returns sensible defaults
func DefaultRetryWorkerConfig() RetryWorkerConfig {
	return RetryWorkerConfig{
		PollInterval: 30 * time.Second,
		MaxRetries:   3,
		RetryIntervals: []time.Duration{
			5 * time.Minute,   // First retry after 5 min
			30 * time.Minute,  // Second retry after 30 min
			2 * time.Hour,     // Third retry after 2 hours
		},
		BatchSize: 100,
	}
}

// RetryWorker handles automatic retry of failed email sends
type RetryWorker struct {
	config       RetryWorkerConfig
	store        *db.Store
	emailService *email.Service

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	retried   atomic.Int64
	succeeded atomic.Int64
	exhausted atomic.Int64 // Permanently failed after max retries
}

// NewRetryWorker creates a new retry worker
func NewRetryWorker(store *db.Store, emailService *email.Service, config RetryWorkerConfig) *RetryWorker {
	ctx, cancel := context.WithCancel(context.Background())

	return &RetryWorker{
		config:       config,
		store:        store,
		emailService: emailService,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start begins the retry worker
func (w *RetryWorker) Start() {
	logx.Infof("Starting retry worker: max_retries=%d, intervals=%v",
		w.config.MaxRetries, w.config.RetryIntervals)

	w.wg.Add(1)
	go w.worker()
}

// Stop gracefully shuts down the retry worker
func (w *RetryWorker) Stop() {
	logx.Info("Stopping retry worker...")
	w.cancel()

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logx.Info("Retry worker stopped gracefully")
	case <-time.After(30 * time.Second):
		logx.Error("Retry worker shutdown timed out")
	}
}

// worker processes failed emails for retry
func (w *RetryWorker) worker() {
	defer w.wg.Done()
	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	logx.Info("Retry worker started")

	for {
		select {
		case <-w.ctx.Done():
			logx.Info("Retry worker stopped")
			return
		case <-ticker.C:
			w.processFailedSends()
		}
	}
}

// processFailedSends fetches and retries failed campaign sends
func (w *RetryWorker) processFailedSends() {
	sends, err := w.store.GetFailedCampaignSendsForRetry(w.ctx, int64(w.config.BatchSize))
	if err != nil {
		if err != sql.ErrNoRows {
			logx.Errorf("Failed to get failed sends for retry: %v", err)
		}
		return
	}

	for _, send := range sends {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		w.retrySend(send)
	}
}

// retrySend attempts to retry a failed send
func (w *RetryWorker) retrySend(send db.GetFailedCampaignSendsForRetryRow) {
	retryCount := send.RetryCount.Int64

	// Check if max retries exceeded
	if retryCount >= int64(w.config.MaxRetries) {
		w.markPermanentlyFailed(send.ID)
		w.exhausted.Add(1)
		return
	}

	// Check if enough time has passed for retry (exponential backoff)
	intervalIdx := int(retryCount)
	if intervalIdx >= len(w.config.RetryIntervals) {
		intervalIdx = len(w.config.RetryIntervals) - 1
	}
	requiredWait := w.config.RetryIntervals[intervalIdx]

	if send.FailedAt.Valid {
		failedTime, err := time.Parse("2006-01-02 15:04:05", send.FailedAt.String)
		if err == nil && time.Since(failedTime) < requiredWait {
			return // Not enough time has passed
		}
	}

	// Attempt retry
	logx.Infof("Retrying send %s (attempt %d/%d)", send.ID, retryCount+1, w.config.MaxRetries)

	// Increment retry count first
	w.store.IncrementCampaignSendRetry(w.ctx, send.ID)
	w.retried.Add(1)

	// Try to send
	err := w.sendEmail(send)
	if err != nil {
		logx.Errorf("Retry failed for send %s: %v", send.ID, err)
		w.store.MarkCampaignSendFailed(w.ctx, db.MarkCampaignSendFailedParams{
			ID:           send.ID,
			ErrorMessage: sql.NullString{String: err.Error(), Valid: true},
		})
	} else {
		logx.Infof("Retry succeeded for send %s", send.ID)
		w.store.MarkCampaignSendSent(w.ctx, send.ID)
		w.store.IncrementCampaignSent(w.ctx, send.CampaignID)
		w.succeeded.Add(1)
	}
}

// sendEmail sends the email for a retry
func (w *RetryWorker) sendEmail(send db.GetFailedCampaignSendsForRetryRow) error {
	fromName := ""
	if send.FromName.Valid {
		fromName = send.FromName.String
	}
	fromEmail := ""
	if send.FromEmail.Valid {
		fromEmail = send.FromEmail.String
	}
	replyTo := ""
	if send.ReplyTo.Valid {
		replyTo = send.ReplyTo.String
	}

	// Personalize and add the unsubscribe link and one-click headers, as on the
	// first attempt
	subject, htmlBody, sendOpts := w.emailService.PrepareCampaignEmail(
		send.Subject, send.HtmlBody, send.Name, send.Email, send.TrackingToken.String)

	return w.emailService.SendCampaignEmail(
		send.Email,
		subject,
		htmlBody,
		fromName,
		fromEmail,
		replyTo,
		sendOpts...,
	)
}

// markPermanentlyFailed marks a send as permanently failed
func (w *RetryWorker) markPermanentlyFailed(id string) {
	err := w.store.MarkCampaignSendPermanentlyFailed(w.ctx, id)
	if err != nil {
		logx.Errorf("Failed to mark send %s as permanently failed: %v", id, err)
	}
}

// Stats returns worker statistics
func (w *RetryWorker) Stats() (retried, succeeded, exhausted int64) {
	return w.retried.Load(), w.succeeded.Load(), w.exhausted.Load()
}

// StartRetryWorker starts the retry worker from service context
func StartRetryWorker(svcCtx *svc.ServiceContext) *RetryWorker {
	config := DefaultRetryWorkerConfig()

	worker := NewRetryWorker(svcCtx.DB, svcCtx.EmailService, config)
	worker.Start()

	return worker
}
