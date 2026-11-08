package outbound

import (
	"context"
	"fmt"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"

	"github.com/google/uuid"
)

// MailboxInput connects one mailbox by SMTP and IMAP. Provider "google" fills
// Google Workspace's hosts; the password is then an app password.
type MailboxInput struct {
	Email        string `json:"email"`
	FromName     string `json:"from_name"`
	Signature    string `json:"signature,omitempty"`
	Provider     string `json:"provider,omitempty"` // google, or empty for custom hosts
	Password     string `json:"password,omitempty"` // used for SMTP and IMAP unless each is given
	SMTPHost     string `json:"smtp_host,omitempty"`
	SMTPPort     int    `json:"smtp_port,omitempty"`
	SMTPUsername string `json:"smtp_username,omitempty"`
	SMTPPassword string `json:"smtp_password,omitempty"`
	IMAPHost     string `json:"imap_host,omitempty"`
	IMAPPort     int    `json:"imap_port,omitempty"`
	IMAPUsername string `json:"imap_username,omitempty"`
	IMAPPassword string `json:"imap_password,omitempty"`
	DailyCap     int    `json:"daily_cap,omitempty"` // at most 30
}

// MailboxResult says what happened to one mailbox in a batch.
type MailboxResult struct {
	Email string `json:"email"`
	ID    string `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

func (in *MailboxInput) fill() error {
	in.Email = normEmail(in.Email)
	if !strings.Contains(in.Email, "@") {
		return refuse("email is required")
	}
	if strings.TrimSpace(in.FromName) == "" {
		return refuse("from_name is required: the real person who stands behind this mailbox")
	}
	switch strings.ToLower(in.Provider) {
	case "google":
		in.SMTPHost, in.SMTPPort = "smtp.gmail.com", 587
		in.IMAPHost, in.IMAPPort = "imap.gmail.com", 993
	case "":
	default:
		return refuse("provider must be google, or leave it empty and pass smtp_host and imap_host")
	}
	if in.SMTPUsername == "" {
		in.SMTPUsername = in.Email
	}
	if in.IMAPUsername == "" {
		in.IMAPUsername = in.Email
	}
	if in.SMTPPassword == "" {
		in.SMTPPassword = in.Password
	}
	if in.IMAPPassword == "" {
		in.IMAPPassword = in.Password
	}
	if in.SMTPPort == 0 {
		in.SMTPPort = 587
	}
	if in.IMAPPort == 0 {
		in.IMAPPort = 993
	}
	if in.SMTPHost == "" || in.IMAPHost == "" || in.SMTPPassword == "" || in.IMAPPassword == "" {
		return refuse("smtp_host, imap_host and a password are required (or provider: google with an app password)")
	}
	if in.DailyCap <= 0 || in.DailyCap > MaxDailyCap {
		in.DailyCap = MaxDailyCap
	}
	return nil
}

// AddMailboxes connects a batch of mailboxes. Each must be on one of the
// brand's outreach domains and must log in to both SMTP and IMAP; a mailbox
// that fails is reported and not saved.
func (s *Service) AddMailboxes(ctx context.Context, org db.Organization, inputs []MailboxInput) ([]MailboxResult, error) {
	if len(inputs) > maxBatch {
		return nil, refuse("at most %d mailboxes per call", maxBatch)
	}
	if s.crypto == nil {
		return nil, refuse("ENCRYPTION_KEY is not set on this server, so it can't store mailbox passwords")
	}
	domains, err := s.db.ListOutreachDomains(ctx, org.ID)
	if err != nil {
		return nil, err
	}
	byName := map[string]db.DomainIdentity{}
	for _, d := range domains {
		byName[d.Domain] = d
	}
	out := make([]MailboxResult, 0, len(inputs))
	for _, in := range inputs {
		res := MailboxResult{Email: normEmail(in.Email)}
		mb, err := s.addMailbox(ctx, org, byName, in)
		if err != nil {
			res.Error = err.Error()
		} else {
			res.ID = mb.ID
		}
		out = append(out, res)
	}
	return out, nil
}

func (s *Service) addMailbox(ctx context.Context, org db.Organization, domains map[string]db.DomainIdentity, in MailboxInput) (db.Mailbox, error) {
	if err := in.fill(); err != nil {
		return db.Mailbox{}, err
	}
	d, ok := domains[domainOf(in.Email)]
	if !ok {
		return db.Mailbox{}, refuse("%s is not on one of this brand's outreach domains; add the domain first with setup_sending(action: add_domain)", in.Email)
	}
	if existing, err := s.db.GetMailboxByEmail(ctx, in.Email); err == nil {
		if existing.OrgID != org.ID {
			return db.Mailbox{}, refuse("%s is connected to another brand", in.Email)
		}
		return existing, nil
	}
	mb := db.Mailbox{
		Email: in.Email, FromName: in.FromName,
		SmtpHost: in.SMTPHost, SmtpPort: int64(in.SMTPPort), SmtpUsername: in.SMTPUsername,
		ImapHost: in.IMAPHost, ImapPort: int64(in.IMAPPort), ImapUsername: in.IMAPUsername,
	}
	if err := email.CheckMailboxLogin(email.MailboxLogin{Email: in.Email, Host: in.SMTPHost, Port: in.SMTPPort, Username: in.SMTPUsername, Password: in.SMTPPassword}); err != nil {
		return db.Mailbox{}, refuse("SMTP login failed for %s: %v. For Google, turn on 2-step verification and use an app password.", in.Email, err)
	}
	if err := testIMAP(mb, in.IMAPPassword); err != nil {
		return db.Mailbox{}, refuse("IMAP login failed for %s: %v", in.Email, err)
	}
	smtpEnc, err := s.crypto.EncryptString(in.SMTPPassword)
	if err != nil {
		return db.Mailbox{}, err
	}
	imapEnc, err := s.crypto.EncryptString(in.IMAPPassword)
	if err != nil {
		return db.Mailbox{}, err
	}
	created, err := s.db.CreateMailbox(ctx, db.CreateMailboxParams{
		ID: uuid.NewString(), OrgID: org.ID, DomainID: d.ID, Email: in.Email,
		FromName: strings.TrimSpace(in.FromName), Signature: in.Signature,
		SmtpHost: in.SMTPHost, SmtpPort: int64(in.SMTPPort), SmtpUsername: in.SMTPUsername, SmtpPasswordEnc: smtpEnc,
		ImapHost: in.IMAPHost, ImapPort: int64(in.IMAPPort), ImapUsername: in.IMAPUsername, ImapPasswordEnc: imapEnc,
		DailyCap: int64(in.DailyCap),
	})
	if err != nil {
		return created, err
	}
	// Start on the domain's DNS state: a failing domain keeps it paused.
	if d.VerificationStatus.String == "failed" {
		_ = s.db.SetMailboxStatus(ctx, db.SetMailboxStatusParams{
			Status: "paused", PausedReason: "DNS: " + d.Domain + " records are failing", ID: created.ID,
		})
	}
	return created, nil
}

// imapPassword reads a mailbox's IMAP password. (SMTP sending reads its own,
// inside email.Send.)
func (s *Service) imapPassword(mb db.Mailbox) (string, error) {
	if s.crypto == nil {
		return "", fmt.Errorf("ENCRYPTION_KEY is not set")
	}
	return s.crypto.DecryptString(mb.ImapPasswordEnc)
}
