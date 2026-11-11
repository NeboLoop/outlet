package listexport

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/middleware"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/rest/pathvar"
	_ "modernc.org/sqlite"
)

func newStore(t *testing.T) (*db.Store, *sql.DB) {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "outlet.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	require.NoError(t, migrations.Run(conn, nil))
	for _, q := range []string{
		`INSERT INTO users (id, email, password_hash, name, role, status) VALUES ('mike', 'mike@example.com', 'x', 'Mike', 'agent', 'active')`,
		`INSERT INTO organizations (id, name, slug, api_key) VALUES ('org1', 'One', 'one', 'k1'), ('org2', 'Two', 'two', 'k2')`,
		`INSERT INTO org_users (org_id, user_id, role) VALUES ('org1', 'mike', 'member')`,
		`INSERT INTO email_lists (id, public_id, org_id, name, slug) VALUES (1, 'p1', 'org1', 'News', 'news'), (2, 'p2', 'org2', 'Theirs', 'theirs')`,
		`INSERT INTO custom_fields (id, list_id, name, field_key) VALUES ('f1', 1, 'Company', 'company'), ('f2', 1, 'Plan', 'plan')`,
		`INSERT INTO contacts (id, org_id, name, email, created_at) VALUES
			('c1', 'org1', 'Ann', 'ann@example.com', '2026-01-01 00:00:00'),
			('c2', 'org1', '=HYPERLINK("http://evil")', 'bo@example.com', '2026-01-02 00:00:00'),
			('c3', 'org1', 'Cy', 'cy@example.com', '2026-01-03 00:00:00'),
			('x1', 'org2', 'X', 'x@example.com', '2026-01-01 00:00:00')`,
		`INSERT INTO list_subscribers (id, list_id, contact_id, status, subscribed_at, verified_at, unsubscribed_at) VALUES
			('s1', 1, 'c1', 'active', '2026-02-01 10:00:00', '2026-02-01 10:05:00', NULL),
			('s2', 1, 'c2', 'unsubscribed', '2026-02-02 10:00:00', NULL, '2026-03-01 00:00:00'),
			('s3', 1, 'c3', 'active', '2026-02-03 10:00:00', NULL, NULL),
			('sx', 2, 'x1', 'active', '2026-02-01 10:00:00', NULL, NULL)`,
		`INSERT INTO custom_field_values (id, subscriber_id, field_id, value) VALUES
			('v1', 's1', 'f1', 'Acme'), ('v2', 's1', 'f2', '+1 555'), ('v3', 's2', 'f1', '@SUM(A1)'), ('v4', 's3', 'f2', '-pro')`,
	} {
		_, err := conn.Exec(q)
		require.NoError(t, err, q)
	}
	return db.NewStore(conn), conn
}

func readCSV(t *testing.T, b []byte) [][]string {
	t.Helper()
	rows, err := csv.NewReader(bytes.NewReader(b)).ReadAll()
	require.NoError(t, err)
	return rows
}

// The export has every subscriber of the list with its status, dates and
// custom fields, in the order they joined, and no cell a spreadsheet would
// run as a formula.
func TestExportWritesSubscribersWithCustomFieldsSafely(t *testing.T) {
	store, _ := newStore(t)
	var buf bytes.Buffer
	res, err := Write(context.Background(), store, &buf, 1, Options{})
	require.NoError(t, err)
	assert.Equal(t, 3, res.Rows)
	assert.Empty(t, res.Next)

	assert.Equal(t, [][]string{
		{"email", "name", "status", "subscribed_at", "verified_at", "unsubscribed_at", "contact_created_at", "company", "plan"},
		{"ann@example.com", "Ann", "active", "2026-02-01 10:00:00", "2026-02-01 10:05:00", "", "2026-01-01 00:00:00", "Acme", "'+1 555"},
		{"bo@example.com", `'=HYPERLINK("http://evil")`, "unsubscribed", "2026-02-02 10:00:00", "", "2026-03-01 00:00:00", "2026-01-02 00:00:00", "'@SUM(A1)", ""},
		{"cy@example.com", "Cy", "active", "2026-02-03 10:00:00", "", "", "2026-01-03 00:00:00", "", "'-pro"},
	}, readCSV(t, buf.Bytes()))

	// Only one status
	buf.Reset()
	res, err = Write(context.Background(), store, &buf, 1, Options{Status: "unsubscribed"})
	require.NoError(t, err)
	assert.Equal(t, 1, res.Rows)
	assert.Equal(t, "bo@example.com", readCSV(t, buf.Bytes())[1][0])
}

