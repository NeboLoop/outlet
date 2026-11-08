package email

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outlet-sh/outlet/internal/db"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/time/rate"
)

// DispatcherConfig configures the email dispatcher
type DispatcherConfig struct {
	// Worker pool settings
	Workers int // Number of concurrent workers (default: 10)

	// Rate limiting (token bucket)
	RateLimit       float64 // Emails per second (default: 14 for SES)
	RateBurst       int     // Max burst size (default: 50)

	// Batch processing
	BatchSize       int           // Emails per batch fetch (default: 100)
	PollInterval    time.Duration // How often to poll for new emails (default: 5s)

	// Retry settings
	MaxRetries      int           // Max retry attempts (default: 3)
	InitialBackoff  time.Duration // Initial retry delay (default: 1s)
	MaxBackoff      time.Duration // Max retry delay (default: 30s)
	BackoffFactor   float64       // Backoff multiplier (default: 2.0)

	// Circuit breaker
	CircuitThreshold   int           // Failures before opening circuit (default: 10)
	CircuitTimeout     time.Duration // How long circuit stays open (default: 60s)
}

// DefaultDispatcherConfig returns sensible defaults for production
func DefaultDispatcherConfig() DispatcherConfig {
	return DispatcherConfig{
		Workers:            10,
		RateLimit:          14.0,    // SES default is 14/sec
		RateBurst:          50,
		BatchSize:          100,
		PollInterval:       5 * time.Second,
		MaxRetries:         3,
		InitialBackoff:     1 * time.Second,
		MaxBackoff:         30 * time.Second,
		BackoffFactor:      2.0,
		CircuitThreshold:   10,
		CircuitTimeout:     60 * time.Second,
	}
}

// EmailJob represents an email to be sent
type EmailJob struct {
	Email       db.GetPendingEmailsRow
	Attempt     int
	NextAttempt time.Time
}

// circuitState represents circuit breaker state
type circuitState int32

const (
	circuitClosed circuitState = iota
	circuitOpen
	circuitHalfOpen
)

// Dispatcher handles high-volume email sending with worker pools,
// rate limiting, retries, and circuit breaker
type Dispatcher struct {
	config          DispatcherConfig
	sequenceService *SequenceService
	db              *db.Store

	// Rate limiter
	limiter *rate.Limiter

	// Circuit breaker state
	circuit          atomic.Int32
	consecutiveFails atomic.Int32
	circuitOpenedAt  atomic.Int64

	// Worker pool
	jobs    chan EmailJob
	retries chan EmailJob

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	sent       atomic.Int64
	failed     atomic.Int64
	retried    atomic.Int64
	rateErrors atomic.Int64
}

// NewDispatcher creates a new email dispatcher
func NewDispatcher(sequenceService *SequenceService, db *db.Store, config DispatcherConfig) *Dispatcher {
	ctx, cancel := context.WithCancel(context.Background())

	d := &Dispatcher{
		config:          config,
		sequenceService: sequenceService,
		db:              db,
		limiter:         rate.NewLimiter(rate.Limit(config.RateLimit), config.RateBurst),
		jobs:            make(chan EmailJob, config.BatchSize*2),
		retries:         make(chan EmailJob, config.BatchSize),
		ctx:             ctx,
		cancel:          cancel,
	}

	return d
}

// Start begins the dispatcher with worker pool
func (d *Dispatcher) Start() {
	logx.Infof("Starting email dispatcher: %d workers, %.1f emails/sec, batch size %d",
		d.config.Workers, d.config.RateLimit, d.config.BatchSize)

	// Start workers
	for i := 0; i < d.config.Workers; i++ {
		d.wg.Add(1)
		go d.worker(i)
	}

	// Start retry processor
	d.wg.Add(1)
	go d.retryProcessor()

	// Start batch fetcher
	d.wg.Add(1)
	go d.batchFetcher()

	// Start metrics reporter
	d.wg.Add(1)
	go d.metricsReporter()
}

// Stop gracefully shuts down the dispatcher
func (d *Dispatcher) Stop() {
	logx.Info("Stopping email dispatcher...")
	// Cancelling stops every goroutine. The channels stay open: the fetcher
	// and retry processor may still be sending, and a send on a closed
	// channel panics.
	d.cancel()

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logx.Info("Email dispatcher stopped gracefully")
	case <-time.After(30 * time.Second):
		logx.Error("Email dispatcher shutdown timed out")
	}
}

