-- Run by 0042_outbound_targeting.go: it alters 0021's sending_pathways, so
-- sqlc must read it after that schema. Kept out of the goose directory so
-- goose never runs it as is.

-- Outbound targeting. Each piece extends what outbound already runs on.

-- A segment is an outreach list with a definition: who it targets, in plain
-- words, and filters (JSON: countries, regions, industries, company_sizes,
-- title_keywords, fields). A list without one is an explicit list.
ALTER TABLE email_lists ADD COLUMN segment_description TEXT NOT NULL DEFAULT '';
ALTER TABLE email_lists ADD COLUMN segment_filters TEXT NOT NULL DEFAULT '{}';

-- What a business sells, to whom, its proof and how it sounds: what an AI
-- writing outreach for the brand starts from. Filled in by that AI from its
-- own research (Outlet does no browsing).
CREATE TABLE IF NOT EXISTS brand_profiles (
    org_id TEXT PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    website TEXT NOT NULL DEFAULT '',
    sells TEXT NOT NULL DEFAULT '',
    audience TEXT NOT NULL DEFAULT '',
    proof TEXT NOT NULL DEFAULT '',
    voice TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now'))
);

-- A brand's materials: case studies, proof points, offers, links and assets,
-- each with a short summary. list_id names the one segment a material is
-- for; null is any segment.
CREATE TABLE IF NOT EXISTS brand_materials (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('case_study', 'proof_point', 'offer', 'link', 'asset')),
    title TEXT NOT NULL,
    summary TEXT NOT NULL,
    details TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL DEFAULT '',
    list_id INTEGER REFERENCES email_lists(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_brand_materials_org ON brand_materials(org_id, kind);

-- The materials a campaign was written from.
CREATE TABLE IF NOT EXISTS campaign_materials (
    sequence_id TEXT NOT NULL REFERENCES email_sequences(id) ON DELETE CASCADE,
    material_id TEXT NOT NULL REFERENCES brand_materials(id) ON DELETE CASCADE,
    PRIMARY KEY (sequence_id, material_id)
);

-- People outreach never writes to: uploaded lists of addresses and domains
-- (existing customers, open deals). Outreach only: unlike the suppression
-- list, they never stop a receipt or a newsletter.
CREATE TABLE IF NOT EXISTS do_not_contact (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    list_name TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('email', 'domain')),
    value TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),
    UNIQUE (org_id, list_name, kind, value)
);
CREATE INDEX IF NOT EXISTS idx_do_not_contact_value ON do_not_contact(org_id, value);

-- The brand's do-not-contact options: also skip everyone at an existing
-- customer's company domain, and an outside check (a URL answering yes or no
-- for one address, signed with the secret, encrypted here).
ALTER TABLE organizations ADD COLUMN dnc_customer_domains INTEGER NOT NULL DEFAULT 0;
ALTER TABLE organizations ADD COLUMN dnc_check_url TEXT NOT NULL DEFAULT '';
ALTER TABLE organizations ADD COLUMN dnc_check_secret_enc BLOB;

-- The pace dial: first emails a day across a mailbox pool (0: no limit).
ALTER TABLE sending_pathways ADD COLUMN daily_new_prospects INTEGER NOT NULL DEFAULT 0;

-- Who wrote an outreach email: ai, ai_edited (an AI wrote it, a person
-- edited it) or person. A campaign's steps carry it to every email queued
-- from them.
ALTER TABLE email_sequences ADD COLUMN authored_by TEXT NOT NULL DEFAULT '';
ALTER TABLE email_queue ADD COLUMN authored_by TEXT NOT NULL DEFAULT '';
