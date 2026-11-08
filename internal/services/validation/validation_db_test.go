package validation

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/config"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/services/emailval"

	_ "modernc.org/sqlite"
)

// newTestService opens a migrated SQLite database in a temp dir with one org.
func newTestService(t *testing.T) (*Service, *db.Store, *sql.DB) {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := migrations.Run(conn); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	mustExec(t, conn, `INSERT INTO organizations (id, name, slug, api_key) VALUES ('org1', 'Org', 'org', 'k1'), ('org2', 'Other', 'other', 'k2')`)
	store := db.NewStore(conn)
	return New(store, config.ValidationConfig{CacheDays: 30, Concurrency: 2}, nil), store, conn
}

func mustExec(t *testing.T, conn *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := conn.Exec(q, args...); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
}

// seedCache stores a probe result so Validate never touches the network.
func seedCache(t *testing.T, store *db.Store, email, status string) {
	t.Helper()
	b, _ := json.Marshal(emailval.Result{Input: email, Normalized: email, SyntaxOK: true, DomainOK: true, HasMX: true, Status: status, Domain: "example.com", Account: "x"})
	if err := store.UpsertValidationCache(context.Background(), db.UpsertValidationCacheParams{EmailLower: email, Result: string(b)}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRecordsUsageAndContact(t *testing.T) {
	ctx := context.Background()
	svc, store, conn := newTestService(t)
	mustExec(t, conn, `INSERT INTO contacts (id, org_id, name, email) VALUES ('c1', 'org1', 'A', 'Ann@Example.com')`)
	seedCache(t, store, "ann@example.com", emailval.StatusValid)
	seedCache(t, store, "bob@example.com", emailval.StatusValid)
	seedCache(t, store, "cat@example.com", emailval.StatusValid)
	// Signals from other orgs on the instance.
	mustExec(t, conn, `INSERT INTO suppression_list (org_id, email, email_lower, source) VALUES ('org2', 'bob@example.com', 'bob@example.com', 'manual')`)
	mustExec(t, conn, `INSERT INTO email_complaint (id, email, email_lower) VALUES ('x1', 'cat@example.com', 'cat@example.com')`)

	res, err := svc.Validate(ctx, "org1", "Ann@Example.com", "api")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != emailval.StatusValid || res.Address != "Ann@Example.com" || !res.MxFound || res.ProcessedAt == "" {
		t.Errorf("unexpected result %+v", res)
	}
	var status, sub sql.NullString
	if err := conn.QueryRow(`SELECT validation_status, validation_sub_status FROM contacts WHERE id = 'c1'`).Scan(&status, &sub); err != nil {
		t.Fatal(err)
	}
	if status.String != emailval.StatusValid {
		t.Errorf("contact validation_status = %q", status.String)
	}

	if res, _ := svc.Validate(ctx, "org1", "bob@example.com", "admin"); res.Status != emailval.StatusDoNotMail || res.SubStatus != emailval.SubGlobalSuppression {
		t.Errorf("bob = %s/%s, want global suppression", res.Status, res.SubStatus)
	}
	if res, _ := svc.Validate(ctx, "org1", "cat@example.com", "mcp"); res.Status != emailval.StatusAbuse {
		t.Errorf("cat = %s, want abuse", res.Status)
	}

	now := time.Now().UTC()
	rows, err := store.CountValidationUsageBySource(ctx, db.CountValidationUsageBySourceParams{
		OrgID:     "org1",
		FromTime:  sql.NullString{String: now.Add(-time.Hour).Format(TimeLayout), Valid: true},
		UntilTime: sql.NullString{String: now.Add(time.Hour).Format(TimeLayout), Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	usage := map[string]int64{}
	for _, r := range rows {
		usage[r.Source] = r.Count
	}
	if usage["api"] != 1 || usage["admin"] != 1 || usage["mcp"] != 1 {
		t.Errorf("usage = %v", usage)
	}
}

func TestJobLifecycle(t *testing.T) {
	ctx := context.Background()
	svc, store, conn := newTestService(t)

	// Upload job: dedupe, append while pending, then start.
	job, err := svc.CreateJob(ctx, "org1", "", "", []string{"a@x.com", "A@X.com", " ", "b@y.com"})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "pending" || job.Total != 2 || job.Source != "upload" {
		t.Fatalf("created job %+v", job)
	}
	if job, err = svc.AppendEmails(ctx, "org1", job.Id, []string{"b@y.com", "c@y.com"}); err != nil || job.Total != 3 {
		t.Fatalf("append: %+v %v", job, err)
	}
	if _, err := svc.Job(ctx, "org2", job.Id); err == nil {
		t.Error("another org can read the job")
	}
	if job, err = svc.StartJob(ctx, "org1", job.Id); err != nil || job.Status != "processing" {
		t.Fatalf("start: %+v %v", job, err)
	}
	if _, err := svc.AppendEmails(ctx, "org1", job.Id, []string{"d@y.com"}); err == nil {
		t.Error("append allowed after start")
	}

	// Worker path: claim, finalize two, schedule a retry for one.
	jobID := sql.NullString{String: job.Id, Valid: true}
	rows, err := store.ClaimPendingValidations(ctx, db.ClaimPendingValidationsParams{JobID: jobID, BatchSize: 10})
	if err != nil || len(rows) != 3 {
		t.Fatalf("claim: %d %v", len(rows), err)
	}
	if again, _ := store.ClaimPendingValidations(ctx, db.ClaimPendingValidationsParams{JobID: jobID, BatchSize: 10}); len(again) != 0 {
		t.Errorf("leased rows claimed twice: %d", len(again))
	}
	for i, r := range rows {
		p := db.UpdateEmailValidationResultParams{ID: r.ID, Status: emailval.StatusInvalid, Domain: r.Domain, Attempts: 1}
		if i == 0 {
			p.Status = "pending"
			p.RetryAt = sql.NullString{String: time.Now().UTC().Add(time.Hour).Format(TimeLayout), Valid: true}
		}
		if err := store.UpdateEmailValidationResult(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.RefreshValidationJobProgress(ctx, job.Id); err != nil {
		t.Fatal(err)
	}
	if n, _ := store.CountPendingValidations(ctx, jobID); n != 1 {
		t.Errorf("pending = %d, want 1", n)
	}
	info, err := svc.Job(ctx, "org1", job.Id)
	if err != nil || info.Total != 3 || info.Processed != 2 || info.Invalid != 2 {
		t.Errorf("job info %+v %v", info, err)
	}

	// Apply invalid results to the suppression list.
	n, err := store.SuppressValidationJobResults(ctx, db.SuppressValidationJobResultsParams{JobID: jobID, OrgID: "org1", Statuses: []string{"invalid"}})
	if err != nil || n != 2 {
		t.Errorf("suppressed %d, %v", n, err)
	}
	// Validation-sourced suppressions do not count as global suppression.
	if g, _ := store.IsEmailGloballySuppressed(ctx, rows[1].Email); g != 0 {
		t.Error("validation suppression fed back into signals")
	}

	if job, err = svc.CancelJob(ctx, "org1", job.Id); err != nil || job.Status != "cancelled" {
		t.Errorf("cancel: %+v %v", job, err)
	}

	// List job: queues active subscribers and starts immediately.
	mustExec(t, conn, `INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (7, 'p7', 'org1', 'News', 'news')`)
	mustExec(t, conn, `INSERT INTO contacts (id, org_id, name, email) VALUES ('c1', 'org1', 'A', 'a@x.com'), ('c2', 'org1', 'B', 'b@x.com')`)
	mustExec(t, conn, `INSERT INTO list_subscribers (id, list_id, contact_id, status) VALUES ('s1', 7, 'c1', 'active'), ('s2', 7, 'c2', 'unsubscribed')`)
	listJob, err := svc.CreateJob(ctx, "org1", "", strconv.Itoa(7), nil)
	if err != nil || listJob.Status != "processing" || listJob.Total != 1 || listJob.Name != "News" || listJob.ListId != "7" {
		t.Errorf("list job %+v %v", listJob, err)
	}
	if _, err := svc.CreateJob(ctx, "org2", "", "7", nil); err == nil {
		t.Error("another org can validate the list")
	}
}

func TestSignalQueries(t *testing.T) {
	ctx := context.Background()
	_, store, conn := newTestService(t)
	mustExec(t, conn, `INSERT INTO email_campaigns (id, org_id, name, subject, html_body) VALUES ('k1', 'org2', 'C', 'S', '<p>')`)
	mustExec(t, conn, `INSERT INTO contacts (id, org_id, name, email) VALUES ('c1', 'org2', 'A', 'Ann@Corp.com'), ('c2', 'org2', 'B', 'bob@corp.com')`)
	mustExec(t, conn, `INSERT INTO campaign_sends (id, campaign_id, contact_id, status, sent_at, opened_at) VALUES
		('s1', 'k1', 'c1', 'sent', datetime('now', '-1 day'), datetime('now', '-1 day')),
		('s2', 'k1', 'c2', 'bounced', datetime('now', '-1 day'), NULL)`)

	e, err := store.GetEmailEngagement(ctx, "ann@corp.com")
	if err != nil || e.Sends != 1 || parseTime(e.LastOpenedAt).IsZero() {
		t.Errorf("engagement %+v %v", e, err)
	}
	d, err := store.GetDomainDeliveryStats(ctx, "corp.com")
	if err != nil || d.Delivered != 1 || d.Bounced != 1 {
		t.Errorf("domain stats %+v %v", d, err)
	}
}
