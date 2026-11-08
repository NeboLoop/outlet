-- +goose Up
-- Outbound: a brand's one-to-one outreach. It runs on Outlet's own lists,
-- contacts, sequences and email queue; what it adds is a second way to
-- deliver (the brand's connected mailboxes, never SES or any other email
-- platform), the rules cold email needs, and reading replies.

-- The brand's outreach settings. Off until the brand's admin turns it on in
-- the admin app; no MCP tool can. The mailing address goes in every
-- outreach email (CAN-SPAM).
ALTER TABLE organizations ADD COLUMN outbound_enabled INTEGER NOT NULL DEFAULT 0;
ALTER TABLE organizations ADD COLUMN outbound_paused_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE organizations ADD COLUMN mailing_address TEXT NOT NULL DEFAULT '';
ALTER TABLE organizations ADD COLUMN outbound_opt_out_line TEXT NOT NULL DEFAULT 'If this isn''t for you, reply "no thanks" and I won''t email again.';
ALTER TABLE organizations ADD COLUMN outbound_timezone TEXT NOT NULL DEFAULT 'America/Denver';
ALTER TABLE organizations ADD COLUMN dns_token_enc BLOB;

-- A domain is for sending through the platform provider (SES), or for
-- outreach through mailboxes. A brand's transactional and campaign mail
-- never sends from an outreach domain, and outreach never uses a sending one.
ALTER TABLE domain_identities ADD COLUMN purpose TEXT NOT NULL DEFAULT 'sending';
ALTER TABLE domain_identities ADD COLUMN dns_host TEXT NOT NULL DEFAULT 'manual';
ALTER TABLE domain_identities ADD COLUMN dkim_selector TEXT NOT NULL DEFAULT '';
-- go.<domain>, a CNAME to this server, when click and open tracking is set up.
ALTER TABLE domain_identities ADD COLUMN tracking_host TEXT NOT NULL DEFAULT '';
-- The last public-DNS check of an outreach domain (MX, SPF, DKIM, DMARC).
ALTER TABLE domain_identities ADD COLUMN dns_check TEXT NOT NULL DEFAULT '';

-- A connected mailbox: outreach leaves only through these.
CREATE TABLE IF NOT EXISTS mailboxes (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    domain_id TEXT NOT NULL REFERENCES domain_identities(id) ON DELETE CASCADE,
    email TEXT NOT NULL UNIQUE,
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
    warmup_started_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now')),
    next_send_at TEXT,
    imap_uid_validity INTEGER NOT NULL DEFAULT 0,
    imap_last_uid INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_mailboxes_org ON mailboxes(org_id);

-- An outreach list holds prospects: people found for outreach, who never
-- signed up. They are list subscribers with status 'prospect', so campaigns
-- and sign-up mail (which go to 'active' subscribers) never reach them, and
-- the list's custom fields carry their details (platform, recent video...).
ALTER TABLE email_lists ADD COLUMN kind TEXT NOT NULL DEFAULT 'subscribers'; -- subscribers, outreach

-- An outreach sequence (sequence_type 'outreach') sends through the brand's
-- mailboxes, one mailbox per person, and stops at the first reply.
ALTER TABLE email_sequences ADD COLUMN approval_mode TEXT NOT NULL DEFAULT 'automatic'; -- every, first, automatic
ALTER TABLE email_sequences ADD COLUMN window_start_hour INTEGER NOT NULL DEFAULT 8;
ALTER TABLE email_sequences ADD COLUMN window_end_hour INTEGER NOT NULL DEFAULT 17;
ALTER TABLE email_sequences ADD COLUMN track_opens INTEGER NOT NULL DEFAULT 1;
ALTER TABLE email_sequences ADD COLUMN track_clicks INTEGER NOT NULL DEFAULT 1;

-- The mailbox a person's outreach comes from, kept for the whole thread, and
-- why the sequence stopped (replied, bounced, unsubscribed, complained).
ALTER TABLE contact_sequence_state ADD COLUMN mailbox_id TEXT REFERENCES mailboxes(id);
ALTER TABLE contact_sequence_state ADD COLUMN stopped_reason TEXT NOT NULL DEFAULT '';

-- Every queued email has a kind, and the kind decides the route: outreach
-- goes only through its mailbox; marketing and transactional through the
-- brand's provider. Outreach is written when it is queued (subject and body
-- here), so a person can review and edit the exact email; status 'draft'
-- waits for that approval.
ALTER TABLE email_queue ADD COLUMN kind TEXT NOT NULL DEFAULT 'marketing'; -- marketing, outreach
ALTER TABLE email_queue ADD COLUMN mailbox_id TEXT REFERENCES mailboxes(id);
ALTER TABLE email_queue ADD COLUMN subject TEXT;
ALTER TABLE email_queue ADD COLUMN body TEXT;
ALTER TABLE email_queue ADD COLUMN message_id TEXT;
ALTER TABLE email_queue ADD COLUMN in_reply_to TEXT;
ALTER TABLE email_queue ADD COLUMN refs TEXT;
CREATE INDEX IF NOT EXISTS idx_email_queue_mailbox ON email_queue(mailbox_id, status, scheduled_for);
CREATE INDEX IF NOT EXISTS idx_email_queue_message_id ON email_queue(message_id);

