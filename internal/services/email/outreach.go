package email

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"

	"github.com/google/uuid"
)

// Outreach sequences (sequence_type 'outreach') run on the same sequence
// engine as every other sequence. What differs is how a step is written and
// queued: from the person's one mailbox, in one thread, as plain text with
// the brand's opt-out line and mailing address, inside a weekday sending
// window, and as a draft when the sequence wants approval.

// TimeLayout is how queue times are stored for outreach: UTC, in order as text.
const TimeLayout = time.DateTime

var spinRe = regexp.MustCompile(`\{([^{}]*\|[^{}]*)\}`)

// Spin picks one option from each {a|b|c} group, so no two emails are
// byte-identical. Merge tags ({{name}}) have no pipe, so they are left alone.
func Spin(s string, r *rand.Rand) string {
	return spinRe.ReplaceAllStringFunc(s, func(m string) string {
		opts := strings.Split(m[1:len(m)-1], "|")
		return opts[r.IntN(len(opts))]
	})
}

// NextWindow is the earliest moment at or after t inside the weekday sending
// window [startHour, endHour) in loc.
func NextWindow(t time.Time, loc *time.Location, startHour, endHour int) time.Time {
	l := t.In(loc)
	for i := 0; i < 8; i++ {
		day := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
		open := day.Add(time.Duration(startHour) * time.Hour)
		closes := day.Add(time.Duration(endHour) * time.Hour)
		if wd := day.Weekday(); wd != time.Saturday && wd != time.Sunday && l.Before(closes) {
			if l.Before(open) {
				return open.UTC()
			}
			return l.UTC()
		}
		l = day.AddDate(0, 0, 1)
	}
	return t.UTC()
}

// LoadLocation is the first valid IANA zone of names, or UTC.
func LoadLocation(names ...string) *time.Location {
	for _, n := range names {
		if n == "" {
			continue
		}
		if loc, err := time.LoadLocation(n); err == nil {
			return loc
		}
	}
	return time.UTC
}

// ErrMissingData means a step names a field the person doesn't have. The
// person's sequence stops rather than send "Hi ,".
var ErrMissingData = errors.New("missing data")

// OutreachStep is one written outreach email, before it is saved.
type OutreachStep struct {
	Subject, Body string
	Mailbox       db.Mailbox
	SendAfter     time.Time
	Status        string // draft or pending
	InReplyTo     string
	References    []string
}

// OutreachTemplate is one step of an outreach sequence as written: plain
// text with merge tags and {a|b} variants. An empty subject on a follow-up
// replies in the thread.
type OutreachTemplate struct {
	Subject, Body string
	DelayHours    int64
}

// WriteOutreachStep writes one step of an outreach sequence for one person,
// without saving it. Draft previews and queueing both use it.
func (s *SequenceService) WriteOutreachStep(ctx context.Context, seq db.EmailSequence, step OutreachTemplate, contact db.Contact, mb db.Mailbox, position int64, now time.Time, r *rand.Rand) (OutreachStep, error) {
	org, err := s.db.GetOrganizationByID(ctx, seq.OrgID.String)
	if err != nil {
		return OutreachStep{}, err
	}
	fields := map[string]string{}
	if seq.ListID.Valid {
		fields = s.GetCustomFieldsForContact(ctx, contact.ID, seq.ListID.Int64)
	}
	fields["sender_name"] = mb.FromName
	if f := strings.Fields(mb.FromName); len(f) > 0 {
		fields["sender_first"] = f[0]
	}

	thread, err := s.db.ListOutreachThread(ctx, db.ListOutreachThreadParams{ContactID: sql.NullString{String: contact.ID, Valid: true}, SequenceID: sql.NullString{String: seq.ID, Valid: true}})
	if err != nil {
		return OutreachStep{}, err
	}
	subjectTmpl := step.Subject
	if strings.TrimSpace(subjectTmpl) == "" && len(thread) > 0 {
		subjectTmpl = "Re: " + strings.TrimPrefix(thread[0].Subject.String, "Re: ")
	}
	subject, missing := FillTags(Spin(subjectTmpl, r), contact.Name, contact.Email, fields)
	body, missingBody := FillTags(Spin(step.Body, r), contact.Name, contact.Email, fields)
	if missing = append(missing, missingBody...); len(missing) > 0 {
		return OutreachStep{}, fmt.Errorf("%w: no value for %s", ErrMissingData, strings.Join(dedupe(missing), ", "))
	}
	body = strings.TrimRight(body, "\n ")
	if sig := strings.TrimSpace(mb.Signature); sig != "" {
		body += "\n\n" + sig
	}
	body += "\n\n" + strings.TrimSpace(org.OutboundOptOutLine) + "\n" + strings.TrimSpace(org.MailingAddress)

	base := now
	if len(thread) > 0 {
		if last, err := time.ParseInLocation(TimeLayout, thread[len(thread)-1].SentAt.String, time.UTC); err == nil {
			base = last.Add(time.Duration(step.DelayHours) * time.Hour)
		}
	}
	if base.Before(now) {
		base = now
	}
	loc := LoadLocation(fields["timezone"], seq.SendTimezone.String, org.OutboundTimezone)
	out := OutreachStep{
		Subject: subject, Body: body, Mailbox: mb,
		SendAfter: NextWindow(base, loc, int(seq.WindowStartHour), int(seq.WindowEndHour)),
		Status:    "pending",
	}
	if seq.ApprovalMode == "every" || (seq.ApprovalMode == "first" && position == 1) {
		out.Status = "draft"
	}
	for _, m := range thread {
		out.References = append(out.References, m.MessageID.String)
	}
	if len(thread) > 0 {
		out.InReplyTo = thread[len(thread)-1].MessageID.String
	}
	return out, nil
}

