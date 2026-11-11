-- Outbound targeting: segments (outreach lists with a definition), the
-- brand profile, materials, do-not-contact checks and the leads export.

-- ---- segments -------------------------------------------------------------------

-- name: SetSegmentDefinition :exec
UPDATE email_lists
SET name = sqlc.arg(name), segment_description = sqlc.arg(segment_description), segment_filters = sqlc.arg(segment_filters),
    updated_at = datetime('now')
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id) AND kind = 'outreach';

-- name: CountSegmentProspects :many
-- Each outreach list's people by status.
SELECT ls.list_id, ls.status, COUNT(*) AS n FROM list_subscribers ls
JOIN email_lists el ON el.id = ls.list_id AND el.kind = 'outreach'
WHERE el.org_id = ? GROUP BY ls.list_id, ls.status;

-- name: GetProspectOutreachSubscription :one
-- A person's subscription on the outreach list they joined first, to copy
-- into another segment.
SELECT ls.* FROM list_subscribers ls
JOIN email_lists el ON el.id = ls.list_id AND el.kind = 'outreach'
WHERE el.org_id = sqlc.arg(org_id) AND ls.contact_id = sqlc.arg(contact_id)
ORDER BY CASE WHEN ls.status = 'prospect' THEN 0 ELSE 1 END, ls.subscribed_at
LIMIT 1;

-- ---- the brand profile --------------------------------------------------------------

-- name: GetBrandProfile :one
SELECT * FROM brand_profiles WHERE org_id = ?;

-- name: UpsertBrandProfile :one
INSERT INTO brand_profiles (org_id, website, sells, audience, proof, voice)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (org_id) DO UPDATE SET
    website = excluded.website, sells = excluded.sells, audience = excluded.audience,
    proof = excluded.proof, voice = excluded.voice, updated_at = strftime('%Y-%m-%d %H:%M:%S', 'now')
RETURNING *;

-- ---- materials ------------------------------------------------------------------------

