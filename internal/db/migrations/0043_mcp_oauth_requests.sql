-- +goose Up
-- An MCP sign-in in progress: the checked OAuth request (client, redirect
-- URI, state, scope, PKCE challenge) kept on the server between the email
-- page, the code page and the emailed link, so none of it rides in the
-- browser's forms. The browser holds only a handle; request_hash is its hash.
-- browser_hash is the hash of a cookie set where the sign-in started, so the
-- emailed link finishes only in that browser. email is where the sign-in was
-- sent. A row lives as long as its emailed code and is removed when the
-- sign-in finishes.
CREATE TABLE IF NOT EXISTS mcp_oauth_requests (
    id TEXT PRIMARY KEY,
    request_hash TEXT NOT NULL UNIQUE,
    browser_hash TEXT NOT NULL,
    client_id TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    scope TEXT NOT NULL,
    state TEXT NOT NULL,
    code_challenge TEXT NOT NULL,
    code_challenge_method TEXT NOT NULL,
    email TEXT NOT NULL DEFAULT '',
    expires_at TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now'))
);

-- +goose Down
DROP TABLE IF EXISTS mcp_oauth_requests;
