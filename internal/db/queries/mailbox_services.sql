-- A mailbox pool's mailbox service, its quotes, and the service's ids for
-- the pool's mailboxes.

-- name: SetPathwayMailboxService :exec
UPDATE sending_pathways SET mailbox_service = ?, mailbox_service_key_enc = ?
WHERE id = ? AND org_id = ? AND provider = 'mailboxes';

-- name: SetMailboxServiceRef :exec
UPDATE mailboxes SET service_ref = ? WHERE id = ? AND org_id = ?;

-- name: CreateMailboxServiceQuote :one
INSERT INTO mailbox_service_quotes (id, org_id, pathway_id, service, kind, request, price_cents, currency, summary, expires_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetMailboxServiceQuote :one
SELECT * FROM mailbox_service_quotes WHERE id = ? AND org_id = ?;

-- name: UseMailboxServiceQuote :execrows
-- Spends a quote once: only an unused, unexpired one.
UPDATE mailbox_service_quotes SET used_at = sqlc.arg(now)
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id) AND used_at IS NULL AND expires_at > sqlc.arg(now);
