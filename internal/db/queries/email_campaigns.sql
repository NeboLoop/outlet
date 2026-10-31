-- Email Campaigns (One-time Broadcasts)
-- Campaign management

-- name: CreateCampaign :one
INSERT INTO email_campaigns (
    id, org_id, design_id, name, subject, preview_text,
    from_name, from_email, reply_to,
    html_body, plain_text,
    list_ids, exclude_list_ids, segment_filter,
    status, scheduled_at, track_opens, track_clicks,
    created_at, updated_at
)
VALUES (sqlc.arg(id), sqlc.arg(org_id), sqlc.arg(design_id), sqlc.arg(name), sqlc.arg(subject), sqlc.arg(preview_text), sqlc.arg(from_name), sqlc.arg(from_email), sqlc.arg(reply_to), sqlc.arg(html_body), sqlc.arg(plain_text), sqlc.arg(list_ids), sqlc.arg(exclude_list_ids), sqlc.arg(segment_filter), sqlc.arg(status), sqlc.arg(scheduled_at), sqlc.arg(track_opens), sqlc.arg(track_clicks), datetime('now'), datetime('now'))
RETURNING *;

-- name: GetCampaign :one
SELECT * FROM email_campaigns
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id);

-- name: ListCampaigns :many
SELECT * FROM email_campaigns
WHERE org_id = sqlc.arg(org_id)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: ListCampaignsByStatus :many
SELECT * FROM email_campaigns
WHERE org_id = sqlc.arg(org_id) AND status = sqlc.arg(status)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: UpdateCampaign :one
UPDATE email_campaigns
SET name = COALESCE(NULLIF(sqlc.arg(name), ''), name),
    subject = COALESCE(NULLIF(sqlc.arg(subject), ''), subject),
    preview_text = COALESCE(NULLIF(sqlc.arg(preview_text), ''), preview_text),
    from_name = COALESCE(NULLIF(sqlc.arg(from_name), ''), from_name),
    from_email = COALESCE(NULLIF(sqlc.arg(from_email), ''), from_email),
    reply_to = COALESCE(NULLIF(sqlc.arg(reply_to), ''), reply_to),
    html_body = COALESCE(NULLIF(sqlc.arg(html_body), ''), html_body),
    plain_text = COALESCE(NULLIF(sqlc.arg(plain_text), ''), plain_text),
    list_ids = COALESCE(NULLIF(sqlc.arg(list_ids), ''), list_ids),
    exclude_list_ids = COALESCE(NULLIF(sqlc.arg(exclude_list_ids), ''), exclude_list_ids),
    segment_filter = COALESCE(NULLIF(sqlc.arg(segment_filter), ''), segment_filter),
    track_opens = COALESCE(sqlc.arg(track_opens), track_opens),
    track_clicks = COALESCE(sqlc.arg(track_clicks), track_clicks),
    updated_at = datetime('now')
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id) AND status = 'draft'
RETURNING *;

-- name: UpdateCampaignStatus :one
UPDATE email_campaigns
SET status = sqlc.arg(status),
    started_at = CASE WHEN sqlc.arg(status) = 'sending' THEN datetime('now') ELSE started_at END,
    completed_at = CASE WHEN sqlc.arg(status) = 'sent' THEN datetime('now') ELSE completed_at END,
    updated_at = datetime('now')
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id)
RETURNING *;

-- name: ScheduleCampaign :one
-- The one way a campaign starts sending: the scheduler picks it up once
-- scheduled_at has passed. No scheduled_at means now.
UPDATE email_campaigns
SET status = 'scheduled',
    scheduled_at = COALESCE(sqlc.narg(scheduled_at), datetime('now')),
    updated_at = datetime('now')
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id) AND status = 'draft'
RETURNING *;

-- name: UpdateCampaignStats :exec
UPDATE email_campaigns
SET recipients_count = sqlc.arg(recipients_count),
    sent_count = sqlc.arg(sent_count),
    delivered_count = sqlc.arg(delivered_count),
    opened_count = sqlc.arg(opened_count),
    clicked_count = sqlc.arg(clicked_count),
    bounced_count = sqlc.arg(bounced_count),
    complained_count = sqlc.arg(complained_count),
    unsubscribed_count = sqlc.arg(unsubscribed_count)
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id);

-- name: IncrementCampaignSent :exec
UPDATE email_campaigns
SET sent_count = sent_count + 1
WHERE id = sqlc.arg(id);

