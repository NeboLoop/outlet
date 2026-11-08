-- name: GetSequenceByID :one
SELECT id, org_id, list_id, slug, name, trigger_event, is_active, send_hour, send_timezone, sequence_type, on_completion_sequence_id, created_at
FROM email_sequences
WHERE id = sqlc.arg(id);

-- name: GetSequenceByListAndSlug :one
SELECT id, org_id, list_id, slug, name, trigger_event, is_active, send_hour, send_timezone, sequence_type, on_completion_sequence_id, created_at
FROM email_sequences
WHERE list_id = sqlc.arg(list_id) AND slug = sqlc.arg(slug);

-- name: GetSequenceByListAndTrigger :one
SELECT id, org_id, list_id, slug, name, trigger_event, is_active, send_hour, send_timezone, sequence_type, on_completion_sequence_id, created_at
FROM email_sequences
WHERE list_id = sqlc.arg(list_id) AND trigger_event = sqlc.arg(trigger_event) AND is_active = 1;

-- name: ListActiveSequencesByListAndTrigger :many
-- Every active sequence of a list that this trigger starts, oldest first.
SELECT id, slug
FROM email_sequences
WHERE list_id = sqlc.arg(list_id) AND trigger_event = sqlc.arg(trigger_event) AND is_active = 1
ORDER BY created_at, id;

-- name: ListSequencesByList :many
SELECT id, org_id, list_id, slug, name, trigger_event, is_active, send_hour, send_timezone, sequence_type, on_completion_sequence_id, created_at
FROM email_sequences
WHERE list_id = sqlc.arg(list_id)
ORDER BY created_at;

-- name: ListSequencesByOrg :many
SELECT es.id, es.org_id, es.list_id, es.slug, es.name, es.trigger_event, es.is_active,
       es.send_hour, es.send_timezone, es.sequence_type, es.on_completion_sequence_id, es.created_at,
       el.name as list_name, el.slug as list_slug,
       ocs.name as on_completion_sequence_name
FROM email_sequences es
LEFT JOIN email_lists el ON el.id = es.list_id
LEFT JOIN email_sequences ocs ON ocs.id = es.on_completion_sequence_id
WHERE es.org_id = sqlc.arg(org_id)
ORDER BY es.created_at;

-- name: ListAllSequences :many
SELECT es.id, es.org_id, es.list_id, es.slug, es.name, es.trigger_event, es.is_active,
       es.send_hour, es.send_timezone, es.sequence_type, es.on_completion_sequence_id, es.created_at,
       el.name as list_name, el.slug as list_slug,
       ocs.name as on_completion_sequence_name
FROM email_sequences es
LEFT JOIN email_lists el ON el.id = es.list_id
LEFT JOIN email_sequences ocs ON ocs.id = es.on_completion_sequence_id
ORDER BY es.created_at;

-- name: CreateSequence :one
-- A new sequence sends each email its delay after the contact joins (no
-- fixed send hour), as the app shows; a send hour is set with UpdateSequence.
INSERT INTO email_sequences (id, org_id, list_id, slug, name, trigger_event, is_active, sequence_type, send_hour, created_at)
VALUES (sqlc.arg(id), sqlc.arg(org_id), sqlc.arg(list_id), sqlc.arg(slug), sqlc.arg(name), sqlc.arg(trigger_event), sqlc.arg(is_active), COALESCE(sqlc.arg(sequence_type), 'lifecycle'), NULL, datetime('now'))
RETURNING *;

-- name: UpdateSequence :exec
UPDATE email_sequences
SET name = sqlc.arg(name), trigger_event = sqlc.arg(trigger_event), is_active = sqlc.arg(is_active), send_hour = sqlc.arg(send_hour), send_timezone = sqlc.arg(send_timezone), sequence_type = COALESCE(sqlc.arg(sequence_type), sequence_type), on_completion_sequence_id = sqlc.arg(on_completion_sequence_id), list_id = COALESCE(sqlc.arg(list_id), list_id)
WHERE id = sqlc.arg(id);

-- name: DeleteSequence :exec
DELETE FROM email_sequences WHERE id = sqlc.arg(id);

-- name: GetTemplateByID :one
SELECT id, sequence_id, position, delay_hours, subject, html_body, plain_text, template_type, is_active, design_id, created_at
FROM email_templates
WHERE id = sqlc.arg(id);

