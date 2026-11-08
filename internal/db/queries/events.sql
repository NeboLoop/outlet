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
SELECT cs.id, ec.org_id, cs.contact_id, c.email, ec.subject, cs.campaign_id,
    COALESCE(cs.list_id, 0) AS list_id
FROM campaign_sends cs
JOIN email_campaigns ec ON ec.id = cs.campaign_id
JOIN contacts c ON c.id = cs.contact_id
WHERE cs.id = sqlc.arg(id);

-- name: GetTransactionalSendEvent :one
SELECT ts.id, ts.org_id, COALESCE(ts.contact_id, '') AS contact_id, ts.to_email, te.subject
FROM transactional_sends ts
JOIN transactional_emails te ON te.id = ts.template_id
WHERE ts.id = sqlc.arg(id);
