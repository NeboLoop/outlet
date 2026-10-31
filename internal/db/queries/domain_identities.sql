-- name: CreateDomainIdentity :one
INSERT INTO domain_identities (
    id, org_id, domain, verification_status, dkim_status,
    verification_token, dkim_tokens, dns_records, mail_from_domain
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetDomainIdentity :one
SELECT * FROM domain_identities
WHERE id = ?;

-- name: GetDomainIdentityByDomain :one
SELECT * FROM domain_identities
WHERE org_id = ? AND domain = ?;

-- name: ListDomainIdentitiesByOrg :many
SELECT * FROM domain_identities
WHERE org_id = ?
ORDER BY created_at DESC;

-- name: UpdateDomainIdentityStatus :one
UPDATE domain_identities
SET verification_status = ?,
    dkim_status = ?,
    last_checked_at = datetime('now'),
    updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: UpdateDomainIdentityFull :one
UPDATE domain_identities
SET verification_status = ?,
    dkim_status = ?,
    verification_token = ?,
    dns_records = ?,
    last_checked_at = datetime('now'),
    updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: UpdateDomainIdentityMailFrom :one
UPDATE domain_identities
SET mail_from_domain = ?,
    mail_from_status = ?,
    dns_records = ?,
    updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: UpdateDomainIdentityDNSRecords :one
UPDATE domain_identities
SET dns_records = ?,
    updated_at = datetime('now')
WHERE id = ?
RETURNING *;

-- name: DeleteDomainIdentity :exec
DELETE FROM domain_identities
WHERE id = ?;

-- name: DeleteDomainIdentityByOrg :exec
DELETE FROM domain_identities
WHERE org_id = ? AND domain = ?;

-- name: ListPendingDomainIdentities :many
SELECT * FROM domain_identities
WHERE verification_status IN ('pending', 'Pending', 'not_started')
   OR dkim_status IN ('pending', 'Pending', 'not_started')
ORDER BY created_at ASC;

-- A domain belongs to one brand: brands verify their domains on a shared
-- SES account, so a second brand holding the same domain could send as it
-- or delete it from SES.

-- name: CountOtherBrandsWithDomain :one
SELECT COUNT(*) FROM domain_identities
WHERE LOWER(domain) = LOWER(sqlc.arg(domain)) AND org_id <> sqlc.arg(org_id);

-- name: CountBrandDomain :one
SELECT COUNT(*) FROM domain_identities
WHERE org_id = sqlc.arg(org_id) AND LOWER(domain) = LOWER(sqlc.arg(domain));
