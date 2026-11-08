-- A mailbox pool's connections to mailbox services, their quotes, and the
-- services' ids for the mailboxes.

-- name: CreateMailboxConnection :one
INSERT INTO mailbox_connections (id, org_id, pathway_id, service, name, api_key_enc, daily_limit, warmup_start, warmup_step)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetMailboxConnection :one
SELECT * FROM mailbox_connections WHERE id = ? AND org_id = ?;

-- name: ListMailboxConnections :many
-- The brand's connections, oldest first.
SELECT * FROM mailbox_connections WHERE org_id = ? ORDER BY created_at, id;

-- name: FirstManualConnection :one
-- A pool's oldest manual connection: where a domain added by hand goes.
SELECT * FROM mailbox_connections WHERE pathway_id = ? AND service = 'manual' ORDER BY created_at, id LIMIT 1;

-- name: UpdateMailboxConnection :exec
UPDATE mailbox_connections SET name = ?, api_key_enc = ?, daily_limit = ?, warmup_start = ?, warmup_step = ?
WHERE id = ? AND org_id = ?;

-- name: DeleteMailboxConnection :execrows
DELETE FROM mailbox_connections WHERE id = ? AND org_id = ?;

-- name: ListConnectionDomains :many
SELECT * FROM domain_identities WHERE connection_id = ? ORDER BY domain;

-- name: ListConnectionMailboxes :many
SELECT m.* FROM mailboxes m
JOIN domain_identities d ON d.id = m.domain_id
WHERE d.connection_id = ? ORDER BY m.email;

-- name: SetDomainConnection :execrows
UPDATE domain_identities SET connection_id = ?, updated_at = datetime('now')
WHERE id = ? AND org_id = ? AND purpose = 'outreach';

-- name: ClearDomainMailboxServiceRefs :exec
-- A domain's mailboxes after it moves to another connection: the old
-- service's ids mean nothing there; a sync links them again.
UPDATE mailboxes SET service_ref = '' WHERE domain_id = ?;

-- name: GetMailboxConnectionLimits :one
-- The limits of a mailbox's connection; each NULL uses its pool's.
SELECT c.daily_limit, c.warmup_start, c.warmup_step FROM mailbox_connections c
JOIN domain_identities d ON d.connection_id = c.id
JOIN mailboxes m ON m.domain_id = d.id
WHERE m.id = ?;

-- name: SetMailboxServiceRef :exec
UPDATE mailboxes SET service_ref = ? WHERE id = ? AND org_id = ?;

-- name: CreateMailboxServiceQuote :one
INSERT INTO mailbox_service_quotes (id, org_id, connection_id, service, kind, request, price_cents, currency, summary, expires_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetMailboxServiceQuote :one
SELECT * FROM mailbox_service_quotes WHERE id = ? AND org_id = ?;

-- name: UseMailboxServiceQuote :execrows
-- Spends a quote once: only an unused, unexpired one.
UPDATE mailbox_service_quotes SET used_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id) AND used_at IS NULL AND expires_at > sqlc.arg(now);
