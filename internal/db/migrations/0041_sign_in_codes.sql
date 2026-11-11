-- +goose Up
-- An emailed sign-in: one link and one 6-digit code, either of which signs
-- the person in once within 15 minutes. A person has at most one: a new one
-- replaces the last, and using either removes the row. Only keyed hashes of
-- the link token and the code are kept. attempts counts code checks; the
-- fifth wrong code removes the row.
CREATE TABLE IF NOT EXISTS sign_in_codes (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    code_hash TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    expires_at TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now'))
);

-- +goose Down
DROP TABLE IF EXISTS sign_in_codes;
