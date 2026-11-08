package db_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/outlet-sh/outlet/internal/db/migrations"

	_ "modernc.org/sqlite"
)

// TestEveryQueryPrepares prepares every generated query against the migrated
// schema. sqlc does not check SQLite queries against the schema, so a missing
// column or an untranslated sqlc.arg() otherwise only fails at runtime.
func TestEveryQueryPrepares(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "q.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := migrations.Run(conn, nil); err != nil {
		t.Fatal(err)
	}

	files, _ := filepath.Glob("*.sql.go")
	query := regexp.MustCompile("(?s)const (\\w+) = `-- name: \\w+ :\\w+\n(.*?)`")
	checked := 0
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range query.FindAllStringSubmatch(string(src), -1) {
			checked++
			// EXPLAIN compiles the statement without running it; the driver's
			// Prepare defers that, so it would miss a missing column
			if _, err := conn.Exec("EXPLAIN "+m[2], make([]any, paramCount(m[2]))...); err != nil {
				t.Errorf("%s %s: %v", f, m[1], err)
			}
		}
	}
	if checked < 100 {
		t.Fatalf("only found %d queries; is the pattern still right?", checked)
	}
}

// paramCount returns how many arguments a query takes: numbered ?N
// placeholders count up to the highest N, and each bare ? takes the next one.
func paramCount(q string) int {
	highest, bare := 0, 0
	for _, m := range regexp.MustCompile(`\?(\d*)`).FindAllStringSubmatch(q, -1) {
		if m[1] == "" {
			bare++
			continue
		}
		if n, _ := strconv.Atoi(m[1]); n > highest {
			highest = n
		}
	}
	return highest + bare
}
