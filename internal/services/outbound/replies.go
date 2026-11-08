package outbound

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"regexp"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/google/uuid"
)

func dialIMAP(mb db.Mailbox, password string) (*imapclient.Client, error) {
	addr := net.JoinHostPort(mb.ImapHost, fmt.Sprint(mb.ImapPort))
	var c *imapclient.Client
	var err error
	if isLoopback(mb.ImapHost) {
		c, err = imapclient.DialInsecure(addr, nil)
	} else {
		c, err = imapclient.DialTLS(addr, nil)
	}
	if err != nil {
		return nil, err
	}
	if err := c.Login(mb.ImapUsername, password).Wait(); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

func testIMAP(mb db.Mailbox, password string) error {
	c, err := dialIMAP(mb, password)
	if err != nil {
		return err
	}
	defer c.Close()
	if _, err := c.Select("INBOX", &imap.SelectOptions{ReadOnly: true}).Wait(); err != nil {
		return err
	}
	return c.Logout().Wait()
}

// PollReplies reads new mail in every connected mailbox, paused ones too.
func (s *Service) PollReplies(ctx context.Context) {
	mailboxes, err := s.db.ListAllMailboxes(ctx)
	if err != nil {
		return
	}
	for _, mb := range mailboxes {
		if err := s.pollMailbox(ctx, mb); err != nil {
			log.Printf("outbound: read %s: %v", mb.Email, err)
			_ = s.db.SetMailboxError(ctx, db.SetMailboxErrorParams{LastError: "IMAP: " + err.Error(), ID: mb.ID})
		}
	}
}

func (s *Service) pollMailbox(ctx context.Context, mb db.Mailbox) error {
	pass, err := s.imapPassword(mb)
	if err != nil {
		return err
	}
	c, err := dialIMAP(mb, pass)
	if err != nil {
		return err
	}
	defer c.Close()
	sel, err := c.Select("INBOX", &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		return err
	}
	// The first read, or a mailbox whose UIDs were reset, starts from now:
	// mail from before outreach began is never classified.
	if mb.ImapUidValidity == 0 || int64(sel.UIDValidity) != mb.ImapUidValidity {
		return s.db.SetMailboxIMAPState(ctx, db.SetMailboxIMAPStateParams{
			ImapUidValidity: int64(sel.UIDValidity), ImapLastUid: int64(sel.UIDNext) - 1, ID: mb.ID,
		})
	}
	if uint32(sel.UIDNext) <= uint32(mb.ImapLastUid)+1 {
		return nil
	}
	var set imap.UIDSet
	set.AddRange(imap.UID(mb.ImapLastUid+1), 0) // 0 is "*"
	msgs, err := c.Fetch(set, &imap.FetchOptions{
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{{Peek: true}},
	}).Collect()
	if err != nil {
		return err
	}
	last := mb.ImapLastUid
	for _, m := range msgs {
		if int64(m.UID) <= mb.ImapLastUid {
			continue // "*" names the last message even when it is old
		}
		if len(m.BodySection) > 0 {
			s.handleIncoming(ctx, mb, m.BodySection[0].Bytes)
		}
		if int64(m.UID) > last {
			last = int64(m.UID)
		}
	}
	return s.db.SetMailboxIMAPState(ctx, db.SetMailboxIMAPStateParams{ImapUidValidity: mb.ImapUidValidity, ImapLastUid: last, ID: mb.ID})
}

// incoming is the part of a received email outbound reads.
type incoming struct {
	From, Subject, MessageID, InReplyTo string
	References                          []string
	Text                                string
	Header                              mail.Header
	Report                              bool   // a delivery status report
	FailedRecipient                     string // from the report
}

func parseIncoming(raw []byte) (incoming, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return incoming{}, err
	}
	in := incoming{Header: msg.Header}
	if a, err := mail.ParseAddress(msg.Header.Get("From")); err == nil {
		in.From = normEmail(a.Address)
	}
	dec := new(mime.WordDecoder)
	in.Subject, _ = dec.DecodeHeader(msg.Header.Get("Subject"))
	in.MessageID = strings.Trim(msg.Header.Get("Message-Id"), "<> ")
	in.InReplyTo = strings.Trim(msg.Header.Get("In-Reply-To"), "<> ")
	for _, r := range strings.Fields(msg.Header.Get("References")) {
		in.References = append(in.References, strings.Trim(r, "<>"))
	}
	ct, params, _ := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	in.Report = ct == "multipart/report"
	body, _ := io.ReadAll(io.LimitReader(msg.Body, 1<<20))
	in.Text = textOf(ct, params, msg.Header.Get("Content-Transfer-Encoding"), body, &in)
	return in, nil
}

var finalRecipientRe = regexp.MustCompile(`(?im)^(?:final|original)-recipient:\s*rfc822;\s*<?([^\s>]+)`)

