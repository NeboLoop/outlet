package lists

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strconv"
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

// A list is reached through its brand: an ID from another brand is refused,
// whatever brand the request names.
func TestListHandlers_CheckTheListsBrand(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn))
	store := db.NewStore(conn)
	s := &svc.ServiceContext{DB: store}
	ctx := context.Background()

	_, err = store.CreateUser(ctx, db.CreateUserParams{ID: "mike", Email: "mike@example.com", PasswordHash: "x", Name: "M", Role: "agent", Status: "active", EmailVerified: int64(1)})
	require.NoError(t, err)
	lists := map[string]string{}
	for _, b := range []string{"alpha", "bravo"} {
		_, err := store.CreateOrganization(ctx, db.CreateOrganizationParams{ID: b + "-id", Name: b, Slug: b, ApiKey: "key-" + b})
		require.NoError(t, err)
		l, err := store.CreateEmailList(ctx, db.CreateEmailListParams{PublicID: "p-" + b, OrgID: b + "-id", Name: b, Slug: b})
		require.NoError(t, err)
		lists[b] = strconv.FormatInt(l.ID, 10)
	}
	require.NoError(t, store.AddUserToOrganization(ctx, db.AddUserToOrganizationParams{OrgID: "alpha-id", UserID: "mike", Role: sql.NullString{String: "member", Valid: true}}))

	user, err := access.Active(ctx, store, "mike")
	require.NoError(t, err)
	// mike names his own brand, then asks for bravo's list by ID.
	asMike := access.WithBrand(access.WithUser(ctx, user), "alpha-id")

	_, err = NewGetListLogic(asMike, s).GetList(&types.GetListRequest{Id: lists["bravo"]})
	var ce *errorx.CodeError
	require.True(t, errors.As(err, &ce), err)
	assert.Equal(t, 403, ce.Code)

	_, err = NewDeleteListLogic(asMike, s).DeleteList(&types.DeleteListRequest{Id: lists["bravo"]})
	require.Error(t, err)
	_, err = store.GetEmailList(ctx, mustInt(t, lists["bravo"]))
	assert.NoError(t, err, "bravo's list is still there")

	got, err := NewGetListLogic(asMike, s).GetList(&types.GetListRequest{Id: lists["alpha"]})
	require.NoError(t, err)
	assert.Equal(t, "alpha", got.Name)

	listed, err := NewListListsLogic(asMike, s).ListLists()
	require.NoError(t, err)
	require.Len(t, listed.Lists, 1)
	assert.Equal(t, "alpha", listed.Lists[0].Name)
}

func mustInt(t *testing.T, s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	require.NoError(t, err)
	return n
}
