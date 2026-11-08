-- Outbound: times are UTC text, 'YYYY-MM-DD HH:MM:SS', so they compare as text.

-- name: GetOutboundSettings :one
SELECT * FROM outbound_settings WHERE org_id = ?;

-- name: UpsertOutboundSettings :one
INSERT INTO outbound_settings (org_id, mailing_address, opt_out_line, timezone)
VALUES (?, ?, ?, ?)
ON CONFLICT (org_id) DO UPDATE SET
    mailing_address = excluded.mailing_address,
    opt_out_line = excluded.opt_out_line,
    timezone = excluded.timezone,
    updated_at = strftime('%Y-%m-%d %H:%M:%S', 'now')
RETURNING *;

-- name: SetOutboundDNSToken :exec
INSERT INTO outbound_settings (org_id, dns_token_enc) VALUES (?, ?)
ON CONFLICT (org_id) DO UPDATE SET dns_token_enc = excluded.dns_token_enc,
    updated_at = strftime('%Y-%m-%d %H:%M:%S', 'now');

-- name: SetOutboundPaused :exec
INSERT INTO outbound_settings (org_id, paused, paused_reason) VALUES (?, ?, ?)
ON CONFLICT (org_id) DO UPDATE SET paused = excluded.paused, paused_reason = excluded.paused_reason,
    updated_at = strftime('%Y-%m-%d %H:%M:%S', 'now');

-- name: CreateOutboundDomain :one
INSERT INTO outbound_domains (id, org_id, domain, dns_host, dkim_selector, tracking_host)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetOutboundDomain :one
SELECT * FROM outbound_domains WHERE id = ? AND org_id = ?;

-- name: GetOutboundDomainByName :one
SELECT * FROM outbound_domains WHERE domain = ?;

-- name: ListOutboundDomains :many
SELECT * FROM outbound_domains WHERE org_id = ? ORDER BY domain;

-- name: ListAllOutboundDomains :many
SELECT * FROM outbound_domains ORDER BY domain;

-- name: SetOutboundDomainCheck :exec
UPDATE outbound_domains
SET status = ?, check_result = ?, checked_at = strftime('%Y-%m-%d %H:%M:%S', 'now')
WHERE id = ?;

