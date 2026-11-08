package sequences

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func status(err error) int {
	var ce *errorx.CodeError
	if errors.As(err, &ce) {
		return ce.Code
	}
	return 0
}

// The admin app enrolls a contact with the member's session, through the
// same Enroll as the SDK route, in the brand the member works in.
func TestAdminEnrollInSequence(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	for _, q := range []string{
		`INSERT INTO users (id, email, password_hash, name, role, status, email_verified) VALUES ('mike', 'mike@example.com', 'x', 'M', 'agent', 'active', 1)`,
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('alpha-id', 'alpha', 'alpha', 'k-a'), ('bravo-id', 'bravo', 'bravo', 'k-b')`,
		`INSERT INTO org_users (org_id, user_id, role) VALUES ('alpha-id', 'mike', 'member')`,
		`INSERT INTO contacts (id, org_id, name, email) VALUES ('c-a', 'alpha-id', 'Ana', 'ana@example.com'), ('c-b', 'bravo-id', 'Ana', 'ana@example.com')`,
		`INSERT INTO email_sequences (id, org_id, slug, name, trigger_event, is_active) VALUES ('seq-a', 'alpha-id', 'nurture', 'Nurture', 'manual', 1), ('seq-b', 'bravo-id', 'nurture', 'Nurture', 'manual', 1)`,
		`INSERT INTO email_templates (id, org_id, sequence_id, position, subject, html_body) VALUES ('t-a', 'alpha-id', 'seq-a', 1, 'Hi', '<p>Hi</p>')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	store := db.NewStore(conn)
	s := &svc.ServiceContext{DB: store}
	user, err := access.Active(context.Background(), store, "mike")
	require.NoError(t, err)
	in := func(brand string) context.Context {
		return access.WithBrand(access.WithUser(context.Background(), user), brand)
	}
	req := &types.EnrollSequenceRequest{SequenceSlug: "nurture", Email: "ana@example.com"}

	resp, err := NewAdminEnrollInSequenceLogic(in("alpha-id"), s).AdminEnrollInSequence(req)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	var queued int
	require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM email_queue WHERE contact_id = 'c-a' AND template_id = 't-a'`).Scan(&queued))
	assert.Equal(t, 1, queued)

	// Not a member of bravo: refused before anything is read.
	_, err = NewAdminEnrollInSequenceLogic(in("bravo-id"), s).AdminEnrollInSequence(req)
	assert.Equal(t, 403, status(err))

	// Errors say what is wrong, for the modal to show.
	_, err = NewAdminEnrollInSequenceLogic(in("alpha-id"), s).AdminEnrollInSequence(&types.EnrollSequenceRequest{SequenceSlug: "nurture", Email: "nobody@example.com"})
	assert.Equal(t, 404, status(err))
	assert.EqualError(t, err, "contact not found")
}