-- name: IncrementCampaignOpened :exec
UPDATE email_campaigns
SET opened_count = opened_count + 1
WHERE id = sqlc.arg(id);

-- name: IncrementCampaignClicked :exec
UPDATE email_campaigns
SET clicked_count = clicked_count + 1
WHERE id = sqlc.arg(id);

-- name: DeleteCampaign :exec
DELETE FROM email_campaigns
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id) AND status = 'draft';

-- name: CountCampaigns :one
SELECT COUNT(*) as count FROM email_campaigns
WHERE org_id = sqlc.arg(org_id);

-- name: CountCampaignsByStatus :one
SELECT COUNT(*) as count FROM email_campaigns
WHERE org_id = sqlc.arg(org_id) AND status = sqlc.arg(status);

-- name: GetScheduledCampaigns :many
SELECT * FROM email_campaigns
WHERE status = 'scheduled' AND datetime(scheduled_at) <= datetime('now')
ORDER BY scheduled_at ASC;

-- Campaign Sends

-- name: CreateCampaignSend :one
INSERT INTO campaign_sends (id, campaign_id, contact_id, list_id, tracking_token, status, created_at)
VALUES (sqlc.arg(id), sqlc.arg(campaign_id), sqlc.arg(contact_id), sqlc.arg(list_id), sqlc.arg(tracking_token), 'pending', datetime('now'))
RETURNING *;

-- name: GetCampaignSend :one
SELECT * FROM campaign_sends
WHERE id = sqlc.arg(id);

-- name: GetCampaignSendByTracking :one
SELECT * FROM campaign_sends
WHERE tracking_token = sqlc.arg(tracking_token);

-- name: UpdateCampaignSendStatus :exec
UPDATE campaign_sends
SET status = sqlc.arg(status),
    sent_at = CASE WHEN sqlc.arg(status) = 'sent' THEN datetime('now') ELSE sent_at END,
    delivered_at = CASE WHEN sqlc.arg(status) = 'delivered' THEN datetime('now') ELSE delivered_at END,
    error_message = sqlc.arg(error_message),
    bounce_type = sqlc.arg(bounce_type)
WHERE id = sqlc.arg(id);

-- name: RecordCampaignOpen :exec
UPDATE campaign_sends
SET opened_at = COALESCE(opened_at, datetime('now')),
    open_count = open_count + 1
WHERE id = sqlc.arg(id);

-- name: RecordCampaignClick :exec
UPDATE campaign_sends
SET clicked_at = COALESCE(clicked_at, datetime('now')),
    click_count = click_count + 1
WHERE id = sqlc.arg(id);

-- name: ListCampaignSends :many
SELECT cs.*, c.email, c.name
FROM campaign_sends cs
JOIN contacts c ON cs.contact_id = c.id
WHERE cs.campaign_id = sqlc.arg(campaign_id)
ORDER BY cs.created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountCampaignSendsByStatus :one
SELECT COUNT(*) as count FROM campaign_sends
WHERE campaign_id = sqlc.arg(campaign_id) AND status = sqlc.arg(status);

-- Campaign Clicks

-- name: CreateCampaignClick :one
INSERT INTO campaign_clicks (id, campaign_send_id, campaign_id, contact_id, link_url, link_name, clicked_at)
VALUES (sqlc.arg(id), sqlc.arg(campaign_send_id), sqlc.arg(campaign_id), sqlc.arg(contact_id), sqlc.arg(link_url), sqlc.arg(link_name), datetime('now'))
RETURNING *;

-- name: ListCampaignClicks :many
SELECT cc.*, c.email
FROM campaign_clicks cc
JOIN contacts c ON cc.contact_id = c.id
WHERE cc.campaign_id = sqlc.arg(campaign_id)
ORDER BY cc.clicked_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: GetCampaignLinkStats :many
SELECT link_url, link_name, COUNT(*) as click_count
FROM campaign_clicks
WHERE campaign_id = sqlc.arg(campaign_id)
GROUP BY link_url, link_name
ORDER BY click_count DESC;

-- Campaign Scheduler Queries

-- name: GetCampaignByID :one
SELECT * FROM email_campaigns
WHERE id = sqlc.arg(id);

