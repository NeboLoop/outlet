package outbound

import (
	"database/sql"
	"testing"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/services/outbound/providers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// other is the harness's second brand.
func (h *harness) other() db.Organization {
	o, err := h.store.GetOrganizationByID(h.ctx, "org-2")
	require.NoError(h.t, err)
	return o
}

// sendAs sends one outreach email from a mailbox as a brand.
func (h *harness) sendAs(orgID string, mb db.Mailbox) error {
	return h.mailer.Send(h.ctx, email.Message{Kind: email.KindOutreach, OrgID: orgID, MailboxID: mb.ID,
		To: "ava@example.org", Subject: "Hello", Text: "Hi Ava"})
}

// A domain added by hand moved to another brand takes its DNS state,
// mailboxes (their credentials, warm-up, own limits) and send counts with
// it, into the brand's outreach pool and its manual connection; the old
// brand can no longer send from its mailboxes and the new one can.
func TestMoveDomainToAnotherBrand(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	sam, pat := h.setup()
	to := h.other()
	toPool, err := h.mailer.CreatePathway(ctx, to.ID, email.PathwayInput{Name: "Outreach", Provider: email.ProviderMailboxes, Kinds: []string{email.KindOutreach.Name()}})
	require.NoError(t, err)
	_, err = h.mailer.CreatePathway(ctx, to.ID, email.PathwayInput{Name: "Spare", Provider: email.ProviderMailboxes})
	require.NoError(t, err)
	toManual, err := h.svc.AddConnection(ctx, to, ConnectionInput{PoolID: toPool.ID, Service: providers.Manual})
	require.NoError(t, err)

	d, err := h.store.GetOutreachDomain(ctx, db.GetOutreachDomainParams{ID: sam.DomainID, OrgID: h.org.ID})
	require.NoError(t, err)
	oldPool, err := h.mailer.GetPathway(ctx, h.org.ID, d.PathwayID.String)
	require.NoError(t, err)
	require.NoError(t, h.mailer.SetMailboxDailyLimit(ctx, h.org.ID, sam.ID, ptr(int64(12))))
	require.NoError(t, h.sendAs(h.org.ID, sam))
	var counted int64
	require.NoError(t, h.conn.QueryRow(`SELECT COALESCE(SUM(n), 0) FROM send_counts WHERE pathway_id = ?`, oldPool.ID).Scan(&counted))
	require.EqualValues(t, 1, counted)

	moved, err := h.svc.MoveDomainsToBrand(ctx, h.org, to, BrandMoveInput{DomainIDs: []string{d.ID}})
	require.NoError(t, err)
	require.Len(t, moved.Domains, 1)
	assert.EqualValues(t, 2, moved.Domains[0].Mailboxes)
	assert.EqualValues(t, 1, moved.Domains[0].SendsCounted)
	assert.Equal(t, toManual.ID, moved.Domains[0].ConnectionID, "a domain added by hand joins the manual connection")
	assert.Equal(t, toPool.ID, moved.PoolID, "the pool carrying the brand's outreach")

	// The domain, as it was, now the other brand's: no new verification.
	_, err = h.store.GetOutreachDomain(ctx, db.GetOutreachDomainParams{ID: d.ID, OrgID: h.org.ID})
	assert.ErrorIs(t, err, sql.ErrNoRows)
	nd, err := h.store.GetOutreachDomain(ctx, db.GetOutreachDomainParams{ID: d.ID, OrgID: to.ID})
	require.NoError(t, err)
	assert.Equal(t, d.VerificationStatus, nd.VerificationStatus)
	assert.Equal(t, d.DnsCheck, nd.DnsCheck)
	assert.Equal(t, d.DkimSelector, nd.DkimSelector)
	assert.Equal(t, moved.PoolID, nd.PathwayID.String)
	newPool, err := h.mailer.GetPathway(ctx, to.ID, moved.PoolID)
	require.NoError(t, err)

	// The mailboxes, credentials and warm-up as they were.
	nsam, err := h.store.GetMailbox(ctx, db.GetMailboxParams{ID: sam.ID, OrgID: to.ID})
	require.NoError(t, err)
	assert.Equal(t, sam.SmtpPasswordEnc, nsam.SmtpPasswordEnc)
	assert.Equal(t, sam.WarmupStartedAt, nsam.WarmupStartedAt)
	assert.EqualValues(t, 12, nsam.DailyLimit.Int64)
	_, err = h.store.GetMailbox(ctx, db.GetMailboxParams{ID: pat.ID, OrgID: to.ID})
	require.NoError(t, err)

	// The old brand's manual connection stays, empty; sam keeps its own limit.
	_, err = h.store.GetMailboxConnection(ctx, db.GetMailboxConnectionParams{ID: d.ConnectionID.String, OrgID: h.org.ID})
	require.NoError(t, err)
	limits, err := h.mailer.MailboxLimits(ctx, newPool, nsam)
	require.NoError(t, err)
	assert.EqualValues(t, 12, limits.DailyLimit)

	// Today's sends counted in the new pool, gone from the old.
	require.NoError(t, h.conn.QueryRow(`SELECT COALESCE(SUM(n), 0) FROM send_counts WHERE pathway_id = ?`, newPool.ID).Scan(&counted))
	assert.EqualValues(t, 1, counted)
	require.NoError(t, h.conn.QueryRow(`SELECT COALESCE(SUM(n), 0) FROM send_counts WHERE pathway_id = ?`, oldPool.ID).Scan(&counted))
	assert.EqualValues(t, 0, counted)

	// Only the new brand sends from them.
	// (pat: sam is still spacing its sends since the one before the move).
	assert.ErrorContains(t, h.sendAs(h.org.ID, pat), "not one of this brand's")
	require.NoError(t, h.sendAs(to.ID, pat))
	o, err := h.svc.Overview(ctx, h.org)
	require.NoError(t, err)
	assert.Empty(t, o.Mailboxes)
	assert.Empty(t, o.Domains)
}

// Nothing moves while the domain's mailboxes carry the old brand's
// outreach: the refusal names the campaign and the waiting emails.
func TestMoveDomainToAnotherBrandRefusedWhileEnrolled(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	sam, _ := h.setup()
	got := h.prospects(creator("ava@example.org", "Ava", "your desk tour"))
	_, err := h.svc.DraftOutreach(ctx, h.org, CampaignInput{Name: "Creators", Steps: threeSteps, ProspectIDs: []string{got["ava@example.org"].ID}})
	require.NoError(t, err)

	_, err = h.svc.MoveDomainsToBrand(ctx, h.org, h.other(), BrandMoveInput{DomainIDs: []string{sam.DomainID, "nope"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Nothing moved.")
	assert.Contains(t, err.Error(), `1 in campaign "Creators"`)
	assert.Contains(t, err.Error(), "1 emails waiting")
	assert.Contains(t, err.Error(), "nope is not one of", "every domain in the way is named at once")
	_, err = h.store.GetOutreachDomain(ctx, db.GetOutreachDomainParams{ID: sam.DomainID, OrgID: h.org.ID})
	assert.NoError(t, err, "the domain stayed")
	_, err = h.store.GetMailbox(ctx, db.GetMailboxParams{ID: sam.ID, OrgID: h.org.ID})
	assert.NoError(t, err, "its mailboxes stayed")
}

// A target brand with several mailbox pools, none carrying its outreach,
// needs pool_id; the refusal lists them.
func TestMoveDomainToAnotherBrandNeedsAPool(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	sam, _ := h.setup()
	to := h.other()
	a, err := h.mailer.CreatePathway(ctx, to.ID, email.PathwayInput{Name: "Pool A", Provider: email.ProviderMailboxes})
	require.NoError(t, err)
	b, err := h.mailer.CreatePathway(ctx, to.ID, email.PathwayInput{Name: "Pool B", Provider: email.ProviderMailboxes})
	require.NoError(t, err)

	_, err = h.svc.MoveDomainsToBrand(ctx, h.org, to, BrandMoveInput{DomainIDs: []string{sam.DomainID}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), a.ID+" (Pool A)")
	assert.Contains(t, err.Error(), b.ID+" (Pool B)")

	moved, err := h.svc.MoveDomainsToBrand(ctx, h.org, to, BrandMoveInput{DomainIDs: []string{sam.DomainID}, PoolID: b.ID})
	require.NoError(t, err)
	assert.Equal(t, b.ID, moved.PoolID)
	pool, err := h.store.GetMailboxPathway(ctx, sam.ID)
	require.NoError(t, err)
	assert.Equal(t, b.ID, pool.ID)
}

// A domain from a mailbox service brings its connection, with its key and
// limits; a brand with no mailbox pool gets one.
func TestMoveServiceDomainToAnotherBrand(t *testing.T) {
	h := newHarness(t)
	ctx := h.ctx
	p := h.mixedPool()
	to := h.other()
	before, err := h.store.GetMailboxConnection(ctx, db.GetMailboxConnectionParams{ID: p.maildoso.ID, OrgID: h.org.ID})
	require.NoError(t, err)

	moved, err := h.svc.MoveDomainsToBrand(ctx, h.org, to, BrandMoveInput{DomainIDs: []string{p.sam.DomainID}})
	require.NoError(t, err)
	assert.Equal(t, p.maildoso.ID, moved.Domains[0].ConnectionID)
	pool, err := h.mailer.GetPathway(ctx, to.ID, moved.PoolID)
	require.NoError(t, err)
	assert.Equal(t, email.ProviderMailboxes, pool.Provider)
	c, err := h.store.GetMailboxConnection(ctx, db.GetMailboxConnectionParams{ID: p.maildoso.ID, OrgID: to.ID})
	require.NoError(t, err)
	assert.Equal(t, pool.ID, c.PathwayID)
	assert.Equal(t, before.ApiKeyEnc, c.ApiKeyEnc)
	assert.Equal(t, before.DailyLimit, c.DailyLimit)
	assert.Equal(t, before.WarmupStart, c.WarmupStart)
	sam, err := h.store.GetMailbox(ctx, db.GetMailboxParams{ID: p.sam.ID, OrgID: to.ID})
	require.NoError(t, err)
	assert.Equal(t, p.sam.ServiceRef, sam.ServiceRef, "still the same account at the service")
}
