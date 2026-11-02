package mailpool

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"regexp"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/events"
)

// report is what a delivery status notification (RFC 3464) or an abuse
// feedback report (RFC 5965) says about one of our sends.
type report struct {
	complaint     bool
	verpID        string // the send id from b+<id>@<BounceDomain>
	origMessageID string // the Message-ID of the email the report is about
	status        string // DSN Status, e.g. 5.1.1
	action        string // DSN Action: failed, delayed, ...
	diagnostic    string
	feedbackType  string // ARF Feedback-Type, e.g. abuse
}

var verpPattern = regexp.MustCompile(`(?i)\bb\+([0-9a-f]{32})@([a-z0-9.-]+)`)

// parseReport reads a DSN or an ARF report. Anything else is not one.
func (s *Service) parseReport(raw []byte) (*report, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	mt, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil || mt != "multipart/report" {
		return nil, errors.New("not a multipart/report message")
	}
	r := &report{}
	switch strings.ToLower(params["report-type"]) {
	case "delivery-status":
	case "feedback-report":
		r.complaint = true
	default:
		return nil, errors.New("report-type is neither delivery-status nor feedback-report")
	}
	// The report's own envelope: catchall keeps the recipient in these.
	verpFrom := []string{msg.Header.Get("To"), msg.Header.Get("Delivered-To"), msg.Header.Get("X-Original-To")}

	mr := multipart.NewReader(msg.Body, params["boundary"])
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(part)
		if err != nil {
			return nil, err
		}
		ct, _, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
		switch strings.ToLower(ct) {
		case "message/delivery-status", "message/global-delivery-status":
			readDeliveryStatus(body, r)
		case "message/feedback-report":
			h := readFields(body)
			r.feedbackType = strings.ToLower(strings.TrimSpace(h.Get("Feedback-Type")))
			verpFrom = append(verpFrom, h.Get("Original-Mail-From"))
		case "message/rfc822", "text/rfc822-headers", "message/rfc822-headers", "message/global", "message/global-headers":
			h := readFields(body)
			r.origMessageID = strings.Trim(strings.TrimSpace(h.Get("Message-Id")), "<>")
			verpFrom = append(verpFrom, h.Get("Return-Path"))
		}
	}
	for _, v := range verpFrom {
		if m := verpPattern.FindStringSubmatch(v); m != nil && strings.EqualFold(m[2], s.cfg.BounceDomain) {
			r.verpID = strings.ToLower(m[1])
			break
		}
	}
	return r, nil
}

// readFields reads one block of header-style fields.
func readFields(b []byte) textproto.MIMEHeader {
	h, _ := textproto.NewReader(bufio.NewReader(bytes.NewReader(append(bytes.TrimLeft(b, "\r\n"), "\r\n\r\n"...)))).ReadMIMEHeader()
	return h
}

// readDeliveryStatus reads the per-recipient blocks of a DSN and keeps the
// first that failed or was delayed.
func readDeliveryStatus(b []byte, r *report) {
	text := strings.ReplaceAll(string(b), "\r\n", "\n")
	for _, block := range strings.Split(text, "\n\n") {
		h := readFields([]byte(block))
		action := strings.ToLower(strings.TrimSpace(h.Get("Action")))
		if action == "" {
			continue
		}
		if r.action == "" || action == "failed" && r.action != "failed" {
			r.action = action
			r.status = strings.TrimSpace(h.Get("Status"))
			// "smtp; 550 5.1.1 ...": drop the diagnostic type.
			diag := h.Get("Diagnostic-Code")
			if i := strings.Index(diag, ";"); i >= 0 {
				diag = diag[i+1:]
			}
			r.diagnostic = strings.TrimSpace(diag)
		}
	}
}

// ReportResult is what a bounce or complaint report was matched to.
type ReportResult struct {
	MessageID string
	MailboxID string
	Tenant    string
	Email     string
	Kind      string // bounce, complaint, or policy (blocked for policy or reputation: no address fault)
	Hard      bool
}

// IngestReport takes a DSN (RFC 3464) or ARF (RFC 5965) message, matches
// it to a pool send by its VERP address or the original Message-ID,
// records it in the existing bounce and complaint tables and re-checks the
// mailbox's health.
func (s *Service) IngestReport(ctx context.Context, orgID string, raw []byte) (*ReportResult, error) {
	if err := s.on(); err != nil {
		return nil, err
	}
	r, err := s.parseReport(raw)
	if err != nil {
		return nil, errorx.NewBadRequestError("not a delivery status or feedback report: " + err.Error())
	}
	send, err := s.matchSend(ctx, r)
	if err != nil {
		return nil, err
	}
	if send.OrgID != orgID {
		return nil, errorx.NewNotFoundError("no pool send matches this report")
	}
	mb, err := s.store.GetPoolMailbox(ctx, send.MailboxID)
	if err != nil {
		return nil, err
	}
	res := &ReportResult{MessageID: send.MessageID, MailboxID: send.MailboxID, Tenant: send.Tenant, Email: send.ToEmail}

	switch {
	case r.complaint:
		res.Kind = "complaint"
		if _, err := s.store.CreateEmailComplaint(ctx, db.CreateEmailComplaintParams{
			ID: newID(), Email: send.ToEmail, EmailForLower: send.ToEmail,
			ComplaintType:   nullString(r.feedbackType),
			SourceEmail:     nullString(mb.Address),
			MessageID:       nullString(send.MessageID),
			RawNotification: nullString(string(raw)),
			PoolSendID:      nullString(send.ID),
		}); err != nil {
			return nil, err
		}
		s.emit(events.TopicPoolMessageComplained, events.PoolEvent{
			OrgID: send.OrgID, Tenant: send.Tenant, MailboxID: send.MailboxID, MessageID: send.MessageID,
			Email: send.ToEmail, Reason: "reported as " + firstNonEmpty(r.feedbackType, "abuse"),
		})
	case strings.HasPrefix(r.status, "5") && isPolicy(r.status+" "+r.diagnostic):
		// Blocked for policy or reputation: the mailbox's fault, not the address's.
		res.Kind = "policy"
		if err := s.store.MarkPoolSendPolicyBlocked(ctx, db.MarkPoolSendPolicyBlockedParams{
			SmtpResponse: nullString(firstNonEmpty(r.diagnostic, r.status)), ID: send.ID,
		}); err != nil {
			return nil, err
		}
	default:
		res.Kind = "bounce"
		res.Hard = strings.HasPrefix(r.status, "5") || (r.status == "" && r.action == "failed")
		if err := s.recordBounce(ctx, send, mb.Address, res.Hard, firstNonEmpty(r.diagnostic, r.status), string(raw)); err != nil {
			return nil, err
		}
	}
	if err := s.evaluate(ctx, send.MailboxID); err != nil {
		return nil, err
	}
	return res, nil
}

func (s *Service) matchSend(ctx context.Context, r *report) (db.PoolSend, error) {
	var send db.PoolSend
	err := sql.ErrNoRows
	if r.verpID != "" {
		send, err = s.store.GetPoolSend(ctx, r.verpID)
	}
	if errors.Is(err, sql.ErrNoRows) && r.origMessageID != "" {
		send, err = s.store.GetPoolSendByMessageID(ctx, strings.ToLower(r.origMessageID))
	}
	if errors.Is(err, sql.ErrNoRows) {
		return send, errorx.NewNotFoundError("no pool send matches this report")
	}
	return send, err
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}
