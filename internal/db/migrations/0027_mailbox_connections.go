package migrations

import (
	"context"
	"database/sql"
	_ "embed"

	"github.com/pressly/goose/v3"
)

// Mailbox connections: a mailbox pool holds any number of mailbox services,
// each a connection with its own key, and each outreach domain belongs to
// one. The schema is in schema/0027_mailbox_connections.sql. This migration
// reads each pool's one service and key (0026) before the schema drops those
// columns, then gives the pool one connection carrying them, with the pool's
// outreach domains attached: a manual pool gets a manual connection. A
// Maildoso connection's mailboxes default to 15 a day, Maildoso's terms.

//go:embed schema/0027_mailbox_connections.sql
var mailboxConnectionsSchema string

func init() {
	goose.AddMigrationContext(upMailboxConnections, downMailboxConnections)
}

// connectionNames are the services' names when this migration ran.
var connectionNames = map[string]string{
	"manual": "Added by hand", "maildoso": "Maildoso", "cheapinboxes": "CheapInboxes", "infraforge": "Infraforge",
}

func upMailboxConnections(ctx context.Context, tx *sql.Tx) error {
	type pool struct {
		id, orgID, service string
		key                []byte
	}
	rows, err := tx.QueryContext(ctx, `SELECT id, org_id, mailbox_service, mailbox_service_key_enc FROM sending_pathways WHERE provider = 'mailboxes'`)
	if err != nil {
		return err
	}
	var pools []pool
	for rows.Next() {
		var p pool
		if err := rows.Scan(&p.id, &p.orgID, &p.service, &p.key); err != nil {
			rows.Close()
			return err
		}
		pools = append(pools, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, mailboxConnectionsSchema); err != nil {
		return err
	}

	for _, p := range pools {
		name, ok := connectionNames[p.service]
		if !ok {
			p.service, name = "manual", connectionNames["manual"]
		}
		key, limit := any(nil), any(nil)
		if p.service != "manual" && len(p.key) > 0 {
			key = p.key
		}
		if p.service == "maildoso" {
			limit = 15
		}
		var connID string
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO mailbox_connections (id, org_id, pathway_id, service, name, api_key_enc, daily_limit)
			VALUES (lower(hex(randomblob(16))), ?, ?, ?, ?, ?, ?) RETURNING id`,
			p.orgID, p.id, p.service, name, key, limit).Scan(&connID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE domain_identities SET connection_id = ? WHERE pathway_id = ? AND purpose = 'outreach'`, connID, p.id); err != nil {
			return err
		}
	}
	return nil
}

// downMailboxConnections puts each pool back on one service: its first
// connection with a key, else manual.
func downMailboxConnections(ctx context.Context, tx *sql.Tx) error {
	for _, stmt := range []string{
		`ALTER TABLE sending_pathways ADD COLUMN mailbox_service TEXT NOT NULL DEFAULT 'manual'`,
		`ALTER TABLE sending_pathways ADD COLUMN mailbox_service_key_enc BLOB`,
		`UPDATE sending_pathways SET
			mailbox_service = COALESCE((SELECT c.service FROM mailbox_connections c WHERE c.pathway_id = sending_pathways.id AND c.api_key_enc IS NOT NULL ORDER BY c.created_at, c.id LIMIT 1), 'manual'),
			mailbox_service_key_enc = (SELECT c.api_key_enc FROM mailbox_connections c WHERE c.pathway_id = sending_pathways.id AND c.api_key_enc IS NOT NULL ORDER BY c.created_at, c.id LIMIT 1)`,
		`DROP TABLE IF EXISTS mailbox_service_quotes`,
		`CREATE TABLE mailbox_service_quotes (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			pathway_id TEXT NOT NULL REFERENCES sending_pathways(id) ON DELETE CASCADE,
			service TEXT NOT NULL,
			kind TEXT NOT NULL,
			request TEXT NOT NULL,
			price_cents INTEGER,
			currency TEXT NOT NULL DEFAULT '',
			summary TEXT NOT NULL DEFAULT '',
			expires_at TEXT NOT NULL,
			used_at TEXT,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mailbox_service_quotes_pathway ON mailbox_service_quotes(pathway_id)`,
		`DROP INDEX IF EXISTS idx_domain_identities_connection`,
		`ALTER TABLE domain_identities DROP COLUMN connection_id`,
		`DROP TABLE IF EXISTS mailbox_connections`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
