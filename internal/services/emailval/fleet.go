package emailval

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"net/url"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/net/proxy"
)

const (
	// proxyQuarantine is how long a proxy is benched after a block signal.
	proxyQuarantine = time.Hour
	// proxyFailureThreshold is the number of consecutive connection failures that count as a block.
	proxyFailureThreshold = 3
)

// SOCKS5Dialer dials SMTP servers through a SOCKS5 proxy on a probe host.
type SOCKS5Dialer struct {
	// ProxyURL is socks5://user:pass@host:port.
	ProxyURL string
	Timeout  time.Duration
}

// DialSMTP connects to an SMTP server through the proxy.
func (d SOCKS5Dialer) DialSMTP(ctx context.Context, address string) (*textproto.Conn, error) {
	u, err := url.Parse(d.ProxyURL)
	if err != nil {
		return nil, fmt.Errorf("parse proxy url: %w", err)
	}
	timeout := d.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	pd, err := proxy.FromURL(u, &net.Dialer{Timeout: timeout})
	if err != nil {
		return nil, err
	}
	cd, ok := pd.(proxy.ContextDialer)
	if !ok {
		return nil, errors.New("proxy dialer does not support context")
	}
	conn, err := cd.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	return textproto.NewConn(conn), nil
}

// probeExit is the network path a probe leaves from.
type probeExit struct {
	key    string // rate-limit and health key: proxy URL, "direct" or "dialer"
	proxy  bool   // key is a proxy URL subject to health tracking
	dialer SMTPDialer
}

// pickExit chooses how the next probe leaves this host. Options.SMTPDialer overrides
// everything; otherwise healthy proxies rotate round-robin, and no proxies means a direct dial.
// ok is false when every proxy is quarantined.
func pickExit(opts *Options) (exit probeExit, ok bool) {
	switch {
	case opts.SMTPDialer != nil:
		return probeExit{key: "dialer", dialer: opts.SMTPDialer}, true
	case len(opts.Proxies) == 0:
		return probeExit{key: "direct", dialer: DefaultSMTPDialer{Timeout: opts.Timeout}}, true
	}
	u, ok := fleetHealth.next(opts.Proxies)
	if !ok {
		return probeExit{}, false
	}
	return probeExit{key: u, proxy: true, dialer: SOCKS5Dialer{ProxyURL: u, Timeout: opts.Timeout}}, true
}

// fleetHealth tracks block signals per proxy URL across all probes in this process.
var fleetHealth = &proxyHealth{}

type proxyHealth struct {
	mu          sync.Mutex
	rr          int
	failures    map[string]int
	quarantined map[string]time.Time // proxy URL -> quarantine end
}

// next returns the next healthy proxy in round-robin order.
func (h *proxyHealth) next(proxies []string) (string, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	for i := 0; i < len(proxies); i++ {
		u := proxies[(h.rr+i)%len(proxies)]
		if until, q := h.quarantined[u]; q && now.Before(until) {
			continue
		}
		h.rr = (h.rr + i + 1) % len(proxies)
		return u, true
	}
	return "", false
}

// report records a session outcome for a proxy, quarantining it on block signals.
func (h *proxyHealth) report(proxyURL string, out probeOutcome) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.failures == nil {
		h.failures = make(map[string]int)
		h.quarantined = make(map[string]time.Time)
	}
	switch {
	case out.blocked:
	case out.connFailed:
		h.failures[proxyURL]++
		if h.failures[proxyURL] < proxyFailureThreshold {
			return
		}
	default:
		h.failures[proxyURL] = 0
		return
	}
	h.failures[proxyURL] = 0
	h.quarantined[proxyURL] = time.Now().Add(proxyQuarantine)
	logx.Errorf("emailval: quarantining probe proxy %s for %s (blocked=%v, sub_status=%s)",
		redactProxy(proxyURL), proxyQuarantine, out.blocked, out.result.SubStatus)
}

// redactProxy hides proxy credentials for logs.
func redactProxy(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "invalid-proxy-url"
	}
	return u.Redacted()
}
