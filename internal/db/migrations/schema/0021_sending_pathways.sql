-- Run by 0021_sending_pathways.go, which then gives every existing brand its
-- pathways (it needs Go to move a brand's own AWS keys into the pathway,
-- encrypted). Kept out of the goose directory so goose never runs it as is;
-- sqlc reads it as part of the schema.

-- A sending pathway is one configured way a brand delivers mail:
--   platform   the server's email settings (SES or the platform SMTP)
--   ses        the brand's own AWS SES account
--   smtp       the brand's own SMTP relay
--   mailboxes  the brand's pool of connected mailboxes
-- Each kind of mail (transactional, marketing, outreach) is carried by at
-- most one of a brand's pathways; the partial unique indexes below keep
-- routing unambiguous. Credentials are encrypted with the server's key.
-- The limits are per sender: a mailbox for a pool, the pathway itself for
-- the other providers. 0 turns a limit off.
CREATE TABLE IF NOT EXISTS sending_pathways (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    provider TEXT NOT NULL CHECK (provider IN ('platform', 'ses', 'smtp', 'mailboxes')),
    carries_transactional INTEGER NOT NULL DEFAULT 0,
    carries_marketing INTEGER NOT NULL DEFAULT 0,
    carries_outreach INTEGER NOT NULL DEFAULT 0,
    aws_region TEXT NOT NULL DEFAULT '',
    aws_access_key_enc BLOB,
    aws_secret_key_enc BLOB,
    smtp_host TEXT NOT NULL DEFAULT '',
    smtp_port INTEGER NOT NULL DEFAULT 0,
    smtp_username TEXT NOT NULL DEFAULT '',
    smtp_password_enc BLOB,
    daily_limit INTEGER NOT NULL DEFAULT 0,
    warmup_start INTEGER NOT NULL DEFAULT 0,
    warmup_step INTEGER NOT NULL DEFAULT 0,
    min_gap_seconds INTEGER NOT NULL DEFAULT 0,
    max_gap_seconds INTEGER NOT NULL DEFAULT 0,
    pause_bounce_rate REAL NOT NULL DEFAULT 0.03,
    pause_min_bounces INTEGER NOT NULL DEFAULT 2,
    pause_spam_blocks INTEGER NOT NULL DEFAULT 2,
    pause_complaints INTEGER NOT NULL DEFAULT 2,
    -- the pathway's own spacing, when it is the sender
    next_send_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_sending_pathways_org ON sending_pathways(org_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_sending_pathways_transactional ON sending_pathways(org_id) WHERE carries_transactional = 1;
CREATE UNIQUE INDEX IF NOT EXISTS idx_sending_pathways_marketing ON sending_pathways(org_id) WHERE carries_marketing = 1;
CREATE UNIQUE INDEX IF NOT EXISTS idx_sending_pathways_outreach ON sending_pathways(org_id) WHERE carries_outreach = 1;

-- How many emails each sender sent on one day (in the brand's time zone).
-- Send raises n only while it is under the sender's allowance, in one
-- statement, so two senders can't both pass the cap.
CREATE TABLE IF NOT EXISTS send_counts (
    pathway_id TEXT NOT NULL REFERENCES sending_pathways(id) ON DELETE CASCADE,
    sender_id TEXT NOT NULL,
    day TEXT NOT NULL,
    n INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (pathway_id, sender_id, day)
);

-- A domain belongs to one pathway: sending domains to a platform, ses or
-- smtp pathway, outreach domains to a mailbox pool. A mailbox belongs to the
-- pool of its domain.
ALTER TABLE domain_identities ADD COLUMN pathway_id TEXT REFERENCES sending_pathways(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_domain_identities_pathway ON domain_identities(pathway_id);

-- A mailbox's own daily limit; NULL uses its pool's.
ALTER TABLE mailboxes ADD COLUMN daily_limit INTEGER;

-- A campaign send held back by a pathway limit waits until this time.
ALTER TABLE campaign_sends ADD COLUMN send_after TEXT;

-- Every new brand starts with the platform pathway carrying its
-- transactional and marketing mail.
CREATE TRIGGER IF NOT EXISTS organizations_default_pathway AFTER INSERT ON organizations
BEGIN
    INSERT INTO sending_pathways (id, org_id, name, provider, carries_transactional, carries_marketing)
    VALUES (lower(hex(randomblob(16))), NEW.id, 'Platform email', 'platform', 1, 1);
END;