-- name: ListTemplatesBySequence :many
SELECT id, sequence_id, position, delay_hours, subject, html_body, plain_text, template_type, is_active, design_id, created_at
FROM email_templates
WHERE sequence_id = sqlc.arg(sequence_id)
ORDER BY position;

-- name: GetNextTemplate :one
SELECT id, sequence_id, position, delay_hours, subject, html_body, plain_text, is_active, is_transactional, design_id, created_at
FROM email_templates
WHERE sequence_id = sqlc.arg(sequence_id) AND position = sqlc.arg(position) AND is_active = 1;

-- name: GetConfirmationTemplate :one
SELECT id, sequence_id, position, delay_hours, subject, html_body, plain_text, template_type, is_active, is_transactional, design_id, created_at
FROM email_templates
WHERE sequence_id = sqlc.arg(sequence_id) AND template_type = 'confirmation' AND is_active = 1;

-- name: CreateTemplate :one
INSERT INTO email_templates (id, sequence_id, position, delay_hours, subject, html_body, plain_text, template_type, is_active, design_id, created_at)
VALUES (sqlc.arg(id), sqlc.arg(sequence_id), sqlc.arg(position), sqlc.arg(delay_hours), sqlc.arg(subject), sqlc.arg(html_body), sqlc.arg(plain_text), sqlc.arg(template_type), sqlc.arg(is_active), sqlc.arg(design_id), datetime('now'))
RETURNING *;

-- name: UpdateTemplate :exec
UPDATE email_templates
SET position = sqlc.arg(position), delay_hours = sqlc.arg(delay_hours), subject = sqlc.arg(subject), html_body = sqlc.arg(html_body), plain_text = sqlc.arg(plain_text), template_type = sqlc.arg(template_type), is_active = sqlc.arg(is_active), design_id = sqlc.arg(design_id)
WHERE id = sqlc.arg(id);

-- name: DeleteTemplate :exec
DELETE FROM email_templates WHERE id = sqlc.arg(id);

-- name: QueueEmail :one
INSERT INTO email_queue (id, contact_id, template_id, scheduled_for, status, tracking_token, created_at)
VALUES (sqlc.arg(id), sqlc.arg(contact_id), sqlc.arg(template_id), sqlc.arg(scheduled_for), 'pending', sqlc.arg(tracking_token), datetime('now'))
RETURNING *;

-- name: ClaimPendingEmails :exec
-- Claims up to limit_count due sequence emails. Skips unsubscribed or blocked
-- contacts, paused or stopped sequences, and contacts who left the list.
UPDATE email_queue
SET claim_token = sqlc.arg(claim_token), claimed_at = datetime('now')
WHERE id IN (
    SELECT eq.id
    FROM email_queue eq
    JOIN email_templates et ON et.id = eq.template_id
    JOIN contacts c ON c.id = eq.contact_id
    LEFT JOIN email_sequences es ON es.id = et.sequence_id
    LEFT JOIN contact_sequence_state css ON css.contact_id = eq.contact_id AND css.sequence_id = es.id
    WHERE eq.status = 'pending' AND eq.kind = 'marketing' AND eq.scheduled_for <= sqlc.arg(scheduled_before)
      AND (eq.claimed_at IS NULL OR eq.claimed_at < datetime('now', '-15 minutes'))
      AND c.unsubscribed_at IS NULL AND c.blocked_at IS NULL
      AND (css.id IS NULL OR css.is_active = 1)
      AND NOT EXISTS (
          SELECT 1 FROM list_subscribers ls
          WHERE ls.list_id = es.list_id AND ls.contact_id = eq.contact_id AND ls.status = 'unsubscribed'
      )
    ORDER BY eq.scheduled_for
    LIMIT sqlc.arg(limit_count)
);

-- name: GetPendingEmails :many
-- Returns the sequence emails claimed with claim_token.
SELECT eq.id, eq.contact_id, eq.template_id, eq.scheduled_for, eq.status, eq.tracking_token,
       et.subject, et.html_body, et.template_type, et.is_transactional,
       c.email, c.name,
       CAST(COALESCE(es.org_id, c.org_id, '') AS TEXT) AS org_id,
       CAST(COALESCE(d.html_body, '') AS TEXT) AS design_html
