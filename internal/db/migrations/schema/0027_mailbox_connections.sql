-- Run by 0027_mailbox_connections.go, which first reads each mailbox pool's
-- mailbox service and key (0026), then gives the pool a connection carrying
-- them. Kept out of the goose directory so goose never runs it as is; sqlc
-- reads it after 0021's and 0026's schemas, which it alters.

-- A mailbox pool's connections: where its domains and mailboxes come from.
-- A pool holds any number, several of one service among them (two Maildoso
-- accounts). manual connects domains and mailboxes by hand (Google
-- Workspace, Microsoft 365, any SMTP and IMAP host); maildoso, cheapinboxes
-- and infraforge are reached with the brand's API key there, encrypted with
-- the server's key and never shown again.
CREATE TABLE IF NOT EXISTS mailbox_connections (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    pathway_id TEXT NOT NULL REFERENCES sending_pathways(id) ON DELETE CASCADE,
    service TEXT NOT NULL, -- manual, maildoso, cheapinboxes, infraforge
    name TEXT NOT NULL,
    api_key_enc BLOB,
    -- The daily limit its mailboxes send under unless a mailbox has its own;
    -- NULL uses the pool's. A Maildoso connection starts at 15, its terms.
    daily_limit INTEGER,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_mailbox_connections_pathway ON mailbox_connections(pathway_id);

-- An outreach domain belongs to one connection of its pool; its mailboxes
-- come from that connection. NULL for a sending domain.
ALTER TABLE domain_identities ADD COLUMN connection_id TEXT REFERENCES mailbox_connections(id);
CREATE INDEX IF NOT EXISTS idx_domain_identities_connection ON domain_identities(connection_id);

-- A quote is a connection's: it buys only there. Quotes live 30 minutes, so
-- the pool-level ones are not carried over.
DROP TABLE IF EXISTS mailbox_service_quotes;
CREATE TABLE IF NOT EXISTS mailbox_service_quotes (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    connection_id TEXT NOT NULL REFERENCES mailbox_connections(id) ON DELETE CASCADE,
    service TEXT NOT NULL,
    kind TEXT NOT NULL, -- domain, connect, mailboxes
    request TEXT NOT NULL, -- the order, as JSON
    price_cents INTEGER,
    currency TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    expires_at TEXT NOT NULL,
    used_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_mailbox_service_quotes_connection ON mailbox_service_quotes(connection_id);

-- The pool's one service and key now live on its connections.
ALTER TABLE sending_pathways DROP COLUMN mailbox_service_key_enc;
ALTER TABLE sending_pathways DROP COLUMN mailbox_service;
