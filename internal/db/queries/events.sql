-- Who and what an email was, for the events published when it is sent,
-- opened or clicked. One query per table that records a send.

-- name: GetQueuedEmailEvent :one
SELECT eq.id, c.org_id, c.id AS contact_id, c.email,
    COALESCE(eq.subject, et.subject, '') AS subject,
    COALESCE(et.sequence_id, '') AS sequence_id
FROM email_queue eq
JOIN contacts c ON c.id = eq.contact_id
LEFT JOIN email_templates et ON et.id = eq.template_id
WHERE eq.id = sqlc.arg(id);

-- name: GetCampaignSendEvent :one
SELECT cs.id, ec.org_id, cs.contact_id, c.email, COALESCE(cs.subject, ec.subject) AS subject, cs.campaign_id,
    COALESCE(cs.list_id, 0) AS list_id
FROM campaign_sends cs
JOIN email_campaigns ec ON ec.id = cs.campaign_id
JOIN contacts c ON c.id = cs.contact_id
WHERE cs.id = sqlc.arg(id);

-- name: GetTransactionalSendEvent :one
SELECT ts.id, ts.org_id, COALESCE(ts.contact_id, '') AS contact_id, ts.to_email, COALESCE(ts.subject, te.subject) AS subject
FROM transactional_sends ts
JOIN transactional_emails te ON te.id = ts.template_id
WHERE ts.id = sqlc.arg(id);

-- name: GetBrandEmailStats :one
-- What a brand's email did in a date range: every kind it sent (campaign
-- sends, queued sequence and outreach email, transactional sends), with
-- opens and clicks, and the brand's contacts who bounced, complained or
-- unsubscribed. Dates are UTC 'YYYY-MM-DD HH:MM:SS', as stored.
WITH sent AS (
    SELECT eq.opened_at, eq.clicked_at
    FROM email_queue eq
    JOIN contacts c ON c.id = eq.contact_id
    WHERE c.org_id = sqlc.arg(org_id)
      AND eq.sent_at >= sqlc.arg(start_date) AND eq.sent_at <= sqlc.arg(end_date)
    UNION ALL
    SELECT cs.opened_at, cs.clicked_at
    FROM campaign_sends cs
    JOIN email_campaigns ec ON ec.id = cs.campaign_id
    WHERE ec.org_id = sqlc.arg(org_id)
      AND cs.sent_at >= sqlc.arg(start_date) AND cs.sent_at <= sqlc.arg(end_date)
    UNION ALL
    SELECT ts.opened_at, ts.clicked_at
    FROM transactional_sends ts
    WHERE ts.org_id = sqlc.arg(org_id)
      AND ts.sent_at >= sqlc.arg(start_date) AND ts.sent_at <= sqlc.arg(end_date)
)
SELECT
    (SELECT COUNT(*) FROM sent) AS sent_count,
    (SELECT COUNT(opened_at) FROM sent) AS opened_count,
    (SELECT COUNT(clicked_at) FROM sent) AS clicked_count,
    (SELECT COUNT(*) FROM email_bounce b
     WHERE b.created_at >= sqlc.arg(start_date) AND b.created_at <= sqlc.arg(end_date)
       AND EXISTS (SELECT 1 FROM contacts c WHERE c.org_id = sqlc.arg(org_id) AND LOWER(c.email) = b.email_lower)) AS bounced_count,
    (SELECT COUNT(*) FROM email_complaint m
     WHERE m.created_at >= sqlc.arg(start_date) AND m.created_at <= sqlc.arg(end_date)
       AND EXISTS (SELECT 1 FROM contacts c WHERE c.org_id = sqlc.arg(org_id) AND LOWER(c.email) = m.email_lower)) AS complaint_count,
    (SELECT COUNT(*) FROM contacts c
     WHERE c.org_id = sqlc.arg(org_id)
       AND ((c.unsubscribed_at >= sqlc.arg(start_date) AND c.unsubscribed_at <= sqlc.arg(end_date))
            OR EXISTS (SELECT 1 FROM list_subscribers ls
                       WHERE ls.contact_id = c.id
                         AND ls.unsubscribed_at >= sqlc.arg(start_date) AND ls.unsubscribed_at <= sqlc.arg(end_date)))) AS unsubscribed_count;
