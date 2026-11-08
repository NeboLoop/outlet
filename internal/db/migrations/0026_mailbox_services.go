package migrations

import (
	"context"
	"database/sql"
	_ "embed"

	"github.com/pressly/goose/v3"
)

// Mailbox services for mailbox pools: the schema is in
// schema/0026_mailbox_services.sql, which alters 0021's sending_pathways, so
// sqlc must read it after that schema.

//go:embed schema/0026_mailbox_services.sql
var mailboxServicesSchema string

func init() {
	goose.AddMigrationContext(upMailboxServices, downMailboxServices)
}

func upMailboxServices(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, mailboxServicesSchema)
	return err
}

func downMailboxServices(ctx context.Context, tx *sql.Tx) error {
	for _, stmt := range []string{
		`DROP TABLE IF EXISTS mailbox_service_quotes`,
		`ALTER TABLE mailboxes DROP COLUMN service_ref`,
		`ALTER TABLE sending_pathways DROP COLUMN mailbox_service_key_enc`,
		`ALTER TABLE sending_pathways DROP COLUMN mailbox_service`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
