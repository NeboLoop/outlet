package migrations

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var embedMigrations embed.FS

// Cipher encrypts a credential with the server's key (crypto.Service).
type Cipher interface {
	EncryptString(plaintext string) ([]byte, error)
}

// credentialCipher is the cipher of the running Run, for migrations that
// move credentials into encrypted columns. Nil when the server has no key.
var credentialCipher Cipher

// Run applies all pending database migrations.
// This is safe to call on every startup - it only applies migrations
// that haven't been applied yet (tracked in goose_db_version table).
// cipher encrypts credentials a migration moves; nil when ENCRYPTION_KEY is
// not set, and a migration that needs it then fails rather than lose them.
func Run(db *sql.DB, cipher Cipher) error {
	goose.SetBaseFS(embedMigrations)
	credentialCipher = cipher

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Status prints the current migration status to stdout.
func Status(db *sql.DB) error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	return goose.Status(db, ".")
}

// Down rolls back the last migration.
func Down(db *sql.DB) error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	return goose.Down(db, ".")
}
