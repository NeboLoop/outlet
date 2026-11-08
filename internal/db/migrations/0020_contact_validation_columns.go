package migrations

import (
	"context"
	"database/sql"
	_ "embed"
	"regexp"
	"strings"

	"github.com/pressly/goose/v3"
)

// Email validation's columns on contacts. outlet-sh databases added them in
// their own 0010 before the two histories were merged, and SQLite can't add
// a column only if it is missing, so each column is checked first.

//go:embed columns/0020_contact_validation_columns.sql
var contactValidationColumns string

var addColumnRe = regexp.MustCompile(`(?i)^ALTER TABLE (\w+) ADD COLUMN (\w+)`)

func init() {
	goose.AddMigrationContext(addMissingColumns(contactValidationColumns), dropColumns(contactValidationColumns))
}

func statements(script string) []string {
	var out []string
	for _, line := range strings.Split(script, "\n") {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "--") {
			out = append(out, strings.TrimSuffix(line, ";"))
		}
	}
	return out
}

func addMissingColumns(script string) goose.GoMigrationContext {
	return func(ctx context.Context, tx *sql.Tx) error {
		for _, stmt := range statements(script) {
			m := addColumnRe.FindStringSubmatch(stmt)
			if m == nil {
				continue
			}
			var n int
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, m[1], m[2]).Scan(&n); err != nil {
				return err
			}
			if n == 0 {
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func dropColumns(script string) goose.GoMigrationContext {
	return func(ctx context.Context, tx *sql.Tx) error {
		stmts := statements(script)
		for i := len(stmts) - 1; i >= 0; i-- {
			if m := addColumnRe.FindStringSubmatch(stmts[i]); m != nil {
				if _, err := tx.ExecContext(ctx, "ALTER TABLE "+m[1]+" DROP COLUMN "+m[2]); err != nil {
					return err
				}
			}
		}
		return nil
	}
}
