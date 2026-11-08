// Package listexport writes a list's subscribers as CSV: email, name,
// status, dates and the list's custom fields. It is the one export the admin
// app, the SDK and MCP use. Rows are read and written a page at a time, so a
// large list never sits in memory.
package listexport

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/middleware"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// Options say which part of a list to export.
type Options struct {
	// Status keeps subscribers with this status (active, pending,
	// unsubscribed, ...). Empty or "all" keeps every subscriber.
	Status string
	// Limit is the most rows to write; 0 writes them all.
	Limit int
	// After continues an export from a Result's Next; empty starts it.
	After string
}

// Result is what one Write wrote.
type Result struct {
	Rows int
	// Next continues the export where this Write stopped. Empty when the
	// list is done.
	Next string
}

// pageSize is how many subscribers one read takes.
const pageSize = 500

// fixedColumns are the columns every export has, before the list's custom
// fields. email and name are the columns an import reads.
var fixedColumns = []string{"email", "name", "status", "subscribed_at", "verified_at", "unsubscribed_at", "contact_created_at"}

// ErrBadCursor is returned for an After that no Write gave.
var ErrBadCursor = errors.New("invalid export cursor")

// Write writes a header row and the list's subscribers to w, in the order
// they joined the list. After each page it flushes, and flushes w when w is
// an http.Flusher, so a download streams.
func Write(ctx context.Context, q db.Querier, w io.Writer, listID int64, opts Options) (Result, error) {
	afterAt, afterID, err := decodeCursor(opts.After)
	if err != nil {
		return Result{}, err
	}
	status := opts.Status
	if status == "all" {
		status = ""
	}
	fields, err := q.ListCustomFieldsByList(ctx, listID)
	if err != nil {
		return Result{}, err
	}

	cw := csv.NewWriter(w)
	header := append([]string{}, fixedColumns...)
	for _, f := range fields {
		header = append(header, f.FieldKey)
	}
	if err := cw.Write(safeRow(header)); err != nil {
		return Result{}, err
	}

	var res Result
	for {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		take := pageSize
		if opts.Limit > 0 && opts.Limit-res.Rows < take {
			take = opts.Limit - res.Rows
		}
		rows, err := q.ListListSubscribersForExport(ctx, db.ListListSubscribersForExportParams{
			ListID:            listID,
			AfterSubscribedAt: afterAt,
			AfterID:           afterID,
			FilterStatus:      status,
			LimitCount:        int64(take),
		})
		if err != nil {
			return res, err
		}
		values, err := customValues(ctx, q, rows, len(fields))
		if err != nil {
			return res, err
		}
		for _, r := range rows {
			record := []string{
				r.Email, r.Name, r.Status.String, r.SubscribedAt.String, r.VerifiedAt.String,
				r.UnsubscribedAt.String, r.ContactCreatedAt.String,
			}
			for _, f := range fields {
				record = append(record, values[r.ID][f.ID])
			}
			if err := cw.Write(safeRow(record)); err != nil {
				return res, err
			}
		}
		res.Rows += len(rows)
		cw.Flush()
		if err := cw.Error(); err != nil {
			return res, err
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		if len(rows) < take {
			res.Next = ""
			return res, nil
		}
		last := rows[len(rows)-1]
		afterAt, afterID = last.SubscribedAt.String, last.ID
		if opts.Limit > 0 && res.Rows >= opts.Limit {
			res.Next = encodeCursor(afterAt, afterID)
			return res, nil
		}
	}
}

// customValues returns the custom field values of a page of subscribers,
// by subscriber and field.
func customValues(ctx context.Context, q db.Querier, rows []db.ListListSubscribersForExportRow, fields int) (map[string]map[string]string, error) {
	out := map[string]map[string]string{}
	if fields == 0 || len(rows) == 0 {
		return out, nil
	}
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	values, err := q.ListCustomFieldValuesForSubscribers(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, v := range values {
		if out[v.SubscriberID] == nil {
			out[v.SubscriberID] = map[string]string{}
		}
		out[v.SubscriberID][v.FieldID] = v.Value.String
	}
	return out, nil
}

// safeRow keeps a spreadsheet from reading a cell as a formula: a cell that
// starts with =, +, -, @, a tab or a carriage return gets a leading '.
func safeRow(record []string) []string {
	for i, cell := range record {
		if cell != "" && strings.ContainsRune("=+-@\t\r", rune(cell[0])) {
			record[i] = "'" + cell
		}
	}
	return record
}

func encodeCursor(subscribedAt, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(subscribedAt + "\x00" + id))
}

func decodeCursor(cursor string) (string, string, error) {
	if cursor == "" {
		return "", "", nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return "", "", ErrBadCursor
	}
	at, id, ok := strings.Cut(string(raw), "\x00")
	if !ok {
		return "", "", ErrBadCursor
	}
	return at, id, nil
}

// exportRequest is what an export download reads from its request.
type exportRequest struct {
	ID     string `path:"id,optional"`
	Slug   string `path:"slug,optional"`
	Status string `form:"status,optional"` // active, pending, unsubscribed or all (default)
}

// AdminHandler serves GET /api/admin/lists/:id/export behind the admin auth
// middleware: the list's CSV, to anyone who may read the list's brand.
func AdminHandler(q db.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req exportRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, errorx.NewBadRequestError(err.Error()))
			return
		}
		if _, err := access.RequireOwner(r.Context(), q, access.List, req.ID, access.Member); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		listID, _ := strconv.ParseInt(req.ID, 10, 64) // RequireOwner found the list by it
		list, err := q.GetEmailList(r.Context(), listID)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, errorx.NewNotFoundError("List not found."))
			return
		}
		serve(w, r, q, list, req.Status)
	}
}

