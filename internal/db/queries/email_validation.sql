-- ========== VALIDATION JOBS ==========

-- name: CreateValidationJob :one
INSERT INTO validation_jobs (id, org_id, name, list_id, source, status)
VALUES (sqlc.arg(id), sqlc.arg(org_id), sqlc.arg(name), sqlc.arg(list_id), sqlc.arg(source), sqlc.arg(status))
RETURNING *;

-- name: GetValidationJob :one
SELECT * FROM validation_jobs WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id);

-- name: ListValidationJobs :many
SELECT * FROM validation_jobs
WHERE org_id = sqlc.arg(org_id)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountValidationJobs :one
SELECT COUNT(*) FROM validation_jobs WHERE org_id = sqlc.arg(org_id);

-- name: ListProcessingValidationJobs :many
SELECT * FROM validation_jobs WHERE status = 'processing' ORDER BY created_at;

-- name: StartValidationJob :execrows
UPDATE validation_jobs
SET status = 'processing', started_at = COALESCE(started_at, datetime('now'))
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id) AND status = 'pending';

-- name: CancelValidationJob :execrows
UPDATE validation_jobs
SET status = 'cancelled', completed_at = datetime('now')
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id) AND status IN ('pending', 'processing');

-- name: CompleteValidationJob :exec
UPDATE validation_jobs
SET status = 'completed', completed_at = datetime('now')
WHERE id = sqlc.arg(id) AND status = 'processing';

-- name: RefreshValidationJobProgress :exec
UPDATE validation_jobs
SET total = (SELECT COUNT(*) FROM email_validations ev WHERE ev.job_id = validation_jobs.id),
    processed = (SELECT COUNT(*) FROM email_validations ev WHERE ev.job_id = validation_jobs.id AND ev.status != 'pending')
WHERE validation_jobs.id = sqlc.arg(id);

-- ========== VALIDATION ROWS ==========

-- name: InsertValidationJobEmail :execrows
-- Queues one address in a job; duplicates within the job are skipped.
INSERT INTO email_validations (id, org_id, job_id, email, email_lower, domain, source, status)
SELECT sqlc.arg(id), sqlc.arg(org_id), sqlc.arg(job_id), sqlc.arg(email), LOWER(sqlc.arg(email)), sqlc.arg(domain), 'job', 'pending'
WHERE NOT EXISTS (
    SELECT 1 FROM email_validations ev
    WHERE ev.job_id = sqlc.arg(job_id) AND ev.email_lower = LOWER(sqlc.arg(email))
);

-- name: InsertEmailValidation :exec
INSERT INTO email_validations (
    id, org_id, job_id, email, email_lower, status, sub_status, free_email, did_you_mean,
    account, domain, domain_age_days, smtp_provider, mx_found, mx_record, source, attempts, processed_at
) VALUES (
    sqlc.arg(id), sqlc.arg(org_id), NULL, sqlc.arg(email), LOWER(sqlc.arg(email)), sqlc.arg(status), sqlc.arg(sub_status),
    sqlc.arg(free_email), sqlc.arg(did_you_mean), sqlc.arg(account), sqlc.arg(domain), sqlc.arg(domain_age_days),
    sqlc.arg(smtp_provider), sqlc.arg(mx_found), sqlc.arg(mx_record), sqlc.arg(source), 1, datetime('now')
);

-- name: ClaimPendingValidations :many
-- Leases due pending rows of a job for 10 minutes so a crash mid-batch only delays them.
UPDATE email_validations
SET retry_at = datetime('now', '+10 minutes')
WHERE id IN (
    SELECT ev.id FROM email_validations ev
    WHERE ev.job_id = sqlc.arg(job_id) AND ev.status = 'pending'
      AND (ev.retry_at IS NULL OR ev.retry_at <= datetime('now'))
    ORDER BY ev.domain, ev.created_at
    LIMIT sqlc.arg(batch_size)
)
RETURNING *;

-- name: UpdateEmailValidationResult :exec
-- status 'pending' with a retry_at schedules a retry; any other status finalizes the row.
UPDATE email_validations
SET status = sqlc.arg(status),
    sub_status = sqlc.arg(sub_status),
    free_email = sqlc.arg(free_email),
    did_you_mean = sqlc.arg(did_you_mean),
    account = sqlc.arg(account),
    domain = sqlc.arg(domain),
    domain_age_days = sqlc.arg(domain_age_days),
    smtp_provider = sqlc.arg(smtp_provider),
    mx_found = sqlc.arg(mx_found),
    mx_record = sqlc.arg(mx_record),
    attempts = sqlc.arg(attempts),
    retry_at = sqlc.arg(retry_at),
    processed_at = CASE WHEN sqlc.arg(status) = 'pending' THEN NULL ELSE datetime('now') END
WHERE id = sqlc.arg(id);

-- name: CountPendingValidations :one
SELECT COUNT(*) FROM email_validations WHERE job_id = sqlc.arg(job_id) AND status = 'pending';

