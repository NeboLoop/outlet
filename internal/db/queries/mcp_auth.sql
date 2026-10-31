-- MCP API Keys

-- name: CreateMCPAPIKey :one
INSERT INTO mcp_api_keys (id, user_id, name, key_hash, key_prefix, scopes, expires_at, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(name), sqlc.arg(key_hash), sqlc.arg(key_prefix), sqlc.arg(scopes), sqlc.arg(expires_at), datetime('now'), datetime('now'))
RETURNING *;

-- name: GetMCPAPIKeyByHash :one
SELECT k.*, u.email as user_email, u.name as user_name, u.role as user_role, u.status as user_status
FROM mcp_api_keys k
JOIN users u ON k.user_id = u.id
WHERE k.key_hash = sqlc.arg(key_hash)
  AND k.revoked_at IS NULL
  AND (k.expires_at IS NULL OR datetime(k.expires_at) > datetime('now'));

-- name: GetMCPAPIKeyByPrefix :many
SELECT * FROM mcp_api_keys
WHERE key_prefix = sqlc.arg(key_prefix)
  AND revoked_at IS NULL;

-- name: ListMCPAPIKeysByUser :many
SELECT * FROM mcp_api_keys
WHERE user_id = sqlc.arg(user_id)
ORDER BY created_at DESC;

-- name: UpdateMCPAPIKeyLastUsed :exec
UPDATE mcp_api_keys
SET last_used_at = datetime('now'), updated_at = datetime('now')
WHERE id = sqlc.arg(id);

-- name: RevokeMCPAPIKey :execrows
-- People revoke their own keys.
UPDATE mcp_api_keys
SET revoked_at = datetime('now'), updated_at = datetime('now')
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) AND revoked_at IS NULL;

-- name: DeleteMCPAPIKey :exec
DELETE FROM mcp_api_keys WHERE id = sqlc.arg(id);

-- MCP OAuth Clients

-- name: CreateMCPOAuthClient :one
INSERT INTO mcp_oauth_clients (id, client_id, client_secret_hash, name, description, redirect_uris, scopes, is_confidential, is_active, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(client_id), sqlc.arg(client_secret_hash), sqlc.arg(name), sqlc.arg(description), sqlc.arg(redirect_uris), sqlc.arg(scopes), sqlc.arg(is_confidential), 1, datetime('now'), datetime('now'))
RETURNING *;

-- name: GetMCPOAuthClientByClientID :one
SELECT * FROM mcp_oauth_clients
WHERE client_id = sqlc.arg(client_id) AND is_active = 1;

-- name: ListMCPOAuthClients :many
SELECT * FROM mcp_oauth_clients
WHERE is_active = 1
ORDER BY created_at DESC;

-- name: UpdateMCPOAuthClient :one
UPDATE mcp_oauth_clients
SET
    name = COALESCE(NULLIF(sqlc.arg(name), ''), name),
    description = COALESCE(NULLIF(sqlc.arg(description), ''), description),
    redirect_uris = COALESCE(NULLIF(sqlc.arg(redirect_uris), ''), redirect_uris),
    scopes = COALESCE(NULLIF(sqlc.arg(scopes), ''), scopes),
    updated_at = datetime('now')
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeactivateMCPOAuthClient :exec
UPDATE mcp_oauth_clients
SET is_active = 0, updated_at = datetime('now')
WHERE id = sqlc.arg(id);

-- MCP OAuth Authorization Codes

-- name: CreateMCPOAuthCode :one
INSERT INTO mcp_oauth_codes (id, client_id, user_id, code_hash, redirect_uri, scopes, code_challenge, code_challenge_method, expires_at, created_at)
VALUES (sqlc.arg(id), sqlc.arg(client_id), sqlc.arg(user_id), sqlc.arg(code_hash), sqlc.arg(redirect_uri), sqlc.arg(scopes), sqlc.arg(code_challenge), sqlc.arg(code_challenge_method), sqlc.arg(expires_at), datetime('now'))
RETURNING *;

-- name: GetMCPOAuthCodeByHash :one
SELECT c.*, cl.client_id as oauth_client_id, cl.name as client_name
FROM mcp_oauth_codes c
JOIN mcp_oauth_clients cl ON c.client_id = cl.id
WHERE c.code_hash = sqlc.arg(code_hash)
  AND c.used_at IS NULL
  AND datetime(c.expires_at) > datetime('now');

