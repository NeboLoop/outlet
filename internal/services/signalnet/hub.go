package signalnet

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/outlet-sh/outlet/internal/db"
)

// ErrUnauthorized is returned by Hub.Authenticate for a missing, unknown or revoked key.
var ErrUnauthorized = errors.New("invalid instance key")

// Hub is the server side of the network, run by outlet.sh (SignalNet.Hub = true).
// It pools reported outcomes per hash and meters lookups and reports per instance key.
type Hub struct {
	store *db.Store
}

// NewHub returns a hub backed by store.
func NewHub(store *db.Store) *Hub { return &Hub{store: store} }

func keyHash(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// CreateInstance issues a new instance key. The key is returned once; only its hash is stored.
func (h *Hub) CreateInstance(ctx context.Context, name string) (db.NetworkInstance, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return db.NetworkInstance{}, "", err
	}
	key := "osn_" + hex.EncodeToString(b)
	inst, err := h.store.CreateNetworkInstance(ctx, db.CreateNetworkInstanceParams{
		ID: uuid.NewString(), Name: name, KeyHash: keyHash(key),
	})
	return inst, key, err
}

// Authenticate returns the active instance for key.
func (h *Hub) Authenticate(ctx context.Context, key string) (db.NetworkInstance, error) {
	if key == "" {
		return db.NetworkInstance{}, ErrUnauthorized
	}
	kh := keyHash(key)
	inst, err := h.store.GetNetworkInstanceByKeyHash(ctx, kh)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && subtle.ConstantTimeCompare([]byte(inst.KeyHash), []byte(kh)) != 1) {
		return db.NetworkInstance{}, ErrUnauthorized
	}
	return inst, err
}

// Lookup returns the pooled signals for hashes the network knows; unknown hashes are
// omitted. Every requested hash is metered against the instance.
func (h *Hub) Lookup(ctx context.Context, instanceID string, hashes []string) ([]Signal, error) {
	rows, err := h.store.GetNetworkSignals(ctx, hashes)
	if err != nil {
		return nil, err
	}
	if err := h.store.MeterNetworkInstance(ctx, db.MeterNetworkInstanceParams{ID: instanceID, Lookups: int64(len(hashes))}); err != nil {
		return nil, err
	}
	signals := make([]Signal, 0, len(rows))
	for _, r := range rows {
		s := Signal{
			Hash:        r.Hash,
			Senders:     int(r.Senders),
			Delivered:   int(r.Delivered),
			HardBounces: int(r.HardBounces),
			Complaints:  int(r.Complaints),
			Engaged:     int(r.Engaged),
		}
		s.LastEngagedAt, _ = time.Parse(time.RFC3339, r.LastEngagedAt.String)
		s.TrapScore, s.Toxic = score(r)
		signals = append(signals, s)
	}
	return signals, nil
}

// Report records events from one instance and returns how many were accepted.
// Events with a malformed hash or unknown kind are skipped; times in the future or
// missing are clamped to now.
func (h *Hub) Report(ctx context.Context, instanceID string, events []Event) (int, error) {
	now := time.Now().UTC()
	accepted := 0
	err := h.store.ExecTx(ctx, func(q *db.Queries) error {
		for _, e := range events {
			p, ok := eventParams(e, now)
			if !ok {
				continue
			}
			if err := q.RecordNetworkEvent(ctx, p); err != nil {
				return err
			}
			if err := q.RecordNetworkSender(ctx, db.RecordNetworkSenderParams{
				Hash: e.Hash, InstanceID: instanceID, Complained: p.Complaints,
			}); err != nil {
				return err
			}
			accepted++
		}
		return q.MeterNetworkInstance(ctx, db.MeterNetworkInstanceParams{ID: instanceID, Reports: int64(accepted)})
	})
	if err != nil {
		return 0, err
	}
	return accepted, nil
}

func eventParams(e Event, now time.Time) (db.RecordNetworkEventParams, bool) {
	if !ValidHash(e.Hash) {
		return db.RecordNetworkEventParams{}, false
	}
	at := e.At.UTC()
	if at.IsZero() || at.After(now) {
		at = now
	}
	ts := at.Format(time.RFC3339)
	when := sql.NullString{String: ts, Valid: true}
	p := db.RecordNetworkEventParams{Hash: e.Hash, At: ts}
	switch e.Kind {
	case KindDelivered:
		p.Delivered, p.LastDeliveredAt = 1, when
	case KindHardBounce:
		p.HardBounces, p.LastHardBounceAt = 1, when
	case KindComplaint:
		p.Complaints = 1
	case KindOpen, KindClick:
		p.Engaged, p.LastEngagedAt = 1, when
	default:
		return db.RecordNetworkEventParams{}, false
	}
	return p, true
}

// score computes the pooled verdict for one address: TrapScore and Toxic.
//
// ponytail: fixed thresholds over lifetime counters, no time decay and no learned model.
// Ceiling: senders that never track opens (tracking off, privacy proxies) make real
// people look unengaged, and machine opens (Apple MPP, link scanners) make traps look
// engaged. Replace with a model trained on confirmed traps once labelled data exists.
func score(r db.GetNetworkSignalsRow) (trap float64, toxic bool) {
	// Toxic: complaints from at least two distinct senders, so one sender's bad
	// list or one angry recipient is not enough to poison the address network-wide.
	toxic = r.ComplaintSenders >= 2

	// Recycled trap: the mailbox hard-bounced (abandoned), later accepted mail again
	// (reactivated by the provider as a trap) and nobody saw engagement since, across
	// at least two senders. RFC 3339 UTC strings compare correctly as text.
	if r.HardBounces > 0 && r.Senders >= 2 &&
		r.LastDeliveredAt.String > r.LastHardBounceAt.String &&
		r.LastEngagedAt.String <= r.LastHardBounceAt.String {
		return 0.9, toxic
	}
	// Pristine-trap pattern: widely mailed, never opened, clicked or complained about.
	if r.Senders >= 3 && r.Delivered >= 10 && r.Engaged == 0 && r.Complaints == 0 {
		return 0.6, toxic
	}
	return 0, toxic
}
