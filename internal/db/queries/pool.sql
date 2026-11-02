-- Mailbox pool queries. Every time is passed in by the pool service (UTC
-- "YYYY-MM-DD HH:MM:SS"), never datetime('now'), so the rules read one clock.


-- name: CreatePoolDomain :one
INSERT INTO pool_domains (id, domain, created_at)
VALUES (sqlc.arg(id), LOWER(sqlc.arg(domain)), sqlc.arg(created_at))
RETURNING *;

-- name: GetPoolDomain :one
SELECT * FROM pool_domains WHERE id = sqlc.arg(id);

-- name: ListPoolDomains :many
SELECT * FROM pool_domains ORDER BY domain;

-- name: PausePoolDomain :execrows
UPDATE pool_domains SET status = 'paused', paused_reason = sqlc.arg(paused_reason)
WHERE id = sqlc.arg(id) AND status = 'active';

-- name: ResumePoolDomain :execrows
UPDATE pool_domains SET status = 'active', paused_reason = NULL
WHERE id = sqlc.arg(id) AND status = 'paused';

-- name: RetirePoolDomain :execrows
UPDATE pool_domains SET status = 'retired'
WHERE id = sqlc.arg(id) AND status <> 'retired';

-- name: CountPoolDomainMailboxes :one
SELECT
    CAST(COUNT(*) AS INTEGER) AS total,
    CAST(COALESCE(SUM(CASE WHEN status = 'paused' THEN 1 ELSE 0 END), 0) AS INTEGER) AS paused
FROM pool_mailboxes
WHERE domain_id = sqlc.arg(domain_id) AND status <> 'retired';


-- name: CreatePoolMailbox :one
INSERT INTO pool_mailboxes (
    id, domain_id, address, smtp_host, smtp_port, smtp_username, smtp_password_enc,
    warm_started_at, health_since, created_at
) VALUES (
    sqlc.arg(id), sqlc.arg(domain_id), LOWER(sqlc.arg(address)), sqlc.arg(smtp_host), sqlc.arg(smtp_port),
    sqlc.arg(smtp_username), sqlc.arg(smtp_password_enc), sqlc.arg(warm_started_at),
    sqlc.arg(created_at), sqlc.arg(created_at)
)
RETURNING *;

-- name: GetPoolMailbox :one
SELECT * FROM pool_mailboxes WHERE id = sqlc.arg(id);

-- name: ListPoolMailboxes :many
SELECT * FROM pool_mailboxes ORDER BY address;

-- name: ListPoolMailboxesByDomain :many
SELECT * FROM pool_mailboxes WHERE domain_id = sqlc.arg(domain_id) ORDER BY address;

-- name: ListTenantPoolMailboxes :many
SELECT * FROM pool_mailboxes
WHERE tenant_org_id = sqlc.arg(org_id) AND tenant = sqlc.arg(tenant)
ORDER BY address;

-- name: PausePoolMailbox :execrows
UPDATE pool_mailboxes
SET status = 'paused', paused_reason = sqlc.arg(paused_reason), paused_kind = sqlc.arg(paused_kind)
WHERE id = sqlc.arg(id) AND status IN ('active', 'resting');

-- name: ResumePoolMailbox :execrows
UPDATE pool_mailboxes
SET status = CASE WHEN tenant IS NULL AND last_tenant IS NOT NULL THEN 'resting' ELSE 'active' END,
    paused_reason = NULL, paused_kind = NULL, health_since = sqlc.arg(health_since)
WHERE id = sqlc.arg(id) AND status = 'paused';

-- name: RetirePoolMailbox :execrows
UPDATE pool_mailboxes SET status = 'retired'
WHERE id = sqlc.arg(id) AND status <> 'retired';

-- AssignablePoolMailboxes lists free mailboxes this tenant may take: never
-- paused or retired, on an active domain, and either never assigned,
-- rested past the cutoff, or last held by this same tenant.
-- name: AssignablePoolMailboxes :many
SELECT m.* FROM pool_mailboxes m
JOIN pool_domains d ON d.id = m.domain_id
WHERE m.tenant IS NULL
  AND m.status IN ('active', 'resting')
  AND d.status = 'active'
  AND (
      m.tenant_released_at IS NULL
      OR m.tenant_released_at <= sqlc.arg(rest_cutoff)
      OR (m.last_tenant_org_id = sqlc.arg(org_id) AND m.last_tenant = sqlc.arg(tenant))
  )
ORDER BY m.warm_started_at, m.address
LIMIT sqlc.arg(max_count);

-- name: AssignPoolMailbox :execrows
UPDATE pool_mailboxes
SET tenant_org_id = sqlc.arg(org_id), tenant = sqlc.arg(tenant),
    tenant_assigned_at = sqlc.arg(assigned_at), status = 'active'
WHERE id = sqlc.arg(id) AND tenant IS NULL AND status IN ('active', 'resting');

-- name: ReleaseTenantPoolMailboxes :execrows
UPDATE pool_mailboxes
SET last_tenant_org_id = tenant_org_id, last_tenant = tenant,
    tenant_org_id = NULL, tenant = NULL, tenant_released_at = sqlc.arg(released_at),
    status = CASE WHEN status = 'active' THEN 'resting' ELSE status END
WHERE tenant_org_id = sqlc.arg(org_id) AND tenant = sqlc.arg(tenant);


-- name: CreatePoolSend :exec
INSERT INTO pool_sends (id, org_id, mailbox_id, tenant, idempotency_key, message_id, to_email, created_at, status)
VALUES (sqlc.arg(id), sqlc.arg(org_id), sqlc.arg(mailbox_id), sqlc.arg(tenant), sqlc.arg(idempotency_key),
        sqlc.arg(message_id), LOWER(sqlc.arg(to_email)), sqlc.arg(created_at), 'sending');