-- Replies read from the brand's mailboxes, matched to the email they answer.
CREATE TABLE IF NOT EXISTS email_replies (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    mailbox_id TEXT NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
    contact_id TEXT REFERENCES contacts(id) ON DELETE SET NULL,
    queue_id TEXT REFERENCES email_queue(id) ON DELETE SET NULL,
    sequence_id TEXT REFERENCES email_sequences(id) ON DELETE SET NULL,
    from_email TEXT NOT NULL,
    subject TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    message_id TEXT NOT NULL DEFAULT '',
    in_reply_to TEXT NOT NULL DEFAULT '',
    refs TEXT NOT NULL DEFAULT '',
    -- interested, not_interested, question, out_of_office, wrong_person,
    -- unsubscribe, complaint, bounce, spam_block, needs_review
    classification TEXT NOT NULL DEFAULT 'needs_review',
    handled INTEGER NOT NULL DEFAULT 0,
    received_at TEXT NOT NULL,
    UNIQUE (mailbox_id, message_id)
);
CREATE INDEX IF NOT EXISTS idx_email_replies_org ON email_replies(org_id, handled, received_at);
CREATE INDEX IF NOT EXISTS idx_email_replies_mailbox ON email_replies(mailbox_id, classification, received_at);

-- +goose Down
DROP TABLE IF EXISTS email_replies;
DROP INDEX IF EXISTS idx_email_queue_message_id;
DROP INDEX IF EXISTS idx_email_queue_mailbox;
ALTER TABLE email_queue DROP COLUMN refs;
ALTER TABLE email_queue DROP COLUMN in_reply_to;
ALTER TABLE email_queue DROP COLUMN message_id;
ALTER TABLE email_queue DROP COLUMN body;
ALTER TABLE email_queue DROP COLUMN subject;
ALTER TABLE email_queue DROP COLUMN mailbox_id;
ALTER TABLE email_queue DROP COLUMN kind;
ALTER TABLE contact_sequence_state DROP COLUMN stopped_reason;
ALTER TABLE contact_sequence_state DROP COLUMN mailbox_id;
ALTER TABLE email_sequences DROP COLUMN track_clicks;
ALTER TABLE email_sequences DROP COLUMN track_opens;
ALTER TABLE email_sequences DROP COLUMN window_end_hour;
ALTER TABLE email_sequences DROP COLUMN window_start_hour;
ALTER TABLE email_sequences DROP COLUMN approval_mode;
ALTER TABLE email_lists DROP COLUMN kind;
DROP TABLE IF EXISTS mailboxes;
ALTER TABLE domain_identities DROP COLUMN dns_check;
ALTER TABLE domain_identities DROP COLUMN tracking_host;
ALTER TABLE domain_identities DROP COLUMN dkim_selector;
ALTER TABLE domain_identities DROP COLUMN dns_host;
ALTER TABLE domain_identities DROP COLUMN purpose;
ALTER TABLE organizations DROP COLUMN dns_token_enc;
ALTER TABLE organizations DROP COLUMN outbound_timezone;
ALTER TABLE organizations DROP COLUMN outbound_opt_out_line;
ALTER TABLE organizations DROP COLUMN mailing_address;
ALTER TABLE organizations DROP COLUMN outbound_paused_reason;
ALTER TABLE organizations DROP COLUMN outbound_enabled;
