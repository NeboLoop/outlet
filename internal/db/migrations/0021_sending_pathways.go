package migrations

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pressly/goose/v3"
)

// Sending pathways. The schema is in schema/0021_sending_pathways.sql; this
// migration runs it, then gives every existing brand the pathways that keep
// its mail flowing as before:
//   - an `ses` pathway with the brand's own AWS keys, when it has them (moved
//     out of organizations.settings and encrypted), else the `platform`
//     pathway, carrying transactional and marketing mail;
//   - a `mailboxes` pathway carrying outreach, for a brand with mailboxes or
//     outreach domains;
//   - each domain attached to the pathway it serves.

//go:embed schema/0021_sending_pathways.sql
var sendingPathwaysSchema string

func init() {
	goose.AddMigrationContext(upSendingPathways, downSendingPathways)
}

// The mailbox pool's defaults when this migration ran. Later changes to the
// defaults live in the email service; a migration keeps what it did.
const poolDefaults = `30, 5, 2, 240, 540`

type orgAWSKeys struct {
	Region, AccessKey, SecretKey string
}

// ownAWSKeys reads a brand's own AWS keys from its settings JSON, and returns
// the settings without them.
func ownAWSKeys(settings string) (orgAWSKeys, string, bool, error) {
	if strings.TrimSpace(settings) == "" {
		return orgAWSKeys{}, settings, false, nil
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal([]byte(settings), &all); err != nil {
		return orgAWSKeys{}, settings, false, nil // not JSON: no keys in it
	}
	raw, ok := all["email"]
	if !ok {
		return orgAWSKeys{}, settings, false, nil
	}
	var emailCfg map[string]any
	if err := json.Unmarshal(raw, &emailCfg); err != nil || emailCfg == nil {
		return orgAWSKeys{}, settings, false, nil
	}
	str := func(k string) string { s, _ := emailCfg[k].(string); return strings.TrimSpace(s) }
	keys := orgAWSKeys{Region: str("aws_region"), AccessKey: str("aws_access_key"), SecretKey: str("aws_secret_key")}
	if keys.AccessKey == "" || keys.SecretKey == "" {
		return orgAWSKeys{}, settings, false, nil
	}
	delete(emailCfg, "aws_access_key")
	delete(emailCfg, "aws_secret_key")
	delete(emailCfg, "aws_region")
	b, err := json.Marshal(emailCfg)
	if err != nil {
		return orgAWSKeys{}, settings, false, err
	}
	all["email"] = b
	out, err := json.Marshal(all)
	if err != nil {
		return orgAWSKeys{}, settings, false, err
	}
	return keys, string(out), true, nil
}

func upSendingPathways(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, sendingPathwaysSchema); err != nil {
		return err
	}

	type org struct{ id, name, settings string }
	rows, err := tx.QueryContext(ctx, `SELECT id, name, COALESCE(settings, '') FROM organizations`)
	if err != nil {
		return err
	}
	var orgs []org
	for rows.Next() {
		var o org
		if err := rows.Scan(&o.id, &o.name, &o.settings); err != nil {
			rows.Close()
			return err
		}
		orgs = append(orgs, o)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, o := range orgs {
		keys, rest, own, err := ownAWSKeys(o.settings)
		if err != nil {
			return err
		}
		var pathwayID string
		if own {
			if credentialCipher == nil {
				return fmt.Errorf("brand %q has its own AWS keys; set ENCRYPTION_KEY so they can move into its sending pathway encrypted", o.name)
			}
			accessEnc, err := credentialCipher.EncryptString(keys.AccessKey)
			if err != nil {
				return err
			}
			secretEnc, err := credentialCipher.EncryptString(keys.SecretKey)
			if err != nil {
				return err
			}
			if keys.Region == "" {
				keys.Region = "us-east-1"
			}
			if err := tx.QueryRowContext(ctx, `
				INSERT INTO sending_pathways (id, org_id, name, provider, carries_transactional, carries_marketing, aws_region, aws_access_key_enc, aws_secret_key_enc)
				VALUES (lower(hex(randomblob(16))), ?, 'Amazon SES', 'ses', 1, 1, ?, ?, ?) RETURNING id`,
				o.id, keys.Region, accessEnc, secretEnc).Scan(&pathwayID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE organizations SET settings = ? WHERE id = ?`, rest, o.id); err != nil {
				return err
			}
		} else if err := tx.QueryRowContext(ctx, `
			INSERT INTO sending_pathways (id, org_id, name, provider, carries_transactional, carries_marketing)
			VALUES (lower(hex(randomblob(16))), ?, 'Platform email', 'platform', 1, 1) RETURNING id`, o.id).Scan(&pathwayID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE domain_identities SET pathway_id = ? WHERE org_id = ? AND purpose = 'sending'`, pathwayID, o.id); err != nil {
			return err
		}

		var pool int
		if err := tx.QueryRowContext(ctx, `
			SELECT (SELECT COUNT(*) FROM mailboxes WHERE org_id = ?) + (SELECT COUNT(*) FROM domain_identities WHERE org_id = ? AND purpose = 'outreach')`,
			o.id, o.id).Scan(&pool); err != nil {
			return err
		}
		if pool == 0 {
			continue
		}
		var poolID string
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO sending_pathways (id, org_id, name, provider, carries_outreach,
				daily_limit, warmup_start, warmup_step, min_gap_seconds, max_gap_seconds)
			VALUES (lower(hex(randomblob(16))), ?, 'Mailboxes', 'mailboxes', 1, `+poolDefaults+`) RETURNING id`, o.id).Scan(&poolID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE domain_identities SET pathway_id = ? WHERE org_id = ? AND purpose = 'outreach'`, poolID, o.id); err != nil {
			return err
		}
	}
	return nil
}

func downSendingPathways(ctx context.Context, tx *sql.Tx) error {
	for _, stmt := range []string{
		`DROP TRIGGER IF EXISTS organizations_default_pathway`,
		`ALTER TABLE campaign_sends DROP COLUMN send_after`,
		`ALTER TABLE mailboxes DROP COLUMN daily_limit`,
		`DROP INDEX IF EXISTS idx_domain_identities_pathway`,
		`ALTER TABLE domain_identities DROP COLUMN pathway_id`,
		`DROP TABLE IF EXISTS send_counts`,
		`DROP TABLE IF EXISTS sending_pathways`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
