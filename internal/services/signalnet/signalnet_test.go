package signalnet

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"

	_ "modernc.org/sqlite"
)

func TestHashNormalizes(t *testing.T) {
	want := Hash("alice@example.com")
	for _, in := range []string{"Alice@Example.COM", "  alice@example.com\n", "\tALICE@EXAMPLE.COM "} {
		if got := Hash(in); got != want {
			t.Errorf("Hash(%q) = %s, want %s", in, got, want)
		}
	}
	// sha256("alice@example.com")
	if want != "ff8d9819fc0e12bf0d24892e45987e249a28dce836a85cad60e28eaaa8c6d976" {
		t.Errorf("unexpected hash %s", want)
	}
	if !ValidHash(want) {
		t.Error("ValidHash rejected a Hash")
	}
	for _, bad := range []string{"", "abc", want[:63], want + "0", "FF8D9819FC0E12BF0D24892E45987E249A28DCE836A85CAD60E28EAAA8C6D976", want[:63] + "g"} {
		if ValidHash(bad) {
			t.Errorf("ValidHash(%q) = true", bad)
		}
	}
}

func ns(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

func TestScore(t *testing.T) {
	const t1, t2, t3 = "2026-01-01T00:00:00Z", "2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z"
	tests := []struct {
		name  string
		row   db.GetNetworkSignalsRow
		trap  float64
		toxic bool
	}{
		{"unknown", db.GetNetworkSignalsRow{}, 0, false},
		{"recycled trap", db.GetNetworkSignalsRow{HardBounces: 1, Delivered: 2, Senders: 2,
			LastHardBounceAt: ns(t1), LastDeliveredAt: ns(t2)}, 0.9, false},
		{"recycled but engaged before abandonment", db.GetNetworkSignalsRow{HardBounces: 1, Delivered: 2, Senders: 2, Engaged: 5,
			LastEngagedAt: ns("2025-06-01T00:00:00Z"), LastHardBounceAt: ns(t1), LastDeliveredAt: ns(t2)}, 0.9, false},
		{"recycled but engaged after", db.GetNetworkSignalsRow{HardBounces: 1, Delivered: 2, Senders: 2, Engaged: 1,
			LastEngagedAt: ns(t3), LastHardBounceAt: ns(t1), LastDeliveredAt: ns(t2)}, 0, false},
		{"recycled single sender", db.GetNetworkSignalsRow{HardBounces: 1, Delivered: 2, Senders: 1,
			LastHardBounceAt: ns(t1), LastDeliveredAt: ns(t2)}, 0, false},
		{"bounced after delivery", db.GetNetworkSignalsRow{HardBounces: 1, Delivered: 2, Senders: 3,
			LastHardBounceAt: ns(t2), LastDeliveredAt: ns(t1)}, 0, false},
		{"pristine pattern", db.GetNetworkSignalsRow{Delivered: 10, Senders: 3}, 0.6, false},
		{"pristine too few deliveries", db.GetNetworkSignalsRow{Delivered: 9, Senders: 3}, 0, false},
		{"pristine with complaint", db.GetNetworkSignalsRow{Delivered: 10, Senders: 3, Complaints: 1, ComplaintSenders: 1}, 0, false},
		{"engaged", db.GetNetworkSignalsRow{Delivered: 50, Senders: 5, Engaged: 1}, 0, false},
		{"toxic", db.GetNetworkSignalsRow{Delivered: 4, Senders: 2, Engaged: 3, Complaints: 2, ComplaintSenders: 2}, 0, true},
		{"one complainer", db.GetNetworkSignalsRow{Delivered: 4, Senders: 2, Complaints: 3, ComplaintSenders: 1}, 0, false},
	}
	for _, tt := range tests {
		trap, toxic := score(tt.row)
		if trap != tt.trap || toxic != tt.toxic {
			t.Errorf("%s: score = (%v, %v), want (%v, %v)", tt.name, trap, toxic, tt.trap, tt.toxic)
		}
	}
}

func TestClientDisabled(t *testing.T) {
	var nilClient *Client
	if s, err := nilClient.Lookup(context.Background(), "a@b.c"); s != nil || err != nil {
		t.Fatalf("nil client Lookup = %v, %v", s, err)
	}
	if err := nilClient.Report(context.Background(), []Event{{Hash: Hash("a@b.c"), Kind: KindOpen}}); err != nil {
		t.Fatal(err)
	}
	if s, _ := (&Client{HubURL: "http://x"}).Lookup(context.Background(), "a@b.c"); s != nil {
		t.Fatal("client without key should be disabled")
	}
	if StartReporter(nil, nil) != nil {
		t.Fatal("reporter should be nil when disabled")
	}
	var r *Reporter
	r.Stop() // nil-safe
}

func TestClientFailOpen(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":500}`, http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := &Client{HubURL: srv.URL, InstanceKey: "k"}
	if s, err := c.Lookup(context.Background(), "a@b.c"); s != nil || err != nil {
		t.Fatalf("Lookup on 500 = %v, %v; want nil, nil", s, err)
	}
	if err := c.Report(context.Background(), []Event{{Hash: Hash("a@b.c"), Kind: KindOpen}}); err == nil {
		t.Fatal("Report on 500 should return an error so the reporter retries")
	}

	// Hub that hangs past the 2s timeout.
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer slow.Close()
	start := time.Now()
	if s, err := (&Client{HubURL: slow.URL, InstanceKey: "k"}).Lookup(context.Background(), "a@b.c"); s != nil || err != nil {
		t.Fatalf("Lookup on timeout = %v, %v", s, err)
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("Lookup took %v, want <= ~2s", d)
	}

	// Unreachable hub.
	if s, err := (&Client{HubURL: "http://127.0.0.1:1", InstanceKey: "k"}).Lookup(context.Background(), "a@b.c"); s != nil || err != nil {
		t.Fatalf("Lookup on unreachable = %v, %v", s, err)
	}
}

func TestClientLookupCaches(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("X-Instance-Key") != "k" || r.URL.Path != "/network/v1/lookup" {
			t.Errorf("bad request: key=%q path=%q", r.Header.Get("X-Instance-Key"), r.URL.Path)
		}
		var req struct{ Hashes []string }
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(map[string]any{"signals": []Signal{{Hash: req.Hashes[0], Senders: 3, TrapScore: 0.6}}})
	}))
	defer srv.Close()
	c := &Client{HubURL: srv.URL + "/", InstanceKey: "k"}
	for i := 0; i < 3; i++ {
		s, err := c.Lookup(context.Background(), "Bob@Example.com")
		if err != nil || s == nil || s.TrapScore != 0.6 || s.Hash != Hash("bob@example.com") {
			t.Fatalf("Lookup = %+v, %v", s, err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("hub called %d times, want 1 (cached)", calls.Load())
	}
}

func newTestHub(t *testing.T) *Hub {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "hub.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := migrations.Run(conn); err != nil {
		t.Fatal(err)
	}
	return NewHub(db.NewStore(conn))
}