func TestSafeRowPrefixesFormulaCells(t *testing.T) {
	assert.Equal(t,
		[]string{"'=1+1", "'+1", "'-1", "'@x", "'\tx", "'\rx", "a=b", "", "plain"},
		SafeRow([]string{"=1+1", "+1", "-1", "@x", "\tx", "\rx", "a=b", "", "plain"}))
}

// flushCounter is a ResponseWriter that counts flushes.
type flushCounter struct {
	*httptest.ResponseRecorder
	flushes int
}

func (f *flushCounter) Flush() { f.flushes++ }

// A large list goes out a page at a time, flushed after each page, and an
// export in parts (Limit and Next) covers every subscriber once.
func TestExportStreamsInPages(t *testing.T) {
	store, conn := newStore(t)
	tx, err := conn.Begin()
	require.NoError(t, err)
	for i := 0; i < 2*pageSize+7; i++ {
		_, err := tx.Exec(`INSERT INTO contacts (id, org_id, name, email) VALUES (?, 'org1', 'N', ?)`, fmt.Sprintf("bulk%d", i), fmt.Sprintf("bulk%d@example.com", i))
		require.NoError(t, err)
		_, err = tx.Exec(`INSERT INTO list_subscribers (id, list_id, contact_id, subscribed_at) VALUES (?, 1, ?, '2026-04-01 00:00:00')`, fmt.Sprintf("b%04d", i), fmt.Sprintf("bulk%d", i))
		require.NoError(t, err)
	}
	require.NoError(t, tx.Commit())
	total := 2*pageSize + 7 + 3

	w := &flushCounter{ResponseRecorder: httptest.NewRecorder()}
	res, err := Write(context.Background(), store, w, 1, Options{})
	require.NoError(t, err)
	assert.Equal(t, total, res.Rows)
	assert.Equal(t, 3, w.flushes, "one flush per page")
	assert.Len(t, readCSV(t, w.Body.Bytes()), total+1)

	seen := map[string]bool{}
	after := ""
	for parts := 0; ; parts++ {
		require.Less(t, parts, 10)
		var buf bytes.Buffer
		res, err := Write(context.Background(), store, &buf, 1, Options{Limit: 400, After: after})
		require.NoError(t, err)
		for _, row := range readCSV(t, buf.Bytes())[1:] {
			assert.False(t, seen[row[0]], "%s twice", row[0])
			seen[row[0]] = true
		}
		if res.Next == "" {
			break
		}
		after = res.Next
	}
	assert.Len(t, seen, total)

	_, err = Write(context.Background(), store, &bytes.Buffer{}, 1, Options{After: "not a cursor"})
	assert.ErrorIs(t, err, ErrBadCursor)
}

// The admin download is for people in the list's brand; the SDK download
// is for the key's brand's lists only.
func TestExportDownloadsRespectTheBrand(t *testing.T) {
	errorx.SetupErrorHandler()
	store, _ := newStore(t)
	user, err := access.Active(context.Background(), store, "mike")
	require.NoError(t, err)

	admin := func(id string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/api/admin/lists/"+id+"/export", nil)
		r = r.WithContext(access.WithUser(r.Context(), user))
		r = pathvar.WithVars(r, map[string]string{"id": id})
		w := httptest.NewRecorder()
		AdminHandler(store)(w, r)
		return w
	}
	w := admin("1")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "text/csv; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), `filename="news-subscribers-`)
	assert.Len(t, readCSV(t, w.Body.Bytes()), 4)

	w = admin("2")
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.NotContains(t, w.Body.String(), "x@example.com")

	sdk := func(orgID, slug string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/sdk/v1/lists/"+slug+"/export?status=active", nil)
		r = r.WithContext(context.WithValue(r.Context(), middleware.OrgIDKey, orgID))
		r = pathvar.WithVars(r, map[string]string{"slug": slug})
		w := httptest.NewRecorder()
		SDKHandler(store)(w, r)
		return w
	}
	w = sdk("org1", "news")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	rows := readCSV(t, w.Body.Bytes())
	assert.Len(t, rows, 3, "active only")
	assert.False(t, strings.Contains(w.Body.String(), "bo@example.com"))

	w = sdk("org1", "theirs")
	assert.Equal(t, http.StatusNotFound, w.Code)
}
