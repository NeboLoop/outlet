-- +goose Up
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
