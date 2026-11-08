-- Sending pathways: the ways a brand delivers mail, which kinds each one
-- carries, and the per-sender limits email.Send enforces.

-- name: CreateSendingPathway :one
INSERT INTO sending_pathways (
    id, org_id, name, provider,
    aws_region, aws_access_key_enc, aws_secret_key_enc,
    smtp_host, smtp_port, smtp_username, smtp_password_enc,
    daily_limit, warmup_start, warmup_step, min_gap_seconds, max_gap_seconds,
    pause_bounce_rate, pause_min_bounces, pause_spam_blocks, pause_complaints
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateSendingPathway :exec
UPDATE sending_pathways
SET name = ?, aws_region = ?, smtp_host = ?, smtp_port = ?, smtp_username = ?,
    daily_limit = ?, warmup_start = ?, warmup_step = ?, min_gap_seconds = ?, max_gap_seconds = ?,
    pause_bounce_rate = ?, pause_min_bounces = ?, pause_spam_blocks = ?, pause_complaints = ?
WHERE id = ? AND org_id = ?;

-- name: SetSendingPathwayAWSKeys :exec
UPDATE sending_pathways SET aws_access_key_enc = ?, aws_secret_key_enc = ? WHERE id = ? AND org_id = ?;

-- name: SetSendingPathwaySMTPPassword :exec
UPDATE sending_pathways SET smtp_password_enc = ? WHERE id = ? AND org_id = ?;

-- name: ReleaseKindsFromOtherPathways :exec
-- Ticking a kind on one pathway takes it off every other pathway of the brand.
UPDATE sending_pathways
SET carries_transactional = CASE WHEN sqlc.arg(transactional) = 1 THEN 0 ELSE carries_transactional END,
    carries_marketing = CASE WHEN sqlc.arg(marketing) = 1 THEN 0 ELSE carries_marketing END,
    carries_outreach = CASE WHEN sqlc.arg(outreach) = 1 THEN 0 ELSE carries_outreach END
WHERE org_id = sqlc.arg(org_id) AND id <> sqlc.arg(id);

-- name: SetPathwayKinds :exec
UPDATE sending_pathways
SET carries_transactional = sqlc.arg(transactional), carries_marketing = sqlc.arg(marketing), carries_outreach = sqlc.arg(outreach)
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id);

-- name: GetSendingPathway :one
SELECT * FROM sending_pathways WHERE id = ? AND org_id = ?;

-- name: ListSendingPathways :many
SELECT * FROM sending_pathways WHERE org_id = ? ORDER BY created_at, name;

-- name: GetPathwayForKind :one
-- The one pathway carrying a kind of mail for a brand.
SELECT * FROM sending_pathways
WHERE org_id = sqlc.arg(org_id)
  AND ((sqlc.arg(kind) = 'transactional' AND carries_transactional = 1)
    OR (sqlc.arg(kind) = 'marketing' AND carries_marketing = 1)
    OR (sqlc.arg(kind) = 'outreach' AND carries_outreach = 1));

-- name: GetMailboxPathway :one
-- The mailbox pool a mailbox belongs to: its domain's pathway.
SELECT p.* FROM sending_pathways p
JOIN domain_identities d ON d.pathway_id = p.id
JOIN mailboxes m ON m.domain_id = d.id
WHERE m.id = ?;

-- name: DeleteSendingPathway :execrows
DELETE FROM sending_pathways WHERE id = ? AND org_id = ?;

-- name: ListPathwayDomains :many
SELECT * FROM domain_identities WHERE pathway_id = ? ORDER BY domain;

-- name: SetDomainPathway :execrows
-- Moves a domain to another of the brand's pathways; an outreach domain's
-- mailboxes go with it (a mailbox belongs to its domain's pool).
UPDATE domain_identities SET pathway_id = sqlc.arg(pathway_id), updated_at = datetime('now')
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id);

