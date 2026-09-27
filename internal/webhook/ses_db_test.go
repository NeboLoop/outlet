package webhook

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// A bounce and a complaint are recorded with an id (the queries used to
// insert NULL, and RETURNING * then failed to scan: nothing was recorded as
// handled and the contact was never blocked).
func TestProcessBounceAndComplaintRecord(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db"))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	svcCtx := &svc.ServiceContext{DB: db.NewStore(conn)}
	ctx := context.Background()

	processBounce(ctx, svcCtx, "org-1", &sesNotification{
		NotificationType: "Bounce",
		Bounce: &sesBounce{BounceType: "Permanent", BounceSubType: "General",
			BouncedRecipients: []sesBounceRecipient{{EmailAddress: "Gone@Example.com"}}},
		Mail: sesMailInfo{MessageId: "m-1", Source: "hello@mail.example.com"},
	}, []byte("{}"))
	processComplaint(ctx, svcCtx, "org-1", &sesNotification{
		NotificationType: "Complaint",
		Complaint:        &sesComplaint{ComplaintFeedbackType: "abuse", ComplainedRecipients: []sesComplaintRecipient{{EmailAddress: "angry@example.com"}}},
		Mail:             sesMailInfo{MessageId: "m-2", Source: "hello@mail.example.com"},
	}, []byte("{}"))

	var id string
	require.NoError(t, conn.QueryRow(`SELECT id FROM email_bounce WHERE email_lower = 'gone@example.com'`).Scan(&id))
	require.NotEmpty(t, id)
	require.NoError(t, conn.QueryRow(`SELECT id FROM email_complaint WHERE email_lower = 'angry@example.com'`).Scan(&id))
	require.NotEmpty(t, id)

	// A second bounce for the same address updates the row and keeps its id.
	b, err := svcCtx.DB.CreateEmailBounce(ctx, db.CreateEmailBounceParams{ID: "other", Email: "gone@example.com", EmailForLower: "gone@example.com", BounceType: "Transient"})
	require.NoError(t, err)
	require.NotEqual(t, "other", b.ID)
}
