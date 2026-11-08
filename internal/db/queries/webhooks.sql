-- name: CreateWebhook :one
INSERT INTO webhooks (id, org_id, url, secret, events, active, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(org_id), sqlc.arg(url), sqlc.arg(secret), sqlc.arg(events), sqlc.arg(active), datetime('now'), datetime('now'))
RETURNING *;

-- name: GetWebhook :one
SELECT * FROM webhooks
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id);

-- name: GetWebhookByID :one
SELECT * FROM webhooks
WHERE id = sqlc.arg(id);

-- name: ListWebhooks :many
SELECT * FROM webhooks
WHERE org_id = sqlc.arg(org_id)
ORDER BY created_at DESC;

-- name: UpdateWebhook :one
UPDATE webhooks
SET url = COALESCE(NULLIF(sqlc.arg(url), ''), url),
    events = COALESCE(NULLIF(sqlc.arg(events), ''), events),
    active = sqlc.arg(active),
    updated_at = datetime('now')
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id)
RETURNING *;

-- name: DeleteWebhook :exec
DELETE FROM webhooks
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id);

-- name: UpdateWebhookDeliveryStats :exec
UPDATE webhooks
SET deliveries_total = deliveries_total + 1,
    deliveries_success = CASE WHEN sqlc.arg(success) = 1 THEN deliveries_success + 1 ELSE deliveries_success END,
    deliveries_failed = CASE WHEN sqlc.arg(success) = 0 THEN deliveries_failed + 1 ELSE deliveries_failed END,
    last_delivery_at = datetime('now'),
    last_status = sqlc.arg(last_status)
WHERE id = sqlc.arg(id);

-- name: CreateWebhookLog :one
INSERT INTO webhook_logs (id, webhook_id, delivery_id, attempt, event, payload, status_code, response, error, duration_ms, delivered_at)
VALUES (sqlc.arg(id), sqlc.arg(webhook_id), sqlc.narg(delivery_id), sqlc.narg(attempt), sqlc.arg(event), sqlc.arg(payload), sqlc.arg(status_code), sqlc.arg(response), sqlc.arg(error), sqlc.arg(duration_ms), datetime('now'))
RETURNING *;

-- name: ListWebhookLogs :many
SELECT * FROM webhook_logs
WHERE webhook_id = sqlc.arg(webhook_id)
ORDER BY delivered_at DESC
LIMIT sqlc.arg(limit_count);

-- name: CountWebhookLogs :one
SELECT COUNT(*) as count FROM webhook_logs
WHERE webhook_id = sqlc.arg(webhook_id);

-- Deliveries: one event for one webhook, tried until it succeeds or runs out
-- of tries (internal/services/webhook).

-- name: CreateWebhookDelivery :one
-- A new delivery starts claimed by the try that follows at once; the claim
-- lapses at claim_until if that try never finishes.
INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status, attempts, next_attempt_at, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(webhook_id), sqlc.arg(event), sqlc.arg(payload), 'sending', 0, sqlc.arg(claim_until), sqlc.arg(now), sqlc.arg(now))
RETURNING *;

-- name: ClaimWebhookDelivery :one
-- Claims a delivery for one try: a waiting one that is due, or one whose
-- earlier claim lapsed. With force, a waiting or finished one is claimed
-- whenever (Send again); one in flight never is.
UPDATE webhook_deliveries
SET status = 'sending', next_attempt_at = sqlc.arg(claim_until), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id)
  AND (
    (status IN ('pending', 'sending') AND next_attempt_at <= sqlc.arg(now))
    OR (sqlc.arg(force) = 1 AND status IN ('pending', 'succeeded', 'failed'))
  )
RETURNING *;

-- name: FinishWebhookDeliveryAttempt :exec
UPDATE webhook_deliveries
SET status = sqlc.arg(status),
    attempts = attempts + 1,
    next_attempt_at = sqlc.narg(next_attempt_at),
    last_status_code = sqlc.narg(last_status_code),
    last_error = sqlc.narg(last_error),
    updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: ListDueWebhookDeliveries :many
-- Deliveries due for a try, oldest first, for active webhooks only.
SELECT d.id, d.webhook_id
FROM webhook_deliveries d
JOIN webhooks w ON w.id = d.webhook_id
WHERE d.status IN ('pending', 'sending')
  AND d.next_attempt_at <= sqlc.arg(now)
  AND COALESCE(w.active, 1) = 1
ORDER BY d.created_at ASC, d.id ASC
LIMIT sqlc.arg(limit_count);

-- name: CountWebhookBacklog :one
-- Deliveries of a webhook still waiting for a try.
SELECT COUNT(*) FROM webhook_deliveries
WHERE webhook_id = sqlc.arg(webhook_id) AND status IN ('pending', 'sending');

-- name: GetWebhookDelivery :one
SELECT * FROM webhook_deliveries
WHERE id = sqlc.arg(id) AND webhook_id = sqlc.arg(webhook_id);

-- name: ListWebhookDeliveries :many
SELECT * FROM webhook_deliveries
WHERE webhook_id = sqlc.arg(webhook_id)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(limit_count);

-- name: ListWebhookRetrySummaries :many
-- For each webhook of a brand with deliveries waiting: how many, and when
-- the next try is.
SELECT d.webhook_id, COUNT(*) AS waiting, CAST(MIN(d.next_attempt_at) AS TEXT) AS next_attempt_at
FROM webhook_deliveries d
JOIN webhooks w ON w.id = d.webhook_id
WHERE w.org_id = sqlc.arg(org_id) AND d.status IN ('pending', 'sending')
GROUP BY d.webhook_id;

-- name: DeleteFinishedWebhookDeliveries :exec
-- Finished deliveries older than the cutoff go, with their tries.
DELETE FROM webhook_deliveries
WHERE status IN ('succeeded', 'failed') AND updated_at < sqlc.arg(before);
