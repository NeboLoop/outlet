-- +goose Up
-- The mailbox pool: warmed mailboxes on pool-only domains that a second
-- Outlet deployment sends customers' outreach from. Off unless Pool.Enabled.
-- Times are UTC "YYYY-MM-DD HH:MM:SS" text written by the pool service, so
-- the ramp, gap and rest rules all read one clock.

CREATE TABLE IF NOT EXISTS pool_domains (
    id TEXT PRIMARY KEY,
    domain TEXT NOT NULL UNIQUE COLLATE NOCASE,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'retired')),
    paused_reason TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS pool_mailboxes (
    id TEXT PRIMARY KEY,
    domain_id TEXT NOT NULL REFERENCES pool_domains(id),
    address TEXT NOT NULL UNIQUE COLLATE NOCASE,
    smtp_host TEXT NOT NULL,
    smtp_port INTEGER NOT NULL,
    smtp_username TEXT NOT NULL,
    smtp_password_enc BLOB NOT NULL,
    warm_started_at TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'resting', 'retired')),
    paused_reason TEXT,
    -- list: bounces or complaints; policy: receiving servers blocked it;
    -- provider: the mailbox's own SMTP server refused the connection or
    -- sign-in (only this kind lets the hub move the tenant's queue); admin.
    paused_kind TEXT CHECK (paused_kind IN ('list', 'policy', 'provider', 'admin')),
    -- Health windows count only sends after this time, so a resumed
    -- mailbox starts clean.
    health_since TEXT NOT NULL,
    tenant_org_id TEXT,
    tenant TEXT,
    tenant_assigned_at TEXT,
    tenant_released_at TEXT,
    last_tenant_org_id TEXT,
    last_tenant TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_pool_mailboxes_domain ON pool_mailboxes(domain_id);
CREATE INDEX IF NOT EXISTS idx_pool_mailboxes_tenant ON pool_mailboxes(tenant_org_id, tenant);

CREATE TABLE IF NOT EXISTS pool_sends (
    -- id is also the VERP token: the envelope sender is b+<id>@<BounceDomain>.
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL,
    mailbox_id TEXT NOT NULL REFERENCES pool_mailboxes(id),
    tenant TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    message_id TEXT NOT NULL UNIQUE,
    to_email TEXT NOT NULL,
    created_at TEXT NOT NULL,
    sent_at TEXT,
    status TEXT NOT NULL CHECK (status IN ('sending', 'sent', 'failed')),
    failure_kind TEXT CHECK (failure_kind IN ('recipient', 'policy', 'provider', 'transient')),
    smtp_code INTEGER,
    smtp_response TEXT,
    UNIQUE (org_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_pool_sends_mailbox ON pool_sends(mailbox_id, created_at);
CREATE INDEX IF NOT EXISTS idx_pool_sends_recipient ON pool_sends(org_id, tenant, to_email, created_at);

-- Each health pause, kept so repeat pauses can hold a tenant until its
-- list is cleaned (cleared_at set by the admin clear-hold route).
CREATE TABLE IF NOT EXISTS pool_pauses (
    id TEXT PRIMARY KEY,
    mailbox_id TEXT NOT NULL REFERENCES pool_mailboxes(id),
    tenant_org_id TEXT,
    tenant TEXT,
    kind TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at TEXT NOT NULL,
    cleared_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_pool_pauses_tenant ON pool_pauses(tenant_org_id, tenant, created_at);

-- Pool bounces and complaints are recorded in the existing tables, linked
-- to the pool send they answer.
ALTER TABLE email_bounce ADD COLUMN pool_send_id TEXT;
ALTER TABLE email_complaint ADD COLUMN pool_send_id TEXT;
CREATE INDEX IF NOT EXISTS idx_email_bounce_pool_send ON email_bounce(pool_send_id);
CREATE INDEX IF NOT EXISTS idx_email_complaint_pool_send ON email_complaint(pool_send_id);

-- +goose Down
DROP INDEX IF EXISTS idx_email_complaint_pool_send;
DROP INDEX IF EXISTS idx_email_bounce_pool_send;
ALTER TABLE email_complaint DROP COLUMN pool_send_id;
ALTER TABLE email_bounce DROP COLUMN pool_send_id;
DROP TABLE IF EXISTS pool_pauses;
DROP TABLE IF EXISTS pool_sends;
DROP TABLE IF EXISTS pool_mailboxes;
DROP TABLE IF EXISTS pool_domains;
