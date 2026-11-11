-- name: ReplaceSignInCode :exec
INSERT INTO sign_in_codes (id, user_id, token_hash, code_hash, attempts, expires_at, created_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(token_hash), sqlc.arg(code_hash), 0, sqlc.arg(expires_at), datetime('now'))
ON CONFLICT (user_id) DO UPDATE SET
    id = EXCLUDED.id,
    token_hash = EXCLUDED.token_hash,
    code_hash = EXCLUDED.code_hash,
    attempts = 0,
    expires_at = EXCLUDED.expires_at,
    created_at = EXCLUDED.created_at;

-- name: GetSignInCodeByTokenHash :one
SELECT id, user_id, token_hash, code_hash, attempts, expires_at, created_at
FROM sign_in_codes
WHERE token_hash = sqlc.arg(token_hash)
  AND datetime(expires_at) > datetime('now');

-- name: CountSignInCodeAttempt :one
UPDATE sign_in_codes
SET attempts = attempts + 1
WHERE user_id = sqlc.arg(user_id)
  AND datetime(expires_at) > datetime('now')
RETURNING id, user_id, token_hash, code_hash, attempts, expires_at, created_at;

-- name: DeleteSignInCode :execrows
DELETE FROM sign_in_codes
WHERE id = sqlc.arg(id);