// worker processes email jobs from the channel
func (d *Dispatcher) worker(id int) {
	defer d.wg.Done()
	logx.Infof("Email worker %d started", id)

	for {
		var job EmailJob
		select {
		case <-d.ctx.Done():
			logx.Infof("Email worker %d stopped", id)
			return
		case job = <-d.jobs:
		}

		// Check circuit breaker
		if d.isCircuitOpen() {
			// Put back in retry queue with delay
			job.NextAttempt = time.Now().Add(d.config.CircuitTimeout / 2)
			select {
			case d.retries <- job:
			default:
				// Retry queue full, mark as failed
				d.markFailed(job.Email, "circuit breaker open, retry queue full")
			}
			continue
		}

		// Wait for rate limiter
		if err := d.limiter.Wait(d.ctx); err != nil {
			if d.ctx.Err() != nil {
				return // Shutting down
			}
			d.rateErrors.Add(1)
			continue
		}

		// Send the email
		err := d.sendEmail(job)
		if err != nil {
			d.handleSendError(job, err)
		} else {
			d.handleSendSuccess(job)
		}
	}
}

// sendEmail processes and sends a single email
func (d *Dispatcher) sendEmail(job EmailJob) error {
	email := job.Email

	// Build template context
	tplCtx := TemplateContext{
		Name:          email.Name,
		Email:         email.Email,
		TrackingToken: email.TrackingToken.String,
		CustomFields:  make(map[string]string),
	}

	// Fetch custom fields if this is a sequence email with a contact
	if email.ContactID.Valid && email.TemplateID.Valid {
		template, err := d.db.GetTemplateByID(d.ctx, email.TemplateID.String)
		if err == nil && template.SequenceID.Valid {
			sequence, err := d.db.GetSequenceByID(d.ctx, template.SequenceID.String)
			if err == nil && sequence.ListID.Valid {
				tplCtx.CustomFields = d.sequenceService.GetCustomFieldsForContact(
					d.ctx, email.ContactID.String, sequence.ListID.Int64)
			}
		}
	}

	// Process template variables
	htmlBody := d.sequenceService.processTemplateVariables(email.HtmlBody, tplCtx, true)
	subject := d.sequenceService.processTemplateVariables(email.Subject, tplCtx, false)

	// Rewrite links for tracking
	if email.TrackingToken.Valid && email.TrackingToken.String != "" {
		htmlBody = d.sequenceService.rewriteLinksForTracking(htmlBody, email.TrackingToken.String)
	}

	// Apply template wrapping
	isTransactional := email.IsTransactional.Valid && email.IsTransactional.Int64 == 1
	templateType := "simple"
	if email.TemplateType.Valid {
		templateType = email.TemplateType.String
	}

	switch templateType {
	case "none":
		// Raw HTML
	case "branded":
		htmlBody = d.sequenceService.wrapWithBrandedTemplate(htmlBody, isTransactional, email.TrackingToken.String)
	case "simple":
		fallthrough
	default:
		htmlBody = d.sequenceService.wrapWithSimpleTemplate(htmlBody, isTransactional, email.TrackingToken.String)
	}

	// Send via SMTP
	return d.sequenceService.sender.Send(d.ctx, d.sequenceService.sequenceMessage(email, subject, htmlBody))
}

// handleSendSuccess processes a successful send
func (d *Dispatcher) handleSendSuccess(job EmailJob) {
	email := job.Email

	// Mark as sent in database
	if err := d.db.MarkEmailSent(d.ctx, email.ID); err != nil {
		logx.Errorf("Failed to mark email %s as sent: %v", email.ID, err)
	}

	// Reset circuit breaker on success
	d.consecutiveFails.Store(0)
	if d.circuit.Load() == int32(circuitHalfOpen) {
		d.circuit.Store(int32(circuitClosed))
		logx.Info("Circuit breaker closed - email sending recovered")
	}

	d.sent.Add(1)

	// Update sequence position and queue next email
	d.updateSequenceState(email)
}

// handleSendError processes a failed send
func (d *Dispatcher) handleSendError(job EmailJob, err error) {
	email := job.Email
	job.Attempt++

	// Increment consecutive failures for circuit breaker
	fails := d.consecutiveFails.Add(1)
	if fails >= int32(d.config.CircuitThreshold) && d.circuit.Load() == int32(circuitClosed) {
		d.circuit.Store(int32(circuitOpen))
		d.circuitOpenedAt.Store(time.Now().UnixNano())
		logx.Errorf("Circuit breaker OPENED after %d consecutive failures", fails)
	}

	// Check if we should retry
	if job.Attempt < d.config.MaxRetries {
		// Calculate backoff with jitter
		backoff := d.calculateBackoff(job.Attempt)
		job.NextAttempt = time.Now().Add(backoff)

		select {
		case d.retries <- job:
			d.retried.Add(1)
			logx.Infof("Email %s queued for retry %d/%d in %v: %v",
				email.ID, job.Attempt, d.config.MaxRetries, backoff, err)
		default:
			// Retry queue full
			d.markFailed(email, fmt.Sprintf("retry queue full after %d attempts: %v", job.Attempt, err))
		}
	} else {
		// Max retries exceeded
		d.markFailed(email, fmt.Sprintf("max retries (%d) exceeded: %v", d.config.MaxRetries, err))
	}
}

