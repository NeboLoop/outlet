-- Email Designs (Global Templates)
-- Reusable email designs at org level

-- name: CreateEmailDesign :one
-- id is AUTOINCREMENT; passing one made every insert use 0
INSERT INTO email_designs (org_id, name, slug, description, category, html_body, plain_text, is_active, created_at, updated_at)
VALUES (sqlc.arg(org_id), sqlc.arg(name), sqlc.arg(slug), sqlc.arg(description), sqlc.arg(category), sqlc.arg(html_body), sqlc.arg(plain_text), sqlc.arg(is_active), datetime('now'), datetime('now'))
RETURNING *;

-- name: GetEmailDesign :one
SELECT * FROM email_designs
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id);

-- name: GetEmailDesignBySlug :one
SELECT * FROM email_designs
WHERE slug = sqlc.arg(slug) AND org_id = sqlc.arg(org_id);

-- name: ListEmailDesigns :many
SELECT * FROM email_designs
WHERE org_id = sqlc.arg(org_id)
ORDER BY created_at DESC;

-- name: ListEmailDesignsByCategory :many
SELECT * FROM email_designs
WHERE org_id = sqlc.arg(org_id) AND category = sqlc.arg(category)
ORDER BY created_at DESC;

-- name: UpdateEmailDesign :one
UPDATE email_designs
SET name = COALESCE(NULLIF(sqlc.arg(name), ''), name),
    slug = COALESCE(NULLIF(sqlc.arg(slug), ''), slug),
    description = COALESCE(NULLIF(sqlc.arg(description), ''), description),
    category = COALESCE(NULLIF(sqlc.arg(category), ''), category),
    html_body = COALESCE(NULLIF(sqlc.arg(html_body), ''), html_body),
    plain_text = COALESCE(NULLIF(sqlc.arg(plain_text), ''), plain_text),
    is_active = COALESCE(sqlc.arg(is_active), is_active),
    updated_at = datetime('now')
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id)
RETURNING *;

-- name: DeleteEmailDesign :exec
DELETE FROM email_designs
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id);

-- name: CountEmailDesigns :one
SELECT COUNT(*) as count FROM email_designs
WHERE org_id = sqlc.arg(org_id);

-- name: CountEmailDesignsByCategory :one
SELECT COUNT(*) as count FROM email_designs
WHERE org_id = sqlc.arg(org_id) AND category = sqlc.arg(category);