func TestHubRoundTrip(t *testing.T) {
	ctx := context.Background()
	hub := newTestHub(t)

	a, keyA, err := hub.CreateInstance(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	b, keyB, _ := hub.CreateInstance(ctx, "b")
	if a.KeyHash == keyA || len(keyA) < 32 {
		t.Fatal("key must be stored hashed")
	}
	if got, err := hub.Authenticate(ctx, keyA); err != nil || got.ID != a.ID {
		t.Fatalf("Authenticate = %v, %v", got.ID, err)
	}
	if _, err := hub.Authenticate(ctx, "nope"); err != ErrUnauthorized {
		t.Fatalf("bad key err = %v", err)
	}

	trap, toxic := Hash("trap@example.com"), Hash("angry@example.com")
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	n, err := hub.Report(ctx, a.ID, []Event{
		{Hash: trap, Kind: KindHardBounce, At: t0},
		{Hash: toxic, Kind: KindDelivered, At: t0},
		{Hash: toxic, Kind: KindComplaint, At: t0},
		{Hash: "not-a-hash", Kind: KindDelivered},
		{Hash: trap, Kind: "bogus"},
	})
	if err != nil || n != 3 {
		t.Fatalf("Report = %d, %v; want 3 accepted", n, err)
	}
	if _, err := hub.Report(ctx, b.ID, []Event{
		{Hash: trap, Kind: KindDelivered, At: t0.Add(24 * time.Hour)},
		{Hash: toxic, Kind: KindOpen, At: t0},
		{Hash: toxic, Kind: KindComplaint, At: t0},
	}); err != nil {
		t.Fatal(err)
	}

	unknown := Hash("nobody@example.com")
	signals, err := hub.Lookup(ctx, b.ID, []string{trap, toxic, unknown})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]Signal{}
	for _, s := range signals {
		got[s.Hash] = s
	}
	if len(got) != 2 {
		t.Fatalf("got %d signals, want 2 (unknown omitted)", len(got))
	}
	if s := got[trap]; s.Senders != 2 || s.HardBounces != 1 || s.Delivered != 1 || s.TrapScore != 0.9 || s.Toxic {
		t.Errorf("trap signal = %+v", s)
	}
	if s := got[toxic]; s.Senders != 2 || s.Complaints != 2 || s.Engaged != 1 || !s.Toxic || !s.LastEngagedAt.Equal(t0) {
		t.Errorf("toxic signal = %+v", s)
	}

	insts, _ := hub.store.ListNetworkInstances(ctx)
	for _, i := range insts {
		switch i.ID {
		case a.ID:
			if i.ReportsCount != 3 || i.LookupsCount != 0 {
				t.Errorf("a metering = %d reports, %d lookups", i.ReportsCount, i.LookupsCount)
			}
		case b.ID:
			if i.ReportsCount != 3 || i.LookupsCount != 3 || !i.LastSeenAt.Valid {
				t.Errorf("b metering = %+v", i)
			}
		}
	}

	if _, err := hub.store.RevokeNetworkInstance(ctx, b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := hub.Authenticate(ctx, keyB); err != ErrUnauthorized {
		t.Fatalf("revoked key err = %v", err)
	}
}

// TestClientAgainstHub wires a real Client to a hub served over HTTP.
func TestClientAgainstHub(t *testing.T) {
	ctx := context.Background()
	hub := newTestHub(t)
	inst, key, _ := hub.CreateInstance(ctx, "c")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := hub.Authenticate(r.Context(), r.Header.Get("X-Instance-Key")); err != nil {
			http.Error(w, "{}", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/network/v1/report":
			var req struct{ Events []Event }
			_ = json.NewDecoder(r.Body).Decode(&req)
			n, err := hub.Report(r.Context(), inst.ID, req.Events)
			if err != nil {
				http.Error(w, "{}", http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]int{"accepted": n})
		case "/network/v1/lookup":
			var req struct{ Hashes []string }
			_ = json.NewDecoder(r.Body).Decode(&req)
			s, _ := hub.Lookup(r.Context(), inst.ID, req.Hashes)
			_ = json.NewEncoder(w).Encode(map[string]any{"signals": s})
		}
	}))
	defer srv.Close()

	c := &Client{HubURL: srv.URL, InstanceKey: key}
	if err := c.Report(ctx, []Event{{Hash: Hash("x@y.z"), Kind: KindClick, At: time.Now()}}); err != nil {
		t.Fatal(err)
	}
	s, _ := c.Lookup(ctx, "X@Y.Z")
	if s == nil || s.Engaged != 1 || s.Senders != 1 {
		t.Fatalf("Lookup = %+v", s)
	}
	if err := (&Client{HubURL: srv.URL, InstanceKey: "wrong"}).Report(ctx, []Event{{Hash: Hash("x@y.z"), Kind: KindOpen}}); err == nil {
		t.Fatal("wrong key should fail")
	}
}
