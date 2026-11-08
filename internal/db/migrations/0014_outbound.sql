-- +goose Up
-- Outbound: one-to-one outreach from connected mailboxes on outreach domains.
-- It never sends through SES or the brand's transactional path. Prospects are
-- not contacts: they never asked to hear from the brand, so they stay apart
-- from lists and subscribers.

-- One row per brand: its outreach settings. Outbound runs only for brands
-- named in the server's config (Outbound.Brands), never by a tool call.
CREATE TABLE IF NOT EXISTS outbound_settings (
    org_id TEXT PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    mailing_address TEXT NOT NULL DEFAULT '',
    opt_out_line TEXT NOT NULL DEFAULT 'If this isn''t for you, reply "no thanks" and I won''t email again.',
    timezone TEXT NOT NULL DEFAULT 'America/Denver',
    dns_token_enc BLOB,
    paused INTEGER NOT NULL DEFAULT 0,
    paused_reason TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now'))
);

CREATE TABLE IF NOT EXISTS outbound_domains (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    domain TEXT NOT NULL,
    dns_host TEXT NOT NULL DEFAULT 'manual', -- digitalocean or manual
    dkim_selector TEXT NOT NULL DEFAULT 'google',
    -- go.<domain>, a CNAME to this server, when click tracking is set up.
    -- Never a shared host: one sender's reputation must not touch another's.
    tracking_host TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending', -- pending, ok, failing
    check_result TEXT NOT NULL DEFAULT '',
    checked_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),
    UNIQUE (domain)
);

CREATE TABLE IF NOT EXISTS outbound_mailboxes (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    domain_id TEXT NOT NULL REFERENCES outbound_domains(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    from_name TEXT NOT NULL,
    signature TEXT NOT NULL DEFAULT '',
    smtp_host TEXT NOT NULL,
    smtp_port INTEGER NOT NULL,
    smtp_username TEXT NOT NULL,
    smtp_password_enc BLOB NOT NULL,
    imap_host TEXT NOT NULL,
    imap_port INTEGER NOT NULL,
    imap_username TEXT NOT NULL,
    imap_password_enc BLOB NOT NULL,
    status TEXT NOT NULL DEFAULT 'active', -- active, paused
    paused_reason TEXT NOT NULL DEFAULT '',
    daily_cap INTEGER NOT NULL DEFAULT 30,
    warmup_started_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),
    next_send_at TEXT,
    imap_uid_validity INTEGER NOT NULL DEFAULT 0,
    imap_last_uid INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),
    UNIQUE (email)
);

CREATE TABLE IF NOT EXISTS outbound_prospects (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    first_name TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL DEFAULT '',
    timezone TEXT NOT NULL DEFAULT '',
    platform TEXT NOT NULL DEFAULT '',
    channel_url TEXT NOT NULL DEFAULT '',
    audience_size INTEGER NOT NULL DEFAULT 0,
    niche TEXT NOT NULL DEFAULT '',
    recent_video TEXT NOT NULL DEFAULT '',
    personal_line TEXT NOT NULL DEFAULT '',
    email_source TEXT NOT NULL,
    custom_fields TEXT NOT NULL DEFAULT '{}',
    -- new, in_sequence, replied, interested, not_interested, partner,
    -- bounced, unsubscribed, held
    status TEXT NOT NULL DEFAULT 'new',
    verify_result TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),
    UNIQUE (org_id, email)
);

