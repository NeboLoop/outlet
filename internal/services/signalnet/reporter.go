package signalnet

import (
	"context"
	"sync"
	"time"

	"github.com/outlet-sh/outlet/internal/events"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	flushInterval = 30 * time.Second
	flushSize     = 500
	bufferMax     = 50000 // oldest events are dropped beyond this while the hub is unreachable
)

// Reporter turns internal email outcome events into hashed network events and
// sends them to the hub in the background. A nil *Reporter is valid and does nothing.
type Reporter struct {
	client *Client
	subs   []events.Subscription
	kick   chan struct{}
	stop   chan struct{}
	wg     sync.WaitGroup

	mu  sync.Mutex
	buf []Event
}

// StartReporter subscribes to the event bus and reports outcomes to the hub.
// Returns nil when the client is disabled (no HubURL or InstanceKey).
func StartReporter(c *Client, bus *events.Subject) *Reporter {
	if !c.enabled() || bus == nil {
		return nil
	}
	r := &Reporter{client: c, kick: make(chan struct{}, 1), stop: make(chan struct{})}
	for topic, kind := range map[string]string{
		events.TopicEmailDelivered:  KindDelivered,
		events.TopicEmailBounced:    KindHardBounce,
		events.TopicEmailComplained: KindComplaint,
		events.TopicEmailOpened:     KindOpen,
		events.TopicEmailClicked:    KindClick,
	} {
		r.subs = append(r.subs, events.Subscribe[events.EmailEvent](bus, topic, func(_ context.Context, e events.EmailEvent) error {
			if e.Email == "" || (kind == KindHardBounce && e.BounceType != "hard") {
				return nil // soft bounces say nothing about the address
			}
			r.add(Event{Hash: Hash(e.Email), Kind: kind, At: e.Timestamp})
			return nil
		}))
	}
	r.wg.Add(1)
	go r.run()
	return r
}

// Stop unsubscribes, flushes what is buffered once, and waits for the reporter to exit.
func (r *Reporter) Stop() {
	if r == nil {
		return
	}
	for _, s := range r.subs {
		s.Unsubscribe()
	}
	close(r.stop)
	r.wg.Wait()
}

func (r *Reporter) add(e Event) {
	if e.At.IsZero() {
		e.At = time.Now()
	}
	r.mu.Lock()
	r.buf = append(r.buf, e)
	if over := len(r.buf) - bufferMax; over > 0 {
		r.buf = r.buf[over:]
	}
	n := len(r.buf)
	r.mu.Unlock()
	if n >= flushSize {
		select {
		case r.kick <- struct{}{}:
		default:
		}
	}
}

func (r *Reporter) run() {
	defer r.wg.Done()
	t := time.NewTicker(flushInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			r.flush()
		case <-r.kick:
			r.flush()
		case <-r.stop:
			r.flush()
			return
		}
	}
}

// flush sends the buffer; on failure the events go back to the front of the buffer
// for the next flush (bounded by bufferMax, oldest dropped).
func (r *Reporter) flush() {
	r.mu.Lock()
	batch := r.buf
	r.buf = nil
	r.mu.Unlock()
	if len(batch) == 0 {
		return
	}
	// Report sends in chunks; a failure part-way resends earlier chunks on retry.
	// The hub counts them twice: acceptable for counters that only feed thresholds.
	if err := r.client.Report(context.Background(), batch); err != nil {
		logx.Errorf("signalnet: report of %d events failed, will retry: %v", len(batch), err)
		r.mu.Lock()
		r.buf = append(batch, r.buf...)
		if over := len(r.buf) - bufferMax; over > 0 {
			r.buf = r.buf[over:]
		}
		r.mu.Unlock()
	}
}
