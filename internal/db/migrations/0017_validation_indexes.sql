-- +goose Up
-- Indexes for the instance-wide signal lookups made by email validation.
CREATE INDEX IF NOT EXISTS idx_contacts_email_lower ON contacts(LOWER(email));
CREATE INDEX IF NOT EXISTS idx_suppression_list_email_lower ON suppression_list(email_lower);

-- +goose Down
DROP INDEX IF EXISTS idx_suppression_list_email_lower;
DROP INDEX IF EXISTS idx_contacts_email_lower;
