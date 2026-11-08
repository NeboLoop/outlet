package workers

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

// CampaignSchedulerConfig configures the campaign scheduler
type CampaignSchedulerConfig struct {
	// How often to check for scheduled campaigns
	ScheduleInterval time.Duration

	// How often to poll for pending sends
	SendPollInterval time.Duration

	// Worker pool settings
	Workers int

	// Rate limiting (emails per second)
	RateLimit int

	// Batch processing - larger = fewer DB round trips
	BatchSize int

	// Error threshold - pause campaign after this many consecutive errors
	ErrorThreshold int

	// SMTP connection pool size
	PoolSize int
}

// DefaultCampaignSchedulerConfig returns sensible defaults
func DefaultCampaignSchedulerConfig() CampaignSchedulerConfig {
	return CampaignSchedulerConfig{
		ScheduleInterval: 10 * time.Second,
		SendPollInterval: 2 * time.Second,
		Workers:          10,   // Increased from 5
		RateLimit:        14,   // SES sandbox default
		BatchSize:        1000, // Increased from 100
		ErrorThreshold:   100,  // Pause after 100 consecutive errors
		PoolSize:         20,   // SMTP connection pool size
	}
}

// CampaignPipe manages state for a single campaign's sending
type CampaignPipe struct {
	CampaignID string
	mu         sync.RWMutex
	wg         sync.WaitGroup

	// Counters
	sent   atomic.Int64
	errors atomic.Int64

	// State
	stopped      atomic.Bool
	paused       atomic.Bool
	errorsPaused atomic.Bool // Paused due to error threshold

	// Rate tracking
	lastMinuteSent atomic.Int64
	lastMinuteTime time.Time
}

// NewCampaignPipe creates a new campaign pipe
func NewCampaignPipe(campaignID string) *CampaignPipe {
	return &CampaignPipe{
		CampaignID:     campaignID,
		lastMinuteTime: time.Now(),
	}
}

// RecordSent records a successful send
func (p *CampaignPipe) RecordSent() {
	p.sent.Add(1)
	p.lastMinuteSent.Add(1)
	p.errors.Store(0) // Reset consecutive errors on success
}

// RecordError records a failed send
func (p *CampaignPipe) RecordError() int64 {
	return p.errors.Add(1)
}

// ShouldPause checks if campaign should be paused due to errors
func (p *CampaignPipe) ShouldPause(threshold int) bool {
	return p.errors.Load() >= int64(threshold)
}

// Stats returns pipe statistics
func (p *CampaignPipe) Stats() (sent, errors int64, rate float64) {
	sent = p.sent.Load()
	errors = p.errors.Load()

	// Calculate rate (messages per minute)
	elapsed := time.Since(p.lastMinuteTime)
	if elapsed >= time.Minute {
		rate = float64(p.lastMinuteSent.Load())
		p.lastMinuteSent.Store(0)
		p.lastMinuteTime = time.Now()
	} else if elapsed > 0 {
		rate = float64(p.lastMinuteSent.Load()) * float64(time.Minute) / float64(elapsed)
	}

	return sent, errors, rate
}

// CampaignScheduler handles scheduled campaigns and sending
type CampaignScheduler struct {
	config       CampaignSchedulerConfig
	store        *db.Store
	emailService *email.Service

	// Rate limiter (sliding window)
	limiter *email.CampaignRateLimiter

	// Campaign pipes
	pipes   map[string]*CampaignPipe
	pipesMu sync.RWMutex

	// Message queue for workers
	msgQueue chan db.GetPendingCampaignSendsRow

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Global metrics
	totalScheduled atomic.Int64
	totalSent      atomic.Int64
	totalFailed    atomic.Int64
}

