package mailpool

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/textproto"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/events"
	"github.com/outlet-sh/outlet/internal/services/email"
)

// SendInput is one email from a tenant's mailbox.
type SendInput struct {
	OrgID                 string
	Tenant                string
	MailboxID             string // optional: must be one of the tenant's mailboxes
	To                    string
	FromName              string
	Subject               string
	Text                  string
	HTML                  string // optional
	InReplyTo             string // optional
	References            string // optional
	Headers               map[string]string
	ListUnsubscribeURL    string // https, RFC 8058 one-click
	ListUnsubscribeMailto string // optional
	IdempotencyKey        string
}

// SendResult is what the send did. A replayed idempotency key answers the
// first result and sends nothing.
type SendResult struct {
	MessageID      string // without angle brackets
	MailboxID      string
	MailboxAddress string
	Status         string // sent, failed, or sending (a first try still in flight)
	FailureKind    string // recipient, policy, provider, transient
	SMTPCode       int
	SMTPResponse   string
}

// Send sends one email from the tenant's sticky mailbox. Refusals before
// anything leaves: 400 bad input, 404 not the tenant's mailbox, 409 the
// mailbox, its domain or the tenant is held, 422 the address is on the pool
// blocklist, 429 over today's allowance or inside the gap, 503 pool off or
// no mailboxes. Once the SMTP server has answered the result is 200 with
// status sent or failed.
func (s *Service) Send(ctx context.Context, in SendInput) (*SendResult, error) {
	if err := s.on(); err != nil {
		return nil, err
	}
	if s.cfg.BounceDomain == "" {
		return nil, errorx.NewCodeError(http.StatusServiceUnavailable, "pool has no bounce domain")
	}
	to, err := checkSendInput(in)
	if err != nil {
		return nil, err
	}

	if prior, err := s.store.GetPoolSendByIdempotencyKey(ctx, db.GetPoolSendByIdempotencyKeyParams{
		OrgID: in.OrgID, IdempotencyKey: in.IdempotencyKey,
	}); err == nil {
		return s.result(ctx, prior)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	now := s.now()
	held, err := s.tenantHeld(ctx, in.OrgID, in.Tenant, now)
	if err != nil {
		return nil, err
	}
	if held {
		return nil, errorx.NewCodeError(http.StatusConflict, "tenant held: its mailboxes were paused for list health too often; clean the list, then an admin clears the hold")
	}
	blocked, err := s.store.IsPoolAddressBlocked(ctx, to)
	if err != nil {
		return nil, err
	}
	if blocked == 1 {
		return nil, errorx.NewCodeError(http.StatusUnprocessableEntity, "address is on the pool blocklist (hard bounce or complaint)")
	}

	mb, err := s.pick(ctx, in, to, now)
	if err != nil {
		return nil, err
	}

	// Check the allowance and gap and reserve the send under the mailbox
	// lock, so two sends can't both take the last slot.
	id := randomID()
	messageID := id + "@" + domainOf(mb.Address)
	lock := s.mailboxLock(mb.ID)
	lock.Lock()
	c, err := s.capacity(ctx, mb, now)
	if err == nil {
		err = readyToSend(c, now)
	}
	if err == nil {
		err = s.store.CreatePoolSend(ctx, db.CreatePoolSendParams{
			ID: id, OrgID: in.OrgID, MailboxID: mb.ID, Tenant: in.Tenant, IdempotencyKey: in.IdempotencyKey,
			MessageID: messageID, ToEmail: to, CreatedAt: fmtTime(now),
		})
		if err != nil {
			// The same key raced in: answer its result.
			if prior, gerr := s.store.GetPoolSendByIdempotencyKey(ctx, db.GetPoolSendByIdempotencyKeyParams{
				OrgID: in.OrgID, IdempotencyKey: in.IdempotencyKey,
			}); gerr == nil {
				lock.Unlock()
				return s.result(ctx, prior)
			}
		}
	}
	lock.Unlock()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{}
	for k, v := range in.Headers {
		headers[k] = v
	}
	if in.InReplyTo != "" {
		headers["In-Reply-To"] = in.InReplyTo
	}
	if in.References != "" {
		headers["References"] = in.References
	}
	raw, err := email.BuildMessage(
		(&mail.Address{Name: in.FromName, Address: mb.Address}).String(), to, in.Subject, in.HTML,
		email.WithTextBody(in.Text),
		email.WithHeaders(headers),
		email.WithHeaders(map[string]string{"Message-ID": "<" + messageID + ">"}), // ours, always
		email.WithListUnsubscribe(in.ListUnsubscribeURL, in.ListUnsubscribeMailto),
	)
	if err == nil {
		var p *email.SMTPPool
		if p, err = s.smtpFor(mb); err == nil {
			err = p.SendWithPool("b+"+id+"@"+s.cfg.BounceDomain, []string{to}, raw)
		}
	}

	res := &SendResult{MessageID: messageID, MailboxID: mb.ID, MailboxAddress: mb.Address, Status: "sent"}
	finish := db.FinishPoolSendParams{ID: id, Status: "sent", SentAt: nullString(fmtTime(s.now()))}
	if err != nil {
		res.Status = "failed"
		res.FailureKind, res.SMTPCode, res.SMTPResponse = classify(err)
		finish = db.FinishPoolSendParams{
			ID: id, Status: "failed",
			FailureKind:  nullString(res.FailureKind),
			SmtpCode:     sql.NullInt64{Int64: int64(res.SMTPCode), Valid: res.SMTPCode != 0},
			SmtpResponse: nullString(res.SMTPResponse),
		}
	}
	if err := s.store.FinishPoolSend(ctx, finish); err != nil {
		return nil, err
	}
	if res.FailureKind == "recipient" {
		// The receiving server refused the address outright: a hard bounce.
		send, err := s.store.GetPoolSend(ctx, id)
		if err != nil {
			return nil, err
		}
		if err := s.recordBounce(ctx, send, mb.Address, true, fmt.Sprintf("%d %s", res.SMTPCode, res.SMTPResponse), ""); err != nil {
			return nil, err
		}
	}
	if err := s.evaluate(ctx, mb.ID); err != nil {
		return nil, err
	}
	return res, nil
}

func checkSendInput(in SendInput) (string, error) {
	if in.Tenant == "" || in.To == "" || in.Subject == "" || in.Text == "" || in.IdempotencyKey == "" {
		return "", errorx.NewBadRequestError("tenant, to, subject, text and idempotency_key are required")
	}
	if !strings.HasPrefix(in.ListUnsubscribeURL, "https://") {
		return "", errorx.NewBadRequestError("list_unsubscribe_url must be an https URL (RFC 8058 one-click)")
	}
	for _, v := range []string{in.FromName, in.Subject, in.To, in.ListUnsubscribeURL, in.ListUnsubscribeMailto} {
		if strings.ContainsAny(v, "\r\n") {
			return "", errorx.NewBadRequestError("a header value has a line break")
		}
	}
	addr, err := mail.ParseAddress(in.To)
	if err != nil {
		return "", errorx.NewBadRequestError("to is not an email address")
	}
	return strings.ToLower(addr.Address), nil
}

// pick chooses the mailbox: the one asked for, else the one that last wrote
// to this recipient (threads stay on one mailbox), else the tenant's ready
// mailbox with the most allowance left.
func (s *Service) pick(ctx context.Context, in SendInput, to string, now time.Time) (db.PoolMailbox, error) {
	mbs, err := s.store.ListTenantPoolMailboxes(ctx, db.ListTenantPoolMailboxesParams{
		OrgID: nullString(in.OrgID), Tenant: nullString(in.Tenant),
	})
	if err != nil {
		return db.PoolMailbox{}, err
	}
	if len(mbs) == 0 {
		return db.PoolMailbox{}, errorx.NewCodeError(http.StatusServiceUnavailable, "no mailboxes: assign the tenant mailboxes first")
	}
	byID := map[string]db.PoolMailbox{}
	for _, mb := range mbs {
		byID[mb.ID] = mb
	}

	want := in.MailboxID
	if want == "" {
		last, err := s.store.LastPoolSendToRecipient(ctx, db.LastPoolSendToRecipientParams{
			OrgID: in.OrgID, Tenant: in.Tenant, ToEmail: to,
		})
		switch {
		case err == nil:
			if _, ok := byID[last.MailboxID]; ok {
				want = last.MailboxID
			}
		case !errors.Is(err, sql.ErrNoRows):
			return db.PoolMailbox{}, err
		}
	}
	if want != "" {
		mb, ok := byID[want]
		if !ok {
			return db.PoolMailbox{}, errorx.NewNotFoundError("mailbox is not one of this tenant's")
		}
		c, err := s.capacity(ctx, mb, now)
		if err != nil {
			return db.PoolMailbox{}, err
		}
		if err := sendable(c); err != nil {
			return db.PoolMailbox{}, err
		}
		return mb, nil
	}

	var best *db.PoolMailbox
	bestLeft := 0
	var soonest *MailboxCapacity
	for i := range mbs {
		c, err := s.capacity(ctx, mbs[i], now)
		if err != nil {
			return db.PoolMailbox{}, err
		}
		if c.Status != "active" {
			continue
		}
		if readyToSend(c, now) == nil && c.RemainingToday > bestLeft {
			best, bestLeft = &mbs[i], c.RemainingToday
		}
		if soonest == nil || c.NextAllowedAt.Before(soonest.NextAllowedAt) {
			cc := c
			soonest = &cc
		}
	}
	if best != nil {
		return *best, nil
	}
	if soonest == nil {
		return db.PoolMailbox{}, errorx.NewCodeError(http.StatusConflict, "every mailbox of this tenant is paused")
	}
	return db.PoolMailbox{}, readyToSend(*soonest, now)
}

// sendable refuses a mailbox that is paused, resting, retired or on a
// paused domain (409); a paused mailbox keeps its queue unless the pause is
// a provider fault.
func sendable(c MailboxCapacity) error {
	if c.Status == "active" {
		return nil
	}
	msg := "mailbox " + c.Status
	if c.PausedReason != "" {
		msg += ": " + c.PausedReason
	}
	if c.Movable {
		msg += " (a provider fault: the queue may move to the tenant's other healthy mailboxes)"
	}
	return errorx.NewCodeError(http.StatusConflict, msg)
}

// readyToSend refuses a mailbox over today's allowance or inside its gap (429).
func readyToSend(c MailboxCapacity, now time.Time) error {
	if err := sendable(c); err != nil {
		return err
	}
	if c.RemainingToday <= 0 {
		return errorx.NewCodeError(http.StatusTooManyRequests, "daily allowance used: next send at "+c.NextAllowedAt.UTC().Format(time.RFC3339))
	}
	if c.NextAllowedAt.After(now) {
		return errorx.NewCodeError(http.StatusTooManyRequests, "inside the gap: next send at "+c.NextAllowedAt.UTC().Format(time.RFC3339))
	}
	return nil
}

// result answers a send already made (an idempotent replay).
func (s *Service) result(ctx context.Context, send db.PoolSend) (*SendResult, error) {
	mb, err := s.store.GetPoolMailbox(ctx, send.MailboxID)
	if err != nil {
		return nil, err
	}
	return &SendResult{
		MessageID:      send.MessageID,
		MailboxID:      send.MailboxID,
		MailboxAddress: mb.Address,
		Status:         send.Status,
		FailureKind:    send.FailureKind.String,
		SMTPCode:       int(send.SmtpCode.Int64),
		SMTPResponse:   send.SmtpResponse.String,
	}, nil
}

// classify sorts an SMTP failure: provider (the mailbox's own server refused
// the account), policy (a receiving server blocked it for policy or
// reputation), recipient (refused the address: a hard bounce) or transient
// (try again later).
func classify(err error) (kind string, code int, response string) {
	var te *textproto.Error
	hasCode := errors.As(err, &te)
	if hasCode {
		code, response = te.Code, te.Msg
	} else {
		response = err.Error()
	}
	var ae *email.AccountError
	switch {
	case errors.As(err, &ae):
		return "provider", code, response
	case !hasCode:
		return "provider", code, response
	case code >= 500 && isPolicy(response):
		return "policy", code, response
	case code >= 500:
		return "recipient", code, response
	default:
		return "transient", code, response
	}
}

// isPolicy: a refusal for policy or sending reputation rather than for the
// address (enhanced status 5.7.x, or the server says so).
func isPolicy(response string) bool {
	r := strings.ToLower(response)
	if strings.HasPrefix(r, "5.7.") || strings.Contains(r, " 5.7.") {
		return true
	}
	for _, w := range []string{"blocked", "blacklist", "blocklist", "spam", "reputation", "policy"} {
		if strings.Contains(r, w) {
			return true
		}
	}
	return false
}

// recordBounce writes a pool bounce into the existing email_bounce table,
// linked to its send, and tells the tenant's brand. A soft bounce never
// replaces a hard one.
func (s *Service) recordBounce(ctx context.Context, send db.PoolSend, from string, hard bool, diagnostic, raw string) error {
	bounceType := "Transient"
	if hard {
		bounceType = "Permanent"
	} else if prior, err := s.store.GetEmailBounce(ctx, send.ToEmail); err == nil && prior.BounceType == "Permanent" {
		return nil
	}
	if _, err := s.store.CreateEmailBounce(ctx, db.CreateEmailBounceParams{
		ID:              newID(),
		Email:           send.ToEmail,
		EmailForLower:   send.ToEmail,
		BounceType:      bounceType,
		BounceSubtype:   nullString("pool"),
		DiagnosticCode:  nullString(diagnostic),
		SourceEmail:     nullString(from),
		MessageID:       nullString(send.MessageID),
		RawNotification: nullString(raw),
		PoolSendID:      nullString(send.ID),
	}); err != nil {
		return err
	}
	s.emit(events.TopicPoolMessageBounced, events.PoolEvent{
		OrgID: send.OrgID, Tenant: send.Tenant, MailboxID: send.MailboxID, MessageID: send.MessageID,
		Email: send.ToEmail, Reason: diagnostic, Hard: hard,
	})
	return nil
}