-- name: CarryDomainSendCounts :exec
-- The send counts of a moved domain's mailboxes go to their new pool, so a
-- move never resets a mailbox's day.
INSERT INTO send_counts (pathway_id, sender_id, day, n)
SELECT sqlc.arg(to_pathway_id), sc.sender_id, sc.day, sc.n FROM send_counts sc
WHERE sc.pathway_id = sqlc.arg(from_pathway_id)
  AND sc.sender_id IN (SELECT m.id FROM mailboxes m WHERE m.domain_id = sqlc.arg(domain_id))
ON CONFLICT (pathway_id, sender_id, day) DO UPDATE SET n = send_counts.n + excluded.n;

-- name: DeleteDomainSendCounts :exec
DELETE FROM send_counts
WHERE pathway_id = sqlc.arg(pathway_id)
  AND sender_id IN (SELECT m.id FROM mailboxes m WHERE m.domain_id = sqlc.arg(domain_id));

-- name: CountPathwayDomain :one
-- Whether a From domain belongs to a pathway.
SELECT COUNT(*) FROM domain_identities
WHERE pathway_id = sqlc.arg(pathway_id) AND LOWER(domain) = LOWER(sqlc.arg(domain));

-- name: ListPoolMailboxes :many
SELECT m.* FROM mailboxes m
JOIN domain_identities d ON d.id = m.domain_id
WHERE d.pathway_id = ? ORDER BY m.email;

-- name: SetMailboxDailyLimit :execrows
UPDATE mailboxes SET daily_limit = ? WHERE id = ? AND org_id = ?;

-- ---- limits -----------------------------------------------------------------

-- name: ClaimPathwaySpacing :execrows
-- Takes the pathway's next send slot, only when its spacing has passed.
UPDATE sending_pathways SET next_send_at = sqlc.arg(next_send_at)
WHERE id = sqlc.arg(id) AND (next_send_at IS NULL OR next_send_at <= sqlc.arg(now));

-- name: ClaimMailboxSpacing :execrows
-- Takes the mailbox's next send slot, only when its spacing has passed.
UPDATE mailboxes SET next_send_at = sqlc.arg(next_send_at)
WHERE id = sqlc.arg(id) AND (next_send_at IS NULL OR next_send_at <= sqlc.arg(now));

-- name: EnsureSendCount :exec
INSERT OR IGNORE INTO send_counts (pathway_id, sender_id, day, n) VALUES (?, ?, ?, 0);

-- name: IncrementSendCount :execrows
-- Counts one send for a sender on a day, only while it is under its
-- allowance: one statement, so two senders can't both pass the cap.
UPDATE send_counts SET n = n + 1
WHERE pathway_id = sqlc.arg(pathway_id) AND sender_id = sqlc.arg(sender_id) AND day = sqlc.arg(day) AND n < sqlc.arg(allowance);

-- name: RefundSendCount :exec
-- A counted send that failed gives its place back.
UPDATE send_counts SET n = n - 1 WHERE pathway_id = ? AND sender_id = ? AND day = ? AND n > 0;

-- name: ListSendCountsForDay :many
SELECT sender_id, n FROM send_counts WHERE pathway_id = ? AND day = ?;

-- name: SumSendCountsSince :one
-- A sender's sends on and after a day.
SELECT CAST(COALESCE(SUM(n), 0) AS INTEGER) FROM send_counts
WHERE pathway_id = sqlc.arg(pathway_id) AND sender_id = sqlc.arg(sender_id) AND day >= sqlc.arg(since_day);

-- ---- deferring held-back sends ---------------------------------------------------

-- name: DeferCampaignSend :exec
-- A send that failed for now (a pathway limit, a 4xx, a dropped connection)
-- waits, unclaimed and pending, to be claimed again after send_after. A sent
-- or failed send is final and never goes back.
UPDATE campaign_sends SET claim_token = NULL, claimed_at = NULL, send_after = ?
WHERE id = ? AND status = 'pending';

-- name: DeferQueuedEmail :exec
-- A sequence email held back by a pathway limit waits, unclaimed and still pending.
UPDATE email_queue SET claim_token = NULL, claimed_at = NULL, scheduled_for = ? WHERE id = ? AND status = 'pending';
