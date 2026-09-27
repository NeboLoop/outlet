-- Email blocklist queries for bounce and complaint management

-- ========== BOUNCES ==========

-- name: CreateEmailBounce :one
INSERT INTO email_bounce (
    id, email, email_lower, bounce_type, bounce_subtype,
    diagnostic_code, source_email, message_id, raw_notification, created_at
) VALUES (sqlc.arg(id), sqlc.arg(email), LOWER(sqlc.arg(email_for_lower)), sqlc.arg(bounce_type), sqlc.arg(bounce_subtype), sqlc.arg(diagnostic_code), sqlc.arg(source_email), sqlc.arg(message_id), sqlc.arg(raw_notification), datetime('now'))
ON CONFLICT (email_lower) DO UPDATE
SET bounce_type = EXCLUDED.bounce_type,
    bounce_subtype = EXCLUDED.bounce_subtype,
    diagnostic_code = EXCLUDED.diagnostic_code,
    source_email = EXCLUDED.source_email,
    message_id = EXCLUDED.message_id,
    raw_notification = EXCLUDED.raw_notification,
    created_at = datetime('now')
RETURNING *;

-- name: GetEmailBounce :one
SELECT * FROM email_bounce
WHERE email_lower = LOWER(sqlc.arg(email));

-- name: IsEmailBounced :one
SELECT CASE WHEN COUNT(*) > 0 THEN 1 ELSE 0 END as found
FROM email_bounce WHERE email_lower = LOWER(sqlc.arg(email));

-- name: DeleteEmailBounce :exec
DELETE FROM email_bounce WHERE email_lower = LOWER(sqlc.arg(email));

-- name: ListRecentBounces :many
SELECT * FROM email_bounce
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountBouncesInDateRange :one
SELECT COUNT(*) FROM email_bounce
WHERE created_at >= sqlc.arg(start_date) AND created_at <= sqlc.arg(end_date);

-- ========== COMPLAINTS ==========

-- name: CreateEmailComplaint :one
INSERT INTO email_complaint (
    id, email, email_lower, complaint_type, feedback_id,
    source_email, message_id, raw_notification, created_at
) VALUES (sqlc.arg(id), sqlc.arg(email), LOWER(sqlc.arg(email_for_lower)), sqlc.arg(complaint_type), sqlc.arg(feedback_id), sqlc.arg(source_email), sqlc.arg(message_id), sqlc.arg(raw_notification), datetime('now'))
ON CONFLICT (email_lower) DO UPDATE
SET complaint_type = EXCLUDED.complaint_type,
    feedback_id = EXCLUDED.feedback_id,
    source_email = EXCLUDED.source_email,
    message_id = EXCLUDED.message_id,
    raw_notification = EXCLUDED.raw_notification,
    created_at = datetime('now')
RETURNING *;

-- name: GetEmailComplaint :one
SELECT * FROM email_complaint
WHERE email_lower = LOWER(sqlc.arg(email));

-- name: IsEmailComplaint :one
SELECT CASE WHEN COUNT(*) > 0 THEN 1 ELSE 0 END as found
FROM email_complaint WHERE email_lower = LOWER(sqlc.arg(email));

-- name: DeleteEmailComplaint :exec
DELETE FROM email_complaint WHERE email_lower = LOWER(sqlc.arg(email));

-- name: ListRecentComplaints :many
SELECT * FROM email_complaint
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountComplaintsInDateRange :one
SELECT COUNT(*) FROM email_complaint
WHERE created_at >= sqlc.arg(start_date) AND created_at <= sqlc.arg(end_date);

-- ========== COMBINED CHECK ==========

-- name: IsEmailBlocked :one
SELECT CASE WHEN (
    (SELECT COUNT(*) FROM email_bounce WHERE email_lower = LOWER(sqlc.arg(email))) +
    (SELECT COUNT(*) FROM email_complaint WHERE email_lower = LOWER(sqlc.arg(email_2)))
) > 0 THEN 1 ELSE 0 END as found;

-- ========== BLOCK CONTACT BY EMAIL ==========

-- name: BlockContactByEmail :exec
UPDATE contacts SET blocked_at = datetime('now'), updated_at = datetime('now')
WHERE LOWER(email) = LOWER(sqlc.arg(email)) AND blocked_at IS NULL;

-- ========== BLOCKED DOMAINS ==========

-- name: CreateBlockedDomain :one
INSERT INTO blocked_domains (org_id, domain, reason)
VALUES (sqlc.arg(org_id), LOWER(sqlc.arg(domain)), sqlc.arg(reason))
ON CONFLICT (org_id, domain) DO UPDATE SET
    reason = COALESCE(EXCLUDED.reason, blocked_domains.reason),
    updated_at = datetime('now')
RETURNING *;

-- name: GetBlockedDomain :one
SELECT * FROM blocked_domains
WHERE org_id = sqlc.arg(org_id) AND domain = LOWER(sqlc.arg(domain));

-- name: ListBlockedDomains :many
SELECT * FROM blocked_domains
WHERE org_id = sqlc.arg(org_id)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountBlockedDomains :one
SELECT COUNT(*) FROM blocked_domains WHERE org_id = sqlc.arg(org_id);

-- name: DeleteBlockedDomain :exec
DELETE FROM blocked_domains
WHERE org_id = sqlc.arg(org_id) AND domain = LOWER(sqlc.arg(domain));

-- name: DeleteBlockedDomainByID :exec
DELETE FROM blocked_domains WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id);

-- name: IsDomainBlocked :one
SELECT CASE WHEN COUNT(*) > 0 THEN 1 ELSE 0 END as found
FROM blocked_domains
WHERE org_id = sqlc.arg(org_id) AND domain = LOWER(sqlc.arg(domain));

