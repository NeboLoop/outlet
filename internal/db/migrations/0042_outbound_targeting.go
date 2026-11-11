package migrations

import (
	"context"
	"database/sql"
	_ "embed"

	"github.com/pressly/goose/v3"
)

// Outbound targeting (segments, the brand profile, materials, do-not-contact,
// the pace dial and who wrote each outreach email): the schema is in
// schema/0042_outbound_targeting.sql, which alters 0021's sending_pathways,
// so sqlc must read it after that schema.

//go:embed schema/0042_outbound_targeting.sql
var outboundTargetingSchema string

func init() {
	goose.AddMigrationContext(upOutboundTargeting, downOutboundTargeting)
}

func upOutboundTargeting(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, outboundTargetingSchema)
	return err
}

func downOutboundTargeting(ctx context.Context, tx *sql.Tx) error {
	for _, stmt := range []string{
		`ALTER TABLE email_queue DROP COLUMN authored_by`,
		`ALTER TABLE email_sequences DROP COLUMN authored_by`,
		`ALTER TABLE sending_pathways DROP COLUMN daily_new_prospects`,
		`ALTER TABLE organizations DROP COLUMN dnc_check_secret_enc`,
		`ALTER TABLE organizations DROP COLUMN dnc_check_url`,
		`ALTER TABLE organizations DROP COLUMN dnc_customer_domains`,
		`DROP INDEX IF EXISTS idx_do_not_contact_value`,
		`DROP TABLE IF EXISTS do_not_contact`,
		`DROP TABLE IF EXISTS campaign_materials`,
		`DROP INDEX IF EXISTS idx_brand_materials_org`,
		`DROP TABLE IF EXISTS brand_materials`,
		`DROP TABLE IF EXISTS brand_profiles`,
		`ALTER TABLE email_lists DROP COLUMN segment_filters`,
		`ALTER TABLE email_lists DROP COLUMN segment_description`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
