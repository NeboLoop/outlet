-- +goose Up
-- The contacts columns this feature adds are in 0020, which adds each only
-- when it is missing (this ran as 0010 on outlet-sh databases).
-- Email validation: per-org validation log (doubles as usage metering), bulk jobs,
-- a global probe-result cache, and the latest validation result on each contact.

CREATE TABLE IF NOT EXISTS validation_jobs (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL DEFAULT '',
    list_id INTEGER REFERENCES email_lists(id) ON DELETE SET NULL,
    source TEXT NOT NULL CHECK (source IN ('list', 'upload')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    total INTEGER NOT NULL DEFAULT 0,
    processed INTEGER NOT NULL DEFAULT 0,
    error TEXT,
    started_at TEXT,
    completed_at TEXT,
    created_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_validation_jobs_org ON validation_jobs(org_id, created_at);
CREATE INDEX IF NOT EXISTS idx_validation_jobs_status ON validation_jobs(status);

-- One row per address checked. status = 'pending' while queued in a job.
-- Statuses: valid, invalid, catch-all, unknown, spamtrap, abuse, do_not_mail (ZeroBounce v2 scheme).
CREATE TABLE IF NOT EXISTS email_validations (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    job_id TEXT REFERENCES validation_jobs(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    email_lower TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    sub_status TEXT NOT NULL DEFAULT '',
    free_email INTEGER NOT NULL DEFAULT 0,
    did_you_mean TEXT NOT NULL DEFAULT '',
    account TEXT NOT NULL DEFAULT '',
    domain TEXT NOT NULL DEFAULT '',
    domain_age_days INTEGER,
    smtp_provider TEXT NOT NULL DEFAULT '',
    mx_found INTEGER NOT NULL DEFAULT 0,
    mx_record TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT 'api' CHECK (source IN ('api', 'admin', 'job', 'mcp')),
    attempts INTEGER NOT NULL DEFAULT 0,
    retry_at TEXT,
    processed_at TEXT,
    created_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_email_validations_org ON email_validations(org_id, processed_at);
CREATE INDEX IF NOT EXISTS idx_email_validations_job ON email_validations(job_id, status);
CREATE INDEX IF NOT EXISTS idx_email_validations_retry ON email_validations(status, retry_at);

-- Instance-wide cache of the network-derived result (DNS + SMTP), shared across orgs.
-- result is the JSON-encoded emailval.Result.
CREATE TABLE IF NOT EXISTS email_validation_cache (
    email_lower TEXT PRIMARY KEY,
    result TEXT NOT NULL,
    checked_at TEXT NOT NULL DEFAULT (datetime('now'))
);


-- +goose Down
DROP TABLE IF EXISTS email_validation_cache;
DROP TABLE IF EXISTS email_validations;
DROP TABLE IF EXISTS validation_jobs;
