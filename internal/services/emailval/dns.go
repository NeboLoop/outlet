package emailval

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"strings"
	"time"
)

// dnsResolver is the subset of *net.Resolver the engine uses; swapped in tests.
type dnsResolver interface {
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

var resolver dnsResolver = net.DefaultResolver

// domainDNSCache holds definitive per-domain DNS answers; transient failures are never cached.
var domainDNSCache = newTTLCache[domainDNS](24 * time.Hour)

// maxMXHostsResolved caps the A/AAAA lookups done for the unroutable check.
const maxMXHostsResolved = 5

// domainDNS is the mail routing picture of one domain.
type domainDNS struct {
	// Hosts are mail hosts in preference order, without trailing dot.
	// With no MX records this is the domain itself (RFC 5321 implicit MX) when it has A/AAAA.
	Hosts []string
	// MXFound is true when the domain publishes real MX records.
	MXFound bool
	// NullMX is true for a single "." MX (RFC 7505): the domain accepts no mail.
	NullMX bool
	// Unroutable is true when every resolved mail host address is private/loopback/reserved.
	Unroutable bool
}

// lookupDomainDNS resolves the mail routing for domain, using the 24h cache.
// A non-nil error means DNS gave no definitive answer (timeout, SERVFAIL).
func lookupDomainDNS(ctx context.Context, domain string) (domainDNS, error) {
	if d, ok := domainDNSCache.get(domain); ok {
		return d, nil
	}

	var d domainDNS
	mxs, err := resolver.LookupMX(ctx, domain)
	switch {
	case err != nil && !isNotFound(err):
		return d, err
	case len(mxs) == 1 && (mxs[0].Host == "." || mxs[0].Host == ""):
		d.NullMX = true
	case len(mxs) > 0:
		d.MXFound = true
		for _, mx := range mxs {
			if h := strings.TrimSuffix(strings.ToLower(mx.Host), "."); h != "" {
				d.Hosts = append(d.Hosts, h)
			}
		}
	default:
		// No MX: fall back to A/AAAA (implicit MX).
		addrs, err := resolver.LookupIPAddr(ctx, domain)
		if err != nil && !isNotFound(err) {
			return d, err
		}
		if len(addrs) > 0 {
			d.Hosts = []string{domain}
			d.Unroutable = !anyRoutable(addrs)
		}
		domainDNSCache.set(domain, d)
		return d, nil
	}

	if len(d.Hosts) > 0 {
		var addrs []net.IPAddr
		for i, h := range d.Hosts {
			if i == maxMXHostsResolved {
				break
			}
			a, err := resolver.LookupIPAddr(ctx, h)
			if err != nil && !isNotFound(err) {
				if ctx.Err() != nil {
					return d, ctx.Err()
				}
				continue
			}
			addrs = append(addrs, a...)
		}
		// Hosts that resolve to nothing are left to the SMTP probe; only flag when addresses exist and none route.
		d.Unroutable = len(addrs) > 0 && !anyRoutable(addrs)
	}
	domainDNSCache.set(domain, d)
	return d, nil
}

func isNotFound(err error) bool {
	var dnsErr *net.DNSError
	return errors.As(err, &dnsErr) && dnsErr.IsNotFound
}

// reservedPrefixes are special-purpose ranges not covered by net.IP helpers (RFC 6890).
var reservedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("100::/64"),
}

// isRoutableIP reports whether ip is a public unicast address a mail server could live on.
func isRoutableIP(ip net.IP) bool {
	if ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() ||
		ip.IsMulticast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() {
		return false
	}
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	addr = addr.Unmap()
	for _, p := range reservedPrefixes {
		if p.Contains(addr) {
			return false
		}
	}
	return true
}

func anyRoutable(addrs []net.IPAddr) bool {
	for _, a := range addrs {
		if isRoutableIP(a.IP) {
			return true
		}
	}
	return false
}

// checkMX looks up MX records for a domain.
// Returns true if at least one MX record exists.
func checkMX(ctx context.Context, domain string) (bool, error) {
	mxRecords, err := resolver.LookupMX(ctx, domain)
	if err != nil {
		// Check if it's a "no such host" error vs network error.
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return len(mxRecords) > 0, nil
}

// getMXHosts returns MX hosts for a domain, sorted by preference.
func getMXHosts(ctx context.Context, domain string) ([]string, error) {
	mxRecords, err := resolver.LookupMX(ctx, domain)
	if err != nil {
		return nil, err
	}

	hosts := make([]string, 0, len(mxRecords))
	for _, mx := range mxRecords {
		hosts = append(hosts, mx.Host)
	}

	return hosts, nil
}
