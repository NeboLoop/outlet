package mailpool

import (
	"context"
	"fmt"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/events"
)

// evaluate checks an active mailbox against the health lines after a send
// result, a bounce or a complaint, and pauses it when one is crossed. Only
// sends since its last resume count. The paused mailbox's queue stays put
// unless the pause is a provider fault.
func (s *Service) evaluate(ctx context.Context, mailboxID string) error {
	mb, err := s.store.GetPoolMailbox(ctx, mailboxID)
	if err != nil {
		return err
	}
	if mb.Status != "active" {
		return nil
	}
	now := s.now()
	kind, reason, err := s.unhealthy(ctx, mb, now)
	if err != nil || kind == "" {
		return err
	}
	return s.pause(ctx, mb, kind, reason, now)
}

func (s *Service) unhealthy(ctx context.Context, mb db.PoolMailbox, now time.Time) (kind, reason string, err error) {
	since := mb.HealthSince

	if n := s.cfg.ProviderFaultsToPause; n > 0 {
		kinds, err := s.store.RecentPoolSendFailureKinds(ctx, db.RecentPoolSendFailureKindsParams{
			MailboxID: mb.ID, Since: since, MaxCount: int64(n),
		})
		if err != nil {
			return "", "", err
		}
		faults := 0
		for _, k := range kinds {
			if k.String == "provider" {
				faults++
			}
		}
		if len(kinds) == n && faults == n {
			return "provider", fmt.Sprintf("its mail provider refused the connection, sign-in or sender on the last %d tries", n), nil
		}
	}

	b, err := s.store.PoolBounceWindow(ctx, db.PoolBounceWindowParams{
		MailboxID: mb.ID, Since: since, WindowSize: int64(s.cfg.BounceWindow),
	})
	if err != nil {
		return "", "", err
	}
	if int(b.Sends) >= s.cfg.BounceMinSample && b.Sends > 0 {
		if pct := 100 * float64(b.HardBounces) / float64(b.Sends); pct > s.cfg.BouncePausePct {
			return "list", fmt.Sprintf("hard bounces were %.1f%% of its last %d emails (the limit is %g%%)", pct, b.Sends, s.cfg.BouncePausePct), nil
		}
	}

	c, err := s.store.PoolComplaintWindow(ctx, db.PoolComplaintWindowParams{
		MailboxID: mb.ID, Since: since, WindowSize: int64(s.cfg.ComplaintWindow),
	})
	if err != nil {
		return "", "", err
	}
	if int(c.Complaints) >= s.cfg.ComplaintMinCount && c.Sends > 0 {
		if pct := 100 * float64(c.Complaints) / float64(c.Sends); pct > s.cfg.ComplaintPausePct {
			return "list", fmt.Sprintf("spam complaints were %.2f%% of its last %d emails (the limit is %g%%)", pct, c.Sends, s.cfg.ComplaintPausePct), nil
		}
	}

	if s.cfg.PolicyBlocksPerDay > 0 {
		from := now.Add(-24 * time.Hour)
		if hs := parseTime(since); hs.After(from) {
			from = hs
		}
		n, err := s.store.CountPoolPolicyBlocksSince(ctx, db.CountPoolPolicyBlocksSinceParams{
			MailboxID: mb.ID, Since: fmtTime(from),
		})
		if err != nil {
			return "", "", err
		}
		if int(n) >= s.cfg.PolicyBlocksPerDay {
			return "policy", fmt.Sprintf("receiving mail servers blocked %d of its emails for policy or reputation in the last day", n), nil
		}
	}
	return "", "", nil
}

// pause stops a mailbox, logs the pause, tells the tenant's brand, and
// pauses the domain when enough of its mailboxes are paused.
func (s *Service) pause(ctx context.Context, mb db.PoolMailbox, kind, reason string, now time.Time) error {
	n, err := s.store.PausePoolMailbox(ctx, db.PausePoolMailboxParams{
		PausedReason: nullString(reason), PausedKind: nullString(kind), ID: mb.ID,
	})
	if err != nil || n == 0 {
		return err
	}
	if err := s.store.CreatePoolPause(ctx, db.CreatePoolPauseParams{
		ID: newID(), MailboxID: mb.ID, TenantOrgID: mb.TenantOrgID, Tenant: mb.Tenant,
		Kind: kind, Reason: reason, CreatedAt: fmtTime(now),
	}); err != nil {
		return err
	}
	held := false
	if mb.Tenant.Valid && (kind == "list" || kind == "policy") {
		if held, err = s.tenantHeld(ctx, mb.TenantOrgID.String, mb.Tenant.String, now); err != nil {
			return err
		}
	}
	s.emit(events.TopicPoolMailboxPaused, events.PoolEvent{
		OrgID: mb.TenantOrgID.String, Tenant: mb.Tenant.String, MailboxID: mb.ID,
		Reason: reason, PauseKind: kind, Movable: kind == "provider", TenantHeld: held,
	})
	return s.checkDomain(ctx, mb.DomainID)
}

// checkDomain pauses a domain once DomainPauseShare of its mailboxes are
// paused, and tells each tenant whose mailbox there stops with it.
func (s *Service) checkDomain(ctx context.Context, domainID string) error {
	counts, err := s.store.CountPoolDomainMailboxes(ctx, domainID)
	if err != nil || counts.Total == 0 {
		return err
	}
	if float64(counts.Paused)/float64(counts.Total) < s.cfg.DomainPauseShare {
		return nil
	}
	reason := fmt.Sprintf("%d of its %d mailboxes are paused", counts.Paused, counts.Total)
	n, err := s.store.PausePoolDomain(ctx, db.PausePoolDomainParams{PausedReason: nullString(reason), ID: domainID})
	if err != nil || n == 0 {
		return err
	}
	return s.announceDomain(ctx, domainID, events.TopicPoolMailboxPaused, "its domain is paused: "+reason)
}

// announceDomain tells the tenant of each active mailbox on the domain that
// it stopped or sends again with its domain.
func (s *Service) announceDomain(ctx context.Context, domainID, topic, reason string) error {
	mbs, err := s.store.ListPoolMailboxesByDomain(ctx, domainID)
	if err != nil {
		return err
	}
	for _, mb := range mbs {
		if mb.Status != "active" || !mb.Tenant.Valid {
			continue
		}
		ev := events.PoolEvent{OrgID: mb.TenantOrgID.String, Tenant: mb.Tenant.String, MailboxID: mb.ID, Reason: reason}
		if topic == events.TopicPoolMailboxPaused {
			ev.PauseKind = "domain"
		}
		s.emit(topic, ev)
	}
	return nil
}
