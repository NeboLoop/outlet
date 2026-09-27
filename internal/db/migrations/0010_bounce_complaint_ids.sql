-- +goose Up
-- CreateEmailBounce and CreateEmailComplaint inserted no id, so every row
-- the SES webhook recorded has a NULL primary key (SQLite allows it) and
-- RETURNING * failed to scan it: the contact was never blocked. Give those
-- rows an id; the queries now always pass one.
UPDATE email_bounce SET id = lower(hex(randomblob(16))) WHERE id IS NULL;
UPDATE email_complaint SET id = lower(hex(randomblob(16))) WHERE id IS NULL;

-- +goose Down
SELECT 1;
