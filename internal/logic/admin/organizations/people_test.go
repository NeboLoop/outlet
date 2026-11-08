package organizations

import (
	"testing"

	"github.com/outlet-sh/outlet/internal/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A brand's admins and super admins see its members with their roles and the
// invitations not yet accepted with when they expire; an expired one says so,
// an accepted one is gone. Members, and admins of another brand, can't.
func TestListOrgPeople(t *testing.T) {
	f := newRESTFixture(t)
	for _, q := range []string{
		`INSERT INTO org_invitations (id, org_id, email, role, token, expires_at, created_at) VALUES ('i1', 'alpha-id', 'new@example.com', 'admin', 't1', datetime('now', '+7 days'), datetime('now'))`,
		`INSERT INTO org_invitations (id, org_id, email, role, token, expires_at, created_at) VALUES ('i2', 'alpha-id', 'late@example.com', 'member', 't2', datetime('now', '-1 day'), datetime('now', '-8 days'))`,
		`INSERT INTO org_invitations (id, org_id, email, role, token, expires_at, accepted_at) VALUES ('i3', 'alpha-id', 'mike@example.com', 'member', 't3', datetime('now', '+7 days'), datetime('now'))`,
		`INSERT INTO org_invitations (id, org_id, email, role, token, expires_at) VALUES ('i4', 'bravo-id', 'other@example.com', 'member', 't4', datetime('now', '+7 days'))`,
	} {
		_, err := f.svc.DB.GetDB().Exec(q)
		require.NoError(t, err)
	}

	for _, who := range []string{"anne", "root"} {
		resp, err := NewListOrgPeopleLogic(f.as(t, who), f.svc).ListOrgPeople(&types.ListOrgPeopleRequest{Id: "alpha-id"})
		require.NoError(t, err, who)
		require.Len(t, resp.Members, 2)
		roles := map[string]string{}
		for _, m := range resp.Members {
			roles[m.Email] = m.Role
		}
		assert.Equal(t, map[string]string{"anne@example.com": "admin", "mike@example.com": "member"}, roles)
		require.Len(t, resp.Invitations, 2)
		assert.Equal(t, "new@example.com", resp.Invitations[0].Email)
		assert.Equal(t, "admin", resp.Invitations[0].Role)
		assert.False(t, resp.Invitations[0].Expired)
		assert.NotEmpty(t, resp.Invitations[0].ExpiresAt)
		assert.Equal(t, "late@example.com", resp.Invitations[1].Email)
		assert.True(t, resp.Invitations[1].Expired)
	}

	_, err := NewListOrgPeopleLogic(f.as(t, "mike"), f.svc).ListOrgPeople(&types.ListOrgPeopleRequest{Id: "alpha-id"})
	assert.Equal(t, 403, status(err), "a member can't list people")
	_, err = NewListOrgPeopleLogic(f.as(t, "anne"), f.svc).ListOrgPeople(&types.ListOrgPeopleRequest{Id: "bravo-id"})
	assert.Error(t, err, "an admin of another brand can't")
}