-- name: UpdateCampaignStatusByID :exec
UPDATE email_campaigns
SET status = sqlc.arg(status),
    started_at = CASE WHEN sqlc.arg(status) = 'sending' THEN datetime('now') ELSE started_at END,
    completed_at = CASE WHEN sqlc.arg(status) = 'sent' THEN datetime('now') ELSE completed_at END,
    updated_at = datetime('now')
WHERE id = sqlc.arg(id);

-- name: SetCampaignRecipientsCount :exec
UPDATE email_campaigns
SET recipients_count = sqlc.arg(recipients_count),
    updated_at = datetime('now')
WHERE id = sqlc.arg(id);

-- name: GetPendingCampaignSends :many
SELECT cs.id, cs.campaign_id, cs.contact_id, cs.list_id, cs.tracking_token, cs.status,
       c.email, c.name,
       ec.subject, ec.html_body, ec.plain_text, ec.from_name, ec.from_email, ec.reply_to,
       ec.track_opens, ec.track_clicks, ec.org_id
FROM campaign_sends cs
JOIN contacts c ON c.id = cs.contact_id
JOIN email_campaigns ec ON ec.id = cs.campaign_id
WHERE cs.status = 'pending'
ORDER BY cs.created_at ASC
LIMIT sqlc.arg(limit_count);

-- name: MarkCampaignSendSent :exec
UPDATE campaign_sends
SET status = 'sent', sent_at = datetime('now')
WHERE id = sqlc.arg(id);

-- name: MarkCampaignSendFailed :exec
UPDATE campaign_sends
SET status = 'failed', error_message = sqlc.arg(error_message)
WHERE id = sqlc.arg(id);

-- name: GetActiveSubscribersForList :many
-- Only a list of the campaign's own brand gives recipients.
SELECT c.id as contact_id, c.email, c.name, ls.list_id
FROM list_subscribers ls
JOIN email_lists l ON l.id = ls.list_id AND l.org_id = sqlc.arg(org_id)
JOIN contacts c ON c.id = ls.contact_id
WHERE ls.list_id = sqlc.arg(list_id)
  AND ls.status = 'active'
  AND c.blocked_at IS NULL
  AND c.unsubscribed_at IS NULL;

-- name: CheckCampaignSendExists :one
SELECT COUNT(*) as count FROM campaign_sends
WHERE campaign_id = sqlc.arg(campaign_id) AND contact_id = sqlc.arg(contact_id);

-- name: CountPendingCampaignSends :one
SELECT COUNT(*) as count FROM campaign_sends
WHERE campaign_id = sqlc.arg(campaign_id) AND status = 'pending';

-- name: CountSentCampaignSends :one
SELECT COUNT(*) as count FROM campaign_sends
WHERE campaign_id = sqlc.arg(campaign_id) AND status = 'sent';

-- Retry Worker Queries

-- name: GetFailedCampaignSendsForRetry :many
SELECT cs.id, cs.campaign_id, cs.contact_id, cs.tracking_token, cs.retry_count, cs.failed_at,
       c.email, c.name,
       ec.subject, ec.html_body, ec.from_name, ec.from_email, ec.reply_to
FROM campaign_sends cs
JOIN contacts c ON c.id = cs.contact_id
JOIN email_campaigns ec ON ec.id = cs.campaign_id
WHERE cs.status = 'failed' AND cs.retry_count < 3
ORDER BY cs.failed_at ASC
LIMIT sqlc.arg(limit_count);

-- name: IncrementCampaignSendRetry :exec
UPDATE campaign_sends
SET retry_count = retry_count + 1
WHERE id = sqlc.arg(id);

-- name: MarkCampaignSendPermanentlyFailed :exec
UPDATE campaign_sends
SET status = 'permanent_failure'
WHERE id = sqlc.arg(id);

-- Dashboard Stats

-- name: GetDashboardEmailStats30Days :one
SELECT
    COALESCE(SUM(sent_count), 0) as emails_sent,
    COALESCE(SUM(opened_count), 0) as emails_opened,
    COALESCE(SUM(clicked_count), 0) as emails_clicked
FROM email_campaigns
WHERE org_id = sqlc.arg(org_id)
  AND status = 'sent'
  AND datetime(completed_at) > datetime('now', '-30 days');

-- name: GetAllOrgsEmailStats30Days :many
SELECT
    org_id,
    COALESCE(SUM(sent_count), 0) as emails_sent
FROM email_campaigns
WHERE status = 'sent'
  AND datetime(completed_at) > datetime('now', '-30 days')
GROUP BY org_id;
