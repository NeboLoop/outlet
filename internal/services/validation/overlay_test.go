package validation

import (
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/services/emailval"
	"github.com/outlet-sh/outlet/internal/services/signalnet"
)

func TestOverlay(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	valid := emailval.Result{Status: emailval.StatusValid, SyntaxOK: true}
	invalid := emailval.Result{Status: emailval.StatusInvalid, SubStatus: emailval.SubMailboxNotFound}
	catchAll := emailval.Result{Status: emailval.StatusCatchAll, CatchAll: true}
	roleCatchAll := emailval.Result{Status: emailval.StatusDoNotMail, SubStatus: emailval.SubRoleBasedCatchAll, RoleBased: true, CatchAll: true}
	goodDomain := LocalSignals{DomainDelivered: 100, DomainBounced: 1}

	tests := []struct {
		name        string
		res         emailval.Result
		local       LocalSignals
		net         *signalnet.Signal
		status, sub string
	}{
		{"engine verdict passes through", valid, LocalSignals{}, nil, emailval.StatusValid, ""},
		{"local complaint beats invalid", invalid, LocalSignals{Complaint: true}, nil, emailval.StatusAbuse, ""},
		{"network complaint", valid, LocalSignals{}, &signalnet.Signal{Complaints: 1}, emailval.StatusAbuse, ""},
		{"abuse beats spamtrap", valid, LocalSignals{Complaint: true}, &signalnet.Signal{TrapScore: 0.9}, emailval.StatusAbuse, ""},
		{"spamtrap", valid, LocalSignals{}, &signalnet.Signal{TrapScore: 0.8}, emailval.StatusSpamtrap, ""},
		{"trap score below threshold", valid, LocalSignals{}, &signalnet.Signal{TrapScore: 0.79}, emailval.StatusValid, ""},
		{"suppression does not override invalid", invalid, LocalSignals{Suppressed: true}, nil, emailval.StatusInvalid, emailval.SubMailboxNotFound},
		{"global suppression", valid, LocalSignals{Suppressed: true}, nil, emailval.StatusDoNotMail, emailval.SubGlobalSuppression},
		{"suppression beats toxic", valid, LocalSignals{Suppressed: true}, &signalnet.Signal{Toxic: true}, emailval.StatusDoNotMail, emailval.SubGlobalSuppression},
		{"toxic", valid, LocalSignals{}, &signalnet.Signal{Toxic: true}, emailval.StatusDoNotMail, emailval.SubToxic},
		{"network hard bounces", valid, LocalSignals{}, &signalnet.Signal{HardBounces: 2}, emailval.StatusInvalid, emailval.SubMailboxNotFound},
		{"network hard bounces with deliveries", valid, LocalSignals{}, &signalnet.Signal{HardBounces: 2, Delivered: 1}, emailval.StatusValid, ""},
		{"catch-all promoted", catchAll, goodDomain, nil, emailval.StatusValid, emailval.SubAcceptAll},
		{"role catch-all promoted", roleCatchAll, goodDomain, nil, emailval.StatusValid, emailval.SubRoleBasedAcceptAll},
		{"catch-all thin history", catchAll, LocalSignals{DomainDelivered: 49}, nil, emailval.StatusCatchAll, ""},
		{"catch-all high bounce rate", catchAll, LocalSignals{DomainDelivered: 98, DomainBounced: 2}, nil, emailval.StatusCatchAll, ""},
		{"gold from local engagement", valid, LocalSignals{LastEngagedAt: now.Add(-89 * 24 * time.Hour)}, nil, emailval.StatusValid, emailval.SubGold},
		{"stale engagement is not gold", valid, LocalSignals{LastEngagedAt: now.Add(-91 * 24 * time.Hour), Sends: 5}, nil, emailval.StatusValid, ""},
		{"gold from network engagement", valid, LocalSignals{}, &signalnet.Signal{Engaged: 1, LastEngagedAt: now.Add(-time.Hour)}, emailval.StatusValid, emailval.SubGold},
		{"gold on promoted catch-all", catchAll, LocalSignals{DomainDelivered: 100, LastEngagedAt: now}, nil, emailval.StatusValid, emailval.SubGold},
		{"alternate", valid, LocalSignals{Sends: 3}, nil, emailval.StatusValid, emailval.SubAlternate},
		{"too few sends for alternate", valid, LocalSignals{Sends: 2}, nil, emailval.StatusValid, ""},
		{"network engagement blocks alternate", valid, LocalSignals{Sends: 3}, &signalnet.Signal{Engaged: 1, LastEngagedAt: now.Add(-200 * 24 * time.Hour)}, emailval.StatusValid, ""},
		{"alternate keeps accept_all", catchAll, LocalSignals{DomainDelivered: 100, Sends: 5}, nil, emailval.StatusValid, emailval.SubAcceptAll},
		{"no enrichment on non-valid", invalid, LocalSignals{LastEngagedAt: now}, nil, emailval.StatusInvalid, emailval.SubMailboxNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.res
			got := Overlay(&in, tt.local, tt.net, now)
			if got.Status != tt.status || got.SubStatus != tt.sub {
				t.Errorf("got %s/%s, want %s/%s", got.Status, got.SubStatus, tt.status, tt.sub)
			}
			if in.Status != tt.res.Status || in.SubStatus != tt.res.SubStatus {
				t.Error("Overlay mutated its input")
			}
		})
	}
}

func TestCleanEmails(t *testing.T) {
	got := CleanEmails([]string{" A@x.com", "a@X.com", "", "  ", "b@y.com"})
	if len(got) != 2 || got[0] != "a@x.com" || got[1] != "b@y.com" {
		t.Errorf("CleanEmails = %v", got)
	}
}
