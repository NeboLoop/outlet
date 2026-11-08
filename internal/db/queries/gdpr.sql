-- GDPR deletions audit log

-- name: CreateGDPRDeletion :exec
INSERT INTO gdpr_deletions (
    id, org_id, contact_id, email_sha256, requested_by, requested_by_email,
    via, providers_requested, provider_results
) VALUES (
    sqlc.arg(id), sqlc.arg(org_id), sqlc.arg(contact_id), sqlc.arg(email_sha256), sqlc.arg(requested_by),
    sqlc.arg(requested_by_email), sqlc.arg(via), sqlc.arg(providers_requested), sqlc.arg(provider_results)
);

-- name: GetGDPRDeletion :one
SELECT * FROM gdpr_deletions
WHERE id = sqlc.arg(id) AND org_id = sqlc.arg(org_id);
