-- name: ListEntryRulesBySequence :many
SELECT ser.*, el.name as list_name, el.slug as list_slug,
       es2.name as source_sequence_name
FROM sequence_entry_rules ser
LEFT JOIN email_lists el ON ser.trigger_type = 'list_join' AND el.id = ser.source_id
LEFT JOIN email_sequences es2 ON ser.trigger_type = 'sequence_complete' AND es2.id = ser.source_id
WHERE ser.sequence_id = sqlc.arg(sequence_id)
ORDER BY ser.priority, ser.created_at;

-- name: ListEntryRulesByList :many
SELECT ser.*, es.name as sequence_name, es.slug as sequence_slug
FROM sequence_entry_rules ser
JOIN email_sequences es ON es.id = ser.sequence_id
WHERE ser.trigger_type = 'list_join' AND ser.source_id = sqlc.arg(source_id) AND ser.is_active = 1
ORDER BY ser.priority;

-- name: GetEntryRule :one
SELECT * FROM sequence_entry_rules WHERE id = sqlc.arg(id);

-- name: GetActiveEntryRuleForTrigger :many
-- The active rules waiting on this trigger and source, for sequences of the
-- contact's own brand only, highest priority first.
SELECT ser.*, es.name as sequence_name, es.sequence_type
FROM sequence_entry_rules ser
JOIN email_sequences es ON es.id = ser.sequence_id
JOIN contacts c ON c.id = sqlc.arg(contact_id) AND c.org_id = es.org_id
WHERE ser.trigger_type = sqlc.arg(trigger_type)
  AND ser.source_id = sqlc.arg(source_id)
  AND ser.is_active = 1
  AND es.is_active = 1
ORDER BY ser.priority DESC, ser.created_at;

-- name: CreateEntryRule :one
INSERT INTO sequence_entry_rules (id, sequence_id, trigger_type, source_id, priority, is_active, created_at)
VALUES (sqlc.arg(id), sqlc.arg(sequence_id), sqlc.arg(trigger_type), sqlc.arg(source_id), sqlc.arg(priority), sqlc.arg(is_active), datetime('now'))
RETURNING *;

-- name: UpdateEntryRule :exec
UPDATE sequence_entry_rules
SET priority = sqlc.arg(priority), is_active = sqlc.arg(is_active)
WHERE id = sqlc.arg(id);

-- name: DeleteEntryRule :exec
DELETE FROM sequence_entry_rules WHERE id = sqlc.arg(id);

-- name: DeleteEntryRulesBySequence :exec
DELETE FROM sequence_entry_rules WHERE sequence_id = sqlc.arg(sequence_id);

-- name: CountActiveSequencesForContact :one
SELECT COUNT(*) as count
FROM contact_sequence_state css
JOIN email_sequences es ON es.id = css.sequence_id
WHERE css.contact_id = sqlc.arg(contact_id)
  AND css.is_active = 1
  AND css.completed_at IS NULL
  AND css.unsubscribed_at IS NULL
  AND es.sequence_type = 'lifecycle';

-- name: GetActiveSequenceForContact :one
SELECT css.*, es.name as sequence_name, es.sequence_type
FROM contact_sequence_state css
JOIN email_sequences es ON es.id = css.sequence_id
WHERE css.contact_id = sqlc.arg(contact_id)
  AND css.is_active = 1
  AND css.completed_at IS NULL
  AND css.unsubscribed_at IS NULL
  AND es.sequence_type = 'lifecycle'
LIMIT 1;

-- Note: PauseContactSequence and ResumeContactSequence are defined in email_sequences.sql
