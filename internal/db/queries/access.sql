-- Which brand a record belongs to, for records that don't carry it directly
-- (or don't always). The access check uses the answer.

-- name: GetSequenceOrgID :one
SELECT COALESCE(s.org_id, l.org_id, '') AS org_id
FROM email_sequences s
LEFT JOIN email_lists l ON l.id = s.list_id
WHERE s.id = sqlc.arg(id);

-- name: GetTemplateOrgID :one
SELECT COALESCE(t.org_id, s.org_id, l.org_id, '') AS org_id
FROM email_templates t
LEFT JOIN email_sequences s ON s.id = t.sequence_id
LEFT JOIN email_lists l ON l.id = s.list_id
WHERE t.id = sqlc.arg(id);

-- name: GetEntryRuleOrgID :one
SELECT COALESCE(s.org_id, l.org_id, '') AS org_id
FROM sequence_entry_rules r
JOIN email_sequences s ON s.id = r.sequence_id
LEFT JOIN email_lists l ON l.id = s.list_id
WHERE r.id = sqlc.arg(id);

-- name: GetQueuedEmailOrgID :one
SELECT COALESCE(c.org_id, '') AS org_id
FROM email_queue q
JOIN contacts c ON c.id = q.contact_id
WHERE q.id = sqlc.arg(id);