// markFailed marks an email as permanently failed
func (d *Dispatcher) markFailed(email db.GetPendingEmailsRow, errMsg string) {
	if err := d.db.MarkEmailFailed(d.ctx, db.MarkEmailFailedParams{
		ID:           email.ID,
		ErrorMessage: sql.NullString{String: errMsg, Valid: true},
	}); err != nil {
		logx.Errorf("Failed to mark email %s as failed: %v", email.ID, err)
	}
	d.failed.Add(1)
	logx.Errorf("Email %s to %s permanently failed: %s", email.ID, email.Email, errMsg)
}

// calculateBackoff returns the backoff duration with jitter
func (d *Dispatcher) calculateBackoff(attempt int) time.Duration {
	backoff := float64(d.config.InitialBackoff) * math.Pow(d.config.BackoffFactor, float64(attempt-1))
	if backoff > float64(d.config.MaxBackoff) {
		backoff = float64(d.config.MaxBackoff)
	}

	// Add 10-20% jitter to prevent thundering herd
	jitter := backoff * (0.1 + 0.1*float64(time.Now().UnixNano()%100)/100)
	return time.Duration(backoff + jitter)
}

// isCircuitOpen checks if the circuit breaker is open
func (d *Dispatcher) isCircuitOpen() bool {
	state := circuitState(d.circuit.Load())

	switch state {
	case circuitOpen:
		// Check if timeout has elapsed
		openedAt := time.Unix(0, d.circuitOpenedAt.Load())
		if time.Since(openedAt) > d.config.CircuitTimeout {
			// Try half-open
			d.circuit.Store(int32(circuitHalfOpen))
			logx.Info("Circuit breaker half-open - testing email delivery")
			return false
		}
		return true
	case circuitHalfOpen:
		// Allow one request through
		return false
	default:
		return false
	}
}

// retryProcessor handles the retry queue
func (d *Dispatcher) retryProcessor() {
	defer d.wg.Done()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var pendingRetries []EmailJob

	for {
		select {
		case <-d.ctx.Done():
			return
		case job, ok := <-d.retries:
			if !ok {
				return
			}
			pendingRetries = append(pendingRetries, job)
		case <-ticker.C:
			// Process due retries
			now := time.Now()
			var remaining []EmailJob
			for _, job := range pendingRetries {
				if job.NextAttempt.Before(now) || job.NextAttempt.Equal(now) {
					select {
					case d.jobs <- job:
					default:
						// Job queue full, keep for later
						remaining = append(remaining, job)
					}
				} else {
					remaining = append(remaining, job)
				}
			}
			pendingRetries = remaining
		}
	}
}

// batchFetcher periodically fetches pending emails from the database
func (d *Dispatcher) batchFetcher() {
	defer d.wg.Done()
	ticker := time.NewTicker(d.config.PollInterval)
	defer ticker.Stop()

	// Process immediately on startup
	d.fetchBatch()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.fetchBatch()
		}
	}
}

// fetchBatch loads pending emails and queues them for sending
func (d *Dispatcher) fetchBatch() {
	// Skip if circuit is fully open
	if circuitState(d.circuit.Load()) == circuitOpen {
		return
	}

	pendingEmails, err := claimDueEmails(d.ctx, d.db, int64(d.config.BatchSize))
	if err != nil {
		logx.Errorf("Failed to fetch pending emails: %v", err)
		return
	}

	if len(pendingEmails) == 0 {
		return
	}

	logx.Infof("Fetched %d pending emails", len(pendingEmails))

	for _, email := range pendingEmails {
		job := EmailJob{
			Email:   email,
			Attempt: 0,
		}
		select {
		case d.jobs <- job:
		case <-d.ctx.Done():
			return
		}
	}
}

// updateSequenceState advances the sequence after a successful send
func (d *Dispatcher) updateSequenceState(email db.GetPendingEmailsRow) {
	d.sequenceService.advanceSequence(d.ctx, email)
}

// metricsReporter periodically logs metrics
func (d *Dispatcher) metricsReporter() {
	defer d.wg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			sent := d.sent.Load()
			failed := d.failed.Load()
			retried := d.retried.Load()
			circuitState := d.circuit.Load()

			stateStr := "closed"
			if circuitState == int32(circuitOpen) {
				stateStr = "OPEN"
			} else if circuitState == int32(circuitHalfOpen) {
				stateStr = "half-open"
			}

			if sent > 0 || failed > 0 || retried > 0 {
				logx.Infof("Email dispatcher stats - sent: %d, failed: %d, retried: %d, circuit: %s",
					sent, failed, retried, stateStr)
			}
		}
	}
}

// Stats returns current dispatcher statistics
func (d *Dispatcher) Stats() (sent, failed, retried int64, isCircuitOpen bool) {
	return d.sent.Load(), d.failed.Load(), d.retried.Load(), d.circuit.Load() == int32(circuitOpen)
}
