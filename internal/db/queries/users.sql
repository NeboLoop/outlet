-- name: GetUserByEmail :one
SELECT id, email, password_hash, role, name, status, email_verified, phone, avatar_url,
       last_login_at, password_changed_at, failed_login_attempts, locked_until,
       created_at, updated_at
FROM users
WHERE LOWER(email) = LOWER(sqlc.arg(email))
LIMIT 1;

-- name: GetUserByID :one
SELECT id, email, password_hash, role, name, status, email_verified, phone, avatar_url,
       last_login_at, password_changed_at, failed_login_attempts, locked_until,
       created_at, updated_at
FROM users
WHERE id = sqlc.arg(id)
LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (
    id, email, password_hash, name, role, status, email_verified, phone, created_at, updated_at
) VALUES (
    sqlc.arg(id), sqlc.arg(email), sqlc.arg(password_hash), sqlc.arg(name), sqlc.arg(role),
    COALESCE(sqlc.arg(status), 'pending'), COALESCE(sqlc.arg(email_verified), 0), sqlc.arg(phone),
    datetime('now'), datetime('now')
)
RETURNING id, email, password_hash, role, name, status, email_verified, phone, avatar_url,
          last_login_at, password_changed_at, failed_login_attempts, locked_until,
          created_at, updated_at;

-- name: UpdateUser :exec
UPDATE users
SET name = COALESCE(sqlc.arg(name), name),
    phone = COALESCE(sqlc.arg(phone), phone),
    avatar_url = COALESCE(sqlc.arg(avatar_url), avatar_url),
    status = COALESCE(sqlc.arg(status), status),
    updated_at = datetime('now')
WHERE id = sqlc.arg(id);

-- name: DeleteUser :execrows
-- The last active super admin is never deleted: with none left, nobody runs
-- the platform and the first-run setup would open to anyone.
DELETE FROM users
WHERE users.id = sqlc.arg(id)
  AND NOT (users.role = 'super_admin' AND users.status = 'active'
           AND (SELECT COUNT(*) FROM users AS admins WHERE admins.role = 'super_admin' AND admins.status = 'active') <= 1);

-- name: ListUsers :many
SELECT id, email, password_hash, role, name, status, email_verified, phone, avatar_url,
       last_login_at, password_changed_at, failed_login_attempts, locked_until,
       created_at, updated_at
FROM users
WHERE (sqlc.arg(filter_role) IS NULL OR sqlc.arg(filter_role) = '' OR role = sqlc.arg(filter_role))
  AND (sqlc.arg(filter_status) IS NULL OR sqlc.arg(filter_status) = '' OR status = sqlc.arg(filter_status))
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: UpdateLastLogin :exec
UPDATE users
SET last_login_at = datetime('now'),
    failed_login_attempts = 0,
    locked_until = NULL,
    updated_at = datetime('now')
WHERE id = sqlc.arg(id);

-- name: IncrementFailedLogins :exec
UPDATE users
SET failed_login_attempts = failed_login_attempts + 1,
    locked_until = CASE
        WHEN failed_login_attempts + 1 >= 5 THEN datetime('now', '+30 minutes')
        ELSE locked_until
    END,
    updated_at = datetime('now')
WHERE id = sqlc.arg(id);

-- name: ResetFailedLogins :exec
UPDATE users
SET failed_login_attempts = 0,
    locked_until = NULL,
    updated_at = datetime('now')
WHERE id = sqlc.arg(id);

-- name: ListActiveAgents :many
SELECT id, email, name, role, status, phone, avatar_url,
       last_login_at, created_at, updated_at
FROM users
WHERE role = 'agent' AND status = 'active'
ORDER BY created_at ASC;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = sqlc.arg(password_hash),
    password_changed_at = datetime('now'),
    updated_at = datetime('now')
WHERE id = sqlc.arg(id);

-- name: SetUserEmailVerified :exec
UPDATE users
SET email_verified = 1,
    status = CASE WHEN status = 'pending' THEN 'active' ELSE status END,
    updated_at = datetime('now')
WHERE id = sqlc.arg(id);

-- name: CheckEmailExists :one
SELECT CASE WHEN COUNT(*) > 0 THEN 1 ELSE 0 END as found
FROM users WHERE LOWER(email) = LOWER(sqlc.arg(email));

-- name: CountUsers :one
SELECT COUNT(*) as count FROM users
WHERE (sqlc.arg(filter_role) IS NULL OR sqlc.arg(filter_role) = '' OR role = sqlc.arg(filter_role))
  AND (sqlc.arg(filter_status) IS NULL OR sqlc.arg(filter_status) = '' OR status = sqlc.arg(filter_status));