FROM email_queue eq
JOIN email_templates et ON et.id = eq.template_id
JOIN contacts c ON c.id = eq.contact_id
LEFT JOIN email_sequences es ON es.id = et.sequence_id
LEFT JOIN email_designs d ON d.id = et.design_id AND d.org_id = COALESCE(es.org_id, c.org_id)
WHERE eq.status = 'pending' AND eq.claim_token = sqlc.arg(claim_token)
ORDER BY eq.scheduled_for;

-- name: MarkEmailSent :exec
UPDATE email_queue
SET status = 'sent', sent_at = datetime('now')
WHERE id = sqlc.arg(id);

-- name: MarkEmailFailed :exec
UPDATE email_queue
SET status = 'failed', error_message = sqlc.arg(error_message)
WHERE id = sqlc.arg(id);

-- name: CancelEmail :exec
UPDATE email_queue
SET status = 'cancelled'
WHERE id = sqlc.arg(id) AND status = 'pending';

-- name: CancelEmailsForContact :exec
UPDATE email_queue
SET status = 'cancelled'
WHERE contact_id = sqlc.arg(contact_id) AND status = 'pending';

-- name: GetEmailQueueForContact :many
SELECT eq.*, et.subject, et.position
FROM email_queue eq
JOIN email_templates et ON et.id = eq.template_id
WHERE eq.contact_id = sqlc.arg(contact_id)
ORDER BY eq.scheduled_for;

-- name: GetContactSequenceState :one
SELECT id, contact_id, sequence_id, current_position, started_at, completed_at, unsubscribed_at, is_active, paused_at
FROM contact_sequence_state
WHERE contact_id = sqlc.arg(contact_id) AND sequence_id = sqlc.arg(sequence_id);

-- name: CreateContactSequenceState :one
INSERT INTO contact_sequence_state (id, contact_id, sequence_id, current_position, started_at)
VALUES (sqlc.arg(id), sqlc.arg(contact_id), sqlc.arg(sequence_id), sqlc.arg(current_position), datetime('now'))
RETURNING *;

-- name: UpdateContactSequencePosition :exec
UPDATE contact_sequence_state
SET current_position = sqlc.arg(current_position)
WHERE contact_id = sqlc.arg(contact_id) AND sequence_id = sqlc.arg(sequence_id);

-- name: CompleteContactSequence :exec
UPDATE contact_sequence_state
SET completed_at = datetime('now')
WHERE contact_id = sqlc.arg(contact_id) AND sequence_id = sqlc.arg(sequence_id);

-- name: UnsubscribeContactFromSequence :exec
UPDATE contact_sequence_state
SET unsubscribed_at = datetime('now')
WHERE contact_id = sqlc.arg(contact_id);

-- name: UnsubscribeContact :exec
UPDATE contacts
SET unsubscribed_at = datetime('now')
WHERE id = sqlc.arg(id);

-- name: GetContactByTrackingToken :one
SELECT c.* FROM contacts c
JOIN email_queue eq ON eq.contact_id = c.id
WHERE eq.tracking_token = sqlc.arg(tracking_token);

-- name: GetSequenceStats :one
SELECT
    COUNT(DISTINCT css.contact_id) as total_subscribers,
    COUNT(DISTINCT CASE WHEN css.completed_at IS NOT NULL THEN css.contact_id END) as completed,
    COUNT(DISTINCT CASE WHEN css.unsubscribed_at IS NOT NULL THEN css.contact_id END) as unsubscribed,
    (SELECT COUNT(*) FROM email_queue eq
     JOIN email_templates et ON et.id = eq.template_id
     WHERE et.sequence_id = sqlc.arg(sequence_id) AND eq.status = 'sent') as emails_sent,
    (SELECT COUNT(*) FROM email_queue eq
     JOIN email_templates et ON et.id = eq.template_id
     WHERE et.sequence_id = sqlc.arg(sequence_id) AND eq.status = 'pending') as emails_pending
FROM contact_sequence_state css
WHERE css.sequence_id = sqlc.arg(sequence_id);

-- Email tracking queries

-- name: QueueEmailWithTracking :one
INSERT INTO email_queue (id, contact_id, template_id, scheduled_for, status, tracking_token, created_at)
VALUES (sqlc.arg(id), sqlc.arg(contact_id), sqlc.arg(template_id), sqlc.arg(scheduled_for), 'pending', sqlc.arg(tracking_token), datetime('now'))
RETURNING *;

