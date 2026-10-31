-- name: GetOrganizationByAPIKey :one
SELECT * FROM organizations
WHERE api_key = sqlc.arg(api_key)
LIMIT 1;

-- name: GetOrganizationByID :one
SELECT * FROM organizations
WHERE id = sqlc.arg(id)
LIMIT 1;

-- name: GetOrganizationBySlug :one
SELECT * FROM organizations
WHERE slug = sqlc.arg(slug)
LIMIT 1;

-- name: ListOrganizations :many
SELECT * FROM organizations
ORDER BY created_at DESC;

-- name: CreateOrganization :one
INSERT INTO organizations (
    id, name, slug, api_key, max_contacts, settings, app_url, created_at, updated_at
) VALUES (
    sqlc.arg(id), sqlc.arg(name), sqlc.arg(slug), sqlc.arg(api_key),
    sqlc.arg(max_contacts), sqlc.arg(settings), sqlc.arg(app_url),
    datetime('now'), datetime('now')
)
RETURNING *;

-- name: UpdateOrganization :one
UPDATE organizations
SET
    name = COALESCE(NULLIF(sqlc.arg(name), ''), name),
    max_contacts = COALESCE(sqlc.arg(max_contacts), max_contacts),
    settings = COALESCE(sqlc.arg(settings), settings),
    app_url = COALESCE(NULLIF(sqlc.arg(app_url), ''), app_url),
    updated_at = datetime('now')
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: RegenerateAPIKey :one
UPDATE organizations
SET
    api_key = sqlc.arg(api_key),
    api_key_created_at = datetime('now'),
    updated_at = datetime('now')
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteOrganization :exec
DELETE FROM organizations
WHERE id = sqlc.arg(id);

-- name: AddUserToOrganization :exec
INSERT INTO org_users (org_id, user_id, role, created_at)
VALUES (sqlc.arg(org_id), sqlc.arg(user_id), sqlc.arg(role), datetime('now'))
ON CONFLICT (org_id, user_id) DO UPDATE SET role = excluded.role;

-- name: RemoveUserFromOrganization :exec
DELETE FROM org_users
WHERE org_id = sqlc.arg(org_id) AND user_id = sqlc.arg(user_id);

-- name: GetOrganizationUsers :many
SELECT u.*, ou.role as org_role, ou.created_at as joined_at
FROM users u
JOIN org_users ou ON u.id = ou.user_id
WHERE ou.org_id = sqlc.arg(org_id)
ORDER BY ou.created_at;

-- name: GetUserOrganizations :many
SELECT o.*, ou.role as user_role
FROM organizations o
JOIN org_users ou ON o.id = ou.org_id
WHERE ou.user_id = sqlc.arg(user_id)
ORDER BY o.created_at;

-- name: UpdateOrgEmailSettings :one
UPDATE organizations
SET
    from_name = COALESCE(NULLIF(sqlc.arg(from_name), ''), from_name),
    from_email = COALESCE(NULLIF(sqlc.arg(from_email), ''), from_email),
    reply_to = COALESCE(NULLIF(sqlc.arg(reply_to), ''), reply_to),
    updated_at = datetime('now')
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: GetOrgEmailSettings :one
SELECT from_name, from_email, reply_to
FROM organizations
WHERE id = sqlc.arg(id);

-- name: UpdateOrgAppUrl :one
UPDATE organizations
SET app_url = sqlc.arg(app_url), updated_at = datetime('now')
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdateOrgMaxContacts :one
UPDATE organizations
SET
    max_contacts = sqlc.arg(max_contacts),
    updated_at = datetime('now')
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: GetOrgUserRole :one
-- The person's role in one brand. No row means they don't belong to it.
SELECT COALESCE(role, 'member') AS role FROM org_users
WHERE org_id = sqlc.arg(org_id) AND user_id = sqlc.arg(user_id);

-- name: CreateOrgInvitation :one
INSERT INTO org_invitations (id, org_id, email, role, token, expires_at, created_at)
VALUES (sqlc.arg(id), sqlc.arg(org_id), sqlc.arg(email), sqlc.arg(role), sqlc.arg(token), datetime('now', '+7 days'), datetime('now'))
RETURNING *;

-- name: DeletePendingOrgInvitations :exec
-- A new invitation replaces any earlier one for the same person and brand.
DELETE FROM org_invitations
WHERE org_id = sqlc.arg(org_id) AND email = sqlc.arg(email) COLLATE NOCASE AND accepted_at IS NULL;

-- name: GetOrgInvitationByToken :one
SELECT i.id, i.org_id, i.email, COALESCE(i.role, 'member') AS role, i.accepted_at,
       CAST(datetime(i.expires_at) <= datetime('now') AS INTEGER) AS expired,
       o.name AS org_name
FROM org_invitations i
JOIN organizations o ON o.id = i.org_id
WHERE i.token = sqlc.arg(token);

-- name: AcceptOrgInvitation :execrows
UPDATE org_invitations
SET accepted_at = datetime('now')
WHERE id = sqlc.arg(id) AND accepted_at IS NULL AND datetime(expires_at) > datetime('now');