// SDKHandler serves GET /sdk/v1/lists/:slug/export behind the API key
// middleware: the CSV of the key's brand's list.
func SDKHandler(q db.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req exportRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, errorx.NewBadRequestError(err.Error()))
			return
		}
		orgID, _ := r.Context().Value(middleware.OrgIDKey).(string)
		if orgID == "" {
			httpx.ErrorCtx(r.Context(), w, errorx.NewUnauthorizedError("API key required."))
			return
		}
		list, err := q.GetEmailListByOrgAndSlug(r.Context(), db.GetEmailListByOrgAndSlugParams{OrgID: orgID, Slug: req.Slug})
		if errors.Is(err, sql.ErrNoRows) {
			httpx.ErrorCtx(r.Context(), w, errorx.NewNotFoundError("List not found."))
			return
		}
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		serve(w, r, q, list, req.Status)
	}
}

// serve streams a list's CSV as a download.
func serve(w http.ResponseWriter, r *http.Request, q db.Querier, list db.EmailList, status string) {
	name := fmt.Sprintf("%s-subscribers-%s.csv", list.Slug, time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	w.Header().Set("Cache-Control", "no-store")
	if err := Unbounded(w); err != nil {
		fmt.Printf("[List Export] list %d: %v\n", list.ID, err)
	}
	cw := &countingWriter{ResponseWriter: w}
	_, err := Write(r.Context(), q, cw, list.ID, Options{Status: status})
	if err == nil || r.Context().Err() != nil {
		return
	}
	if cw.n == 0 {
		w.Header().Del("Content-Disposition")
		httpx.ErrorCtx(r.Context(), w, err)
		return
	}
	// The download has started, so its status can't change; the file is
	// cut off
	fmt.Printf("[List Export] list %d: %v\n", list.ID, err)
}

// countingWriter counts what has gone out, so an export that fails before
// its first byte can still answer with an error.
type countingWriter struct {
	http.ResponseWriter
	n int
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.ResponseWriter.Write(p)
	c.n += n
	return n, err
}

func (c *countingWriter) Flush() {
	if f, ok := c.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unbounded lifts the server's write deadline for this one response, so a
// download of any size finishes: a download has no time limit. Its route must
// also have no go-zero timeout (rest.WithTimeout(0)), which would otherwise
// buffer the whole response and cut it off.
func Unbounded(w http.ResponseWriter) error {
	return http.NewResponseController(w).SetWriteDeadline(time.Time{})
}

// Unwrap lets http.ResponseController reach the connection.
func (c *countingWriter) Unwrap() http.ResponseWriter { return c.ResponseWriter }