-- name: GetEmailByTrackingToken :one
SELECT eq.*, c.email, c.name
FROM email_queue eq
JOIN contacts c ON c.id = eq.contact_id
WHERE eq.tracking_token = sqlc.arg(tracking_token);

-- name: RecordEmailOpen :exec
UPDATE email_queue
SET opened_at = COALESCE(opened_at, datetime('now')), open_count = open_count + 1
WHERE id = sqlc.arg(id);

-- name: RecordEmailClick :exec
UPDATE email_queue
SET clicked_at = COALESCE(clicked_at, datetime('now')), click_count = click_count + 1
WHERE id = sqlc.arg(id);

-- name: CreateEmailClick :one
INSERT INTO email_clicks (id, email_queue_id, contact_id, link_url, link_name, user_agent, ip_address, clicked_at)
VALUES (sqlc.arg(id), sqlc.arg(email_queue_id), sqlc.arg(contact_id), sqlc.arg(link_url), sqlc.arg(link_name), sqlc.arg(user_agent), sqlc.arg(ip_address), datetime('now'))
RETURNING *;

-- name: GetEmailStatsForSequence :one
SELECT
    SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) as sent_count,
    SUM(CASE WHEN opened_at IS NOT NULL THEN 1 ELSE 0 END) as opened_count,
    SUM(CASE WHEN clicked_at IS NOT NULL THEN 1 ELSE 0 END) as clicked_count,
    COALESCE(SUM(open_count), 0) as total_opens,
    COALESCE(SUM(click_count), 0) as total_clicks
FROM email_queue eq
JOIN email_templates et ON et.id = eq.template_id
WHERE et.sequence_id = sqlc.arg(sequence_id);

-- name: GetEmailStatsForTemplate :one
SELECT
    SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) as sent_count,
    SUM(CASE WHEN opened_at IS NOT NULL THEN 1 ELSE 0 END) as opened_count,
    SUM(CASE WHEN clicked_at IS NOT NULL THEN 1 ELSE 0 END) as clicked_count,
    COALESCE(SUM(open_count), 0) as total_opens,
    COALESCE(SUM(click_count), 0) as total_clicks
FROM email_queue
WHERE template_id = sqlc.arg(template_id);

-- name: ListEmailQueueByOrg :many
SELECT eq.id, eq.contact_id, eq.template_id, eq.scheduled_for, eq.sent_at, eq.status, eq.error_message,
       et.subject, c.email as contact_email, c.name as contact_name
FROM email_queue eq
JOIN email_templates et ON et.id = eq.template_id
JOIN email_sequences es ON es.id = et.sequence_id
JOIN contacts c ON c.id = eq.contact_id
WHERE es.org_id = sqlc.arg(org_id)
  AND (sqlc.arg(filter_status) IS NULL OR sqlc.arg(filter_status) = '' OR eq.status = sqlc.arg(filter_status))
  AND (sqlc.arg(filter_contact_id) IS NULL OR sqlc.arg(filter_contact_id) = '' OR eq.contact_id = sqlc.arg(filter_contact_id))
ORDER BY eq.scheduled_for DESC;

-- SDK Sequence queries

-- name: GetSequenceByOrgAndSlug :one
SELECT id, org_id, list_id, slug, name, trigger_event, is_active, send_hour, send_timezone, sequence_type, on_completion_sequence_id, created_at
FROM email_sequences
WHERE org_id = sqlc.arg(org_id) AND slug = sqlc.arg(slug);

-- name: ListContactSequenceStatesWithDetails :many
SELECT
    css.id, css.contact_id, css.sequence_id, css.current_position,
    css.started_at, css.completed_at, css.unsubscribed_at, css.is_active, css.paused_at,
    es.slug as sequence_slug, es.name as sequence_name,
    (SELECT COUNT(*) FROM email_templates et WHERE et.sequence_id = css.sequence_id AND et.is_active = 1) as total_steps,
    (SELECT COUNT(*) FROM email_queue eq
     JOIN email_templates et ON et.id = eq.template_id
     WHERE eq.contact_id = css.contact_id AND et.sequence_id = css.sequence_id AND eq.status = 'sent') as emails_sent,
    (SELECT COUNT(*) FROM email_queue eq
     JOIN email_templates et ON et.id = eq.template_id
     WHERE eq.contact_id = css.contact_id AND et.sequence_id = css.sequence_id AND eq.opened_at IS NOT NULL) as emails_opened,
    (SELECT COUNT(*) FROM email_queue eq
     JOIN email_templates et ON et.id = eq.template_id
     WHERE eq.contact_id = css.contact_id AND et.sequence_id = css.sequence_id AND eq.clicked_at IS NOT NULL) as emails_clicked,
    (SELECT eq.scheduled_for FROM email_queue eq
     JOIN email_templates et ON et.id = eq.template_id
     WHERE eq.contact_id = css.contact_id AND et.sequence_id = css.sequence_id AND eq.status = 'pending'
     ORDER BY eq.scheduled_for LIMIT 1) as next_email_at