// NewCampaignScheduler creates a new campaign scheduler
func NewCampaignScheduler(store *db.Store, emailService *email.Service, config CampaignSchedulerConfig) *CampaignScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &CampaignScheduler{
		config:       config,
		store:        store,
		emailService: emailService,
		limiter:      email.NewCampaignRateLimiter(config.RateLimit),
		pipes:        make(map[string]*CampaignPipe),
		msgQueue:     make(chan db.GetPendingCampaignSendsRow, config.BatchSize),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start begins the campaign scheduler
func (s *CampaignScheduler) Start() {
	logx.Infof("Starting campaign scheduler: %d workers, %d emails/sec, batch=%d, pool=%d",
		s.config.Workers, s.config.RateLimit, s.config.BatchSize, s.config.PoolSize)

	// Enable SMTP connection pooling (loads SMTP config from platform_settings)
	if err := s.emailService.EnablePool(s.ctx, s.config.PoolSize); err != nil {
		logx.Errorf("Warning: Failed to enable SMTP connection pooling: %v", err)
	}

	// Start schedule checker
	s.wg.Add(1)
	go s.scheduleChecker()

	// Start batch fetcher
	s.wg.Add(1)
	go s.batchFetcher()

	// Start send workers
	for i := 0; i < s.config.Workers; i++ {
		s.wg.Add(1)
		go s.sendWorker(i)
	}

	// Start metrics reporter
	s.wg.Add(1)
	go s.metricsReporter()

	// Start pipe cleanup
	s.wg.Add(1)
	go s.pipeCleanup()
}

// Stop gracefully shuts down the scheduler
func (s *CampaignScheduler) Stop() {
	logx.Info("Stopping campaign scheduler...")
	// Cancelling stops every goroutine; the queue stays open because the
	// batch fetcher may still be sending to it
	s.cancel()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logx.Info("Campaign scheduler stopped gracefully")
	case <-time.After(30 * time.Second):
		logx.Error("Campaign scheduler shutdown timed out")
	}

	// Close SMTP pool
	s.emailService.ClosePool()
}

// scheduleChecker polls for scheduled campaigns due to send
func (s *CampaignScheduler) scheduleChecker() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.config.ScheduleInterval)
	defer ticker.Stop()

	// Check immediately on startup
	s.checkScheduledCampaigns()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkScheduledCampaigns()
		}
	}
}

// checkScheduledCampaigns finds due campaigns and queues them for sending
func (s *CampaignScheduler) checkScheduledCampaigns() {
	campaigns, err := s.store.GetScheduledCampaigns(s.ctx)
	if err != nil {
		logx.Errorf("Failed to get scheduled campaigns: %v", err)
		return
	}

	for _, campaign := range campaigns {
		if err := s.processCampaign(campaign); err != nil {
			logx.Errorf("Failed to process campaign %s: %v", campaign.ID, err)
		}
	}
}

// processCampaign queues sends for a campaign
func (s *CampaignScheduler) processCampaign(campaign db.EmailCampaign) error {
	logx.Infof("Processing campaign %s: %s", campaign.ID, campaign.Name)

	// A campaign sends only from a domain of its brand's marketing pathway.
	// One that can't send is paused (the admin shows it as an error);
	// "failed" is not a campaign status.
	if err := email.CheckSender(s.ctx, s.store, campaign.OrgID, email.KindMarketing, campaign.FromEmail.String); err != nil {
		logx.Errorf("Campaign %s can't send: %v", campaign.ID, err)
		return s.store.UpdateCampaignStatusByID(s.ctx, db.UpdateCampaignStatusByIDParams{
			ID:     campaign.ID,
			Status: sql.NullString{String: "paused", Valid: true},
		})
	}

	// Create pipe for this campaign
	s.getOrCreatePipe(campaign.ID)

	// The campaign stays 'scheduled' while its sends are created and only then
	// becomes 'sending'. Workers claim sends of 'sending' campaigns only, so
	// none can finish (and mark the campaign sent) while rows are still being
	// added. If this stops halfway, the next check resumes it; existing sends
	// are skipped.
	listIDs := db.DecodeListIDs(campaign.ListIds.String)
	if len(listIDs) == 0 {
		logx.Errorf("Campaign %s has no target lists", campaign.ID)
		// 'failed' is not an allowed campaign status; cancel it instead
		return s.store.UpdateCampaignStatusByID(s.ctx, db.UpdateCampaignStatusByIDParams{
			ID:     campaign.ID,
			Status: sql.NullString{String: "cancelled", Valid: true},
		})
	}

	// Collect all unique subscribers from target lists
	subscriberMap := make(map[string]db.GetActiveSubscribersForListRow)

	for _, listID := range listIDs {
		subscribers, err := s.store.GetActiveSubscribersForList(s.ctx, db.GetActiveSubscribersForListParams{
			OrgID:  campaign.OrgID,
			ListID: listID,
		})
		if err != nil {
			logx.Errorf("Failed to get subscribers for list %d: %v", listID, err)
			continue
		}

		for _, sub := range subscribers {
			// Dedupe by contact ID
			if _, exists := subscriberMap[sub.ContactID]; !exists {
				subscriberMap[sub.ContactID] = sub
			}
		}
	}

	// Remove the contacts of the excluded lists
	excludeListIDs := db.DecodeListIDs(campaign.ExcludeListIds.String)
	for _, excludeListID := range excludeListIDs {
		excludedSubs, err := s.store.GetActiveSubscribersForList(s.ctx, db.GetActiveSubscribersForListParams{
			OrgID:  campaign.OrgID,
			ListID: excludeListID,
		})
		if err != nil {
			logx.Errorf("Failed to get excluded subscribers for list %d: %v", excludeListID, err)
			continue
		}
		for _, sub := range excludedSubs {
			delete(subscriberMap, sub.ContactID)
		}
	}

	// Create campaign_sends for each subscriber
	var recipientCount int64
	for _, sub := range subscriberMap {
		// Check if send already exists (idempotency)
		exists, err := s.store.CheckCampaignSendExists(s.ctx, db.CheckCampaignSendExistsParams{
			CampaignID: campaign.ID,
			ContactID:  sub.ContactID,
		})
		if err != nil {
			logx.Errorf("Failed to check existing send: %v", err)
			continue
		}
		if exists > 0 {
			recipientCount++ // created by an earlier, interrupted run
			continue
		}

		// Create campaign send
		trackingToken := uuid.NewString()
		_, err = s.store.CreateCampaignSend(s.ctx, db.CreateCampaignSendParams{
			ID:            uuid.NewString(),
			CampaignID:    campaign.ID,
			ContactID:     sub.ContactID,
			ListID:        sql.NullInt64{Int64: sub.ListID, Valid: true},
			TrackingToken: sql.NullString{String: trackingToken, Valid: true},
		})
		if err != nil {
			logx.Errorf("Failed to create campaign send: %v", err)
			continue
		}
		recipientCount++
	}

	// Update recipient count
	err := s.store.SetCampaignRecipientsCount(s.ctx, db.SetCampaignRecipientsCountParams{
		ID:              campaign.ID,
		RecipientsCount: sql.NullInt64{Int64: recipientCount, Valid: true},
	})
	if err != nil {
		logx.Errorf("Failed to update recipient count: %v", err)
	}

	s.totalScheduled.Add(1)
	logx.Infof("Campaign %s queued with %d recipients", campaign.ID, recipientCount)

	// Release it to the workers; with nothing to send it is finished at once
	status := "sending"
	if recipientCount == 0 {
		status = "sent"
	}
	return s.store.UpdateCampaignStatusByID(s.ctx, db.UpdateCampaignStatusByIDParams{
		ID:     campaign.ID,
		Status: sql.NullString{String: status, Valid: true},
	})
}

