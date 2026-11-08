-- +goose Up
-- Each send keeps the subject the person got, its merge tags filled, so the
-- email.sent, email.opened and email.clicked webhooks report it. Queued
-- email (email_queue) already has a subject column.
ALTER TABLE campaign_sends ADD COLUMN subject TEXT;
ALTER TABLE transactional_sends ADD COLUMN subject TEXT;

-- +goose Down
ALTER TABLE transactional_sends DROP COLUMN subject;
ALTER TABLE campaign_sends DROP COLUMN subject;
