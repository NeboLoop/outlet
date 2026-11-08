package validation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/services/emailval"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/google/uuid"
)

// Request limits.
const (
	MaxBatchEmails = 100   // addresses per synchronous batch call
	MaxJobEmails   = 10000 // addresses per job create/append call
)

// CreateJob creates a bulk validation job. With listID it queues the list's active
// subscribers and starts immediately; otherwise it queues emails (upload) and stays
// pending until more are appended and StartJob is called.
func (s *Service) CreateJob(ctx context.Context, orgID, name, listID string, emails []string) (*types.ValidationJobInfo, error) {
	params := db.CreateValidationJobParams{
		ID:     uuid.New().String(),
		OrgID:  orgID,
		Name:   strings.TrimSpace(name),
		Source: "upload",
		Status: "pending",
	}

	if listID != "" {
		if len(emails) > 0 {
			return nil, errorx.NewBadRequestError("provide list_id or emails, not both")
		}
		id, err := strconv.ParseInt(listID, 10, 64)
		if err != nil {
			return nil, errorx.NewBadRequestError("invalid list_id")
		}
		list, err := s.store.GetEmailList(ctx, id)
		if err != nil || list.OrgID != orgID {
			return nil, errorx.NewNotFoundError("list not found")
		}
		emails, err = s.store.ListActiveListSubscriberEmails(ctx, id)
		if err != nil {
			return nil, err
		}
		params.ListID = sql.NullInt64{Int64: id, Valid: true}
		params.Source = "list"
		if params.Name == "" {
			params.Name = list.Name
		}
	} else if len(emails) > MaxJobEmails {
		return nil, errorx.NewBadRequestError(fmt.Sprintf("at most %d emails per call; append the rest", MaxJobEmails))
	}
	if params.Name == "" {
		params.Name = "Upload " + time.Now().UTC().Format("2006-01-02 15:04")
	}

	job, err := s.store.CreateValidationJob(ctx, params)
	if err != nil {
		return nil, err
	}
	if err := s.queue(ctx, job, emails); err != nil {
		return nil, err
	}
	if job.Source == "list" {
		return s.StartJob(ctx, orgID, job.ID)
	}
	return s.Job(ctx, orgID, job.ID)
}

// AppendEmails queues more addresses on a pending upload job.
func (s *Service) AppendEmails(ctx context.Context, orgID, jobID string, emails []string) (*types.ValidationJobInfo, error) {
	if len(emails) > MaxJobEmails {
		return nil, errorx.NewBadRequestError(fmt.Sprintf("at most %d emails per call", MaxJobEmails))
	}
	job, err := s.job(ctx, orgID, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != "pending" {
		return nil, errorx.NewBadRequestError("emails can only be appended to a pending job")
	}
	if err := s.queue(ctx, job, emails); err != nil {
		return nil, err
	}
	return s.Job(ctx, orgID, jobID)
}

// StartJob hands a pending job to the worker.
func (s *Service) StartJob(ctx context.Context, orgID, jobID string) (*types.ValidationJobInfo, error) {
	n, err := s.store.StartValidationJob(ctx, db.StartValidationJobParams{ID: jobID, OrgID: orgID})
	if err != nil {
		return nil, err
	}
	if n == 0 {
		if _, err := s.job(ctx, orgID, jobID); err != nil {
			return nil, err
		}
		return nil, errorx.NewBadRequestError("only a pending job can be started")
	}
	return s.Job(ctx, orgID, jobID)
}

// CancelJob stops a pending or processing job. Addresses already checked keep their results.
func (s *Service) CancelJob(ctx context.Context, orgID, jobID string) (*types.ValidationJobInfo, error) {
	n, err := s.store.CancelValidationJob(ctx, db.CancelValidationJobParams{ID: jobID, OrgID: orgID})
	if err != nil {
		return nil, err
	}
	if n == 0 {
		if _, err := s.job(ctx, orgID, jobID); err != nil {
			return nil, err
		}
		return nil, errorx.NewBadRequestError("job has already finished")
	}
	return s.Job(ctx, orgID, jobID)
}

// Job returns a job with its per-status breakdown.
func (s *Service) Job(ctx context.Context, orgID, jobID string) (*types.ValidationJobInfo, error) {
	job, err := s.job(ctx, orgID, jobID)
	if err != nil {
		return nil, err
	}
	info, err := s.JobInfo(ctx, job)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// JobInfo maps a job row to the API type; counts come from its validation rows.
func (s *Service) JobInfo(ctx context.Context, job db.ValidationJob) (types.ValidationJobInfo, error) {
	info := types.ValidationJobInfo{
		Id:          job.ID,
		Name:        job.Name,
		Source:      job.Source,
		Status:      job.Status,
		Error:       job.Error.String,
		StartedAt:   job.StartedAt.String,
		CompletedAt: job.CompletedAt.String,
		CreatedAt:   job.CreatedAt.String,
	}
	if job.ListID.Valid {
		info.ListId = strconv.FormatInt(job.ListID.Int64, 10)
	}
	counts, err := s.store.CountValidationJobStatuses(ctx, sql.NullString{String: job.ID, Valid: true})
	if err != nil {
		return info, err
	}
	for _, c := range counts {
		n := int(c.Count)
		info.Total += n
		if c.Status != "pending" {
			info.Processed += n
		}
		switch c.Status {
		case emailval.StatusValid:
			info.Valid = n
		case emailval.StatusInvalid:
			info.Invalid = n
		case emailval.StatusCatchAll:
			info.CatchAll = n
		case emailval.StatusUnknown:
			info.Unknown = n
		case emailval.StatusSpamtrap:
			info.Spamtrap = n
		case emailval.StatusAbuse:
			info.Abuse = n
		case emailval.StatusDoNotMail:
			info.DoNotMail = n
		}
	}
	return info, nil
}

func (s *Service) job(ctx context.Context, orgID, jobID string) (db.ValidationJob, error) {
	job, err := s.store.GetValidationJob(ctx, db.GetValidationJobParams{ID: jobID, OrgID: orgID})
	if errors.Is(err, sql.ErrNoRows) {
		return job, errorx.NewNotFoundError("validation job not found")
	}
	return job, err
}

// queue inserts addresses as pending rows of the job (lowercased, blanks and duplicates skipped).
func (s *Service) queue(ctx context.Context, job db.ValidationJob, emails []string) error {
	emails = CleanEmails(emails)
	if len(emails) == 0 {
		return nil
	}
	jobID := sql.NullString{String: job.ID, Valid: true}
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		for _, email := range emails {
			if _, err := q.InsertValidationJobEmail(ctx, db.InsertValidationJobEmailParams{
				ID:     uuid.New().String(),
				OrgID:  job.OrgID,
				JobID:  jobID,
				Email:  email,
				Domain: domainOf(email),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return s.store.RefreshValidationJobProgress(ctx, job.ID)
}

// CleanEmails trims and lowercases addresses, dropping blanks and duplicates.
func CleanEmails(emails []string) []string {
	seen := make(map[string]bool, len(emails))
	out := make([]string, 0, len(emails))
	for _, e := range emails {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" || seen[e] {
			continue
		}
		seen[e] = true
		out = append(out, e)
	}
	return out
}

func domainOf(email string) string {
	if i := strings.LastIndex(email, "@"); i >= 0 {
		return email[i+1:]
	}
	return ""
}
