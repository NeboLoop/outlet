-- Public Pages Queries
-- These support the server-side rendered public subscribe/confirm/unsubscribe pages

-- name: GetListBySlugForPublicPage :one
SELECT
    el.id,
    el.org_id,
    el.name,
    el.slug,
    el.description,
    el.double_optin,
    el.public_page_enabled,
    el.thank_you_url,
    el.confirm_redirect_url,
    el.unsubscribe_redirect_url,
    el.confirmation_email_subject,
    el.confirmation_email_body,
    o.name as org_name,
    o.slug as org_slug,
    o.from_name,
    o.from_email,
    o.settings as org_settings
FROM email_lists el
JOIN organizations o ON o.id = el.org_id
WHERE el.slug = sqlc.arg(slug)
  AND (el.public_page_enabled IS NULL OR el.public_page_enabled = 1)
LIMIT 1;

-- name: GetListByPublicIDForPublicPage :one
SELECT
    el.id,
    el.org_id,
    el.name,
    el.slug,
    el.public_id,
    el.description,
    el.double_optin,
    el.public_page_enabled,
    el.thank_you_url,
    el.confirm_redirect_url,
    el.unsubscribe_redirect_url,
    el.confirmation_email_subject,
    el.confirmation_email_body,
    o.name as org_name,
    o.slug as org_slug,
    o.from_name,
    o.from_email,
    o.settings as org_settings
FROM email_lists el
JOIN organizations o ON o.id = el.org_id
WHERE el.public_id = sqlc.arg(public_id)
  AND (el.public_page_enabled IS NULL OR el.public_page_enabled = 1)
LIMIT 1;

-- name: GetListByIDForPublicPage :one
SELECT
    el.id,
    el.org_id,
    el.name,
    el.slug,
    el.description,
    el.double_optin,
    el.public_page_enabled,
    el.thank_you_url,
    el.confirm_redirect_url,
    el.unsubscribe_redirect_url,
    el.confirmation_email_subject,
    el.confirmation_email_body,
    o.name as org_name,
    o.slug as org_slug,
    o.from_name,
    o.from_email,
    o.settings as org_settings
FROM email_lists el
JOIN organizations o ON o.id = el.org_id
WHERE el.id = sqlc.arg(id)
  AND (el.public_page_enabled IS NULL OR el.public_page_enabled = 1)
LIMIT 1;

-- name: GetContactByEmailForPublicPage :one
SELECT c.*, ls.status as subscription_status, ls.list_id
FROM contacts c
LEFT JOIN list_subscribers ls ON ls.contact_id = c.id AND ls.list_id = sqlc.arg(list_id)
WHERE c.email = sqlc.arg(email) AND c.org_id = sqlc.arg(org_id)
LIMIT 1;

-- name: GetEmailQueueByTrackingToken :one
SELECT
    eq.id,
    eq.tracking_token,
    eq.contact_id,
    eq.template_id,
    eq.status,
    c.email,
    c.name,
    et.subject,
    et.html_body,
    et.template_type,
    et.is_transactional,
    el.name as list_name,
    el.unsubscribe_redirect_url,
    o.name as org_name
FROM email_queue eq
JOIN contacts c ON c.id = eq.contact_id
LEFT JOIN email_templates et ON et.id = eq.template_id
LEFT JOIN email_sequences es ON es.id = et.sequence_id
LEFT JOIN email_lists el ON el.id = es.list_id
LEFT JOIN organizations o ON o.id = el.org_id
WHERE eq.tracking_token = sqlc.arg(token)
LIMIT 1;

-- name: GetCampaignSendByTrackingToken :one
-- The send with all CampaignMessage needs to build it, design included.
SELECT
    cs.id,
    cs.campaign_id,
    cs.contact_id,
    cs.tracking_token,
    cs.status,
    c.email,
    c.name,
    ec.org_id,
    ec.subject,
    ec.html_body,
    ec.plain_text,
    ec.from_name,
    ec.from_email,
    ec.reply_to,
    ec.track_opens,
    ec.track_clicks,
    CAST(COALESCE(d.html_body, '') AS TEXT) AS design_html,
    CAST(COALESCE(d.plain_text, '') AS TEXT) AS design_text,
    el.name as list_name,
    el.unsubscribe_redirect_url,
    o.name as org_name
FROM campaign_sends cs
JOIN contacts c ON c.id = cs.contact_id
JOIN email_campaigns ec ON ec.id = cs.campaign_id
LEFT JOIN email_designs d ON d.id = ec.design_id AND d.org_id = ec.org_id
LEFT JOIN email_lists el ON el.id = cs.list_id
LEFT JOIN organizations o ON o.id = ec.org_id
WHERE cs.tracking_token = sqlc.arg(token)
LIMIT 1;

-- name: GetTransactionalSendByTrackingToken :one
SELECT
    ts.id,
    ts.contact_id,
    ts.tracking_token,
    ts.status,
    ts.to_email as email,
    ts.to_name as name,
    te.subject,
    te.html_body,
    o.name as org_name
FROM transactional_sends ts
JOIN transactional_emails te ON te.id = ts.template_id
JOIN organizations o ON o.id = ts.org_id
WHERE ts.tracking_token = sqlc.arg(token)
LIMIT 1;

-- name: UpdateListPublicPageSettings :exec
UPDATE email_lists
SET public_page_enabled = sqlc.arg(public_page_enabled),
    thank_you_url = sqlc.arg(thank_you_url),
    confirm_redirect_url = sqlc.arg(confirm_redirect_url),
    unsubscribe_redirect_url = sqlc.arg(unsubscribe_redirect_url),
    updated_at = datetime('now')
WHERE id = sqlc.arg(id);

-- name: GetOrgEmailConfig :one
SELECT id, settings FROM organizations WHERE id = sqlc.arg(id);

-- name: UpdateOrgSettings :exec
UPDATE organizations
SET settings = sqlc.arg(settings), updated_at = datetime('now')
WHERE id = sqlc.arg(id);
