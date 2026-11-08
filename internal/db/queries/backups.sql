-- name: CreateBackupRecord :one
INSERT INTO backup_history (
    id, filename, file_path, file_size, backup_type, storage_type,
    s3_bucket, s3_key, status, created_by, started_at
) VALUES (
    sqlc.arg(id), sqlc.arg(filename), sqlc.arg(file_path), sqlc.arg(file_size),
    sqlc.arg(backup_type), sqlc.arg(storage_type), sqlc.arg(s3_bucket),
    sqlc.arg(s3_key), sqlc.arg(status), sqlc.arg(created_by), datetime('now')
)
RETURNING *;

-- name: UpdateBackupStatus :one
UPDATE backup_history
SET status = sqlc.arg(status),
    error_message = sqlc.arg(error_message),
    completed_at = CASE WHEN sqlc.arg(status) = 'completed' OR sqlc.arg(status) = 'failed' THEN datetime('now') ELSE completed_at END
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdateBackupComplete :one
UPDATE backup_history
SET status = 'completed',
    file_size = sqlc.arg(file_size),
    completed_at = datetime('now')
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: GetBackup :one
SELECT * FROM backup_history
WHERE id = sqlc.arg(id)
LIMIT 1;

-- name: ListBackups :many
SELECT * FROM backup_history
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_val) OFFSET sqlc.arg(offset_val);

-- name: CountBackups :one
SELECT COUNT(*) as total FROM backup_history;

-- name: GetLatestBackup :one
SELECT * FROM backup_history
WHERE status = 'completed'
ORDER BY created_at DESC
LIMIT 1;

-- name: GetBackupsInProgress :many
SELECT * FROM backup_history
WHERE status IN ('pending', 'in_progress')
ORDER BY started_at DESC;

-- name: DeleteBackup :exec
DELETE FROM backup_history
WHERE id = sqlc.arg(id);

-- name: DeleteOldBackups :exec
DELETE FROM backup_history
WHERE created_at < datetime('now', sqlc.arg(days_ago) || ' days')
  AND status = 'completed';

-- name: GetBackupsByType :many
SELECT * FROM backup_history
WHERE backup_type = sqlc.arg(backup_type)
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_val);

-- name: UpdateBackupS3Key :exec
UPDATE backup_history
SET s3_key = sqlc.arg(s3_key)
WHERE id = sqlc.arg(id);
