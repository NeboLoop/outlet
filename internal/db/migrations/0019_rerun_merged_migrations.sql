-- +goose Up
-- outlet-sh/outlet and NeboLoop/outlet both numbered migrations 0010 to 0012.
-- When the two were merged, outlet-sh's moved to 0015 to 0017. A database
-- that ran outlet-sh's under 0010 to 0012 never ran these, so they run
-- again here. Each is safe to repeat: it creates only what is missing and
-- fixes only rows still in the old form.

-- From 0010_bounce_complaint_ids
-- CreateEmailBounce and CreateEmailComplaint inserted no id, so every row
-- the SES webhook recorded has a NULL primary key (SQLite allows it) and
-- RETURNING * failed to scan it: the contact was never blocked. Give those
-- rows an id; the queries now always pass one.
UPDATE email_bounce SET id = lower(hex(randomblob(16))) WHERE id IS NULL;
UPDATE email_complaint SET id = lower(hex(randomblob(16))) WHERE id IS NULL;


-- From 0011_access_audit
-- One row for every change made through MCP: who made it, with which tool,
-- in which brand, on what, and how it ended.
CREATE TABLE IF NOT EXISTS mcp_audit_log (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    user_email TEXT NOT NULL,
    tool TEXT NOT NULL,
    action TEXT NOT NULL,
    org_id TEXT,
    target TEXT,
    result TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_mcp_audit_log_created_at ON mcp_audit_log(created_at);
CREATE INDEX IF NOT EXISTS idx_mcp_audit_log_org_id ON mcp_audit_log(org_id);
CREATE INDEX IF NOT EXISTS idx_mcp_audit_log_user_id ON mcp_audit_log(user_id);


-- From 0012_one_list_ids_form_and_default_scope
-- Campaigns keep their lists in one form, a JSON array of list IDs like
-- ["12","14"]. Campaigns made over MCP kept them comma-separated ("12,14");
-- rewrite those.
UPDATE email_campaigns
SET list_ids = '["' || REPLACE(REPLACE(TRIM(list_ids), ' ', ''), ',', '","') || '"]'
WHERE list_ids IS NOT NULL AND TRIM(list_ids) <> '' AND SUBSTR(TRIM(list_ids), 1, 1) <> '[';

UPDATE email_campaigns
SET exclude_list_ids = '["' || REPLACE(REPLACE(TRIM(exclude_list_ids), ' ', ''), ',', '","') || '"]'
WHERE exclude_list_ids IS NOT NULL AND TRIM(exclude_list_ids) <> '' AND SUBSTR(TRIM(exclude_list_ids), 1, 1) <> '[';

UPDATE email_campaigns SET list_ids = '[]' WHERE list_ids IS NULL OR TRIM(list_ids) = '';

-- An OAuth grant made without a scope gets the default the sign-in step now
-- gives: full access, as the person who signed in. Grants made before that
-- default kept an empty scope, which the MCP gate refuses.
UPDATE mcp_oauth_tokens SET scopes = 'mcp:full' WHERE TRIM(scopes) = '';
UPDATE mcp_oauth_codes SET scopes = 'mcp:full' WHERE TRIM(scopes) = '';


-- +goose Down
SELECT 1;