-- name: CreateOutboundMailbox :one
INSERT INTO outbound_mailboxes (
    id, org_id, domain_id, email, from_name, signature,
    smtp_host, smtp_port, smtp_username, smtp_password_enc,
    imap_host, imap_port, imap_username, imap_password_enc, daily_cap
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetOutboundMailbox :one
SELECT * FROM outbound_mailboxes WHERE id = ? AND org_id = ?;

-- name: GetOutboundMailboxByEmail :one
SELECT * FROM outbound_mailboxes WHERE email = ?;

-- name: ListOutboundMailboxes :many
SELECT * FROM outbound_mailboxes WHERE org_id = ? ORDER BY email;

-- name: ListActiveOutboundMailboxes :many
-- Every mailbox the sender and the reply reader work, across brands.
SELECT * FROM outbound_mailboxes WHERE status = 'active' ORDER BY email;

-- name: ListAllOutboundMailboxes :many
SELECT * FROM outbound_mailboxes ORDER BY email;

-- name: SetOutboundMailboxStatus :exec
UPDATE outbound_mailboxes SET status = ?, paused_reason = ? WHERE id = ?;

-- name: PauseOutboundMailboxesOnDomain :exec
UPDATE outbound_mailboxes SET status = 'paused', paused_reason = ?
WHERE domain_id = ? AND status = 'active';

-- name: ResumeOutboundMailboxesPausedForDNS :exec
UPDATE outbound_mailboxes SET status = 'active', paused_reason = ''
WHERE domain_id = ? AND status = 'paused' AND paused_reason LIKE 'DNS:%';

-- name: SetOutboundMailboxNextSend :exec
UPDATE outbound_mailboxes SET next_send_at = ? WHERE id = ?;

-- name: SetOutboundMailboxError :exec
UPDATE outbound_mailboxes SET last_error = ? WHERE id = ?;

-- name: SetOutboundMailboxIMAPState :exec
UPDATE outbound_mailboxes SET imap_uid_validity = ?, imap_last_uid = ? WHERE id = ?;

-- name: CountOutboundSentSince :one
SELECT COUNT(*) FROM outbound_messages
WHERE mailbox_id = ? AND status = 'sent' AND sent_at >= ?;

-- name: CountOutboundBouncesSince :one
SELECT COUNT(*) FROM outbound_replies
WHERE mailbox_id = ? AND classification = 'bounce' AND received_at >= ?;

-- name: CountOutboundComplaintsSince :one
SELECT COUNT(*) FROM outbound_replies
WHERE mailbox_id = ? AND classification = 'complaint' AND received_at >= ?;

-- name: CreateOutboundProspect :one
INSERT INTO outbound_prospects (
    id, org_id, email, first_name, name, timezone, platform, channel_url,
    audience_size, niche, recent_video, personal_line, email_source,
    custom_fields, status, verify_result
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (org_id, email) DO NOTHING
RETURNING *;

-- name: GetOutboundProspect :one
SELECT * FROM outbound_prospects WHERE id = ? AND org_id = ?;

-- name: GetOutboundProspectByEmail :one
SELECT * FROM outbound_prospects WHERE org_id = ? AND email = ?;

-- name: ListOutboundProspectsByIDs :many
SELECT * FROM outbound_prospects WHERE org_id = sqlc.arg(org_id) AND id IN (sqlc.slice(ids));

-- name: SetOutboundProspectStatus :exec
UPDATE outbound_prospects SET status = ?, updated_at = strftime('%Y-%m-%d %H:%M:%S', 'now')
WHERE id = ?;

-- name: CountOutboundProspectsByStatus :many
SELECT status, COUNT(*) AS n FROM outbound_prospects WHERE org_id = ? GROUP BY status;

-- name: CreateOutboundCampaign :one
INSERT INTO outbound_campaigns (id, org_id, name, approval_mode, track_clicks, window_start_hour, window_end_hour, idempotency_key)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetOutboundCampaign :one
SELECT * FROM outbound_campaigns WHERE id = ? AND org_id = ?;

-- name: GetOutboundCampaignByKey :one
SELECT * FROM outbound_campaigns WHERE org_id = ? AND idempotency_key = ?;

-- name: ListOutboundCampaigns :many
SELECT * FROM outbound_campaigns WHERE org_id = ? ORDER BY created_at DESC;

-- name: SetOutboundCampaignStatus :exec
UPDATE outbound_campaigns SET status = ? WHERE id = ? AND org_id = ?;

-- name: CreateOutboundStep :one
INSERT INTO outbound_steps (id, campaign_id, position, delay_days, subject, body)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListOutboundSteps :many
SELECT * FROM outbound_steps WHERE campaign_id = ? ORDER BY position;

-- name: CreateOutboundEnrollment :one
INSERT INTO outbound_enrollments (id, org_id, campaign_id, prospect_id, mailbox_id, next_position, next_at, thread_subject)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (campaign_id, prospect_id) DO NOTHING
RETURNING *;

-- name: GetOutboundEnrollment :one
SELECT * FROM outbound_enrollments WHERE id = ?;

-- name: ListActiveOutboundEnrollmentsForProspect :many
SELECT * FROM outbound_enrollments WHERE prospect_id = ? AND status IN ('active', 'paused');

-- name: ListDueOutboundEnrollments :many
-- Enrollments whose next step is due and has no draft yet. A step's draft
-- is written once; next_at is cleared when it is.
SELECT e.* FROM outbound_enrollments e
JOIN outbound_campaigns c ON c.id = e.campaign_id
WHERE e.status = 'active' AND c.status = 'active'
  AND e.next_at IS NOT NULL AND e.next_at <= ?
ORDER BY e.next_at
LIMIT ?;

-- name: SetOutboundEnrollmentNext :exec
UPDATE outbound_enrollments SET next_position = ?, next_at = ? WHERE id = ?;

-- name: SetOutboundEnrollmentStatus :exec
UPDATE outbound_enrollments SET status = ?, next_at = NULL WHERE id = ?;

-- name: CreateOutboundMessage :one
INSERT INTO outbound_messages (
    id, org_id, enrollment_id, mailbox_id, prospect_id, position, to_email,
    subject, body, status, message_id, in_reply_to, refs, not_before
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetOutboundMessage :one
SELECT * FROM outbound_messages WHERE id = ? AND org_id = ?;

-- name: GetOutboundMessageByMessageID :one
SELECT * FROM outbound_messages WHERE message_id = ? LIMIT 1;

-- name: ListOutboundMessagesByStatus :many
SELECT * FROM outbound_messages WHERE org_id = ? AND status = ? ORDER BY created_at LIMIT ?;

-- name: ListOutboundThread :many
SELECT * FROM outbound_messages
WHERE enrollment_id = ? AND status = 'sent'
ORDER BY sent_at;

-- name: CountOutboundMessagesByStatus :many
SELECT status, COUNT(*) AS n FROM outbound_messages WHERE org_id = ? GROUP BY status;

-- name: ApproveOutboundMessage :execrows
UPDATE outbound_messages SET status = 'approved'
WHERE id = ? AND org_id = ? AND status = 'draft';

-- name: EditOutboundMessage :execrows
UPDATE outbound_messages SET subject = ?, body = ?
WHERE id = ? AND org_id = ? AND status IN ('draft', 'approved');

-- name: SkipOutboundMessage :execrows
UPDATE outbound_messages SET status = 'skipped'
WHERE id = ? AND org_id = ? AND status IN ('draft', 'approved');

-- name: SkipOpenOutboundMessagesForEnrollment :exec
UPDATE outbound_messages SET status = 'skipped'
WHERE enrollment_id = ? AND status IN ('draft', 'approved');

-- name: NextOutboundMessageForMailbox :one
-- The oldest approved email due on this mailbox whose enrollment, if any, is
-- still running.
SELECT m.* FROM outbound_messages m
LEFT JOIN outbound_enrollments e ON e.id = m.enrollment_id
LEFT JOIN outbound_campaigns c ON c.id = e.campaign_id
WHERE m.mailbox_id = ? AND m.status = 'approved' AND m.not_before <= ?
  AND (m.enrollment_id IS NULL OR (e.status = 'active' AND c.status = 'active'))
ORDER BY m.not_before
LIMIT 1;

-- name: ClaimOutboundMessage :execrows
-- Claimed once: only one sender can move a row from approved to sending.
UPDATE outbound_messages SET status = 'sending', claimed_at = ?
WHERE id = ? AND status = 'approved';

-- name: MarkOutboundMessageSent :exec
UPDATE outbound_messages SET status = 'sent', sent_at = ?, error = '' WHERE id = ?;

-- name: MarkOutboundMessageFailed :exec
UPDATE outbound_messages SET status = 'failed', error = ? WHERE id = ?;

-- name: ReleaseOutboundMessage :exec
-- A transient failure: back to approved, tried again later from the same mailbox.
UPDATE outbound_messages SET status = 'approved', claimed_at = NULL, error = ?, not_before = ?
WHERE id = ? AND status = 'sending';

-- name: ListStuckOutboundMessages :many
-- Rows left in sending by a crash. We can't know whether they went, so they
-- are marked failed and never sent again.
SELECT * FROM outbound_messages WHERE status = 'sending' AND claimed_at < ?;

-- name: CreateOutboundReply :one
INSERT INTO outbound_replies (
    id, org_id, mailbox_id, enrollment_id, prospect_id, from_email, subject,
    body, message_id, in_reply_to, refs, classification, handled, received_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (mailbox_id, message_id) DO NOTHING
RETURNING *;

-- name: GetOutboundReply :one
SELECT * FROM outbound_replies WHERE id = ? AND org_id = ?;

-- name: ListOutboundReplies :many
SELECT * FROM outbound_replies WHERE org_id = ? AND handled = ? ORDER BY received_at DESC LIMIT ?;

-- name: SetOutboundReplyClassification :exec
UPDATE outbound_replies SET classification = ?, handled = ? WHERE id = ? AND org_id = ?;

-- name: SetOutboundReplyHandled :exec
UPDATE outbound_replies SET handled = 1 WHERE id = ? AND org_id = ?;

-- name: CountUnhandledOutboundReplies :one
SELECT COUNT(*) FROM outbound_replies WHERE org_id = ? AND handled = 0;

-- name: ListTransactionalDomains :many
-- Every domain any brand sends transactional mail from; outreach may use none.
SELECT domain FROM domain_identities;

-- name: SetOutboundMessageNotBefore :exec
UPDATE outbound_messages SET not_before = ? WHERE id = ? AND status = 'approved';

-- Campaign results. Reply rate is the measure: opens aren't tracked (pixels
-- hurt cold delivery and Apple Mail fakes them).

-- name: OutboundCampaignEnrollmentCounts :many
SELECT status, COUNT(*) AS n FROM outbound_enrollments WHERE campaign_id = ? GROUP BY status;

-- name: OutboundCampaignSentByStep :many
SELECT m.position, COUNT(*) AS n FROM outbound_messages m
JOIN outbound_enrollments e ON e.id = m.enrollment_id
WHERE e.campaign_id = ? AND m.status = 'sent' AND m.position > 0
GROUP BY m.position;

-- name: OutboundCampaignRepliesByClass :many
SELECT r.classification, COUNT(DISTINCT r.prospect_id) AS n FROM outbound_replies r
JOIN outbound_enrollments e ON e.id = r.enrollment_id
WHERE e.campaign_id = ?
GROUP BY r.classification;

-- name: OutboundCampaignRepliesByStep :many
-- Which step each person answered: the last one sent to them before their
-- first human reply.
SELECT CAST(COALESCE((SELECT MAX(m.position) FROM outbound_messages m
        WHERE m.enrollment_id = r.enrollment_id AND m.status = 'sent' AND m.position > 0
          AND m.sent_at <= r.received_at), 0) AS INTEGER) AS step,
    COUNT(DISTINCT r.prospect_id) AS n
FROM outbound_replies r
JOIN outbound_enrollments e ON e.id = r.enrollment_id
WHERE e.campaign_id = ? AND r.classification NOT IN ('bounce', 'out_of_office')
GROUP BY step;

-- name: OutboundCampaignPartners :one
SELECT COUNT(*) FROM outbound_enrollments e
JOIN outbound_prospects p ON p.id = e.prospect_id
WHERE e.campaign_id = ? AND p.status = 'partner';

-- name: LatestOutboundEnrollmentForProspect :one
SELECT * FROM outbound_enrollments WHERE prospect_id = ? ORDER BY created_at DESC LIMIT 1;

-- name: SetOutboundDomainTrackingHost :exec
UPDATE outbound_domains SET tracking_host = ? WHERE id = ?;

-- name: CreateOutboundLink :exec
INSERT INTO outbound_links (token, org_id, message_id, url) VALUES (?, ?, ?, ?);

-- name: GetOutboundLinkForClick :one
SELECT l.token, l.url, m.sent_at FROM outbound_links l
JOIN outbound_messages m ON m.id = l.message_id
WHERE l.token = ?;

-- name: RecordOutboundClick :exec
UPDATE outbound_links SET clicks = clicks + 1,
    first_clicked_at = COALESCE(first_clicked_at, ?)
WHERE token = ?;

-- name: OutboundCampaignClicksByStep :many
-- People who clicked, by the step whose link they clicked.
SELECT m.position, COUNT(DISTINCT m.prospect_id) AS n FROM outbound_links l
JOIN outbound_messages m ON m.id = l.message_id
JOIN outbound_enrollments e ON e.id = m.enrollment_id
WHERE e.campaign_id = ? AND l.clicks > 0
GROUP BY m.position;

-- name: OutboundCampaignClickers :one
SELECT COUNT(DISTINCT m.prospect_id) FROM outbound_links l
JOIN outbound_messages m ON m.id = l.message_id
JOIN outbound_enrollments e ON e.id = m.enrollment_id
WHERE e.campaign_id = ? AND l.clicks > 0;