FROM contact_sequence_state css
JOIN email_sequences es ON es.id = css.sequence_id
WHERE css.contact_id = sqlc.arg(contact_id) AND es.org_id = sqlc.arg(org_id);

-- name: GetContactSequenceStateWithDetails :one
SELECT
    css.id, css.contact_id, css.sequence_id, css.current_position,
    css.started_at, css.completed_at, css.unsubscribed_at, css.is_active, css.paused_at,
    es.slug as sequence_slug, es.name as sequence_name,
    (SELECT COUNT(*) FROM email_templates et WHERE et.sequence_id = css.sequence_id AND et.is_active = 1) as total_steps,
    (SELECT COUNT(*) FROM email_queue eq
     JOIN email_templates et ON et.id = eq.template_id
     WHERE eq.contact_id = css.contact_id AND et.sequence_id = css.sequence_id AND eq.status = 'sent') as emails_sent,
    (SELECT COUNT(*) FROM email_queue eq
     JOIN email_templates et ON et.id = eq.template_id
     WHERE eq.contact_id = css.contact_id AND et.sequence_id = css.sequence_id AND eq.opened_at IS NOT NULL) as emails_opened,
    (SELECT COUNT(*) FROM email_queue eq
     JOIN email_templates et ON et.id = eq.template_id
     WHERE eq.contact_id = css.contact_id AND et.sequence_id = css.sequence_id AND eq.clicked_at IS NOT NULL) as emails_clicked,
    (SELECT eq.scheduled_for FROM email_queue eq
     JOIN email_templates et ON et.id = eq.template_id
     WHERE eq.contact_id = css.contact_id AND et.sequence_id = css.sequence_id AND eq.status = 'pending'
     ORDER BY eq.scheduled_for LIMIT 1) as next_email_at
FROM contact_sequence_state css
JOIN email_sequences es ON es.id = css.sequence_id
WHERE css.contact_id = sqlc.arg(contact_id) AND es.org_id = sqlc.arg(org_id) AND es.slug = sqlc.arg(slug);

-- name: PauseContactSequence :exec
UPDATE contact_sequence_state
SET paused_at = datetime('now'), is_active = 0
WHERE contact_id = sqlc.arg(contact_id) AND sequence_id = sqlc.arg(sequence_id);

-- name: ResumeContactSequence :exec
UPDATE contact_sequence_state
SET paused_at = NULL, is_active = 1
WHERE contact_id = sqlc.arg(contact_id) AND sequence_id = sqlc.arg(sequence_id);

-- name: CancelContactSequence :exec
UPDATE contact_sequence_state
SET is_active = 0, unsubscribed_at = datetime('now')
WHERE contact_id = sqlc.arg(contact_id) AND sequence_id = sqlc.arg(sequence_id);

-- name: CancelAllContactSequences :exec
UPDATE contact_sequence_state
SET is_active = 0, unsubscribed_at = datetime('now')
WHERE contact_id = sqlc.arg(contact_id)
  AND sequence_id IN (SELECT id FROM email_sequences WHERE org_id = sqlc.arg(org_id));

-- name: CancelPendingEmailsForContactSequence :exec
UPDATE email_queue
SET status = 'cancelled'
WHERE contact_id = sqlc.arg(contact_id)
  AND template_id IN (SELECT id FROM email_templates WHERE sequence_id = sqlc.arg(sequence_id))
  AND status = 'pending';

-- name: CountTemplatesBySequence :one
SELECT COUNT(*) as count
FROM email_templates
WHERE sequence_id = sqlc.arg(sequence_id) AND is_active = 1;
