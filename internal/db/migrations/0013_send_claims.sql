-- +goose Up
-- Workers claim rows before sending, so a row polled again while it waits in
-- memory is never sent twice. A claim older than 15 minutes is treated as
-- abandoned (a crash mid-send) and can be claimed again.
ALTER TABLE campaign_sends ADD COLUMN claim_token TEXT;
ALTER TABLE campaign_sends ADD COLUMN claimed_at TEXT;
CREATE INDEX IF NOT EXISTS idx_campaign_sends_claim ON campaign_sends(claim_token);

ALTER TABLE email_queue ADD COLUMN claim_token TEXT;
ALTER TABLE email_queue ADD COLUMN claimed_at TEXT;
CREATE INDEX IF NOT EXISTS idx_email_queue_claim ON email_queue(claim_token);
CREATE INDEX IF NOT EXISTS idx_email_queue_pending ON email_queue(status, scheduled_for);

-- +goose Down
DROP INDEX IF EXISTS idx_email_queue_pending;
DROP INDEX IF EXISTS idx_email_queue_claim;
DROP INDEX IF EXISTS idx_campaign_sends_claim;
ALTER TABLE email_queue DROP COLUMN claimed_at;
ALTER TABLE email_queue DROP COLUMN claim_token;
ALTER TABLE campaign_sends DROP COLUMN claimed_at;
ALTER TABLE campaign_sends DROP COLUMN claim_token;
