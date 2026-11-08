-- name: GetEmailListByOrgAndSlug :one
SELECT * FROM email_lists
WHERE org_id = sqlc.arg(org_id) AND slug = sqlc.arg(slug)
LIMIT 1;

-- name: CreateEmailList :one
INSERT INTO email_lists (
    public_id, org_id, name, slug, description, double_optin, created_at, updated_at
) VALUES (
    sqlc.arg(public_id), sqlc.arg(org_id), sqlc.arg(name), sqlc.arg(slug), sqlc.arg(description),
    sqlc.arg(double_optin), datetime('now'), datetime('now')
) RETURNING *;

-- name: ListEmailLists :many
SELECT * FROM email_lists
WHERE org_id = sqlc.arg(org_id)
ORDER BY created_at DESC;

-- name: GetAllOrgsListCounts :many
SELECT
    org_id,
    COUNT(*) as list_count
FROM email_lists
GROUP BY org_id;

-- name: GetEmailList :one
SELECT * FROM email_lists
WHERE id = sqlc.arg(id);

-- name: GetEmailListByPublicID :one
SELECT * FROM email_lists
WHERE public_id = sqlc.arg(public_id);

-- name: UpdateEmailList :one
UPDATE email_lists
SET name = COALESCE(sqlc.arg(name), name),
    description = COALESCE(sqlc.arg(description), description),
    double_optin = COALESCE(sqlc.arg(double_optin), double_optin),
    confirmation_email_subject = COALESCE(NULLIF(sqlc.arg(confirmation_subject), ''), confirmation_email_subject),
    confirmation_email_body = COALESCE(NULLIF(sqlc.arg(confirmation_body), ''), confirmation_email_body),
    thank_you_url = COALESCE(NULLIF(sqlc.arg(thank_you_url), ''), thank_you_url),
    confirm_redirect_url = COALESCE(NULLIF(sqlc.arg(confirm_redirect_url), ''), confirm_redirect_url),
    already_subscribed_url = COALESCE(NULLIF(sqlc.arg(already_subscribed_url), ''), already_subscribed_url),
    unsubscribe_redirect_url = COALESCE(NULLIF(sqlc.arg(unsubscribe_redirect_url), ''), unsubscribe_redirect_url),
    thank_you_email_enabled = COALESCE(sqlc.arg(thank_you_email_enabled), thank_you_email_enabled),
    thank_you_email_subject = COALESCE(NULLIF(sqlc.arg(thank_you_email_subject), ''), thank_you_email_subject),
    thank_you_email_body = COALESCE(NULLIF(sqlc.arg(thank_you_email_body), ''), thank_you_email_body),
    goodbye_email_enabled = COALESCE(sqlc.arg(goodbye_email_enabled), goodbye_email_enabled),
    goodbye_email_subject = COALESCE(NULLIF(sqlc.arg(goodbye_email_subject), ''), goodbye_email_subject),
    goodbye_email_body = COALESCE(NULLIF(sqlc.arg(goodbye_email_body), ''), goodbye_email_body),
    unsubscribe_behavior = COALESCE(NULLIF(sqlc.arg(unsubscribe_behavior), ''), unsubscribe_behavior),
    unsubscribe_scope = COALESCE(NULLIF(sqlc.arg(unsubscribe_scope), ''), unsubscribe_scope),
    updated_at = datetime('now')
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteEmailList :exec
DELETE FROM email_lists
WHERE id = sqlc.arg(id);

-- name: SubscribeToList :one
INSERT INTO list_subscribers (
    id, list_id, contact_id, status, subscribed_at
) VALUES (
    sqlc.arg(id), sqlc.arg(list_id), sqlc.arg(contact_id), 'active', datetime('now')
)
ON CONFLICT (list_id, contact_id) DO UPDATE
SET status = 'active', unsubscribed_at = NULL
RETURNING *;

-- name: UnsubscribeFromList :exec
UPDATE list_subscribers
SET status = 'unsubscribed', unsubscribed_at = datetime('now')
WHERE list_id = sqlc.arg(list_id) AND contact_id = sqlc.arg(contact_id);

-- name: GetListSubscriber :one
SELECT * FROM list_subscribers
WHERE list_id = sqlc.arg(list_id) AND contact_id = sqlc.arg(contact_id);

-- name: ListListSubscribers :many
SELECT ls.*, c.email, c.name, c.validation_status, c.validation_sub_status
FROM list_subscribers ls
JOIN contacts c ON c.id = ls.contact_id
WHERE ls.list_id = sqlc.arg(list_id)
  AND (sqlc.arg(filter_status) IS NULL OR sqlc.arg(filter_status) = '' OR ls.status = sqlc.arg(filter_status))
ORDER BY ls.subscribed_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountListSubscribers :one
SELECT COUNT(*) as count FROM list_subscribers
WHERE list_id = sqlc.arg(list_id)
  AND (sqlc.arg(filter_status) IS NULL OR sqlc.arg(filter_status) = '' OR status = sqlc.arg(filter_status));

