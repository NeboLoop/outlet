package lists

import (
	"context"
	"database/sql"
	"path/filepath"
	"strconv"
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

// Adding refuses an address that isn't one; removing takes the subscriber ID
// the list of subscribers gives, and takes that person off the list.
func TestListSubscribers_AddAndRemove(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	store := db.NewStore(conn)
	s := &svc.ServiceContext{DB: store}
	ctx := context.Background()

	_, err = store.CreateUser(ctx, db.CreateUserParams{ID: "mike", Email: "mike@example.com", PasswordHash: "x", Name: "M", Role: "agent", Status: "active", EmailVerified: int64(1)})
	require.NoError(t, err)
	_, err = store.CreateOrganization(ctx, db.CreateOrganizationParams{ID: "alpha-id", Name: "alpha", Slug: "alpha", ApiKey: "key"})
	require.NoError(t, err)
	l, err := store.CreateEmailList(ctx, db.CreateEmailListParams{PublicID: "p", OrgID: "alpha-id", Name: "News", Slug: "news"})
	require.NoError(t, err)
	listID := strconv.FormatInt(l.ID, 10)
	require.NoError(t, store.AddUserToOrganization(ctx, db.AddUserToOrganizationParams{OrgID: "alpha-id", UserID: "mike", Role: sql.NullString{String: "member", Valid: true}}))
	user, err := access.Active(ctx, store, "mike")
	require.NoError(t, err)
	asMike := access.WithBrand(access.WithUser(ctx, user), "alpha-id")

	_, err = NewAddListSubscriberLogic(asMike, s).AddListSubscriber(&types.AddListSubscriberRequest{Id: listID, Email: "not-an-email"})
	assert.ErrorContains(t, err, "valid email")
	_, err = NewAddListSubscriberLogic(asMike, s).AddListSubscriber(&types.AddListSubscriberRequest{Id: listID, Email: "ann@example.com"})
	require.NoError(t, err)

	subs, err := NewListListSubscribersLogic(asMike, s).ListListSubscribers(&types.ListSubscribersRequest{Id: listID})
	require.NoError(t, err)
	require.Len(t, subs.Subscribers, 1)

	_, err = NewRemoveListSubscriberLogic(asMike, s).RemoveListSubscriber(&types.RemoveSubscriberRequest{Id: listID, SubscriberId: subs.Subscribers[0].Id})
	require.NoError(t, err)
	subs, err = NewListListSubscribersLogic(asMike, s).ListListSubscribers(&types.ListSubscribersRequest{Id: listID})
	require.NoError(t, err)
	assert.Empty(t, subs.Subscribers, "removed")
}

// Adding someone by hand makes their subscription active, so it starts the
// list's sign-up emails like a sign-up form does.
func TestAddListSubscriber_StartsSignupEmails(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	store := db.NewStore(conn)
	var started []int64
	store.OnSubscriptionActivated(func(_ context.Context, _ string, listID int64) error {
		started = append(started, listID)
		return nil
	})
	s := &svc.ServiceContext{DB: store}
	ctx := context.Background()

	_, err = store.CreateUser(ctx, db.CreateUserParams{ID: "mike", Email: "mike@example.com", PasswordHash: "x", Name: "M", Role: "agent", Status: "active", EmailVerified: int64(1)})
	require.NoError(t, err)
	_, err = store.CreateOrganization(ctx, db.CreateOrganizationParams{ID: "alpha-id", Name: "alpha", Slug: "alpha", ApiKey: "key"})
	require.NoError(t, err)
	l, err := store.CreateEmailList(ctx, db.CreateEmailListParams{PublicID: "p", OrgID: "alpha-id", Name: "News", Slug: "news"})
	require.NoError(t, err)
	require.NoError(t, store.AddUserToOrganization(ctx, db.AddUserToOrganizationParams{OrgID: "alpha-id", UserID: "mike", Role: sql.NullString{String: "member", Valid: true}}))
	user, err := access.Active(ctx, store, "mike")
	require.NoError(t, err)
	asMike := access.WithBrand(access.WithUser(ctx, user), "alpha-id")

	for range 2 {
		_, err = NewAddListSubscriberLogic(asMike, s).AddListSubscriber(&types.AddListSubscriberRequest{Id: strconv.FormatInt(l.ID, 10), Email: "ann@example.com"})
		require.NoError(t, err)
	}
	assert.Equal(t, []int64{l.ID}, started, "started once, for the one subscription added")
}
