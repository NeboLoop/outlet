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

func dialIMAP(mb db.OutboundMailbox, password string) (*imapclient.Client, error) {
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

func testIMAP(mb db.OutboundMailbox, password string) error {
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
	mailboxes, err := s.db.ListAllOutboundMailboxes(ctx)
	if err != nil {
		return
	}
	for _, mb := range mailboxes {
		if err := s.pollMailbox(ctx, mb); err != nil {
			log.Printf("outbound: read %s: %v", mb.Email, err)
			_ = s.db.SetOutboundMailboxError(ctx, db.SetOutboundMailboxErrorParams{LastError: "IMAP: " + err.Error(), ID: mb.ID})
		}
	}
}

func (s *Service) pollMailbox(ctx context.Context, mb db.OutboundMailbox) error {
	_, pass, err := s.passwords(mb)
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
		return s.db.SetOutboundMailboxIMAPState(ctx, db.SetOutboundMailboxIMAPStateParams{
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
	return s.db.SetOutboundMailboxIMAPState(ctx, db.SetOutboundMailboxIMAPStateParams{ImapUidValidity: mb.ImapUidValidity, ImapLastUid: last, ID: mb.ID})
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
	oooSubjectRe = regexp.MustCompile(`(?i)\b(out of (the )?office|automatic reply|auto-?reply|autoreply|on vacation|away from)\b`)
	optOutRe     = regexp.MustCompile(`(?i)\b(unsubscribe|remove me|take me off|stop (emailing|e-mailing|contacting)|don'?t (email|contact) me|no thanks|no, thanks|not interested)\b`)
	notInterRe   = regexp.MustCompile(`(?i)\bnot interested\b`)
	angryRe      = regexp.MustCompile(`(?i)\b(spam|reported|reporting you|how did you get my (email|address))\b`)
)

// classify applies the simple rules. Everything they don't settle is
// needs_review, which the AI classifies through the conversations tool.
func classify(mb db.OutboundMailbox, in incoming) string {
	from := in.From
	if in.Report || strings.HasPrefix(from, "mailer-daemon@") || strings.HasPrefix(from, "postmaster@") {
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
func (s *Service) handleIncoming(ctx context.Context, mb db.OutboundMailbox, raw []byte) {
	in, err := parseIncoming(raw)
	if err != nil || in.From == normEmail(mb.Email) {
		return
	}
	class := classify(mb, in)

	// Match: threading headers first, then the failed recipient of a
	// bounce, then the sender's address.
	var sent *db.OutboundMessage
	for _, id := range append([]string{in.InReplyTo}, in.References...) {
		if id == "" {
			continue
		}
		if m, err := s.db.GetOutboundMessageByMessageID(ctx, id); err == nil && m.MailboxID == mb.ID {
			sent = &m
			break
		}
	}
	if sent == nil && class == "bounce" {
		for _, m := range msgIDsIn(in.Text) {
			if found, err := s.db.GetOutboundMessageByMessageID(ctx, m); err == nil && found.MailboxID == mb.ID {
				sent = &found
				break
			}
		}
	}
	var prospect db.OutboundProspect
	switch {
	case sent != nil:
		prospect, err = s.db.GetOutboundProspect(ctx, db.GetOutboundProspectParams{ID: sent.ProspectID, OrgID: mb.OrgID})
	case class == "bounce" && in.FailedRecipient != "":
		prospect, err = s.db.GetOutboundProspectByEmail(ctx, db.GetOutboundProspectByEmailParams{OrgID: mb.OrgID, Email: in.FailedRecipient})
	case class != "bounce":
		prospect, err = s.db.GetOutboundProspectByEmail(ctx, db.GetOutboundProspectByEmailParams{OrgID: mb.OrgID, Email: in.From})
	default:
		return
	}
	if err != nil {
		return // not a reply to outreach
	}
	if class == "bounce" && in.FailedRecipient != "" && in.FailedRecipient != prospect.Email {
		return // a bounce for some other mail
	}
	enrollment := sql.NullString{}
	if sent != nil && sent.EnrollmentID.Valid {
		enrollment = sent.EnrollmentID
	}
	handled := int64(1)
	if class == "needs_review" || class == "complaint" {
		handled = 0
	}
	if in.MessageID == "" {
		in.MessageID = uuid.NewString() + "@missing"
	}
	if _, err := s.db.CreateOutboundReply(ctx, db.CreateOutboundReplyParams{
		ID: uuid.NewString(), OrgID: mb.OrgID, MailboxID: mb.ID, EnrollmentID: enrollment,
		ProspectID: sql.NullString{String: prospect.ID, Valid: true}, FromEmail: in.From, Subject: in.Subject,
		Body: truncate(newTextOrAll(in.Text), 20000), MessageID: in.MessageID, InReplyTo: in.InReplyTo,
		Refs: strings.Join(in.References, " "), Classification: class, Handled: handled,
	}); err != nil {
		return // already recorded (ON CONFLICT) or failed; effects ran the first time
	}
	s.applyClassification(ctx, mb.OrgID, prospect, class)
}

// applyClassification stops the prospect's sequences and records what the
// reply means. Any human reply stops them; opt-outs, bounces and complaints
// also go on the brand's suppression list.
func (s *Service) applyClassification(ctx context.Context, orgID string, p db.OutboundProspect, class string) {
	enrollStatus, prospectStatus, suppressReason := "replied", "replied", ""
	switch class {
	case "out_of_office":
		return // the sequence carries on; its follow-up arrives after they're back
	case "bounce":
		enrollStatus, prospectStatus, suppressReason = "bounced", "bounced", "outbound bounce"
	case "unsubscribe":
		enrollStatus, prospectStatus, suppressReason = "unsubscribed", "unsubscribed", "outbound opt-out"
	case "complaint":
		enrollStatus, prospectStatus, suppressReason = "unsubscribed", "unsubscribed", "outbound complaint"
	case "not_interested":
		prospectStatus = "not_interested"
	case "interested":
		prospectStatus = "interested"
	case "wrong_person":
		prospectStatus = "not_interested"
	}
	if suppressReason != "" {
		s.suppress(ctx, orgID, p.Email, suppressReason)
	}
	if p.Status != "partner" {
		_ = s.db.SetOutboundProspectStatus(ctx, db.SetOutboundProspectStatusParams{Status: prospectStatus, ID: p.ID})
	}
	enrollments, err := s.db.ListActiveOutboundEnrollmentsForProspect(ctx, p.ID)
	if err != nil {
		return
	}
	for _, e := range enrollments {
		_ = s.db.SkipOpenOutboundMessagesForEnrollment(ctx, sql.NullString{String: e.ID, Valid: true})
		_ = s.db.SetOutboundEnrollmentStatus(ctx, db.SetOutboundEnrollmentStatusParams{Status: enrollStatus, ID: e.ID})
	}
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