CREATE TABLE IF NOT EXISTS outbound_campaigns (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active', -- active, paused, done
    approval_mode TEXT NOT NULL DEFAULT 'every', -- every, first, automatic
    track_clicks INTEGER NOT NULL DEFAULT 0,
    window_start_hour INTEGER NOT NULL DEFAULT 8,
    window_end_hour INTEGER NOT NULL DEFAULT 17,
    idempotency_key TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now'))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_outbound_campaigns_idem
    ON outbound_campaigns(org_id, idempotency_key) WHERE idempotency_key <> '';

CREATE TABLE IF NOT EXISTS outbound_steps (
    id TEXT PRIMARY KEY,
    campaign_id TEXT NOT NULL REFERENCES outbound_campaigns(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    delay_days INTEGER NOT NULL DEFAULT 0, -- business days after the previous step
    subject TEXT NOT NULL DEFAULT '', -- empty: a reply in the same thread
    body TEXT NOT NULL,
    UNIQUE (campaign_id, position)
);

-- One prospect in one campaign. The mailbox is chosen once and kept, so
-- every step goes from the same sender in the same thread.
CREATE TABLE IF NOT EXISTS outbound_enrollments (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    campaign_id TEXT NOT NULL REFERENCES outbound_campaigns(id) ON DELETE CASCADE,
    prospect_id TEXT NOT NULL REFERENCES outbound_prospects(id) ON DELETE CASCADE,
    mailbox_id TEXT NOT NULL REFERENCES outbound_mailboxes(id),
    -- active, replied, bounced, unsubscribed, done, stopped, paused
    status TEXT NOT NULL DEFAULT 'active',
    next_position INTEGER NOT NULL DEFAULT 1,
    next_at TEXT,
    thread_subject TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),
    UNIQUE (campaign_id, prospect_id)
);
CREATE INDEX IF NOT EXISTS idx_outbound_enrollments_due ON outbound_enrollments(status, next_at);

-- One email: drafted, approved, then claimed and sent exactly once.
CREATE TABLE IF NOT EXISTS outbound_messages (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    enrollment_id TEXT REFERENCES outbound_enrollments(id) ON DELETE CASCADE,
    mailbox_id TEXT NOT NULL REFERENCES outbound_mailboxes(id),
    prospect_id TEXT NOT NULL REFERENCES outbound_prospects(id) ON DELETE CASCADE,
    position INTEGER NOT NULL DEFAULT 0, -- 0: a reply written in a conversation
    to_email TEXT NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    -- draft, approved, sending, sent, skipped, failed
    status TEXT NOT NULL DEFAULT 'draft',
    message_id TEXT NOT NULL,
    in_reply_to TEXT NOT NULL DEFAULT '',
    refs TEXT NOT NULL DEFAULT '',
    not_before TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),
    claimed_at TEXT,
    sent_at TEXT,
    error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_outbound_messages_due ON outbound_messages(mailbox_id, status, not_before);
CREATE INDEX IF NOT EXISTS idx_outbound_messages_msgid ON outbound_messages(message_id);
CREATE INDEX IF NOT EXISTS idx_outbound_messages_org_status ON outbound_messages(org_id, status);

CREATE TABLE IF NOT EXISTS outbound_replies (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    mailbox_id TEXT NOT NULL REFERENCES outbound_mailboxes(id) ON DELETE CASCADE,
    enrollment_id TEXT REFERENCES outbound_enrollments(id) ON DELETE SET NULL,
    prospect_id TEXT REFERENCES outbound_prospects(id) ON DELETE SET NULL,
    from_email TEXT NOT NULL,
    subject TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    message_id TEXT NOT NULL DEFAULT '',
    in_reply_to TEXT NOT NULL DEFAULT '',
    refs TEXT NOT NULL DEFAULT '',
    -- interested, not_interested, question, out_of_office, wrong_person,
    -- unsubscribe, bounce, complaint, needs_review
    classification TEXT NOT NULL DEFAULT 'needs_review',
    handled INTEGER NOT NULL DEFAULT 0,
    received_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),
    UNIQUE (mailbox_id, message_id)
);
CREATE INDEX IF NOT EXISTS idx_outbound_replies_org ON outbound_replies(org_id, handled, received_at);

-- A tracked link in a sent email. The token is random and the address is
-- stored here, so the redirect can only go where the email pointed.
CREATE TABLE IF NOT EXISTS outbound_links (
    token TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    message_id TEXT NOT NULL REFERENCES outbound_messages(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    clicks INTEGER NOT NULL DEFAULT 0,
    first_clicked_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_outbound_links_message ON outbound_links(message_id);

-- +goose Down
DROP TABLE IF EXISTS outbound_links;
DROP TABLE IF EXISTS outbound_replies;
DROP TABLE IF EXISTS outbound_messages;
DROP TABLE IF EXISTS outbound_enrollments;
DROP TABLE IF EXISTS outbound_steps;
DROP TABLE IF EXISTS outbound_campaigns;
DROP TABLE IF EXISTS outbound_prospects;
DROP TABLE IF EXISTS outbound_mailboxes;
DROP TABLE IF EXISTS outbound_domains;
DROP TABLE IF EXISTS outbound_settings;