// batchFetcher continuously fetches pending sends and queues them
func (s *CampaignScheduler) batchFetcher() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.config.SendPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.fetchAndQueueSends()
		}
	}
}

// fetchAndQueueSends fetches pending sends and adds them to the worker queue
func (s *CampaignScheduler) fetchAndQueueSends() {
	// Claim the batch first, so polling again while these wait for the rate
	// limiter cannot queue them twice
	claim := sql.NullString{String: uuid.NewString(), Valid: true}
	if err := s.store.ClaimPendingCampaignSends(s.ctx, db.ClaimPendingCampaignSendsParams{
		ClaimToken: claim,
		LimitCount: int64(s.config.BatchSize),
	}); err != nil {
		logx.Errorf("Failed to claim pending campaign sends: %v", err)
		return
	}
	sends, err := s.store.GetPendingCampaignSends(s.ctx, claim)
	if err != nil {
		logx.Errorf("Failed to get pending campaign sends: %v", err)
		return
	}

	for _, send := range sends {
		// Check if campaign is paused
		pipe := s.getPipe(send.CampaignID)
		if pipe != nil && (pipe.paused.Load() || pipe.errorsPaused.Load()) {
			continue
		}

		select {
		case <-s.ctx.Done():
			return
		case s.msgQueue <- send:
			// Queued successfully
		}
	}
}