// textOf pulls the plain text out of a message, and notes a failed recipient
// in a delivery report.
func textOf(ct string, params map[string]string, cte string, body []byte, in *incoming) string {
	if strings.EqualFold(cte, "quoted-printable") {
		if b, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body))); err == nil {
			body = b
		}
	}
	if strings.HasPrefix(ct, "multipart/") {
		r := multipart.NewReader(bytes.NewReader(body), params["boundary"])
		var text string
		for {
			p, err := r.NextPart()
			if err != nil {
				break
			}
			pct, pparams, _ := mime.ParseMediaType(p.Header.Get("Content-Type"))
			pb, _ := io.ReadAll(io.LimitReader(p, 1<<20))
			switch {
			case pct == "message/delivery-status":
				if m := finalRecipientRe.FindSubmatch(pb); m != nil {
					in.FailedRecipient = normEmail(string(m[1]))
				}
			case pct == "text/rfc822-headers" || pct == "message/rfc822":
				text += "\n" + string(pb)
			default:
				if t := textOf(pct, pparams, p.Header.Get("Content-Transfer-Encoding"), pb, in); text == "" || pct == "text/plain" {
					if strings.TrimSpace(t) != "" {
						text = t + text
					}
				}
			}
		}
		return text
	}
	if ct == "text/html" {
		return stripTags(string(body))
	}
	return string(body)
}

var tagRe = regexp.MustCompile(`<[^>]*>`)

func stripTags(s string) string { return tagRe.ReplaceAllString(s, " ") }