func dedupe(s []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// PickMailbox is the mailbox a person's outreach comes from: the one already
// chosen for them in this sequence, or the least-used active one of the pool
// carrying the brand's outreach (any of the brand's mailboxes when outreach
// rides another provider: the mailbox is then the sender's address).
func (s *SequenceService) PickMailbox(ctx context.Context, seq db.EmailSequence, state db.ContactSequenceState) (db.Mailbox, error) {
	if state.MailboxID.Valid {
		return s.db.GetMailbox(ctx, db.GetMailboxParams{ID: state.MailboxID.String, OrgID: seq.OrgID.String})
	}
	p, err := s.db.GetPathwayForKind(ctx, db.GetPathwayForKindParams{OrgID: seq.OrgID.String, Kind: KindOutreach.Name()})
	if errors.Is(err, sql.ErrNoRows) {
		return db.Mailbox{}, errors.New("no sending pathway carries this brand's outreach")
	}
	if err != nil {
		return db.Mailbox{}, err
	}
	var all []db.Mailbox
	if p.Provider == ProviderMailboxes {
		all, err = s.db.ListPoolMailboxes(ctx, sql.NullString{String: p.ID, Valid: true})
	} else {
		all, err = s.db.ListMailboxes(ctx, seq.OrgID.String)
	}
	if err != nil {
		return db.Mailbox{}, err
	}
	load := map[string]int64{}
	if rows, err := s.db.CountMailboxStatesForSequence(ctx, sql.NullString{String: seq.ID, Valid: true}); err == nil {
		for _, r := range rows {
			load[r.MailboxID.String] = r.N
		}
	}
	var best *db.Mailbox
	for i := range all {
		if all[i].Status != "active" {
			continue
		}
		if best == nil || load[all[i].ID] < load[best.ID] {
			best = &all[i]
		}
	}
	if best == nil {
		return db.Mailbox{}, errors.New("no active mailbox")
	}
	if err := s.db.SetSequenceStateMailbox(ctx, db.SetSequenceStateMailboxParams{MailboxID: sql.NullString{String: best.ID, Valid: true}, ID: state.ID}); err != nil {
		return db.Mailbox{}, err
	}
	return *best, nil
}

// queueOutreach writes and queues the next step of a person's outreach
// sequence. A step naming a field the person lacks stops their sequence.
func (s *SequenceService) queueOutreach(ctx context.Context, seq db.EmailSequence, contactID string, position int64) error {
	state, err := s.db.GetSequenceStateFull(ctx, db.GetSequenceStateFullParams{ContactID: sql.NullString{String: contactID, Valid: true}, SequenceID: sql.NullString{String: seq.ID, Valid: true}})
	if err != nil || state.IsActive.Int64 != 1 || state.StoppedReason != "" {
		return err
	}
	step, err := s.db.GetNextTemplate(ctx, db.GetNextTemplateParams{SequenceID: sql.NullString{String: seq.ID, Valid: true}, Position: position})
	if err != nil {
		return nil // no more steps
	}
	contact, err := s.db.GetContactByID(ctx, contactID)
	if err != nil {
		return err
	}
	if s.gate != nil {
		org, err := s.db.GetOrganizationByID(ctx, seq.OrgID.String)
		if err != nil {
			return err
		}
		if stop := s.gate(ctx, org, contact); stop != "" {
			return s.db.StopSequenceState(ctx, db.StopSequenceStateParams{StoppedReason: stop, ID: state.ID})
		}
	}
	mb, err := s.PickMailbox(ctx, seq, state)
	if err != nil {
		return err
	}
	tmpl := OutreachTemplate{Subject: step.Subject, Body: step.PlainText.String, DelayHours: step.DelayHours}
	w, err := s.WriteOutreachStep(ctx, seq, tmpl, contact, mb, position, s.now().UTC(), rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0)))
	if errors.Is(err, ErrMissingData) {
		_ = s.db.StopSequenceState(ctx, db.StopSequenceStateParams{StoppedReason: err.Error(), ID: state.ID})
		return err
	}
	if err != nil {
		return err
	}
	_, domain, _ := cutAt(mb.Email)
	_, err = s.db.QueueOutreachEmail(ctx, db.QueueOutreachEmailParams{
		ID: uuid.NewString(), ContactID: sql.NullString{String: contactID, Valid: true},
		TemplateID: sql.NullString{String: step.ID, Valid: true}, ScheduledFor: w.SendAfter.Format(TimeLayout),
		Status: sql.NullString{String: w.Status, Valid: true}, TrackingToken: sql.NullString{String: generateTrackingToken(), Valid: true},
		MailboxID: sql.NullString{String: mb.ID, Valid: true},
		Subject:   sql.NullString{String: w.Subject, Valid: true}, Body: sql.NullString{String: w.Body, Valid: true},
		MessageID:  sql.NullString{String: uuid.NewString() + "@" + domain, Valid: true},
		InReplyTo:  sql.NullString{String: w.InReplyTo, Valid: w.InReplyTo != ""},
		Refs:       sql.NullString{String: strings.Join(w.References, " "), Valid: len(w.References) > 0},
		AuthoredBy: seq.AuthoredBy,
	})
	return err
}

// AdvanceAfterSend moves a person's sequence on after one of its emails went:
// the next step is queued, or the sequence completes. Every sequence, outreach
// or not, advances this one way.
func (s *SequenceService) AdvanceAfterSend(ctx context.Context, contactID, templateID string) {
	s.advanceSequence(ctx, db.GetPendingEmailsRow{
		ContactID:  sql.NullString{String: contactID, Valid: true},
		TemplateID: sql.NullString{String: templateID, Valid: true},
	})
}