// sendWorker processes messages from the queue
func (s *CampaignScheduler) sendWorker(id int) {
	defer s.wg.Done()
	logx.Infof("Campaign send worker %d started", id)

	for {
		var send db.GetPendingCampaignSendsRow
		select {
		case <-s.ctx.Done():
			logx.Infof("Campaign send worker %d stopped", id)
			return
		case send = <-s.msgQueue:
		}

		// Get or create pipe for this campaign
		pipe := s.getOrCreatePipe(send.CampaignID)

		// Check if paused
		if pipe.paused.Load() || pipe.errorsPaused.Load() {
			continue
		}

		// Rate limit (sliding window)
		if err := s.limiter.WaitGlobal(s.ctx); err != nil {
			if s.ctx.Err() != nil {
				return
			}
			continue
		}

		// Send the email. One held back by its pathway's limit waits, still
		// pending, until the limit lifts; it never fails or pauses the campaign.
		subject, err := s.sendCampaignEmail(send)
		if deferCampaignSend(s.ctx, s.store, send.ID, err) {
			continue
		} else if errors.Is(err, email.ErrSuppressed) || errors.Is(err, email.ErrInvalidMessage) {
			// The recipient can never be sent to: record it, and don't count
			// it toward auto-pausing the campaign
			s.markSendFailed(send.ID, err.Error())
		} else if err != nil {
			s.markSendFailed(send.ID, err.Error())
			s.totalFailed.Add(1)

			// Record error and check threshold
			errCount := pipe.RecordError()
			if errCount >= int64(s.config.ErrorThreshold) {
				logx.Errorf("Campaign %s auto-paused: %d consecutive errors", send.CampaignID, errCount)
				pipe.errorsPaused.Store(true)
				s.pauseCampaign(send.CampaignID, "Too many consecutive errors")
			}
		} else {
			s.markSendSent(send.ID, subject)
			s.totalSent.Add(1)
			pipe.RecordSent()

			// Increment campaign sent count
			_ = s.store.IncrementCampaignSent(s.ctx, send.CampaignID)
		}

		// Check if campaign is complete
		s.checkCampaignComplete(send.CampaignID)
	}
}

// sendCampaignEmail sends a single campaign email and returns the subject it
// went with, its merge tags filled
func (s *CampaignScheduler) sendCampaignEmail(send db.GetPendingCampaignSendsRow) (string, error) {
	msg := s.emailService.CampaignMessage(email.CampaignEmail{
		OrgID:         send.OrgID,
		FromName:      send.FromName.String,
		FromEmail:     send.FromEmail.String,
		ReplyTo:       send.ReplyTo.String,
		To:            send.Email,
		Name:          send.Name,
		Subject:       send.Subject,
		HTML:          send.HtmlBody,
		Text:          send.PlainText.String,
		DesignHTML:    send.DesignHtml,
		DesignText:    send.DesignText,
		TrackingToken: send.TrackingToken.String,
		TrackOpens:    send.TrackOpens.Valid && send.TrackOpens.Int64 == 1,
		TrackClicks:   send.TrackClicks.Valid && send.TrackClicks.Int64 == 1,
	})
	return msg.Subject, s.emailService.Send(s.ctx, msg)
}

// deferCampaignSend leaves a campaign send held back by a temporary failure
// (its pathway's limit, a 4xx, a dropped connection, SES throttling) pending
// and unclaimed, to go again at RetryAt, or in 15 minutes. It is the one way
// a campaign send is tried again; a send that failed for good, or that the
// mail server kept refusing for GiveUpAfter, is marked failed and never
// retried. It reports whether it deferred.
func deferCampaignSend(ctx context.Context, store *db.Store, id string, err error) bool {
	if err == nil || !email.IsTemporary(err) {
		return false
	}
	now := time.Now()
	refused := email.RefusedAt(err, now)
	if refused.Valid {
		if since, _ := store.CampaignSendRefusedSince(ctx, id); email.GaveUp(since, now) {
			logx.Infof("Send %s: refused for %s, giving up: %v", id, email.GiveUpAfter, err)
			return false
		}
	}
	at := email.RetryAt(err, now.Add(15*time.Minute))
	if derr := store.DeferCampaignSend(ctx, db.DeferCampaignSendParams{SendAfter: sql.NullString{String: at.UTC().Format(time.DateTime), Valid: true}, RefusedAt: refused, ID: id}); derr != nil {
		logx.Errorf("Failed to defer send %s: %v", id, derr)
	}
	return true
}

// markSendSent marks a campaign send as sent with the subject it went with
func (s *CampaignScheduler) markSendSent(id, subject string) {
	if err := s.store.MarkCampaignSendSent(s.ctx, db.MarkCampaignSendSentParams{ID: id, Subject: sql.NullString{String: subject, Valid: true}}); err != nil {
		logx.Errorf("Failed to mark send %s as sent: %v", id, err)
	}
}

// markSendFailed marks a campaign send as failed
func (s *CampaignScheduler) markSendFailed(id, errMsg string) {
	if err := s.store.MarkCampaignSendFailed(s.ctx, db.MarkCampaignSendFailedParams{
		ID:           id,
		ErrorMessage: sql.NullString{String: errMsg, Valid: true},
	}); err != nil {
		logx.Errorf("Failed to mark send %s as failed: %v", id, err)
	}
}

