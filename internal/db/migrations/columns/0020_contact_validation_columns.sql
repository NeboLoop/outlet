-- Run by 0020_contact_validation_columns.go, which skips any column the table
-- already has. Kept out of the goose directory so goose never runs it as is;
-- sqlc reads it as part of the schema.
ALTER TABLE contacts ADD COLUMN validation_status TEXT;
ALTER TABLE contacts ADD COLUMN validation_sub_status TEXT;
ALTER TABLE contacts ADD COLUMN validated_at TEXT;
