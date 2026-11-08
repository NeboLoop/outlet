-- name: CreateNetworkInstance :one
INSERT INTO network_instances (id, name, key_hash)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(key_hash))
RETURNING *;

-- name: ListNetworkInstances :many
SELECT * FROM network_instances ORDER BY created_at;

-- name: RevokeNetworkInstance :execrows
UPDATE network_instances SET active = 0 WHERE id = sqlc.arg(id);

-- name: GetNetworkInstanceByKeyHash :one
SELECT * FROM network_instances WHERE key_hash = sqlc.arg(key_hash) AND active = 1;

-- name: MeterNetworkInstance :exec
UPDATE network_instances
SET lookups_count = lookups_count + sqlc.arg(lookups),
    reports_count = reports_count + sqlc.arg(reports),
    last_seen_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = sqlc.arg(id);

-- name: RecordNetworkEvent :exec
INSERT INTO network_signals (hash, delivered, hard_bounces, complaints, engaged,
    last_engaged_at, last_hard_bounce_at, last_delivered_at, first_seen_at, updated_at)
VALUES (sqlc.arg(hash), sqlc.arg(delivered), sqlc.arg(hard_bounces), sqlc.arg(complaints), sqlc.arg(engaged),
    sqlc.narg(last_engaged_at), sqlc.narg(last_hard_bounce_at), sqlc.narg(last_delivered_at), sqlc.arg(at), sqlc.arg(at))
ON CONFLICT (hash) DO UPDATE SET
    delivered = network_signals.delivered + excluded.delivered,
    hard_bounces = network_signals.hard_bounces + excluded.hard_bounces,
    complaints = network_signals.complaints + excluded.complaints,
    engaged = network_signals.engaged + excluded.engaged,
    last_engaged_at = CASE WHEN network_signals.last_engaged_at IS NULL OR excluded.last_engaged_at > network_signals.last_engaged_at
        THEN COALESCE(excluded.last_engaged_at, network_signals.last_engaged_at) ELSE network_signals.last_engaged_at END,
    last_hard_bounce_at = CASE WHEN network_signals.last_hard_bounce_at IS NULL OR excluded.last_hard_bounce_at > network_signals.last_hard_bounce_at
        THEN COALESCE(excluded.last_hard_bounce_at, network_signals.last_hard_bounce_at) ELSE network_signals.last_hard_bounce_at END,
    last_delivered_at = CASE WHEN network_signals.last_delivered_at IS NULL OR excluded.last_delivered_at > network_signals.last_delivered_at
        THEN COALESCE(excluded.last_delivered_at, network_signals.last_delivered_at) ELSE network_signals.last_delivered_at END,
    first_seen_at = MIN(network_signals.first_seen_at, excluded.first_seen_at),
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now');

-- name: RecordNetworkSender :exec
INSERT INTO network_signal_senders (hash, instance_id, complained)
VALUES (sqlc.arg(hash), sqlc.arg(instance_id), sqlc.arg(complained))
ON CONFLICT (hash, instance_id) DO UPDATE SET
    complained = MAX(network_signal_senders.complained, excluded.complained);

-- name: GetNetworkSignals :many
SELECT s.hash, s.delivered, s.hard_bounces, s.complaints, s.engaged,
    s.last_engaged_at, s.last_hard_bounce_at, s.last_delivered_at,
    CAST((SELECT COUNT(*) FROM network_signal_senders ns WHERE ns.hash = s.hash) AS INTEGER) AS senders,
    CAST((SELECT COUNT(*) FROM network_signal_senders ns WHERE ns.hash = s.hash AND ns.complained = 1) AS INTEGER) AS complaint_senders
FROM network_signals s
WHERE s.hash IN (sqlc.slice(hashes));
