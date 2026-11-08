package emailval

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// rdapBaseURL is the RDAP bootstrap redirector; swapped in tests.
var rdapBaseURL = "https://rdap.org/domain/"

// httpClient fetches RDAP and the disposable list; redirects are followed by default.
var httpClient = &http.Client{}

// rdapCache holds registration dates per registrable domain. Failures are not cached.
var rdapCache = newTTLCache[time.Time](7 * 24 * time.Hour)

// domainAgeDays returns how many days ago domain's registrable domain was registered, 0 when unknown.
func domainAgeDays(ctx context.Context, domain string) int {
	reg := registrableDomain(domain)
	created, ok := rdapCache.get(reg)
	if !ok {
		var err error
		created, err = fetchRegistrationDate(ctx, reg)
		if err != nil {
			return 0
		}
		rdapCache.set(reg, created)
	}
	days := int(time.Since(created).Hours() / 24)
	return max(days, 0)
}

func fetchRegistrationDate(ctx context.Context, domain string) (time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rdapBaseURL+domain, nil)
	if err != nil {
		return time.Time{}, err
	}
	req.Header.Set("Accept", "application/rdap+json, application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("rdap %s: HTTP %d", domain, resp.StatusCode)
	}
	var body struct {
		Events []struct {
			Action string `json:"eventAction"`
			Date   string `json:"eventDate"`
		} `json:"events"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return time.Time{}, err
	}
	for _, e := range body.Events {
		if e.Action == "registration" {
			return time.Parse(time.RFC3339, e.Date)
		}
	}
	return time.Time{}, fmt.Errorf("rdap %s: no registration event", domain)
}

// multiPartSuffixes are public suffixes with two labels, so registration happens one level deeper.
var multiPartSuffixes = toSet(
	"co.uk", "org.uk", "me.uk", "ltd.uk", "plc.uk", "net.uk", "ac.uk", "gov.uk",
	"com.au", "net.au", "org.au", "edu.au", "gov.au", "asn.au", "id.au",
	"co.nz", "net.nz", "org.nz", "co.za", "org.za", "co.jp", "ne.jp", "or.jp", "ac.jp",
	"co.in", "net.in", "org.in", "firm.in", "co.kr", "or.kr", "co.il", "org.il", "co.id", "co.th",
	"com.br", "net.br", "org.br", "com.mx", "org.mx", "com.ar", "com.co", "com.pe", "com.ve", "com.ec",
	"com.cn", "net.cn", "org.cn", "com.hk", "com.tw", "com.sg", "com.my", "com.ph", "com.vn", "com.pk",
	"com.tr", "com.ua", "com.pl", "com.ru", "com.eg", "com.sa", "com.ng", "com.gh", "co.ke",
)

// registrableDomain reduces a host to the domain a registrant owns (mail.example.co.uk -> example.co.uk).
// ponytail: last-two-labels plus a small multi-part suffix list, not the full Public Suffix List.
func registrableDomain(domain string) string {
	labels := strings.Split(domain, ".")
	n := 2
	if len(labels) >= 3 && multiPartSuffixes[strings.Join(labels[len(labels)-2:], ".")] {
		n = 3
	}
	if len(labels) <= n {
		return domain
	}
	return strings.Join(labels[len(labels)-n:], ".")
}
