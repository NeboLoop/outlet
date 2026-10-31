-- +goose Up
-- One row for every change made through MCP: who made it, with which tool,
-- in which brand, on what, and how it ended.
CREATE TABLE IF NOT EXISTS mcp_audit_log (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    user_email TEXT NOT NULL,
    tool TEXT NOT NULL,
    action TEXT NOT NULL,
    org_id TEXT,
    target TEXT,
    result TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_mcp_audit_log_created_at ON mcp_audit_log(created_at);
CREATE INDEX IF NOT EXISTS idx_mcp_audit_log_org_id ON mcp_audit_log(org_id);
CREATE INDEX IF NOT EXISTS idx_mcp_audit_log_user_id ON mcp_audit_log(user_id);

-- +goose Down
DROP TABLE IF EXISTS mcp_audit_log;
