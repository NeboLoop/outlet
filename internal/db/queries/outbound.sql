-- Outbound runs on Outlet's lists, contacts, sequences and email queue.
-- These are the queries only outreach needs. Times are UTC text,
-- 'YYYY-MM-DD HH:MM:SS', so they compare in order.

-- ---- brand settings --------------------------------------------------------

-- name: SetOutboundEnabled :exec
UPDATE organizations SET outbound_enabled = ?, updated_at = datetime('now') WHERE id = ?;

-- name: SetOutboundPausedReason :exec
UPDATE organizations SET outbound_paused_reason = ?, updated_at = datetime('now') WHERE id = ?;

-- name: UpdateOutboundSettings :exec
UPDATE organizations
SET mailing_address = ?, outbound_opt_out_line = ?, outbound_timezone = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: SetOrganizationDNSToken :exec
UPDATE organizations SET dns_token_enc = ?, updated_at = datetime('now') WHERE id = ?;

-- ---- outreach domains (domain_identities with purpose 'outreach') -------------

-- name: CreateOutreachDomain :one
INSERT INTO domain_identities (id, org_id, domain, verification_status, dkim_status, purpose, dns_host, dkim_selector, tracking_host, pathway_id)
VALUES (?, ?, ?, 'pending', 'pending', 'outreach', ?, ?, ?, ?)
RETURNING *;

-- name: GetOutreachDomain :one
SELECT * FROM domain_identities WHERE id = ? AND org_id = ? AND purpose = 'outreach';

-- name: GetAnyDomainIdentity :one
-- A domain belongs to one brand and one purpose.
SELECT * FROM domain_identities WHERE LOWER(domain) = LOWER(?) LIMIT 1;

-- name: ListOutreachDomains :many
SELECT * FROM domain_identities WHERE org_id = ? AND purpose = 'outreach' ORDER BY domain;

-- name: ListAllOutreachDomains :many
SELECT * FROM domain_identities WHERE purpose = 'outreach' ORDER BY domain;

-- name: SetOutreachDomainCheck :exec
UPDATE domain_identities
SET verification_status = ?, dns_check = ?, last_checked_at = datetime('now'), updated_at = datetime('now')
WHERE id = ? AND purpose = 'outreach';

-- name: SetOutreachDomainTrackingHost :exec
UPDATE domain_identities SET tracking_host = ?, updated_at = datetime('now') WHERE id = ? AND purpose = 'outreach';

-- ---- mailboxes ----------------------------------------------------------------

