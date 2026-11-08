-- +goose Up
-- The audit log of GDPR deletions: one row for each contact deleted at its
-- owner's request, who asked for it, and what each email provider was asked
-- to delete and did. The address itself isn't kept, only its SHA-256, so a
-- later request about the same person can be matched to this one.
CREATE TABLE IF NOT EXISTS gdpr_deletions (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    contact_id TEXT NOT NULL,
    email_sha256 TEXT NOT NULL,
    requested_by TEXT NOT NULL, -- the person's user ID, or "api-key" for a brand key over MCP
    requested_by_email TEXT NOT NULL DEFAULT '',
    via TEXT NOT NULL, -- admin or mcp
    providers_requested INTEGER NOT NULL DEFAULT 0,
    provider_results TEXT NOT NULL DEFAULT '[]', -- JSON: what each sending pathway's provider did
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_gdpr_deletions_org ON gdpr_deletions(org_id, created_at);
CREATE INDEX IF NOT EXISTS idx_gdpr_deletions_email ON gdpr_deletions(email_sha256);

-- +goose Down
DROP TABLE IF EXISTS gdpr_deletions;