-- name: ListValidationResults :many
SELECT * FROM email_validations
WHERE job_id = sqlc.arg(job_id) AND org_id = sqlc.arg(org_id)
  AND (sqlc.arg(filter_status) = '' OR status = sqlc.arg(filter_status))
ORDER BY email_lower
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountValidationResults :one
SELECT COUNT(*) FROM email_validations
WHERE job_id = sqlc.arg(job_id) AND org_id = sqlc.arg(org_id)
  AND (sqlc.arg(filter_status) = '' OR status = sqlc.arg(filter_status));

-- name: CountValidationJobStatuses :many
SELECT status, COUNT(*) AS count FROM email_validations
WHERE job_id = sqlc.arg(job_id)
GROUP BY status;

-- name: SuppressValidationJobResults :execrows
-- Adds a job's results with the given statuses to the org's suppression list.
INSERT INTO suppression_list (org_id, email, email_lower, reason, source)
SELECT ev.org_id, ev.email, ev.email_lower, 'validation: ' || ev.status || CASE WHEN ev.sub_status != '' THEN '/' || ev.sub_status ELSE '' END, 'validation'
FROM email_validations ev
WHERE ev.job_id = sqlc.arg(job_id) AND ev.org_id = sqlc.arg(org_id) AND ev.status IN (sqlc.slice(statuses))
ON CONFLICT (org_id, email_lower) DO NOTHING;

-- name: CountValidationUsageBySource :many
-- Checks completed in [from_time, until_time), grouped by source.
SELECT source, COUNT(*) AS count FROM email_validations
WHERE org_id = sqlc.arg(org_id) AND status != 'pending'
  AND processed_at >= sqlc.arg(from_time) AND processed_at < sqlc.arg(until_time)
GROUP BY source;

-- ========== CACHE ==========

-- name: GetValidationCache :one
SELECT result FROM email_validation_cache
WHERE email_lower = sqlc.arg(email_lower)
  AND checked_at >= sqlc.arg(fresh_after);

-- name: UpsertValidationCache :exec
INSERT INTO email_validation_cache (email_lower, result, checked_at)
VALUES (sqlc.arg(email_lower), sqlc.arg(result), datetime('now'))
ON CONFLICT (email_lower) DO UPDATE SET result = EXCLUDED.result, checked_at = EXCLUDED.checked_at;

-- ========== CONTACTS ==========

-- name: UpdateContactValidation :exec
UPDATE contacts
SET validation_status = sqlc.arg(status), validation_sub_status = sqlc.arg(sub_status), validated_at = datetime('now')
WHERE org_id = sqlc.arg(org_id) AND LOWER(email) = LOWER(sqlc.arg(email));

-- name: ListActiveListSubscriberEmails :many
SELECT c.email FROM list_subscribers ls
JOIN contacts c ON c.id = ls.contact_id
WHERE ls.list_id = sqlc.arg(list_id) AND ls.status = 'active';

-- ========== INSTANCE-WIDE SIGNALS (all orgs on this instance) ==========

-- name: HasEmailComplaint :one
SELECT COUNT(*) FROM email_complaint WHERE email_lower = LOWER(sqlc.arg(email));

-- name: IsEmailGloballySuppressed :one
-- Permanent hard bounce, or suppressed by any org. Suppressions created by validation
-- itself are ignored so a validation verdict never feeds back into the next one.
SELECT CAST((
    (SELECT COUNT(*) FROM email_bounce b WHERE b.email_lower = LOWER(sqlc.arg(email)) AND b.bounce_type = 'Permanent') +
    (SELECT COUNT(*) FROM suppression_list sl WHERE sl.email_lower = LOWER(sqlc.arg(email)) AND COALESCE(sl.source, '') != 'validation')
) AS INTEGER) AS found;

-- name: GetEmailEngagement :one
-- Campaign sends to this address across all orgs in the last 12 months, and its last open/click.
SELECT
    COUNT(cs.id) AS sends,
    CAST(COALESCE(MAX(cs.opened_at), '') AS TEXT) AS last_opened_at,
    CAST(COALESCE(MAX(cs.clicked_at), '') AS TEXT) AS last_clicked_at
FROM campaign_sends cs
JOIN contacts c ON c.id = cs.contact_id
WHERE LOWER(c.email) = LOWER(sqlc.arg(email))
  AND cs.status IN ('sent', 'delivered')
  AND COALESCE(cs.sent_at, cs.created_at) >= datetime('now', '-12 months');

-- name: GetDomainDeliveryStats :one
-- Campaign outcomes for every contact on a domain across all orgs. 'sent' counts as
-- delivered because SMTP sending never reports delivery, only bounces.
SELECT
    CAST(COALESCE(SUM(CASE WHEN cs.status IN ('sent', 'delivered') THEN 1 ELSE 0 END), 0) AS INTEGER) AS delivered,
    CAST(COALESCE(SUM(CASE WHEN cs.status = 'bounced' THEN 1 ELSE 0 END), 0) AS INTEGER) AS bounced
FROM campaign_sends cs
JOIN contacts c ON c.id = cs.contact_id
WHERE LOWER(c.email) LIKE '%@' || LOWER(sqlc.arg(domain));