// newText is what the person wrote, without the quoted thread below it.
func newText(text string) string {
	var out []string
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, ">") {
			continue
		}
		if (strings.HasPrefix(line, "On ") && strings.HasSuffix(line, "wrote:")) || strings.HasPrefix(line, "-----Original Message") ||
			strings.HasPrefix(line, "From: ") {
			break
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

var (
	spamBlockRe  = regexp.MustCompile(`(?i)(\b5\.7\.\d+\b|\bspam\b|blocked|blacklist|blocklist|policy|reputation|not accepted for policy)`)
	oooSubjectRe = regexp.MustCompile(`(?i)\b(out of (the )?office|automatic reply|auto-?reply|autoreply|on vacation|away from)\b`)
	optOutRe     = regexp.MustCompile(`(?i)\b(unsubscribe|remove me|take me off|stop (emailing|e-mailing|contacting)|don'?t (email|contact) me|no thanks|no, thanks|not interested)\b`)
	notInterRe   = regexp.MustCompile(`(?i)\bnot interested\b`)
	angryRe      = regexp.MustCompile(`(?i)\b(spam|reported|reporting you|how did you get my (email|address))\b`)
)

// classify applies the simple rules. Everything they don't settle is
// needs_review, which the AI classifies through the conversations tool.
func classify(mb db.Mailbox, in incoming) string {
	from := in.From
	if in.Report || strings.HasPrefix(from, "mailer-daemon@") || strings.HasPrefix(from, "postmaster@") {
		// A rejection for spam or policy says the mailbox (or its domain) is
		// being blocked, not that the address is bad.
		if spamBlockRe.MatchString(in.Text) {
			return "spam_block"
		}
		return "bounce"
	}
	if as := strings.ToLower(in.Header.Get("Auto-Submitted")); (as != "" && as != "no") || in.Header.Get("X-Autoreply") != "" ||
		in.Header.Get("X-Autorespond") != "" || strings.EqualFold(in.Header.Get("Precedence"), "auto_reply") || oooSubjectRe.MatchString(in.Subject) {
		return "out_of_office"
	}
	if strings.EqualFold(strings.TrimSpace(in.Subject), "unsubscribe") {
		return "unsubscribe"
	}
	text := newText(in.Text)
	switch {
	case angryRe.MatchString(text):
		return "complaint"
	case notInterRe.MatchString(text) && len(text) < 200:
		return "not_interested"
	case optOutRe.MatchString(text) && len(text) < 200:
		return "unsubscribe"
	}
	return "needs_review"
}

// handleIncoming matches one received email to the outreach it answers and
// applies its effects. Mail that answers no outreach is left alone.
func (s *Service) handleIncoming(ctx context.Context, mb db.Mailbox, raw []byte) {
	in, err := parseIncoming(raw)
	if err != nil || in.From == normEmail(mb.Email) {
		return
	}
	class := classify(mb, in)
	isBounce := class == "bounce" || class == "spam_block"

	// Match: threading headers first, then a message id inside a bounce,
	// then the failed recipient of a bounce, then the sender's address.
	var sent *db.GetOutreachEmailByMessageIDRow
	ids := append([]string{in.InReplyTo}, in.References...)
	if isBounce {
		ids = append(ids, msgIDsIn(in.Text)...)
	}
	for _, id := range ids {
		if id == "" {
			continue
		}
		if m, err := s.db.GetOutreachEmailByMessageID(ctx, sql.NullString{String: id, Valid: true}); err == nil && m.MailboxID.String == mb.ID {
			sent = &m
			break
		}
	}
	var contact db.Contact
	switch {
	case sent != nil:
		contact, err = s.db.GetContactByID(ctx, sent.ContactID.String)
	case isBounce && in.FailedRecipient != "":
		contact, err = s.db.GetContactByOrgAndEmail(ctx, db.GetContactByOrgAndEmailParams{OrgID: sql.NullString{String: mb.OrgID, Valid: true}, Email: in.FailedRecipient})
	case !isBounce:
		contact, err = s.db.GetContactByOrgAndEmail(ctx, db.GetContactByOrgAndEmailParams{OrgID: sql.NullString{String: mb.OrgID, Valid: true}, Email: in.From})
	default:
		return
	}
	if err != nil || contact.OrgID.String != mb.OrgID {
		return // not an answer to outreach
	}
	if isBounce && in.FailedRecipient != "" && in.FailedRecipient != normEmail(contact.Email) {
		return // a bounce for some other mail
	}
	queueID, sequenceID := sql.NullString{}, sql.NullString{}
	if sent != nil {
		queueID, sequenceID = sql.NullString{String: sent.ID, Valid: true}, sent.SequenceID
	} else if states, err := s.db.ListActiveOutreachStatesForContact(ctx, sql.NullString{String: contact.ID, Valid: true}); err == nil && len(states) > 0 {
		sequenceID = states[0].SequenceID
	}
	if sent == nil && !sequenceID.Valid && !isBounce {
		return // mail from a contact with no outreach going on is ordinary mail
	}
	handled := int64(1)
	if class == "needs_review" || class == "complaint" {
		handled = 0
	}
	if in.MessageID == "" {
		in.MessageID = uuid.NewString() + "@missing"
	}
	if _, err := s.db.CreateEmailReply(ctx, db.CreateEmailReplyParams{
		ID: uuid.NewString(), OrgID: mb.OrgID, MailboxID: mb.ID,
		ContactID: sql.NullString{String: contact.ID, Valid: true}, QueueID: queueID, SequenceID: sequenceID,
		FromEmail: in.From, Subject: in.Subject, Body: truncate(newTextOrAll(in.Text), 20000),
		MessageID: in.MessageID, InReplyTo: in.InReplyTo, Refs: strings.Join(in.References, " "),
		Classification: class, Handled: handled, ReceivedAt: ts(s.now()),
	}); err != nil {
		return // already recorded (ON CONFLICT) or failed; effects ran the first time
	}
	s.applyClassification(ctx, mb.OrgID, contact, class)
}

// applyClassification records what a reply means. Any human reply ends the
// person's outreach. Opt-outs go on the brand's suppression list; bounces and
// complaints into Outlet's bounce and complaint records, which stop every
// kind of mail to the address. A spam block stops the person's outreach but
// says nothing about the address; it counts against the mailbox's health.
func (s *Service) applyClassification(ctx context.Context, orgID string, c db.Contact, class string) {
	reason := "replied"
	switch class {
	case "out_of_office":
		return // the sequence carries on; its follow-up arrives after they're back
	case "bounce":
		reason = "bounced"
		_, _ = s.db.CreateEmailBounce(ctx, db.CreateEmailBounceParams{ID: uuid.NewString(), Email: c.Email, EmailForLower: c.Email, BounceType: "Permanent",
			BounceSubtype: sql.NullString{String: "outreach", Valid: true}})
	case "spam_block":
		reason = "spam_blocked"
	case "unsubscribe":
		reason = "unsubscribed"
		s.suppress(ctx, orgID, c.Email, "outreach opt-out")
	case "complaint":
		reason = "complained"
		_, _ = s.db.CreateEmailComplaint(ctx, db.CreateEmailComplaintParams{ID: uuid.NewString(), Email: c.Email, EmailForLower: c.Email,
			ComplaintType: sql.NullString{String: "outreach", Valid: true}})
	case "not_interested", "wrong_person":
		reason = class
	case "interested":
		_, _ = s.db.AddContactTag(ctx, db.AddContactTagParams{ContactID: sql.NullString{String: c.ID, Valid: true}, Tag: "interested"})
	}
	s.stopContact(ctx, c.ID, reason)
}

var msgIDRe = regexp.MustCompile(`<([^<>@\s]+@[^<>\s]+)>`)

func msgIDsIn(text string) []string {
	var ids []string
	for _, m := range msgIDRe.FindAllStringSubmatch(text, 20) {
		ids = append(ids, m[1])
	}
	return ids
}

func newTextOrAll(text string) string {
	if t := newText(text); t != "" {
		return t
	}
	return strings.TrimSpace(text)
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
