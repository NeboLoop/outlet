package outbound

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/services/outbound/providers"
)

// Outreach domains move to another brand of the same caller (cold outreach
// kept apart from a company's own mail) with everything that belongs to
// them: the domain itself, DNS checks and DKIM included, so it needs no new
// verification; its mailboxes with their credentials (encrypted with the
// server's key, the same for every brand), warm-up start, own daily limits
// and reading state; and their send counts. As within a brand, a domain
// added by hand joins the target pool's manual connection, while one from a
// mailbox service brings its connection, with its key and limits. What the
// old brand did with them stays its own: sent emails, replies and threads,
// prospects, campaigns and their results.

// BrandMoveInput moves outreach domains to another brand. PoolID names the
// target brand's mailbox pool; left out, the one carrying its outreach, else
// its only one (a new one when it has none). ConnectionID names a connection
// of the target brand the domains join, their mailboxes then sending under
// its limits. Left out, a domain added by hand joins the pool's manual
// connection (made when it has none), and one from a mailbox service brings
// its connection, which takes every domain on it in the same move.
type BrandMoveInput struct {
	DomainIDs    []string
	PoolID       string
	ConnectionID string
}

// BrandMove is what moved, and where.
type BrandMove struct {
	Brand   string        `json:"brand"`
	PoolID  string        `json:"pool_id"`
	Pool    string        `json:"pool"`
	Domains []MovedDomain `json:"domains"`
}

// MovedDomain is one domain that moved: its mailboxes and the sends counted
// for them, carried to the new pool so no mailbox's day starts over.
type MovedDomain struct {
	ID           string `json:"id"`
	Domain       string `json:"domain"`
	Mailboxes    int64  `json:"mailboxes"`
	SendsCounted int64  `json:"sends_counted"`
	ConnectionID string `json:"connection_id"`
}