-- name: CreateBrandMaterial :one
INSERT INTO brand_materials (id, org_id, kind, title, summary, details, url, list_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetBrandMaterial :one
SELECT * FROM brand_materials WHERE id = ? AND org_id = ?;

-- name: UpdateBrandMaterial :exec
UPDATE brand_materials
SET kind = ?, title = ?, summary = ?, details = ?, url = ?, list_id = ?, updated_at = strftime('%Y-%m-%d %H:%M:%S', 'now')
WHERE id = ? AND org_id = ?;

-- name: DeleteBrandMaterial :execrows
DELETE FROM brand_materials WHERE id = ? AND org_id = ?;

-- name: ListBrandMaterials :many
-- A brand's materials, of one kind (empty: any), for one segment (0: any):
-- a segment's own and the brand-wide ones.
SELECT * FROM brand_materials
WHERE org_id = sqlc.arg(org_id)
  AND (CAST(sqlc.arg(kind) AS TEXT) = '' OR kind = sqlc.arg(kind))
  AND (CAST(sqlc.arg(list_id) AS INTEGER) = 0 OR list_id IS NULL OR list_id = sqlc.arg(list_id))
ORDER BY kind, title;

-- name: PinCampaignMaterial :exec
INSERT INTO campaign_materials (sequence_id, material_id) VALUES (?, ?) ON CONFLICT DO NOTHING;

-- name: ListCampaignMaterials :many
SELECT m.* FROM campaign_materials cm JOIN brand_materials m ON m.id = cm.material_id
WHERE cm.sequence_id = ? ORDER BY m.kind, m.title;

-- ---- do not contact -------------------------------------------------------------------

-- name: AddDoNotContact :execrows
INSERT INTO do_not_contact (org_id, list_name, kind, value) VALUES (?, ?, ?, ?)
ON CONFLICT DO NOTHING;

-- name: DeleteDoNotContactList :execrows
DELETE FROM do_not_contact WHERE org_id = ? AND list_name = ?;

-- name: ListDoNotContactLists :many
SELECT list_name,
       CAST(SUM(CASE WHEN kind = 'email' THEN 1 ELSE 0 END) AS INTEGER) AS emails,
       CAST(SUM(CASE WHEN kind = 'domain' THEN 1 ELSE 0 END) AS INTEGER) AS domains,
       CAST(MAX(created_at) AS TEXT) AS updated_at
FROM do_not_contact WHERE org_id = ? GROUP BY list_name ORDER BY list_name;

-- name: FindDoNotContact :one
-- The first uploaded entry naming this address or its domain.
SELECT list_name, kind, value FROM do_not_contact
WHERE org_id = sqlc.arg(org_id)
  AND ((kind = 'email' AND value = LOWER(sqlc.arg(email))) OR (kind = 'domain' AND value = LOWER(sqlc.arg(domain))))
ORDER BY kind DESC LIMIT 1;

-- name: FindCustomer :one
-- Where the brand already knows this address outside outreach: a list that
-- isn't an outreach list (any status), transactional mail it was sent, or a
-- contact the brand had before outreach (on no outreach list).
SELECT CAST(COALESCE(
    (SELECT 'list ' || el.name FROM contacts c
       JOIN list_subscribers ls ON ls.contact_id = c.id
       JOIN email_lists el ON el.id = ls.list_id AND el.kind <> 'outreach'
     WHERE c.org_id = sqlc.arg(org_id) AND LOWER(c.email) = LOWER(sqlc.arg(email)) LIMIT 1),
    (SELECT 'transactional email ' || te.name FROM transactional_sends ts
       JOIN transactional_emails te ON te.id = ts.template_id
     WHERE ts.org_id = sqlc.arg(org_id) AND LOWER(ts.to_email) = LOWER(sqlc.arg(email)) LIMIT 1),
    (SELECT 'a contact since ' || COALESCE(c.created_at, 'before outreach') FROM contacts c
     WHERE c.org_id = sqlc.arg(org_id) AND LOWER(c.email) = LOWER(sqlc.arg(email))
       AND NOT EXISTS (SELECT 1 FROM list_subscribers ls JOIN email_lists el ON el.id = ls.list_id AND el.kind = 'outreach'
                       WHERE ls.contact_id = c.id) LIMIT 1),
    '') AS TEXT) AS found;

-- name: FindCustomerAtDomain :one
-- A customer (as FindCustomer reads one) at this email domain.
SELECT CAST(COALESCE(
    (SELECT c.email FROM contacts c
       JOIN list_subscribers ls ON ls.contact_id = c.id
       JOIN email_lists el ON el.id = ls.list_id AND el.kind <> 'outreach'
     WHERE c.org_id = sqlc.arg(org_id) AND LOWER(c.email) LIKE '%@' || LOWER(sqlc.arg(domain)) LIMIT 1),
    (SELECT ts.to_email FROM transactional_sends ts
     WHERE ts.org_id = sqlc.arg(org_id) AND LOWER(ts.to_email) LIKE '%@' || LOWER(sqlc.arg(domain)) LIMIT 1),
    (SELECT c.email FROM contacts c
     WHERE c.org_id = sqlc.arg(org_id) AND LOWER(c.email) LIKE '%@' || LOWER(sqlc.arg(domain))
       AND NOT EXISTS (SELECT 1 FROM list_subscribers ls JOIN email_lists el ON el.id = ls.list_id AND el.kind = 'outreach'
                       WHERE ls.contact_id = c.id) LIMIT 1),
    '') AS TEXT) AS found;

-- name: UpdateDoNotContactSettings :exec
UPDATE organizations SET dnc_customer_domains = ?, dnc_check_url = ?, dnc_check_secret_enc = ? WHERE id = ?;

-- name: ListOwnDomains :many
-- The brand's own domains: its sending and outreach domains.
SELECT LOWER(domain) AS domain FROM domain_identities WHERE org_id = ?;

-- ---- pace -----------------------------------------------------------------------------

-- name: SetPathwayDailyNewProspects :exec
UPDATE sending_pathways SET daily_new_prospects = ? WHERE id = ? AND org_id = ?;

-- name: AverageOutreachSteps :one
-- The average number of steps in the brand's active outreach campaigns.
SELECT CAST(COALESCE(AVG(n), 0) AS REAL) FROM (
    SELECT COUNT(et.id) AS n FROM email_sequences es
    JOIN email_templates et ON et.sequence_id = es.id
    WHERE es.org_id = ? AND es.sequence_type = 'outreach' AND es.is_active = 1
    GROUP BY es.id
);

-- ---- leads ----------------------------------------------------------------------------

-- name: ListOutreachLeads :many
-- One row per person per outreach campaign, for the leads export: the
-- segment, where their sequence stands and their last reply.
SELECT c.id AS contact_id, c.email, c.name, es.id AS campaign_id, es.name AS campaign_name,
       CAST(COALESCE(el.name, '') AS TEXT) AS segment_name, CAST(COALESCE(es.list_id, 0) AS INTEGER) AS list_id,
       css.is_active, css.stopped_reason, css.completed_at, css.started_at,
       CAST(COALESCE(r.classification, '') AS TEXT) AS reply_classification,
       CAST(COALESCE(r.received_at, '') AS TEXT) AS replied_at,
       CAST(css.sequence_id || ':' || css.contact_id AS TEXT) AS cursor_key
FROM contact_sequence_state css
JOIN email_sequences es ON es.id = css.sequence_id AND es.sequence_type = 'outreach'
JOIN contacts c ON c.id = css.contact_id
LEFT JOIN email_lists el ON el.id = es.list_id
LEFT JOIN email_replies r ON r.id = (
    SELECT r2.id FROM email_replies r2
    WHERE r2.contact_id = css.contact_id AND r2.sequence_id = css.sequence_id
      AND r2.classification NOT IN ('bounce', 'spam_block')
    ORDER BY r2.received_at DESC LIMIT 1)
WHERE es.org_id = sqlc.arg(org_id)
  AND (CAST(sqlc.arg(campaign_id) AS TEXT) = '' OR es.id = sqlc.arg(campaign_id))
  AND (CAST(sqlc.arg(classification) AS TEXT) = '' OR r.classification = sqlc.arg(classification))
  AND css.sequence_id || ':' || css.contact_id > CAST(sqlc.arg(after) AS TEXT)
ORDER BY cursor_key
LIMIT sqlc.arg(limit_count);