-- name: CreateMailbox :one
INSERT INTO mailboxes (
    id, org_id, domain_id, email, from_name, signature,
    smtp_host, smtp_port, smtp_username, smtp_password_enc,
    imap_host, imap_port, imap_username, imap_password_enc, daily_limit
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetMailbox :one
SELECT * FROM mailboxes WHERE id = ? AND org_id = ?;

-- name: GetMailboxByEmail :one
SELECT * FROM mailboxes WHERE email = ?;

-- name: ListMailboxes :many
SELECT * FROM mailboxes WHERE org_id = ? ORDER BY email;

-- name: ListActiveMailboxes :many
-- Every mailbox the sender works, across brands.
SELECT * FROM mailboxes WHERE status = 'active' ORDER BY email;

-- name: ListAllMailboxes :many
SELECT * FROM mailboxes ORDER BY email;

-- name: SetMailboxStatus :exec
UPDATE mailboxes SET status = ?, paused_reason = ? WHERE id = ?;

-- name: PauseMailboxesOnDomain :exec
UPDATE mailboxes SET status = 'paused', paused_reason = ? WHERE domain_id = ? AND status = 'active';

-- name: ResumeMailboxesPausedForDNS :exec
UPDATE mailboxes SET status = 'active', paused_reason = ''
WHERE domain_id = ? AND status = 'paused' AND paused_reason LIKE 'DNS:%';

-- name: SetMailboxNextSend :exec
UPDATE mailboxes SET next_send_at = ? WHERE id = ?;

-- name: SetMailboxError :exec
UPDATE mailboxes SET last_error = ? WHERE id = ?;

-- name: SetMailboxIMAPState :exec
UPDATE mailboxes SET imap_uid_validity = ?, imap_last_uid = ? WHERE id = ?;

-- name: CountMailboxSentSince :one
SELECT COUNT(*) FROM email_queue
WHERE mailbox_id = sqlc.arg(mailbox_id) AND status = 'sent' AND sent_at >= sqlc.arg(since);

-- name: CountMailboxRepliesSince :one
-- Bounces, spam blocks or complaints on one mailbox since a time.
SELECT COUNT(*) FROM email_replies
WHERE mailbox_id = sqlc.arg(mailbox_id) AND classification = sqlc.arg(classification) AND received_at >= sqlc.arg(since);

-- ---- outreach lists, prospects and sequences ----------------------------------

-- name: SetListKind :exec
UPDATE email_lists SET kind = ?, updated_at = datetime('now') WHERE id = ? AND org_id = ?;

-- name: ListOutreachLists :many
SELECT * FROM email_lists WHERE org_id = ? AND kind = 'outreach' ORDER BY created_at;

-- name: ListProspects :many
-- One row per person on a brand's outreach lists, newest first, with what
-- decides whether they can be enrolled: a 'prospect' status (on the list
-- filtered by, else on any), an active outreach campaign, and suppression
-- (the check IsSendBlocked makes). total counts every matching person.
SELECT p.id, p.email, p.name, p.source, p.validation_status, p.created_at, p.added_at,
       p.prospect, p.in_campaign, p.suppressed, CAST(COUNT(*) OVER () AS INTEGER) AS total
FROM (
    SELECT c.id, c.email, c.name, c.source, c.validation_status, c.created_at,
           CAST(MAX(ls.subscribed_at) AS TEXT) AS added_at,
           CAST(MAX(CASE WHEN ls.status = 'prospect' THEN 1 ELSE 0 END) AS INTEGER) AS prospect,
           CAST(EXISTS (
               SELECT 1 FROM contact_sequence_state css
               JOIN email_sequences es ON es.id = css.sequence_id AND es.sequence_type = 'outreach'
               WHERE css.contact_id = c.id AND css.is_active = 1
           ) AS INTEGER) AS in_campaign,
           CAST((
               (SELECT COUNT(*) FROM email_bounce b WHERE b.email_lower = LOWER(c.email) AND b.bounce_type = 'Permanent') +
               (SELECT COUNT(*) FROM email_complaint ec WHERE ec.email_lower = LOWER(c.email)) +
               (SELECT COUNT(*) FROM suppression_list sl WHERE sl.org_id = el.org_id AND sl.email_lower = LOWER(c.email)) +
               (SELECT COUNT(*) FROM blocked_domains bd WHERE bd.org_id = el.org_id AND bd.domain = LOWER(SUBSTR(c.email, INSTR(c.email, '@') + 1)))
           ) > 0 AS INTEGER) AS suppressed
    FROM list_subscribers ls
    JOIN email_lists el ON el.id = ls.list_id AND el.kind = 'outreach'
    JOIN contacts c ON c.id = ls.contact_id
    WHERE el.org_id = sqlc.arg(org_id)
      AND (CAST(sqlc.arg(list_id) AS INTEGER) = 0 OR ls.list_id = sqlc.arg(list_id))
      AND (CAST(sqlc.arg(status) AS TEXT) = '' OR ls.status = sqlc.arg(status))
      AND (CAST(sqlc.arg(search) AS TEXT) = '' OR c.email LIKE '%' || sqlc.arg(search) || '%' OR c.name LIKE '%' || sqlc.arg(search) || '%')
    GROUP BY c.id
) p
WHERE (CAST(sqlc.arg(enrollable) AS INTEGER) = 0 OR (p.prospect = 1 AND p.in_campaign = 0 AND p.suppressed = 0))
ORDER BY p.added_at DESC, p.id
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: ListProspectSubscriptions :many
-- The outreach lists of some people, with their status on each.
SELECT ls.contact_id, el.id AS list_id, el.name AS list_name, ls.status
FROM list_subscribers ls
JOIN email_lists el ON el.id = ls.list_id AND el.kind = 'outreach'
WHERE el.org_id = sqlc.arg(org_id) AND ls.contact_id IN (sqlc.slice(contact_ids))
ORDER BY el.name;

-- name: ListActiveOutreachCampaignsFor :many
-- The outreach campaigns some people are in now.
SELECT css.contact_id, es.id, es.name
FROM contact_sequence_state css
JOIN email_sequences es ON es.id = css.sequence_id AND es.sequence_type = 'outreach'
WHERE es.org_id = sqlc.arg(org_id) AND css.is_active = 1 AND css.contact_id IN (sqlc.slice(contact_ids))
ORDER BY es.name;

-- name: SetOutreachSequenceOptions :exec
UPDATE email_sequences
SET sequence_type = 'outreach', approval_mode = ?, window_start_hour = ?, window_end_hour = ?,
    track_opens = ?, track_clicks = ?, send_timezone = ?
WHERE id = ? AND org_id = ?;

-- name: GetOutreachSequence :one
SELECT * FROM email_sequences WHERE id = ? AND org_id = ? AND sequence_type = 'outreach';

-- name: GetSequenceOptions :one
SELECT * FROM email_sequences WHERE id = ?;

-- name: ListOutreachSequences :many
SELECT * FROM email_sequences WHERE org_id = ? AND sequence_type = 'outreach' ORDER BY created_at DESC;

-- name: SetSequenceActive :exec
UPDATE email_sequences SET is_active = ? WHERE id = ? AND org_id = ?;

-- name: ListSequenceSteps :many
SELECT * FROM email_templates WHERE sequence_id = ? ORDER BY position;

-- name: GetSequenceStateFull :one
SELECT * FROM contact_sequence_state WHERE contact_id = ? AND sequence_id = ?;

-- name: SetSequenceStateMailbox :exec
UPDATE contact_sequence_state SET mailbox_id = ? WHERE id = ?;

-- name: StopSequenceState :exec
-- Ends a person's sequence: a reply, bounce, opt-out or complaint.
UPDATE contact_sequence_state
SET is_active = 0, stopped_reason = ?, completed_at = COALESCE(completed_at, datetime('now'))
WHERE id = ?;

-- name: ListActiveOutreachStatesForContact :many
SELECT css.* FROM contact_sequence_state css
JOIN email_sequences es ON es.id = css.sequence_id AND es.sequence_type = 'outreach'
WHERE css.contact_id = ? AND css.is_active = 1;

-- name: CountMailboxStatesForSequence :many
SELECT mailbox_id, COUNT(*) AS n FROM contact_sequence_state
WHERE sequence_id = ? AND mailbox_id IS NOT NULL GROUP BY mailbox_id;

-- ---- the outreach queue (email_queue rows of kind 'outreach') -----------------

-- name: QueueOutreachEmail :one
INSERT INTO email_queue (
    id, contact_id, template_id, scheduled_for, status, tracking_token,
    kind, mailbox_id, subject, body, message_id, in_reply_to, refs
) VALUES (?, ?, ?, ?, ?, ?, 'outreach', ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetOutreachEmail :one
-- One outreach email of a brand, with its person and sequence.
SELECT eq.*, c.email AS to_email, c.name AS to_name, c.org_id AS org_id,
       COALESCE(et.position, 0) AS position, et.sequence_id AS sequence_id
FROM email_queue eq
JOIN contacts c ON c.id = eq.contact_id
LEFT JOIN email_templates et ON et.id = eq.template_id
WHERE eq.id = sqlc.arg(id) AND c.org_id = sqlc.arg(org_id) AND eq.kind = 'outreach';

-- name: GetOutreachEmailByMessageID :one
SELECT eq.*, c.org_id AS org_id, et.sequence_id AS sequence_id
FROM email_queue eq
JOIN contacts c ON c.id = eq.contact_id
LEFT JOIN email_templates et ON et.id = eq.template_id
WHERE eq.message_id = ? AND eq.kind = 'outreach' LIMIT 1;

-- name: ListOutreachEmails :many
-- A brand's outreach emails with one status, oldest first, with the
-- campaign and step each belongs to.
SELECT eq.*, c.email AS to_email, c.name AS to_name, COALESCE(et.position, 0) AS position, m.email AS from_email,
       CAST(COALESCE(es.id, '') AS TEXT) AS campaign_id, CAST(COALESCE(es.name, '') AS TEXT) AS campaign_name
FROM email_queue eq
JOIN contacts c ON c.id = eq.contact_id
JOIN mailboxes m ON m.id = eq.mailbox_id
LEFT JOIN email_templates et ON et.id = eq.template_id
LEFT JOIN email_sequences es ON es.id = et.sequence_id
WHERE c.org_id = sqlc.arg(org_id) AND eq.kind = 'outreach' AND eq.status = sqlc.arg(status)
ORDER BY eq.scheduled_for
LIMIT sqlc.arg(limit_count);

-- name: CountOutreachEmailsByStatus :many
SELECT eq.status, COUNT(*) AS n FROM email_queue eq
JOIN contacts c ON c.id = eq.contact_id
WHERE c.org_id = ? AND eq.kind = 'outreach'
GROUP BY eq.status;

-- name: ListOutreachThread :many
-- The sent emails of one person's outreach sequence, oldest first.
SELECT eq.* FROM email_queue eq
JOIN email_templates et ON et.id = eq.template_id
WHERE eq.contact_id = ? AND et.sequence_id = ? AND eq.kind = 'outreach' AND eq.status = 'sent'
ORDER BY eq.sent_at;

-- name: NextOutreachForMailbox :one
-- The oldest approved outreach email due on this mailbox, whose person is
-- still in an active sequence (or which answers a reply).
SELECT eq.*, c.email AS to_email, c.org_id AS org_id, COALESCE(et.position, 0) AS position, et.sequence_id AS sequence_id
FROM email_queue eq
JOIN contacts c ON c.id = eq.contact_id
LEFT JOIN email_templates et ON et.id = eq.template_id
LEFT JOIN email_sequences es ON es.id = et.sequence_id
LEFT JOIN contact_sequence_state css ON css.contact_id = eq.contact_id AND css.sequence_id = es.id
WHERE eq.mailbox_id = sqlc.arg(mailbox_id) AND eq.kind = 'outreach' AND eq.status = 'pending'
  AND eq.scheduled_for <= sqlc.arg(now)
  AND (eq.template_id IS NULL OR (es.is_active = 1 AND css.is_active = 1))
ORDER BY eq.scheduled_for
LIMIT 1;

-- name: ClaimOutreachEmail :execrows
-- Claimed once: only one sender moves a row from pending to sending.
UPDATE email_queue SET status = 'sending', claimed_at = ? WHERE id = ? AND status = 'pending' AND kind = 'outreach';

-- name: ReleaseOutreachEmail :exec
-- A transient failure: back to pending, tried again later from the same mailbox.
UPDATE email_queue SET status = 'pending', claimed_at = NULL, error_message = ?, scheduled_for = ?
WHERE id = ? AND status = 'sending';

-- name: SetOutreachSchedule :exec
UPDATE email_queue SET scheduled_for = ? WHERE id = ? AND status = 'pending' AND kind = 'outreach';

-- name: MarkOutreachSent :exec
UPDATE email_queue SET status = 'sent', sent_at = ?, error_message = NULL WHERE id = ?;

-- name: ListStuckOutreach :many
-- Rows left in sending by a crash. Nobody knows whether they went, so they
-- are marked failed and never sent again.
SELECT * FROM email_queue WHERE kind = 'outreach' AND status = 'sending' AND claimed_at < ?;

-- name: ApproveOutreachDraft :execrows
UPDATE email_queue SET status = 'pending'
WHERE email_queue.id = sqlc.arg(id) AND status = 'draft' AND kind = 'outreach'
  AND contact_id IN (SELECT c.id FROM contacts c WHERE c.org_id = sqlc.arg(org_id));

-- name: EditOutreachEmail :execrows
UPDATE email_queue SET subject = sqlc.arg(subject), body = sqlc.arg(body)
WHERE email_queue.id = sqlc.arg(id) AND status IN ('draft', 'pending') AND kind = 'outreach'
  AND contact_id IN (SELECT c.id FROM contacts c WHERE c.org_id = sqlc.arg(org_id));

-- name: SkipOutreachEmail :execrows
UPDATE email_queue SET status = 'cancelled'
WHERE email_queue.id = sqlc.arg(id) AND status IN ('draft', 'pending') AND kind = 'outreach'
  AND contact_id IN (SELECT c.id FROM contacts c WHERE c.org_id = sqlc.arg(org_id));

-- name: CancelOpenOutreachForContact :exec
-- A reply, bounce or opt-out cancels everything still waiting for the person.
UPDATE email_queue SET status = 'cancelled'
WHERE contact_id = ? AND kind = 'outreach' AND status IN ('draft', 'pending');

-- ---- replies ------------------------------------------------------------------

-- name: CreateEmailReply :one
INSERT INTO email_replies (
    id, org_id, mailbox_id, contact_id, queue_id, sequence_id, from_email, subject,
    body, message_id, in_reply_to, refs, classification, handled, received_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (mailbox_id, message_id) DO NOTHING
RETURNING *;

-- name: GetEmailReply :one
SELECT * FROM email_replies WHERE id = ? AND org_id = ?;

-- name: ListEmailReplies :many
-- Replies with the name of the campaign each answers.
SELECT r.*, CAST(COALESCE(es.name, '') AS TEXT) AS campaign_name
FROM email_replies r
LEFT JOIN email_sequences es ON es.id = r.sequence_id
WHERE r.org_id = ? AND r.handled = ? ORDER BY r.received_at DESC LIMIT ?;

-- name: SetEmailReplyClassification :exec
UPDATE email_replies SET classification = ?, handled = ? WHERE id = ? AND org_id = ?;

-- name: SetEmailReplyHandled :exec
UPDATE email_replies SET handled = 1 WHERE id = ? AND org_id = ?;

-- name: CountUnhandledEmailReplies :one
SELECT COUNT(*) FROM email_replies WHERE org_id = ? AND handled = 0;

-- ---- results ------------------------------------------------------------------
-- Reply rate is the main measure; opens and clicks count when the sequence
-- tracks them. People, not emails: each counts once.

-- name: OutreachSequenceStates :many
SELECT CAST(CASE WHEN stopped_reason <> '' THEN stopped_reason WHEN completed_at IS NOT NULL THEN 'done'
                 WHEN is_active = 1 THEN 'active' ELSE 'done' END AS TEXT) AS state,
       COUNT(*) AS n
FROM contact_sequence_state WHERE sequence_id = ? GROUP BY state;

-- name: OutreachSentByStep :many
SELECT et.position, COUNT(*) AS sent,
       CAST(SUM(CASE WHEN eq.open_count > 0 THEN 1 ELSE 0 END) AS INTEGER) AS opened,
       CAST(SUM(CASE WHEN eq.click_count > 0 THEN 1 ELSE 0 END) AS INTEGER) AS clicked
FROM email_queue eq JOIN email_templates et ON et.id = eq.template_id
WHERE et.sequence_id = ? AND eq.kind = 'outreach' AND eq.status = 'sent'
GROUP BY et.position;

-- name: OutreachPeopleEngaged :one
SELECT COUNT(DISTINCT CASE WHEN eq.open_count > 0 THEN eq.contact_id END) AS opened,
       COUNT(DISTINCT CASE WHEN eq.click_count > 0 THEN eq.contact_id END) AS clicked
FROM email_queue eq JOIN email_templates et ON et.id = eq.template_id
WHERE et.sequence_id = ? AND eq.kind = 'outreach' AND eq.status = 'sent';

-- name: OutreachRepliesByKind :many
SELECT classification, COUNT(DISTINCT contact_id) AS n FROM email_replies
WHERE sequence_id = ? GROUP BY classification;

-- name: OutreachRepliesByStep :many
-- The step each person answered: the last one sent to them before their reply.
SELECT CAST(COALESCE((SELECT MAX(et.position) FROM email_queue eq JOIN email_templates et ON et.id = eq.template_id
        WHERE eq.contact_id = r.contact_id AND et.sequence_id = r.sequence_id AND eq.status = 'sent'
          AND eq.sent_at <= r.received_at), 0) AS INTEGER) AS step,
       COUNT(DISTINCT r.contact_id) AS n
FROM email_replies r
WHERE r.sequence_id = sqlc.arg(sequence_id) AND r.classification NOT IN ('bounce', 'spam_block', 'out_of_office')
GROUP BY step;

-- name: OutreachPartners :one
SELECT COUNT(*) FROM contact_sequence_state css
JOIN contact_tags ct ON ct.contact_id = css.contact_id AND ct.tag = 'partner'
WHERE css.sequence_id = ?;

-- ---- opens and clicks ---------------------------------------------------------
-- Recorded by the one tracking endpoint (/api/e/o, /api/e/c) by tracking
-- token. Mail scanners open links and images within seconds of delivery, so
-- nothing before counted_after is counted.

-- name: RecordOutreachClick :execrows
UPDATE email_queue
SET click_count = click_count + 1, clicked_at = COALESCE(clicked_at, sqlc.arg(at))
WHERE tracking_token = sqlc.arg(token) AND kind = 'outreach' AND status = 'sent' AND sent_at <= sqlc.arg(counted_after);

-- name: RecordOutreachOpen :execrows
UPDATE email_queue
SET open_count = open_count + 1, opened_at = COALESCE(opened_at, sqlc.arg(at))
WHERE tracking_token = sqlc.arg(token) AND kind = 'outreach' AND status = 'sent' AND sent_at <= sqlc.arg(counted_after);

-- name: ListOutreachEmailsForSequence :many
SELECT eq.*, c.email AS to_email, m.email AS from_email, et.position
FROM email_queue eq
JOIN email_templates et ON et.id = eq.template_id
JOIN contacts c ON c.id = eq.contact_id
JOIN mailboxes m ON m.id = eq.mailbox_id
WHERE et.sequence_id = ? AND eq.kind = 'outreach'
ORDER BY eq.created_at;

-- name: CountProspectsByStatus :many
SELECT ls.status, COUNT(*) AS n FROM list_subscribers ls
JOIN email_lists el ON el.id = ls.list_id AND el.kind = 'outreach'
WHERE el.org_id = ? GROUP BY ls.status;

-- name: GetOutreachSequenceBySlug :one
SELECT * FROM email_sequences WHERE org_id = ? AND slug = ? AND sequence_type = 'outreach';
