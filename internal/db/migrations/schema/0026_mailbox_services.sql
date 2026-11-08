-- Run by 0026_mailbox_services.go. Kept out of the goose directory so goose
-- never runs it as is; sqlc reads it after 0021's schema, which it alters.

-- A mailbox pool's mailbox service: where its domains and mailboxes come
-- from. manual connects them by hand; maildoso, cheapinboxes and infraforge
-- are reached with the brand's API key there, encrypted with the server's
-- key and never shown again.
ALTER TABLE sending_pathways ADD COLUMN mailbox_service TEXT NOT NULL DEFAULT 'manual';
ALTER TABLE sending_pathways ADD COLUMN mailbox_service_key_enc BLOB;

-- The mailbox service's own id for a mailbox, to delete it there and to
-- match it in a sync. Empty for a mailbox connected by hand.
ALTER TABLE mailboxes ADD COLUMN service_ref TEXT NOT NULL DEFAULT '';

-- A price for something a mailbox service sells (a domain, connecting a
-- domain, mailboxes). Buying needs an unexpired, unused quote and the
-- person's confirmation; nothing spends money otherwise. price_cents is
-- NULL when the service's API gives no price.
CREATE TABLE IF NOT EXISTS mailbox_service_quotes (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    pathway_id TEXT NOT NULL REFERENCES sending_pathways(id) ON DELETE CASCADE,
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
CREATE INDEX IF NOT EXISTS idx_mailbox_service_quotes_pathway ON mailbox_service_quotes(pathway_id);
