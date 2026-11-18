package email

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAddress = "Acme Inc\n1 Main St, Denver, CO 80202"

var testAddressHTML = "Acme Inc<br>1 Main St, Denver, CO 80202"

// Every marketing sequence step carries the brand's mailing address and the
// unsubscribe link, once: the built-in layouts in their footer, a step with
// template_type 'none' or its own design appended. A transactional step
// carries neither.
func TestSequenceEmail_CarriesTheMailingAddress(t *testing.T) {
	sender, store := newTestService(t)
	seq := NewSequenceServiceWithBaseURL(store, sender, "https://mail.example.com")
	link := `href="https://mail.example.com/u/tok"`
	base := db.GetPendingEmailsRow{
		Name: "Ana", Email: "ana@x.com", Subject: "Hi", HtmlBody: "<p>Hello</p>",
		TrackingToken:  sql.NullString{String: "tok", Valid: true},
		MailingAddress: testAddress,
	}
	for name, mutate := range map[string]func(*db.GetPendingEmailsRow){
		"simple":  func(r *db.GetPendingEmailsRow) {},
		"branded": func(r *db.GetPendingEmailsRow) { r.TemplateType = sql.NullString{String: "branded", Valid: true} },
		"none":    func(r *db.GetPendingEmailsRow) { r.TemplateType = sql.NullString{String: "none", Valid: true} },
		"own design": func(r *db.GetPendingEmailsRow) {
			r.DesignHtml = "<html><body><div>BRAND</div>{{content}}</body></html>"
		},
		"own design with its own unsubscribe link": func(r *db.GetPendingEmailsRow) {
			r.DesignHtml = `<html><body>{{content}}<a href="{{unsubscribe_url}}">Leave</a></body></html>`
		},
	} {
		t.Run(name, func(t *testing.T) {
			row := base
			mutate(&row)
			_, html := seq.renderQueued(context.Background(), row)
			assert.Equal(t, 1, strings.Count(html, testAddressHTML), html)
			assert.Equal(t, 1, strings.Count(html, link), html)
			assert.Contains(t, html, "<p>Hello</p>")
		})
	}

	t.Run("transactional", func(t *testing.T) {
		for _, tt := range []string{"simple", "branded", "none"} {
			row := base
			row.TemplateType = sql.NullString{String: tt, Valid: true}
			row.IsTransactional = sql.NullInt64{Int64: 1, Valid: true}
			_, html := seq.renderQueued(context.Background(), row)
			assert.NotContains(t, html, "Acme Inc", tt)
			assert.NotContains(t, html, "/u/tok", tt)
		}
	})
}

// A newsletter campaign carries the brand's mailing address in its HTML
// footer and its plain text, once, design or not.
func TestCampaignEmail_CarriesTheMailingAddress(t *testing.T) {
	s := NewService(nil, nil)
	s.baseURL = "https://mail.example.com"
	m := s.CampaignMessage(CampaignEmail{
		To: "ana@x.com", Name: "Ana", Subject: "Hi", TrackingToken: "tok",
		HTML: "<html><body><p>News</p></body></html>", Text: "News",
		MailingAddress: testAddress,
	})
	assert.Equal(t, 1, strings.Count(m.HTML, testAddressHTML), m.HTML)
	assert.Less(t, strings.Index(m.HTML, testAddressHTML), strings.Index(m.HTML, "</body>"))
	assert.Contains(t, m.Text, "1 Main St, Denver, CO 80202")

	// A design that already shows the address gets no second one.
	m = s.CampaignMessage(CampaignEmail{
		To: "ana@x.com", Subject: "Hi", TrackingToken: "tok", HTML: "<p>News</p>",
		DesignHTML:     "<html><body>{{content}}<footer>" + testAddressHTML + "</footer></body></html>",
		MailingAddress: testAddress,
	})
	assert.Equal(t, 1, strings.Count(m.HTML, testAddressHTML), m.HTML)
}

// Outreach and opt-in mail format the address the one same way.
func TestMailingAddress_OneFormat(t *testing.T) {
	assert.Equal(t, "Acme Inc\n1 Main St", MailingAddress("  Acme Inc \r\n 1 Main St \n"))
	assert.Equal(t, "A &amp; B<br>1 Main St", mailingAddressHTML("A & B\n1 Main St"))
	assert.Equal(t, "", MailingAddress("  "))
}

// A marketing step of a brand with no mailing address (a sequence turned on
// before activation needed one) is not sent: it is marked failed with the
// reason, never sent without the address.
func TestSequenceEmail_RefusedWithoutAMailingAddress(t *testing.T) {
	sender, store := newTestService(t)
	for _, q := range []string{
		`INSERT INTO contacts (id, org_id, name, email) VALUES ('c1', 'org-a', 'C', 'c1@x.example')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'p1', 'org-a', 'L', 'l')`,
		`INSERT INTO email_sequences (id, org_id, list_id, slug, name, trigger_event) VALUES ('seq', 'org-a', 1, 's', 'S', 'manual')`,
		`INSERT INTO email_templates (id, org_id, sequence_id, position, subject, html_body) VALUES ('t1', 'org-a', 'seq', 1, 's', 'b')`,
		`INSERT INTO email_queue (id, contact_id, template_id, scheduled_for, tracking_token) VALUES ('q1', 'c1', 't1', '2000-01-01T00:00:00Z', 'tok1')`,
	} {
		_, err := store.GetDB().Exec(q)
		require.NoError(t, err, q)
	}
	seq := NewSequenceServiceWithBaseURL(store, sender, "https://outlet.example.com")
	sent, err := seq.ProcessPendingEmails(context.Background(), 10)
	require.NoError(t, err)
	assert.Equal(t, 0, sent)

	var status, reason string
	require.NoError(t, store.GetDB().QueryRow(`SELECT status, error_message FROM email_queue WHERE id = 'q1'`).Scan(&status, &reason))
	assert.Equal(t, "failed", status)
	assert.Equal(t, NoMailingAddress, reason)
}

// Activating opt-in mail is refused, with the reason, for a brand without a
// mailing address.
func TestCheckMailingAddress(t *testing.T) {
	_, store := newTestService(t)
	ctx := context.Background()
	err := CheckMailingAddress(ctx, store, "org-a")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mailing address")

	_, err = store.GetDB().Exec(`UPDATE organizations SET mailing_address = '1 Main St' WHERE id = 'org-a'`)
	require.NoError(t, err)
	assert.NoError(t, CheckMailingAddress(ctx, store, "org-a"))
}
