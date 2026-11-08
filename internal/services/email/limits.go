package email

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
)

// ErrOverLimit means a sender is over its pathway's daily limit or inside
// its spacing. Send returns it as a temporary DeliveryError with RetryAt:
// callers defer the email to then, never fail it.
var ErrOverLimit = errors.New("sending limit")

func overLimit(retryAt time.Time, format string, args ...any) error {
	return &DeliveryError{Temporary: true, RetryAt: retryAt, Err: fmt.Errorf("%w: %s", ErrOverLimit, fmt.Sprintf(format, args...))}
}

// RetryAt is when a temporary failure is worth trying again: the time a
// limit lifts, or fallback.
func RetryAt(err error, fallback time.Time) time.Time {
	var d *DeliveryError
	if errors.As(err, &d) && !d.RetryAt.IsZero() {
		return d.RetryAt
	}
	return fallback
}

// sender is who sends within a pathway: a mailbox of a pool, or the pathway.
type sender struct {
	id       string
	label    string
	start    time.Time     // where the warm-up ramp starts
	limits   SenderLimits // PathwayLimits, or a pool mailbox's MailboxLimits
	next     string        // its next-send time as stored, for the retry estimate
	mailbox  bool
}

// reserve takes one send for a sender, or refuses with ErrOverLimit: the
// spacing slot and today's count are claimed together in one transaction,
// with conditional updates, so two workers can't both pass a limit. The
// returned release gives the count back when the send then fails.
func (s *Service) reserve(ctx context.Context, org db.Organization, p db.SendingPathway, snd sender) (func(), error) {
	now := s.now()
	day := Day(now, org)
	loc := LoadLocation(org.OutboundTimezone)
	l := now.In(loc)
	tomorrow := time.Date(l.Year(), l.Month(), l.Day()+1, 0, 0, 0, 0, loc)

	allow, limited := Allowance(snd.limits, snd.start, now)
	if !limited {
		allow = math.MaxInt64
	} else if allow <= 0 {
		return nil, overLimit(tomorrow, "over the pathway's daily limit (%s may send none today)", snd.label)
	}
	err := s.db.ExecTx(ctx, func(q *db.Queries) error {
		if p.MaxGapSeconds > 0 {
			gap := time.Duration(p.MinGapSeconds) * time.Second
			if spread := p.MaxGapSeconds - p.MinGapSeconds; spread > 0 {
				gap += time.Duration(rand.Int64N(spread+1)) * time.Second
			}
			next := sql.NullString{String: now.Add(gap).UTC().Format(TimeLayout), Valid: true}
			at := sql.NullString{String: now.UTC().Format(TimeLayout), Valid: true}
			var n int64
			var err error
			if snd.mailbox {
				n, err = q.ClaimMailboxSpacing(ctx, db.ClaimMailboxSpacingParams{NextSendAt: next, ID: snd.id, Now: at})
			} else {
				n, err = q.ClaimPathwaySpacing(ctx, db.ClaimPathwaySpacingParams{NextSendAt: next, ID: snd.id, Now: at})
			}
			if err != nil {
				return err
			}
			if n == 0 {
				retry, perr := time.ParseInLocation(TimeLayout, snd.next, time.UTC)
				if perr != nil || !retry.After(now) {
					retry = now.Add(time.Duration(max(p.MinGapSeconds, 1)) * time.Second)
				}
				return overLimit(retry, "%s is spacing its sends; the next may go at %s UTC", snd.label, retry.UTC().Format(TimeLayout))
			}
		}
		if err := q.EnsureSendCount(ctx, db.EnsureSendCountParams{PathwayID: p.ID, SenderID: snd.id, Day: day}); err != nil {
			return err
		}
		n, err := q.IncrementSendCount(ctx, db.IncrementSendCountParams{PathwayID: p.ID, SenderID: snd.id, Day: day, Allowance: allow})
		if err != nil {
			return err
		}
		if n == 0 {
			return overLimit(tomorrow, "over the pathway's daily limit (%s sent its %d for today)", snd.label, allow)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	release := func() {
		_ = s.db.RefundSendCount(context.WithoutCancel(ctx), db.RefundSendCountParams{PathwayID: p.ID, SenderID: snd.id, Day: day})
	}
	return release, nil
}

// GiveUpAfter is how long Outlet keeps trying a send the mail server refuses
// for now, as a mail server does before it bounces.
const GiveUpAfter = 5 * 24 * time.Hour

// RefusedAt is now when err is a refusal for now (a 4xx, a dropped
// connection, throttling), to record as a send's first refusal. A limit hold
// (a daily limit, warm-up, a paused pool) carries a RetryAt and is not a
// refusal: it waits as long as it must.
func RefusedAt(err error, now time.Time) sql.NullString {
	var d *DeliveryError
	if !errors.As(err, &d) || !d.Temporary || !d.RetryAt.IsZero() {
		return sql.NullString{}
	}
	return sql.NullString{String: now.UTC().Format(time.RFC3339), Valid: true}
}

// GaveUp reports whether a send first refused at since has been refused for
// longer than GiveUpAfter.
func GaveUp(since sql.NullString, now time.Time) bool {
	t, err := time.Parse(time.RFC3339, since.String)
	return since.Valid && err == nil && now.Sub(t) > GiveUpAfter
}