-- name: IncrementBlockedDomainAttempts :exec
UPDATE blocked_domains
SET block_attempts = block_attempts + 1, updated_at = datetime('now')
WHERE org_id = sqlc.arg(org_id) AND domain = LOWER(sqlc.arg(domain));

-- name: BulkInsertBlockedDomains :exec
INSERT INTO blocked_domains (org_id, domain, reason)
VALUES (sqlc.arg(org_id), LOWER(sqlc.arg(domain)), sqlc.arg(reason))
ON CONFLICT (org_id, domain) DO NOTHING;

-- ========== SUPPRESSION LIST ==========

-- name: AddToSuppressionList :one
INSERT INTO suppression_list (org_id, email, email_lower, reason, source)
VALUES (sqlc.arg(org_id), sqlc.arg(email), LOWER(sqlc.arg(email)), sqlc.arg(reason), sqlc.arg(source))
ON CONFLICT (org_id, email_lower) DO UPDATE SET
    reason = COALESCE(EXCLUDED.reason, suppression_list.reason),
    source = EXCLUDED.source
RETURNING *;

-- name: GetSuppressedEmail :one
SELECT * FROM suppression_list
WHERE org_id = sqlc.arg(org_id) AND email_lower = LOWER(sqlc.arg(email));

-- name: ListSuppressedEmails :many
SELECT * FROM suppression_list
WHERE org_id = sqlc.arg(org_id)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountSuppressedEmails :one
SELECT COUNT(*) FROM suppression_list WHERE org_id = sqlc.arg(org_id);

-- name: DeleteFromSuppressionList :exec
DELETE FROM suppression_list
WHERE org_id = sqlc.arg(org_id) AND email_lower = LOWER(sqlc.arg(email));

-- name: DeleteSuppressionByID :exec
DELETE FROM suppression_list WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id);

-- name: IsEmailSuppressed :one
SELECT CASE WHEN COUNT(*) > 0 THEN 1 ELSE 0 END as found
FROM suppression_list
WHERE org_id = sqlc.arg(org_id) AND email_lower = LOWER(sqlc.arg(email));

-- name: IncrementSuppressionAttempts :exec
UPDATE suppression_list
SET block_attempts = block_attempts + 1
WHERE org_id = sqlc.arg(org_id) AND email_lower = LOWER(sqlc.arg(email));

-- name: ClearSuppressionList :exec
DELETE FROM suppression_list WHERE org_id = sqlc.arg(org_id);

-- ========== COMBINED BLOCK CHECK (Enhanced) ==========

-- name: IsEmailFullyBlocked :one
SELECT CASE WHEN (
    (SELECT COUNT(*) FROM email_bounce WHERE email_lower = LOWER(sqlc.arg(check_email))) +
    (SELECT COUNT(*) FROM email_complaint WHERE email_lower = LOWER(sqlc.arg(check_email))) +
    (SELECT COUNT(*) FROM suppression_list sl WHERE sl.org_id = sqlc.arg(check_org_id) AND sl.email_lower = LOWER(sqlc.arg(check_email))) +
    (SELECT COUNT(*) FROM blocked_domains bd WHERE bd.org_id = sqlc.arg(check_org_id) AND bd.domain = LOWER(SUBSTR(sqlc.arg(check_email), INSTR(sqlc.arg(check_email), '@') + 1)))
) > 0 THEN 1 ELSE 0 END as found;

-- ========== IMPORT JOBS ==========

-- name: CreateImportJob :one
INSERT INTO import_jobs (id, org_id, list_id, type, filename, options)
VALUES (sqlc.arg(id), sqlc.arg(org_id), sqlc.arg(list_id), sqlc.arg(type), sqlc.arg(filename), sqlc.arg(options))
RETURNING *;

-- name: GetImportJob :one
SELECT * FROM import_jobs WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id);

-- name: ListImportJobs :many
SELECT * FROM import_jobs
WHERE org_id = sqlc.arg(org_id)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: UpdateImportJobStatus :exec
UPDATE import_jobs
SET status = sqlc.arg(status),
    started_at = CASE WHEN sqlc.arg(status) = 'processing' THEN datetime('now') ELSE started_at END,
    completed_at = CASE WHEN sqlc.arg(status) IN ('completed', 'failed', 'cancelled') THEN datetime('now') ELSE completed_at END
WHERE id = sqlc.arg(id);

-- name: UpdateImportJobProgress :exec
UPDATE import_jobs
SET processed_rows = sqlc.arg(processed_rows),
    success_count = sqlc.arg(success_count),
    error_count = sqlc.arg(error_count),
    skip_count = sqlc.arg(skip_count)
WHERE id = sqlc.arg(id);

-- name: SetImportJobTotalRows :exec
UPDATE import_jobs
SET total_rows = sqlc.arg(total_rows)
WHERE id = sqlc.arg(id);

-- name: SetImportJobErrors :exec
UPDATE import_jobs
SET errors = sqlc.arg(errors)
WHERE id = sqlc.arg(id);

-- name: DeleteImportJob :exec
DELETE FROM import_jobs WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id);

-- name: CancelImportJob :exec
UPDATE import_jobs
SET status = 'cancelled', completed_at = datetime('now')
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id) AND status IN ('pending', 'processing');

-- name: ListPendingImportJobs :many
SELECT * FROM import_jobs
WHERE status = 'pending'
ORDER BY created_at ASC
LIMIT 10;

-- name: AddBlockedDomain :one
INSERT INTO blocked_domains (id, org_id, domain)
VALUES (sqlc.arg(id), sqlc.arg(org_id), LOWER(sqlc.arg(domain)))
ON CONFLICT (org_id, domain) DO NOTHING
RETURNING *;