// MoveDomainsToBrand moves outreach domains of one brand to another, all or
// none, in one transaction. It refuses, naming every domain in the way at
// once, while a domain's mailboxes still carry the old brand's outreach (a
// person in a campaign, paused ones too, or an email waiting), when the
// target brand may not send from the domain (a protected company domain, or
// its own sending domain), or when a domain's connection holds domains left
// behind. The caller's right to manage both brands is checked before.
func (s *Service) MoveDomainsToBrand(ctx context.Context, from, to db.Organization, in BrandMoveInput) (BrandMove, error) {
	if to.ID == from.ID {
		return BrandMove{}, refuse("The domains are already %s's; name the brand to move them to.", to.Name)
	}
	var ids []string
	for _, id := range in.DomainIDs {
		if id = strings.TrimSpace(id); id != "" && !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return BrandMove{}, refuse("Name the outreach domains to move.")
	}
	var join *db.MailboxConnection
	if in.ConnectionID != "" {
		c, err := s.db.GetMailboxConnection(ctx, db.GetMailboxConnectionParams{ID: in.ConnectionID, OrgID: to.ID})
		if err != nil {
			return BrandMove{}, refuse("%s has no connection %s.", to.Name, in.ConnectionID)
		}
		join = &c
	}
	pool, newPool, err := s.brandMovePool(ctx, to, in.PoolID, join)
	if err != nil {
		return BrandMove{}, err
	}

	var domains []db.DomainIdentity
	var problems []string
	for _, id := range ids {
		d, err := s.db.GetOutreachDomain(ctx, db.GetOutreachDomainParams{ID: id, OrgID: from.ID})
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s is not one of %s's outreach domains.", id, from.Name))
			continue
		}
		domains = append(domains, d)
	}
	for _, d := range domains {
		p, err := s.brandMoveProblems(ctx, s.db, to, d, join == nil, ids)
		if err != nil {
			return BrandMove{}, err
		}
		problems = append(problems, p...)
	}
	if len(problems) > 0 {
		return BrandMove{}, refuse("Nothing moved. %s", strings.Join(problems, " "))
	}

	// A mailbox service's connection moves whole and keeps the limits its
	// mailboxes send under: the ones it took from its old pool are written
	// on it.
	conns := map[string]db.MoveMailboxConnectionToBrandParams{}
	if join == nil {
		for _, d := range domains {
			if !d.ConnectionID.Valid || conns[d.ConnectionID.String].ID != "" {
				continue
			}
			c, err := s.db.GetMailboxConnection(ctx, db.GetMailboxConnectionParams{ID: d.ConnectionID.String, OrgID: from.ID})
			if err != nil {
				return BrandMove{}, err
			}
			if c.Service == providers.Manual {
				continue
			}
			old, err := s.mailer.GetPathway(ctx, from.ID, c.PathwayID)
			if err != nil {
				return BrandMove{}, err
			}
			l := email.PathwayLimits(old)
			fill := func(v sql.NullInt64, pool int64) sql.NullInt64 {
				if v.Valid {
					return v
				}
				return sql.NullInt64{Int64: pool, Valid: true}
			}
			conns[c.ID] = db.MoveMailboxConnectionToBrandParams{
				ToOrgID: to.ID, DailyLimit: fill(c.DailyLimit, l.DailyLimit), WarmupStart: fill(c.WarmupStart, l.WarmupStart), WarmupStep: fill(c.WarmupStep, l.WarmupStep),
				ID: c.ID, FromOrgID: from.ID,
			}
		}
	}
	if newPool {
		if pool, err = s.mailer.DomainPathway(ctx, to.ID, "outreach", ""); err != nil {
			return BrandMove{}, err
		}
	}

	out := BrandMove{Brand: to.Name, PoolID: pool.ID, Pool: pool.Name}
	err = s.db.ExecTx(ctx, func(q *db.Queries) error {
		// Checked again inside the move: a campaign may have taken a
		// mailbox since.
		for _, d := range domains {
			p, err := s.brandMoveProblems(ctx, q, to, d, join == nil, ids)
			if err != nil {
				return err
			}
			if len(p) > 0 {
				return refuse("Nothing moved. %s", strings.Join(p, " "))
			}
		}
		for _, c := range conns {
			c.PathwayID = pool.ID
			if n, err := q.MoveMailboxConnectionToBrand(ctx, c); err != nil || n == 0 {
				return moveErr(err)
			}
			if err := q.MoveConnectionQuotesToBrand(ctx, db.MoveConnectionQuotesToBrandParams{ToOrgID: to.ID, ConnectionID: c.ID}); err != nil {
				return err
			}
		}
		for _, d := range domains {
			conn := d.ConnectionID
			switch {
			case join != nil:
				conn = sql.NullString{String: join.ID, Valid: true}
			case conns[conn.String].ID == "":
				c, err := email.ManualConnection(ctx, q, to.ID, pool.ID)
				if err != nil {
					return err
				}
				conn = sql.NullString{String: c.ID, Valid: true}
			}
			moved := MovedDomain{ID: d.ID, Domain: d.Domain, ConnectionID: conn.String}
			if d.PathwayID.Valid {
				n, err := q.SumDomainSendCounts(ctx, db.SumDomainSendCountsParams{PathwayID: d.PathwayID.String, DomainID: d.ID})
				if err != nil {
					return err
				}
				moved.SendsCounted = n
			}
			if n, err := q.MoveOutreachDomainToBrand(ctx, db.MoveOutreachDomainToBrandParams{
				ToOrgID: to.ID, PathwayID: sql.NullString{String: pool.ID, Valid: true}, ConnectionID: conn, ID: d.ID, FromOrgID: from.ID,
			}); err != nil || n == 0 {
				return moveErr(err)
			}
			n, err := q.MoveDomainMailboxesToBrand(ctx, db.MoveDomainMailboxesToBrandParams{ToOrgID: to.ID, DomainID: d.ID, FromOrgID: from.ID})
			if err != nil {
				return err
			}
			moved.Mailboxes = n
			if join != nil {
				// The old service's ids mean nothing at the joined
				// connection; a sync there links them again.
				if err := q.ClearDomainMailboxServiceRefs(ctx, d.ID); err != nil {
					return err
				}
			}
			if d.PathwayID.Valid {
				if err := q.CarryDomainSendCounts(ctx, db.CarryDomainSendCountsParams{ToPathwayID: pool.ID, FromPathwayID: d.PathwayID.String, DomainID: d.ID}); err != nil {
					return err
				}
				if err := q.DeleteDomainSendCounts(ctx, db.DeleteDomainSendCountsParams{PathwayID: d.PathwayID.String, DomainID: d.ID}); err != nil {
					return err
				}
			}
			out.Domains = append(out.Domains, moved)
		}
		return nil
	})
	if err != nil {
		return BrandMove{}, err
	}
	return out, nil
}

