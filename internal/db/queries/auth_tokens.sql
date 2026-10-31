-- name: CreateAuthToken :one
INSERT INTO auth_tokens (id, user_id, token, token_type, expires_at, created_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(token), sqlc.arg(token_type), sqlc.arg(expires_at), datetime('now'))
RETURNING id, user_id, token, token_type, expires_at, used_at, created_at;

-- name: GetAuthTokenByToken :one
SELECT id, user_id, token, token_type, expires_at, used_at, created_at
FROM auth_tokens
WHERE token = sqlc.arg(token)
  AND used_at IS NULL
  AND datetime(expires_at) > datetime('now')
LIMIT 1;

-- name: MarkAuthTokenUsed :exec
UPDATE auth_tokens
SET used_at = datetime('now')
WHERE id = sqlc.arg(id);

-- name: DeleteExpiredAuthTokens :exec
DELETE FROM auth_tokens
WHERE datetime(expires_at) < datetime('now') OR used_at IS NOT NULL;

-- name: DeleteAuthTokensByUser :exec
DELETE FROM auth_tokens
WHERE user_id = sqlc.arg(user_id) AND token_type = sqlc.arg(token_type);

-- name: GetPendingAuthToken :one
SELECT id, user_id, token, token_type, expires_at, used_at, created_at
FROM auth_tokens
WHERE user_id = sqlc.arg(user_id)
  AND token_type = sqlc.arg(token_type)
  AND used_at IS NULL
  AND datetime(expires_at) > datetime('now')
ORDER BY created_at DESC
LIMIT 1;