-- name: SubscribeToListPending :one
INSERT INTO list_subscribers (
    id, list_id, contact_id, status, verification_token, verification_sent_at
) VALUES (
    sqlc.arg(id), sqlc.arg(list_id), sqlc.arg(contact_id), 'pending',
    sqlc.arg(verification_token), datetime('now')
)
ON CONFLICT (list_id, contact_id) DO UPDATE
SET status = CASE
    WHEN list_subscribers.status = 'active' THEN 'active'
    ELSE 'pending'
END,
verification_token = CASE
    WHEN list_subscribers.status = 'active' THEN list_subscribers.verification_token
    ELSE EXCLUDED.verification_token
END,
verification_sent_at = CASE
    WHEN list_subscribers.status = 'active' THEN list_subscribers.verification_sent_at
    ELSE datetime('now')
END,
unsubscribed_at = NULL
RETURNING *;

-- name: ConfirmListSubscription :one
UPDATE list_subscribers
-- The token is kept so opening the link again (a link scanner often opens
-- it first) shows "confirmed" instead of "invalid link"
SET status = 'active', verified_at = datetime('now')
WHERE verification_token = sqlc.arg(token) AND status = 'pending'
RETURNING *;

-- name: GetListSubscriberByToken :one
SELECT ls.*, el.name as list_name, el.org_id
FROM list_subscribers ls
JOIN email_lists el ON el.id = ls.list_id
WHERE ls.verification_token = sqlc.arg(token);

-- name: RemoveSubscriberFromList :exec
DELETE FROM list_subscribers
WHERE list_id = sqlc.arg(list_id) AND contact_id = sqlc.arg(contact_id);

-- name: GetListSubscriberByID :one
SELECT ls.*, c.email, c.name
FROM list_subscribers ls
JOIN contacts c ON c.id = ls.contact_id
WHERE ls.id = sqlc.arg(id);

-- name: CountActiveSubscribers :one
SELECT COUNT(*) as count FROM list_subscribers
WHERE list_id = sqlc.arg(list_id) AND status = 'active';

-- name: CheckListSubscription :one
SELECT COUNT(*) as count FROM list_subscribers
WHERE list_id = sqlc.arg(list_id) AND contact_id = sqlc.arg(contact_id);

-- name: CreateListSubscriber :one
INSERT INTO list_subscribers (id, list_id, contact_id, status, subscribed_at)
VALUES (sqlc.arg(id), sqlc.arg(list_id), sqlc.arg(contact_id), sqlc.arg(status), datetime('now'))
ON CONFLICT (list_id, contact_id) DO NOTHING
RETURNING *;

-- name: GetListSubscriberDetail :one
SELECT
    ls.id,
    ls.list_id,
    ls.contact_id,
    ls.status,
    ls.subscribed_at,
    ls.verified_at,
    ls.unsubscribed_at,
    c.email,
    c.name,
    c.source,
    c.created_at as contact_created_at,
    c.email_verified,
    c.gdpr_consent,
    c.gdpr_consent_at,
    c.blocked_at,
    -- Email stats: count from campaign_sends for this contact
    COALESCE((SELECT COUNT(*) FROM campaign_sends cs WHERE cs.contact_id = c.id), 0) as emails_sent,
    COALESCE((SELECT COUNT(*) FROM campaign_sends cs WHERE cs.contact_id = c.id AND cs.opened_at IS NOT NULL), 0) as emails_opened,
    COALESCE((SELECT COUNT(*) FROM campaign_sends cs WHERE cs.contact_id = c.id AND cs.clicked_at IS NOT NULL), 0) as emails_clicked,
    -- Last activity timestamps
    (SELECT MAX(sent_at) FROM campaign_sends cs WHERE cs.contact_id = c.id) as last_email_at,
    (SELECT MAX(opened_at) FROM campaign_sends cs WHERE cs.contact_id = c.id AND opened_at IS NOT NULL) as last_open_at,
    (SELECT MAX(clicked_at) FROM campaign_sends cs WHERE cs.contact_id = c.id AND clicked_at IS NOT NULL) as last_click_at
FROM list_subscribers ls
JOIN contacts c ON c.id = ls.contact_id
WHERE ls.id = sqlc.arg(id);

-- name: GetSubscriberCampaignActivity :many
SELECT
    cs.id,
    cs.campaign_id,
    ec.name as campaign_name,
    ec.subject as campaign_subject,
    cs.status,
    cs.sent_at,
    cs.opened_at,
    cs.open_count,
    cs.clicked_at,
    cs.click_count
FROM campaign_sends cs
LEFT JOIN email_campaigns ec ON ec.id = cs.campaign_id
WHERE cs.contact_id = sqlc.arg(contact_id)
ORDER BY cs.sent_at DESC
LIMIT 50;

-- name: GetSubscriberSequenceEnrollments :many
SELECT
    css.id,
    css.sequence_id,
    es.name as sequence_name,
    css.current_position,
    css.is_active,
    css.started_at,
    css.completed_at,
    css.paused_at,
    css.unsubscribed_at
FROM contact_sequence_state css
LEFT JOIN email_sequences es ON es.id = css.sequence_id
WHERE css.contact_id = sqlc.arg(contact_id)
ORDER BY css.started_at DESC;