-- name: FinishPoolSend :exec
UPDATE pool_sends
SET status = sqlc.arg(status), sent_at = sqlc.narg(sent_at), failure_kind = sqlc.narg(failure_kind),
    smtp_code = sqlc.narg(smtp_code), smtp_response = sqlc.narg(smtp_response)
WHERE id = sqlc.arg(id);

-- name: MarkPoolSendPolicyBlocked :exec
UPDATE pool_sends SET failure_kind = 'policy', smtp_response = sqlc.narg(smtp_response)
WHERE id = sqlc.arg(id);

-- name: GetPoolSend :one
SELECT * FROM pool_sends WHERE id = sqlc.arg(id);

-- name: GetPoolSendByMessageID :one
SELECT * FROM pool_sends WHERE message_id = sqlc.arg(message_id);

-- name: GetPoolSendByIdempotencyKey :one
SELECT * FROM pool_sends WHERE org_id = sqlc.arg(org_id) AND idempotency_key = sqlc.arg(idempotency_key);

-- name: LastPoolSendToRecipient :one
SELECT * FROM pool_sends
WHERE org_id = sqlc.arg(org_id) AND tenant = sqlc.arg(tenant) AND to_email = LOWER(sqlc.arg(to_email))
ORDER BY created_at DESC
LIMIT 1;

-- CountPoolMailboxAttemptsSince counts the sends that reached the mail
-- provider (a refused connection or sign-in never did).
-- name: CountPoolMailboxAttemptsSince :one
SELECT CAST(COUNT(*) AS INTEGER) FROM pool_sends
WHERE mailbox_id = sqlc.arg(mailbox_id) AND created_at >= sqlc.arg(since)
  AND COALESCE(failure_kind, '') <> 'provider';

-- name: LastPoolMailboxAttemptAt :one
SELECT created_at FROM pool_sends
WHERE mailbox_id = sqlc.arg(mailbox_id)
ORDER BY created_at DESC
LIMIT 1;

-- name: CountPoolPolicyBlocksSince :one
SELECT CAST(COUNT(*) AS INTEGER) FROM pool_sends
WHERE mailbox_id = sqlc.arg(mailbox_id) AND created_at >= sqlc.arg(since) AND failure_kind = 'policy';

-- name: RecentPoolSendFailureKinds :many
SELECT failure_kind FROM pool_sends
WHERE mailbox_id = sqlc.arg(mailbox_id) AND created_at >= sqlc.arg(since) AND status <> 'sending'
ORDER BY created_at DESC
LIMIT sqlc.arg(max_count);

-- PoolBounceWindow counts the mailbox's last sends since its health reset
-- that reached the provider, and the hard bounces among them.
-- name: PoolBounceWindow :one
SELECT
    CAST(COUNT(*) AS INTEGER) AS sends,
    CAST(COALESCE(SUM(CASE WHEN b.id IS NOT NULL THEN 1 ELSE 0 END), 0) AS INTEGER) AS hard_bounces
FROM (
    SELECT s.id FROM pool_sends s
    WHERE s.mailbox_id = sqlc.arg(mailbox_id) AND s.created_at >= sqlc.arg(since)
      AND s.status <> 'sending' AND COALESCE(s.failure_kind, '') <> 'provider'
    ORDER BY s.created_at DESC
    LIMIT sqlc.arg(window_size)
) w
LEFT JOIN email_bounce b ON b.pool_send_id = w.id AND b.bounce_type = 'Permanent';

-- name: PoolComplaintWindow :one
SELECT
    CAST(COUNT(*) AS INTEGER) AS sends,
    CAST(COALESCE(SUM(CASE WHEN c.id IS NOT NULL THEN 1 ELSE 0 END), 0) AS INTEGER) AS complaints
FROM (
    SELECT s.id FROM pool_sends s
    WHERE s.mailbox_id = sqlc.arg(mailbox_id) AND s.created_at >= sqlc.arg(since)
      AND s.status <> 'sending' AND COALESCE(s.failure_kind, '') <> 'provider'
    ORDER BY s.created_at DESC
    LIMIT sqlc.arg(window_size)
) w
LEFT JOIN email_complaint c ON c.pool_send_id = w.id;

-- IsPoolAddressBlocked: a hard bounce or a complaint blocks an address for
-- the whole pool. Soft bounces do not.
-- name: IsPoolAddressBlocked :one
SELECT CAST((
    (SELECT COUNT(*) FROM email_bounce WHERE email_lower = LOWER(sqlc.arg(email)) AND bounce_type = 'Permanent') +
    (SELECT COUNT(*) FROM email_complaint WHERE email_lower = LOWER(sqlc.arg(email)))
) > 0 AS INTEGER) AS blocked;


-- name: CreatePoolPause :exec
INSERT INTO pool_pauses (id, mailbox_id, tenant_org_id, tenant, kind, reason, created_at)
VALUES (sqlc.arg(id), sqlc.arg(mailbox_id), sqlc.narg(tenant_org_id), sqlc.narg(tenant), sqlc.arg(kind), sqlc.arg(reason), sqlc.arg(created_at));

-- name: CountTenantPoolPausesSince :one
SELECT CAST(COUNT(*) AS INTEGER) FROM pool_pauses
WHERE tenant_org_id = sqlc.arg(org_id) AND tenant = sqlc.arg(tenant)
  AND kind IN ('list', 'policy') AND created_at >= sqlc.arg(since) AND cleared_at IS NULL;

-- name: ClearTenantPoolPauses :execrows
UPDATE pool_pauses SET cleared_at = sqlc.arg(cleared_at)
WHERE tenant_org_id = sqlc.arg(org_id) AND tenant = sqlc.arg(tenant) AND cleared_at IS NULL;