// moveErr is a move that found its row gone or changed under it.
func moveErr(err error) error {
	if err != nil {
		return err
	}
	return refuse("Nothing moved: a domain or connection changed while moving; try again.")
}

// brandMovePool is the target brand's mailbox pool the domains join: the
// named one (the joined connection's), else the one carrying its outreach,
// else its only one. newPool says it has none, so one is made.
func (s *Service) brandMovePool(ctx context.Context, to db.Organization, poolID string, join *db.MailboxConnection) (pool db.SendingPathway, newPool bool, err error) {
	if join != nil {
		if poolID != "" && poolID != join.PathwayID {
			return pool, false, refuse("Connection %q is in another mailbox pool of %s than pool_id; pass one of them.", join.Name, to.Name)
		}
		poolID = join.PathwayID
	}
	if poolID != "" {
		pool, err = s.mailboxPool(ctx, to.ID, poolID)
		return pool, false, err
	}
	all, err := s.db.ListSendingPathways(ctx, to.ID)
	if err != nil {
		return pool, false, err
	}
	var pools []db.SendingPathway
	for _, p := range all {
		if p.Provider != email.ProviderMailboxes {
			continue
		}
		if p.CarriesOutreach == 1 {
			return p, false, nil
		}
		pools = append(pools, p)
	}
	switch len(pools) {
	case 0:
		return pool, true, nil
	case 1:
		return pools[0], false, nil
	}
	names := make([]string, 0, len(pools))
	for _, p := range pools {
		names = append(names, fmt.Sprintf("%s (%s)", p.ID, p.Name))
	}
	return pool, false, refuse("%s has %d mailbox pools and none carries its outreach; pass pool_id, one of: %s.", to.Name, len(pools), strings.Join(names, ", "))
}

// brandMoveProblems says why one domain can't move to the brand, if it
// can't. wholeConn: a mailbox service's connection moves with its domain,
// so every domain on it must be among moving.
func (s *Service) brandMoveProblems(ctx context.Context, q db.Querier, to db.Organization, d db.DomainIdentity, wholeConn bool, moving []string) ([]string, error) {
	var out []string
	if err := s.allowedDomain(ctx, to, d.Domain); err != nil {
		out = append(out, fmt.Sprintf("%s can't move to %s: %s", d.Domain, to.Name, err.Error()))
	}
	enrolled, err := q.ListDomainActiveEnrollments(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	waiting, err := q.CountDomainWaitingOutreach(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	if len(enrolled) > 0 || waiting > 0 {
		var in []string
		for _, e := range enrolled {
			in = append(in, fmt.Sprintf("%d in campaign %q (%s)", e.N, e.CampaignName, e.CampaignID))
		}
		msg := d.Domain + "'s mailboxes still carry this brand's outreach:"
		if len(in) > 0 {
			msg += " people enrolled, " + strings.Join(in, ", ") + ";"
		}
		if waiting > 0 {
			msg += fmt.Sprintf(" %d emails waiting (drafts, approved or sending);", waiting)
		}
		out = append(out, strings.TrimSuffix(msg, ";")+". Let those campaigns finish and skip the waiting emails first, so nothing of this brand's sends from another's mailbox.")
	}
	if wholeConn && d.ConnectionID.Valid {
		on, err := q.ListConnectionDomains(ctx, d.ConnectionID)
		if err != nil {
			return nil, err
		}
		var left []string
		for _, o := range on {
			if !slices.Contains(moving, o.ID) {
				left = append(left, o.Domain)
			}
		}
		c, err := q.GetMailboxConnection(ctx, db.GetMailboxConnectionParams{ID: d.ConnectionID.String, OrgID: d.OrgID})
		if err != nil {
			return nil, err
		}
		if len(left) > 0 && c.Service != providers.Manual {
			out = append(out, fmt.Sprintf("%s's connection %q moves with it but also holds %s: move them together, or pass connection_id, a connection of %s to join.", d.Domain, c.Name, strings.Join(left, ", "), to.Name))
		}
	}
	return out, nil
}
