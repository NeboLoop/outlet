-- +goose Up
-- When a send was first refused for now by the mail server (a 4xx, a dropped
-- connection, SES throttling). Outlet keeps trying it, like a mail server
-- does, and gives up five days after this. A limit hold (a daily limit,
-- warm-up, a paused pool) is not a refusal and never sets it.
ALTER TABLE email_queue ADD COLUMN first_refused_at TEXT;
ALTER TABLE campaign_sends ADD COLUMN first_refused_at TEXT;

-- +goose Down
ALTER TABLE campaign_sends DROP COLUMN first_refused_at;
ALTER TABLE email_queue DROP COLUMN first_refused_at;