// checkCampaignComplete checks if all sends are done and updates campaign status
func (s *CampaignScheduler) checkCampaignComplete(campaignID string) {
	pending, err := s.store.CountPendingCampaignSends(s.ctx, campaignID)
	if err != nil {
		return
	}

	if pending == 0 {
		err := s.store.UpdateCampaignStatusByID(s.ctx, db.UpdateCampaignStatusByIDParams{
			ID:     campaignID,
			Status: sql.NullString{String: "sent", Valid: true},
		})
		if err != nil {
			logx.Errorf("Failed to mark campaign %s as sent: %v", campaignID, err)
		} else {
			logx.Infof("Campaign %s completed", campaignID)
			// Clean up pipe
			s.removePipe(campaignID)
		}
	}
}

// pauseCampaign pauses a campaign due to errors
func (s *CampaignScheduler) pauseCampaign(campaignID, reason string) {
	err := s.store.UpdateCampaignStatusByID(s.ctx, db.UpdateCampaignStatusByIDParams{
		ID:     campaignID,
		Status: sql.NullString{String: "paused", Valid: true},
	})
	if err != nil {
		logx.Errorf("Failed to pause campaign %s: %v", campaignID, err)
	}
}

// getOrCreatePipe gets or creates a pipe for a campaign
func (s *CampaignScheduler) getOrCreatePipe(campaignID string) *CampaignPipe {
	s.pipesMu.Lock()
	defer s.pipesMu.Unlock()

	if pipe, ok := s.pipes[campaignID]; ok {
		return pipe
	}

	pipe := NewCampaignPipe(campaignID)
	s.pipes[campaignID] = pipe
	return pipe
}

// getPipe gets a pipe for a campaign (or nil if not exists)
func (s *CampaignScheduler) getPipe(campaignID string) *CampaignPipe {
	s.pipesMu.RLock()
	defer s.pipesMu.RUnlock()
	return s.pipes[campaignID]
}

// removePipe removes a pipe for a completed campaign
func (s *CampaignScheduler) removePipe(campaignID string) {
	s.pipesMu.Lock()
	defer s.pipesMu.Unlock()
	delete(s.pipes, campaignID)
}

// pipeCleanup periodically cleans up stale pipes
func (s *CampaignScheduler) pipeCleanup() {
	defer s.wg.Done()
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// Clean up pipes for completed campaigns
			s.pipesMu.Lock()
			for id := range s.pipes {
				pending, err := s.store.CountPendingCampaignSends(s.ctx, id)
				if err == nil && pending == 0 {
					delete(s.pipes, id)
				}
			}
			s.pipesMu.Unlock()
		}
	}
}

// metricsReporter periodically logs metrics
func (s *CampaignScheduler) metricsReporter() {
	defer s.wg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			scheduled := s.totalScheduled.Load()
			sent := s.totalSent.Load()
			failed := s.totalFailed.Load()
			pooled, created := s.emailService.PoolStats()
			current, max := s.limiter.GlobalStats()

			if scheduled > 0 || sent > 0 || failed > 0 {
				logx.Infof("Campaign stats - campaigns: %d, sent: %d, failed: %d, pool: %d/%d, rate: %d/%d",
					scheduled, sent, failed, pooled, created, current, max)
			}

			// Log per-campaign stats
			s.pipesMu.RLock()
			for id, pipe := range s.pipes {
				pSent, pErrors, rate := pipe.Stats()
				if pSent > 0 || pErrors > 0 {
					logx.Debugf("Campaign %s: sent=%d, errors=%d, rate=%.1f/min",
						id, pSent, pErrors, rate)
				}
			}
			s.pipesMu.RUnlock()
		}
	}
}

// Stats returns current scheduler statistics
func (s *CampaignScheduler) Stats() (scheduled, sent, failed int64) {
	return s.totalScheduled.Load(), s.totalSent.Load(), s.totalFailed.Load()
}

// SetRateLimit dynamically updates the rate limit
func (s *CampaignScheduler) SetRateLimit(rate int) {
	s.limiter.SetGlobalRate(rate)
	logx.Infof("Campaign scheduler rate limit updated to %d/sec", rate)
}

// StartCampaignScheduler starts the campaign scheduler from service context
func StartCampaignScheduler(svcCtx *svc.ServiceContext) *CampaignScheduler {
	config := DefaultCampaignSchedulerConfig()

	// Override with config values if present
	if svcCtx.Config.Email.WorkerCount > 0 {
		config.Workers = svcCtx.Config.Email.WorkerCount
	}
	if svcCtx.Config.Email.RateLimit > 0 {
		config.RateLimit = int(svcCtx.Config.Email.RateLimit)
	}
	if svcCtx.Config.Email.RateBurst > 0 {
		config.PoolSize = svcCtx.Config.Email.RateBurst // Use burst as pool size hint
	}

	scheduler := NewCampaignScheduler(svcCtx.DB, svcCtx.EmailService, config)
	scheduler.Start()

	return scheduler
}
