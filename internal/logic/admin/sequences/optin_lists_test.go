package sequences

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// Only an outreach sequence goes on an outreach list: an autoresponder is
// refused on one when it is created, moved there or retyped, and an entry
// rule never starts one from an outreach list. A sequence already there can
// still be renamed or turned off.
func TestSequences_StayOffOutreachLists(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	for _, q := range []string{
		`INSERT INTO users (id, email, password_hash, name, role, status, email_verified) VALUES ('mike', 'mike@example.com', 'x', 'M', 'agent', 'active', 1)`,
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('alpha-id', 'alpha', 'alpha', 'k-a')`,
		`INSERT INTO org_users (org_id, user_id, role) VALUES ('alpha-id', 'mike', 'admin')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'p1', 'alpha-id', 'News', 'news')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug, kind) VALUES (2, 'p2', 'alpha-id', 'Utah dental', 'utah-dental', 'outreach')`,
		// An autoresponder put on the outreach list before the rule.
		`INSERT INTO email_sequences (id, org_id, list_id, slug, name, trigger_event, sequence_type) VALUES ('old', 'alpha-id', 2, 'old', 'Old', 'signup', 'lifecycle')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	store := db.NewStore(conn)
	user, err := access.Active(context.Background(), store, "mike")
	require.NoError(t, err)
	ctx := access.WithBrand(access.WithUser(context.Background(), user), "alpha-id")
	s := &svc.ServiceContext{DB: store}
	const refusal = `"Utah dental" is an outreach list`
	create := func(list, kind string) (*types.SequenceInfo, error) {
		return NewCreateSequenceLogic(ctx, s).CreateSequence(&types.CreateSequenceRequest{
			ListId: list, Slug: list + kind, Name: kind, TriggerEvent: "signup", SequenceType: kind,
		})
	}

	_, err = create("2", "lifecycle")
	require.Error(t, err)
	assert.Contains(t, err.Error(), refusal)
	_, err = create("2", "")
	require.Error(t, err, "no type is lifecycle")
	welcome, err := create("1", "lifecycle")
	require.NoError(t, err)
	cold, err := create("2", "outreach")
	require.NoError(t, err)

	outreachList := "2"
	_, err = NewUpdateSequenceLogic(ctx, s).UpdateSequence(&types.UpdateSequenceRequest{Id: welcome.Id, ListId: &outreachList, IsActive: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), refusal)
	_, err = NewUpdateSequenceLogic(ctx, s).UpdateSequence(&types.UpdateSequenceRequest{Id: cold.Id, SequenceType: "lifecycle", IsActive: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), refusal)
	_, err = NewUpdateSequenceLogic(ctx, s).UpdateSequence(&types.UpdateSequenceRequest{Id: "old", Name: "Old, off", IsActive: false})
	require.NoError(t, err, "turning off one already there")

	_, err = NewCreateEntryRuleLogic(ctx, s).CreateEntryRule(&types.CreateEntryRuleRequest{SequenceId: welcome.Id, TriggerType: "list_join", SourceId: "2"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), refusal)
	_, err = NewCreateEntryRuleLogic(ctx, s).CreateEntryRule(&types.CreateEntryRuleRequest{SequenceId: welcome.Id, TriggerType: "list_join", SourceId: "1"})
	require.NoError(t, err)
}
