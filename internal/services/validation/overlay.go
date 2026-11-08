package validation

import (
	"time"

	"github.com/outlet-sh/outlet/internal/services/emailval"
	"github.com/outlet-sh/outlet/internal/services/signalnet"
)

// Thresholds for the signal overlays.
const (
	engagedWithin         = 90 * 24 * time.Hour
	alternateMinSends     = 3
	catchAllMinDeliveries = 50
	catchAllMaxBounceRate = 0.02
	trapScoreThreshold    = 0.8
	networkMinHardBounces = 2
)

// LocalSignals is what this instance (across all its orgs) knows about an address.
type LocalSignals struct {
	Complaint       bool      // any org received a complaint from it
	Suppressed      bool      // permanent hard bounce, or on any org's suppression list
	Sends           int       // campaign sends to it in the last 12 months
	LastEngagedAt   time.Time // last open or click, zero if never
	DomainDelivered int       // campaign deliveries to its domain (only filled for catch-alls)
	DomainBounced   int       // campaign bounces on its domain (only filled for catch-alls)
}

// Overlay applies instance and Signal Network signals to the engine result and returns
// the final verdict. It never mutates res. net may be nil.
//
// Precedence, first match wins:
//  1. abuse                         - complaint seen locally or anywhere on the network
//  2. spamtrap                      - network trap score >= 0.8
//  3. invalid (engine)              - a mailbox that does not exist stays invalid
//  4. do_not_mail/global_suppression - permanent hard bounce or any org's suppression list
//  5. do_not_mail/toxic             - network flags the address as toxic
//  6. invalid/mailbox_not_found     - network saw >= 2 hard bounces and no delivery
//  7. engine verdict, where a catch-all on a domain with >= 50 local deliveries and a
//     bounce rate under 2% is promoted to valid/accept_all (valid/role_based_accept_all
//     for role addresses).
//
// Then, for a valid verdict only: an open or click within 90 days (local or network)
// sets sub-status gold; otherwise a plain valid with >= 3 local sends in 12 months and
// no engagement at all sets sub-status alternate.
func Overlay(res *emailval.Result, local LocalSignals, net *signalnet.Signal, now time.Time) *emailval.Result {
	out := *res
	if net == nil {
		net = &signalnet.Signal{}
	}

	switch {
	case local.Complaint || net.Complaints > 0:
		out.Status, out.SubStatus = emailval.StatusAbuse, ""
	case net.TrapScore >= trapScoreThreshold:
		out.Status, out.SubStatus = emailval.StatusSpamtrap, ""
	case out.Status == emailval.StatusInvalid:
		// keep the engine's invalid verdict
	case local.Suppressed:
		out.Status, out.SubStatus = emailval.StatusDoNotMail, emailval.SubGlobalSuppression
	case net.Toxic:
		out.Status, out.SubStatus = emailval.StatusDoNotMail, emailval.SubToxic
	case net.HardBounces >= networkMinHardBounces && net.Delivered == 0:
		out.Status, out.SubStatus = emailval.StatusInvalid, emailval.SubMailboxNotFound
	case isCatchAll(&out) && goodDomainHistory(local):
		out.Status, out.SubStatus = emailval.StatusValid, emailval.SubAcceptAll
		if out.RoleBased {
			out.SubStatus = emailval.SubRoleBasedAcceptAll
		}
	}

	if out.Status != emailval.StatusValid {
		return &out
	}
	lastEngaged := local.LastEngagedAt
	if net.Engaged > 0 && net.LastEngagedAt.After(lastEngaged) {
		lastEngaged = net.LastEngagedAt
	}
	switch {
	case !lastEngaged.IsZero() && now.Sub(lastEngaged) <= engagedWithin:
		out.SubStatus = emailval.SubGold
	case out.SubStatus == "" && local.Sends >= alternateMinSends && local.LastEngagedAt.IsZero() && net.Engaged == 0:
		out.SubStatus = emailval.SubAlternate
	}
	return &out
}

// isCatchAll reports whether the engine found the domain accepts every address.
func isCatchAll(res *emailval.Result) bool {
	return res.Status == emailval.StatusCatchAll ||
		(res.Status == emailval.StatusDoNotMail && res.SubStatus == emailval.SubRoleBasedCatchAll)
}

func goodDomainHistory(local LocalSignals) bool {
	if local.DomainDelivered < catchAllMinDeliveries {
		return false
	}
	total := local.DomainDelivered + local.DomainBounced
	return float64(local.DomainBounced)/float64(total) < catchAllMaxBounceRate
}