-- name: MarkMCPOAuthCodeUsed :exec
UPDATE mcp_oauth_codes
SET used_at = datetime('now')
WHERE id = sqlc.arg(id);

-- name: CleanupExpiredMCPOAuthCodes :exec
DELETE FROM mcp_oauth_codes
WHERE datetime(expires_at) < datetime('now') OR used_at IS NOT NULL;

-- MCP OAuth Tokens

-- name: CreateMCPOAuthToken :one
INSERT INTO mcp_oauth_tokens (id, client_id, user_id, access_token_hash, refresh_token_hash, scopes, expires_at, refresh_expires_at, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(client_id), sqlc.arg(user_id), sqlc.arg(access_token_hash), sqlc.arg(refresh_token_hash), sqlc.arg(scopes), sqlc.arg(expires_at), sqlc.arg(refresh_expires_at), datetime('now'), datetime('now'))
RETURNING *;

-- name: GetMCPOAuthTokenByAccessHash :one
SELECT t.*, u.email as user_email, u.name as user_name, u.role as user_role, u.status as user_status,
       cl.client_id as oauth_client_id, cl.name as client_name
FROM mcp_oauth_tokens t
JOIN users u ON t.user_id = u.id
JOIN mcp_oauth_clients cl ON t.client_id = cl.id
WHERE t.access_token_hash = sqlc.arg(access_token_hash)
  AND t.revoked_at IS NULL
  AND datetime(t.expires_at) > datetime('now');

-- name: GetMCPOAuthTokenByRefreshHash :one
SELECT t.*, u.email as user_email, u.name as user_name, u.role as user_role, u.status as user_status,
       cl.client_id as oauth_client_id, cl.name as client_name
FROM mcp_oauth_tokens t
JOIN users u ON t.user_id = u.id
JOIN mcp_oauth_clients cl ON t.client_id = cl.id
WHERE t.refresh_token_hash = sqlc.arg(refresh_token_hash)
  AND t.revoked_at IS NULL
  AND (t.refresh_expires_at IS NULL OR datetime(t.refresh_expires_at) > datetime('now'));

-- name: RevokeMCPOAuthToken :exec
UPDATE mcp_oauth_tokens
SET revoked_at = datetime('now'), updated_at = datetime('now')
WHERE id = sqlc.arg(id);

-- name: RevokeMCPOAuthTokensByUser :exec
UPDATE mcp_oauth_tokens
SET revoked_at = datetime('now'), updated_at = datetime('now')
WHERE user_id = sqlc.arg(user_id) AND revoked_at IS NULL;

-- name: CleanupExpiredMCPOAuthTokens :exec
DELETE FROM mcp_oauth_tokens
WHERE (datetime(expires_at) < datetime('now') AND (refresh_expires_at IS NULL OR datetime(refresh_expires_at) < datetime('now')))
   OR datetime(revoked_at) < datetime('now', '-30 days');

-- MCP Sessions (for persisting org selection across server restarts)

-- name: UpsertMCPSession :exec
INSERT INTO mcp_sessions (session_id, user_id, org_id, updated_at)
VALUES (sqlc.arg(session_id), sqlc.arg(user_id), sqlc.arg(org_id), datetime('now'))
ON CONFLICT (session_id) DO UPDATE SET
    org_id = EXCLUDED.org_id,
    updated_at = datetime('now');

-- name: GetMCPSession :one
SELECT * FROM mcp_sessions
WHERE session_id = sqlc.arg(session_id);

-- name: DeleteMCPSession :exec
DELETE FROM mcp_sessions WHERE session_id = sqlc.arg(session_id);

-- name: CleanupOldMCPSessions :exec
-- Delete sessions older than 30 days
DELETE FROM mcp_sessions
WHERE updated_at < datetime('now', '-30 days');

-- MCP audit log (every change made through MCP)

-- name: CreateMCPAuditLog :exec
INSERT INTO mcp_audit_log (id, user_id, user_email, tool, action, org_id, target, result, created_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(user_email), sqlc.arg(tool), sqlc.arg(action), sqlc.arg(org_id), sqlc.arg(target), sqlc.arg(result), datetime('now'));
