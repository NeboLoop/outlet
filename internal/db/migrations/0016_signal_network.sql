-- +goose Up
-- Outlet Signal Network hub: subscribed instances and pooled, hashed deliverability
-- signals. Only used when SignalNet.Hub is enabled. No email addresses are stored,
-- only SHA-256 hex of the trimmed, lowercased address.

-- An instance key is a subscription. Only the SHA-256 of the key is stored.
CREATE TABLE IF NOT EXISTS network_instances (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    active INTEGER NOT NULL DEFAULT 1,
    lookups_count INTEGER NOT NULL DEFAULT 0, -- hashes looked up (metering)
    reports_count INTEGER NOT NULL DEFAULT 0, -- events accepted (metering)
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    last_seen_at TEXT
);

-- Timestamps are RFC 3339 UTC so they compare correctly as text.
CREATE TABLE IF NOT EXISTS network_signals (
    hash TEXT PRIMARY KEY,
    delivered INTEGER NOT NULL DEFAULT 0,
    hard_bounces INTEGER NOT NULL DEFAULT 0,
    complaints INTEGER NOT NULL DEFAULT 0,
    engaged INTEGER NOT NULL DEFAULT 0,
    last_engaged_at TEXT,
    last_hard_bounce_at TEXT,
    last_delivered_at TEXT,
    first_seen_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- One row per (address, instance) for distinct-sender counts. complained marks
-- instances that received a complaint, for the distinct-complainer toxic rule.
CREATE TABLE IF NOT EXISTS network_signal_senders (
    hash TEXT NOT NULL,
    instance_id TEXT NOT NULL REFERENCES network_instances(id) ON DELETE CASCADE,
    complained INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (hash, instance_id)
);

-- +goose Down
DROP TABLE IF EXISTS network_signal_senders;
DROP TABLE IF EXISTS network_signals;
DROP TABLE IF EXISTS network_instances;
