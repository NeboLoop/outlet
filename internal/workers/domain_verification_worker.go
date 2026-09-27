package workers

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/svc"
)

// DomainVerificationWorker periodically checks pending domain verifications
type DomainVerificationWorker struct {
	svcCtx   *svc.ServiceContext
	interval time.Duration
	stop     chan struct{}
	wg       sync.WaitGroup
}

// NewDomainVerificationWorker creates a new domain verification worker
func NewDomainVerificationWorker(svcCtx *svc.ServiceContext, interval time.Duration) *DomainVerificationWorker {
	return &DomainVerificationWorker{
		svcCtx:   svcCtx,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

// Start starts the domain verification worker
func (w *DomainVerificationWorker) Start() {
	w.wg.Add(1)
	go w.run()
}

// Stop stops the domain verification worker
func (w *DomainVerificationWorker) Stop() {
	close(w.stop)
	w.wg.Wait()
}

func (w *DomainVerificationWorker) run() {
	defer w.wg.Done()

	// Run immediately on start
	w.checkPendingDomains()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.checkPendingDomains()
		case <-w.stop:
			log.Println("Domain verification worker stopping...")
			return
		}
	}
}

func (w *DomainVerificationWorker) checkPendingDomains() {
	ctx := context.Background()

	// Get all domain identities with pending status
	identities, err := w.svcCtx.DB.ListPendingDomainIdentities(ctx)
	if err != nil {
		log.Printf("Failed to list pending domain identities: %v", err)
		return
	}

	if len(identities) == 0 {
		return
	}

	log.Printf("Checking %d pending domain identities...", len(identities))

	for _, identity := range identities {
		w.checkDomainStatus(ctx, identity)
	}
}

func (w *DomainVerificationWorker) checkDomainStatus(ctx context.Context, identity db.DomainIdentity) {
	// Get AWS credentials
	region, accessKey, secretKey, err := email.AWSCredentials(ctx, w.svcCtx.DB, w.svcCtx.CryptoService, identity.OrgID)
	if err != nil {
		log.Printf("Failed to get AWS credentials for org %s: %v", identity.OrgID, err)
		return
	}

	// Get current status from AWS
	status, err := email.GetDomainIdentityStatus(ctx, region, accessKey, secretKey, identity.Domain)
	if err != nil {
		log.Printf("Failed to check domain status for %s: %v", identity.Domain, err)
		return
	}

	// Check if status changed
	statusChanged := status.VerificationStatus != identity.VerificationStatus.String ||
		status.DKIMStatus != identity.DkimStatus.String

	if !statusChanged {
		return
	}

	log.Printf("Domain %s status changed: verification=%s->%s, dkim=%s->%s",
		identity.Domain,
		identity.VerificationStatus.String, status.VerificationStatus,
		identity.DkimStatus.String, status.DKIMStatus)

	// Update the database
	updated, err := w.svcCtx.DB.UpdateDomainIdentityStatus(ctx, db.UpdateDomainIdentityStatusParams{
		ID:                 identity.ID,
		VerificationStatus: sql.NullString{String: status.VerificationStatus, Valid: true},
		DkimStatus:         sql.NullString{String: status.DKIMStatus, Valid: true},
	})
	if err != nil {
		log.Printf("Failed to update domain identity status: %v", err)
		return
	}

	// Broadcast update via WebSocket
	if w.svcCtx.WebSocketHub != nil {
		w.svcCtx.WebSocketHub.BroadcastDomainIdentityUpdate(
			updated.ID,
			updated.OrgID,
			updated.Domain,
			updated.VerificationStatus.String,
			updated.DkimStatus.String,
			updated.MailFromStatus.String,
			updated.LastCheckedAt.String,
		)
	}
}

// StartDomainVerificationWorker starts the domain verification worker with a 1-minute interval
func StartDomainVerificationWorker(svcCtx *svc.ServiceContext) *DomainVerificationWorker {
	worker := NewDomainVerificationWorker(svcCtx, 1*time.Minute)
	worker.Start()
	return worker
}
