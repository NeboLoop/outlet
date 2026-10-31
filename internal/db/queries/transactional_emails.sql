-- Transactional Emails
-- API-triggered transactional email templates and sends

-- Templates

-- name: CreateTransactionalEmail :one
INSERT INTO transactional_emails (
    id, org_id, design_id, name, slug, description,
    subject, html_body, plain_text,
    from_name, from_email, reply_to, is_active,
    created_at, updated_at
)
VALUES (sqlc.arg(id), sqlc.arg(org_id), sqlc.arg(design_id), sqlc.arg(name), sqlc.arg(slug), sqlc.arg(description), sqlc.arg(subject), sqlc.arg(html_body), sqlc.arg(plain_text), sqlc.arg(from_name), sqlc.arg(from_email), sqlc.arg(reply_to), sqlc.arg(is_active), datetime('now'), datetime('now'))
RETURNING *;

-- name: GetTransactionalEmail :one
SELECT * FROM transactional_emails
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id);

-- name: GetTransactionalEmailBySlug :one
SELECT * FROM transactional_emails
WHERE slug = sqlc.arg(slug) AND org_id = sqlc.arg(org_id);

-- name: ListTransactionalEmails :many
SELECT * FROM transactional_emails
WHERE org_id = sqlc.arg(org_id)
ORDER BY created_at DESC;

-- name: UpdateTransactionalEmail :one
UPDATE transactional_emails
SET name = COALESCE(NULLIF(sqlc.arg(name), ''), name),
    slug = COALESCE(NULLIF(sqlc.arg(slug), ''), slug),
    description = COALESCE(NULLIF(sqlc.arg(description), ''), description),
    subject = COALESCE(NULLIF(sqlc.arg(subject), ''), subject),
    html_body = COALESCE(NULLIF(sqlc.arg(html_body), ''), html_body),
    plain_text = COALESCE(NULLIF(sqlc.arg(plain_text), ''), plain_text),
    from_name = COALESCE(NULLIF(sqlc.arg(from_name), ''), from_name),
    from_email = COALESCE(NULLIF(sqlc.arg(from_email), ''), from_email),
    reply_to = COALESCE(NULLIF(sqlc.arg(reply_to), ''), reply_to),
    is_active = COALESCE(sqlc.arg(is_active), is_active),
    updated_at = datetime('now')
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id)
RETURNING *;

-- name: DeleteTransactionalEmail :exec
DELETE FROM transactional_emails
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id);

-- name: CountTransactionalEmails :one
SELECT COUNT(*) as count FROM transactional_emails
WHERE org_id = sqlc.arg(org_id);

-- Sends (Logging)

-- name: CreateTransactionalSend :one
INSERT INTO transactional_sends (
    id, template_id, org_id, to_email, to_name, contact_id,
    status, tracking_token, context_data, created_at
)
VALUES (sqlc.arg(id), sqlc.arg(template_id), sqlc.arg(org_id), sqlc.arg(to_email), sqlc.arg(to_name), sqlc.arg(contact_id), sqlc.arg(status), sqlc.arg(tracking_token), sqlc.arg(context_data), datetime('now'))
RETURNING *;

-- name: GetTransactionalSend :one
SELECT * FROM transactional_sends
WHERE id = sqlc.arg(id);

-- name: GetTransactionalSendByTracking :one
SELECT * FROM transactional_sends
WHERE tracking_token = sqlc.arg(tracking_token);

-- name: GetTransactionalSendByTrackingAndOrg :one
SELECT ts.*, te.subject
FROM transactional_sends ts
JOIN transactional_emails te ON ts.template_id = te.id
WHERE ts.tracking_token = sqlc.arg(tracking_token) AND ts.org_id = sqlc.arg(org_id);

-- name: UpdateTransactionalSendStatus :exec
UPDATE transactional_sends
SET status = sqlc.arg(status),
    sent_at = CASE WHEN sqlc.arg(status) = 'sent' THEN datetime('now') ELSE sent_at END,
    delivered_at = CASE WHEN sqlc.arg(status) = 'delivered' THEN datetime('now') ELSE delivered_at END,
    error_message = sqlc.arg(error_message)
WHERE id = sqlc.arg(id);

-- name: RecordTransactionalOpen :exec
UPDATE transactional_sends
SET opened_at = COALESCE(opened_at, datetime('now'))
WHERE id = sqlc.arg(id);

-- name: RecordTransactionalClick :exec
UPDATE transactional_sends
SET clicked_at = COALESCE(clicked_at, datetime('now'))
WHERE id = sqlc.arg(id);

-- name: ListTransactionalSends :many
SELECT * FROM transactional_sends
WHERE template_id = sqlc.arg(template_id)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: ListTransactionalSendsByOrg :many
SELECT ts.*, te.name as template_name, te.slug as template_slug
FROM transactional_sends ts
JOIN transactional_emails te ON ts.template_id = te.id
WHERE ts.org_id = sqlc.arg(org_id)
ORDER BY ts.created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountTransactionalSends :one
SELECT COUNT(*) as count FROM transactional_sends
WHERE template_id = sqlc.arg(template_id);

-- name: GetTransactionalStats :one
SELECT
    COUNT(*) as total,
    SUM(CASE WHEN status = 'sent' OR status = 'delivered' THEN 1 ELSE 0 END) as sent,
    SUM(CASE WHEN status = 'delivered' THEN 1 ELSE 0 END) as delivered,
    SUM(CASE WHEN opened_at IS NOT NULL THEN 1 ELSE 0 END) as opened,
    SUM(CASE WHEN clicked_at IS NOT NULL THEN 1 ELSE 0 END) as clicked,
    SUM(CASE WHEN status = 'bounced' THEN 1 ELSE 0 END) as bounced,
    SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed
FROM transactional_sends
WHERE template_id = sqlc.arg(template_id);

-- name: GetRecentAPIKeySends :one
-- Mail sent with the brand's API key (the SDK and SMTP) in the last 7 days.
SELECT COUNT(*) AS sends, CAST(COALESCE(MAX(created_at), '') AS TEXT) AS last_sent_at
FROM transactional_sends
WHERE org_id = sqlc.arg(org_id) AND created_at >= datetime('now', '-7 days');
