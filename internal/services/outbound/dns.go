package outbound

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"

	"github.com/google/uuid"
)

// Resolver is the part of DNS outbound reads. net.DefaultResolver in
// production, a fake in tests.
type Resolver interface {
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
	LookupTXT(ctx context.Context, name string) ([]string, error)
	LookupCNAME(ctx context.Context, name string) (string, error)
}

type netResolver struct{}

func (netResolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	return net.DefaultResolver.LookupMX(ctx, name)
}

func (netResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	return net.DefaultResolver.LookupTXT(ctx, name)
}

func (netResolver) LookupCNAME(ctx context.Context, name string) (string, error) {
	return net.DefaultResolver.LookupCNAME(ctx, name)
}

// DNSCheck is what one domain's mail records look like from public DNS.
type DNSCheck struct {
	MX      bool     `json:"mx"`
	SPF     bool     `json:"spf"`
	DKIM    bool     `json:"dkim"`
	DMARC   bool     `json:"dmarc"`
	Missing []string `json:"missing,omitempty"`
	// Tracking is whether the click-tracking subdomain points at this server.
	// It never pauses sending; campaigns with click tracking need it.
	Tracking *bool `json:"tracking,omitempty"`
}

func (c DNSCheck) OK() bool { return len(c.Missing) == 0 }

func (s *Service) checkDNS(ctx context.Context, d db.OutboundDomain) DNSCheck {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var c DNSCheck
	if mx, err := s.dns.LookupMX(ctx, d.Domain); err == nil && len(mx) > 0 {
		c.MX = true
	}
	c.SPF = hasTXT(ctx, s.dns, d.Domain, "v=spf1")
	c.DMARC = hasTXT(ctx, s.dns, "_dmarc."+d.Domain, "v=dmarc1")
	c.DKIM = hasTXT(ctx, s.dns, d.DkimSelector+"._domainkey."+d.Domain, "v=dkim1") ||
		hasTXT(ctx, s.dns, d.DkimSelector+"._domainkey."+d.Domain, "p=")
	for name, ok := range map[string]bool{"MX": c.MX, "SPF": c.SPF, "DKIM (" + d.DkimSelector + "._domainkey)": c.DKIM, "DMARC": c.DMARC} {
		if !ok {
			c.Missing = append(c.Missing, name)
		}
	}
	sort.Strings(c.Missing)
	if d.TrackingHost != "" {
		cname, err := s.dns.LookupCNAME(ctx, d.TrackingHost)
		ok := err == nil && s.tracking != "" && normDomain(cname) == s.tracking
		c.Tracking = &ok
	}
	return c
}

func hasTXT(ctx context.Context, r Resolver, name, prefix string) bool {
	txts, err := r.LookupTXT(ctx, name)
	if err != nil {
		return false
	}
	for _, t := range txts {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(t)), prefix) {
			return true
		}
	}
	return false
}

// CheckDomain rechecks one domain and pauses or resumes its mailboxes.
func (s *Service) CheckDomain(ctx context.Context, d db.OutboundDomain) (DNSCheck, error) {
	c := s.checkDNS(ctx, d)
	raw, _ := json.Marshal(c)
	status := "ok"
	if !c.OK() {
		status = "failing"
	}
	if err := s.db.SetOutboundDomainCheck(ctx, db.SetOutboundDomainCheckParams{Status: status, CheckResult: string(raw), ID: d.ID}); err != nil {
		return c, err
	}
	if c.OK() {
		return c, s.db.ResumeOutboundMailboxesPausedForDNS(ctx, d.ID)
	}
	return c, s.db.PauseOutboundMailboxesOnDomain(ctx, db.PauseOutboundMailboxesOnDomainParams{
		PausedReason: "DNS: " + d.Domain + " is missing " + strings.Join(c.Missing, ", "),
		DomainID:     d.ID,
	})
}

// CheckAllDomains is the daily recheck.
func (s *Service) CheckAllDomains(ctx context.Context) {
	domains, err := s.db.ListAllOutboundDomains(ctx)
	if err != nil {
		return
	}
	for _, d := range domains {
		_, _ = s.CheckDomain(ctx, d)
	}
}

// DomainInput adds one outreach domain.
type DomainInput struct {
	Domain       string   `json:"domain"`
	DNSHost      string   `json:"dns_host,omitempty"`      // digitalocean or manual (default)
	MailProvider string   `json:"mail_provider,omitempty"` // google, or other
	MXHosts      []string `json:"mx_hosts,omitempty"`      // for other providers
	SPFInclude   string   `json:"spf_include,omitempty"`   // for other providers, e.g. _spf.mailforge.ai
	DKIMSelector string   `json:"dkim_selector,omitempty"` // default google
	DKIMValue    string   `json:"dkim_value,omitempty"`    // the TXT value from the mail provider's admin
	DMARCReport  string   `json:"dmarc_report,omitempty"`  // rua address, optional
	// ClickTracking sets up go.<domain>, a CNAME to this server, for tracked links.
	ClickTracking bool `json:"click_tracking,omitempty"`

	trackingTarget string // this server's host, filled by AddDomain
}

// DNSRecord is one record outbound wants on a domain.
type DNSRecord struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Data     string `json:"data"`
	Priority int    `json:"priority,omitempty"`
}

// wantedRecords lists the mail records a domain needs. DKIM is included only
// when its value is known; it comes from the mail provider's admin console.
func wantedRecords(in DomainInput) ([]DNSRecord, error) {
	var recs []DNSRecord
	spf := ""
	switch strings.ToLower(in.MailProvider) {
	case "", "google":
		recs = append(recs, DNSRecord{Type: "MX", Name: "@", Data: "smtp.google.com.", Priority: 1})
		spf = "v=spf1 include:_spf.google.com ~all"
	default:
		if len(in.MXHosts) == 0 || in.SPFInclude == "" {
			return nil, refuse("For mail provider %q, pass mx_hosts and spf_include (from the provider's setup page).", in.MailProvider)
		}
		for i, h := range in.MXHosts {
			recs = append(recs, DNSRecord{Type: "MX", Name: "@", Data: strings.TrimSuffix(h, ".") + ".", Priority: 10 * (i + 1)})
		}
		spf = "v=spf1 include:" + in.SPFInclude + " ~all"
	}
	recs = append(recs, DNSRecord{Type: "TXT", Name: "@", Data: spf})
	dmarc := "v=DMARC1; p=none"
	if in.DMARCReport != "" {
		dmarc += "; rua=mailto:" + in.DMARCReport
	}
	recs = append(recs, DNSRecord{Type: "TXT", Name: "_dmarc", Data: dmarc})
	if in.DKIMValue != "" {
		recs = append(recs, DNSRecord{Type: "TXT", Name: in.DKIMSelector + "._domainkey", Data: in.DKIMValue})
	}
	if in.ClickTracking {
		recs = append(recs, DNSRecord{Type: "CNAME", Name: "go", Data: in.trackingTarget + "."})
	}
	return recs, nil
}

// AddDomain registers an outreach domain, refusing the brand's own and
// protected domains. With dns_host digitalocean it writes the missing mail
// records; otherwise it returns them to paste.
func (s *Service) AddDomain(ctx context.Context, org db.Organization, in DomainInput) (db.OutboundDomain, []DNSRecord, DNSCheck, error) {
	in.Domain = normDomain(in.Domain)
	if in.DKIMSelector == "" {
		in.DKIMSelector = "google"
	}
	if in.DNSHost == "" {
		in.DNSHost = "manual"
	}
	if in.DNSHost != "manual" && in.DNSHost != "digitalocean" {
		return db.OutboundDomain{}, nil, DNSCheck{}, refuse("dns_host must be digitalocean or manual.")
	}
	if err := s.allowedDomain(ctx, org, in.Domain); err != nil {
		return db.OutboundDomain{}, nil, DNSCheck{}, err
	}
	trackingHost := ""
	if in.ClickTracking {
		if s.tracking == "" {
			return db.OutboundDomain{}, nil, DNSCheck{}, refuse("Click tracking needs this server's public host (APP_DOMAIN) in its config.")
		}
		in.trackingTarget, trackingHost = s.tracking, "go."+in.Domain
	}
	recs, err := wantedRecords(in)
	if err != nil {
		return db.OutboundDomain{}, nil, DNSCheck{}, err
	}
	d, err := s.db.GetOutboundDomainByName(ctx, in.Domain)
	switch {
	case err == nil && d.OrgID != org.ID:
		return db.OutboundDomain{}, nil, DNSCheck{}, refuse("%s is already an outreach domain of another brand.", in.Domain)
	case err != nil:
		d, err = s.db.CreateOutboundDomain(ctx, db.CreateOutboundDomainParams{
			ID: uuid.NewString(), OrgID: org.ID, Domain: in.Domain, DnsHost: in.DNSHost, DkimSelector: in.DKIMSelector,
			TrackingHost: trackingHost,
		})
		if err != nil {
			return d, nil, DNSCheck{}, err
		}
	}
	if trackingHost != "" && d.TrackingHost != trackingHost {
		if err := s.db.SetOutboundDomainTrackingHost(ctx, db.SetOutboundDomainTrackingHostParams{TrackingHost: trackingHost, ID: d.ID}); err != nil {
			return d, nil, DNSCheck{}, err
		}
		d.TrackingHost = trackingHost
	}
	var pending []DNSRecord
	if in.DNSHost == "digitalocean" {
		pending, err = s.writeDigitalOcean(ctx, org.ID, in.Domain, recs)
		if err != nil {
			return d, recs, DNSCheck{}, err
		}
	} else {
		pending = recs
	}
	c, err := s.CheckDomain(ctx, d)
	if err == nil {
		d, err = s.db.GetOutboundDomain(ctx, db.GetOutboundDomainParams{ID: d.ID, OrgID: org.ID})
	}
	return d, pending, c, err
}

// allowedDomain refuses protected domains, the brand's own sending domain
// and any domain the platform sends transactional mail from.
func (s *Service) allowedDomain(ctx context.Context, org db.Organization, domain string) error {
	if domain == "" || !strings.Contains(domain, ".") {
		return refuse("Pass a domain, like getexample.com.")
	}
	for _, p := range s.protected {
		if domain == p || strings.HasSuffix(domain, "."+p) {
			return refuse("%s is a protected company domain; outreach never uses it. Use a separate outreach domain.", domain)
		}
	}
	if org.FromEmail.Valid && domainOf(org.FromEmail.String) == domain {
		return refuse("%s is this brand's own sending domain; outreach must use a separate domain so its reputation never touches the brand's mail.", domain)
	}
	if names, err := s.db.ListTransactionalDomains(ctx); err == nil {
		for _, n := range names {
			if normDomain(n) == domain {
				return refuse("%s sends transactional mail on this server; outreach must use a separate domain.", domain)
			}
		}
	}
	return nil
}

// SetDNSToken stores the brand's DigitalOcean API token, encrypted. Give it
// a token limited to domain read and write.
func (s *Service) SetDNSToken(ctx context.Context, orgID, token string) error {
	if s.crypto == nil {
		return refuse("ENCRYPTION_KEY is not set on this server, so it can't store a token.")
	}
	enc, err := s.crypto.EncryptString(strings.TrimSpace(token))
	if err != nil {
		return err
	}
	return s.db.SetOutboundDNSToken(ctx, db.SetOutboundDNSTokenParams{OrgID: orgID, DnsTokenEnc: enc})
}

type doRecord struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Data     string `json:"data"`
	Priority *int   `json:"priority"`
	TTL      int    `json:"ttl,omitempty"`
}

// writeDigitalOcean adds the records the domain lacks. It never adds a second
// SPF or DMARC record, and never touches records it didn't ask for. It
// returns the records it could not write (none, when all went in).
func (s *Service) writeDigitalOcean(ctx context.Context, orgID, domain string, want []DNSRecord) ([]DNSRecord, error) {
	st, err := s.settings(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if len(st.DnsTokenEnc) == 0 || s.crypto == nil {
		return want, refuse("No DigitalOcean token is set. Call setup_sending(action: dns_token, token: ...) with a token limited to domain read and write, or use dns_host: manual.")
	}
	token, err := s.crypto.DecryptString(st.DnsTokenEnc)
	if err != nil {
		return want, err
	}
	var list struct {
		Records []doRecord `json:"domain_records"`
	}
	if err := s.doCall(ctx, token, http.MethodGet, "/v2/domains/"+domain+"/records?per_page=200", nil, &list); err != nil {
		return want, err
	}
	var failed []DNSRecord
	for _, r := range want {
		if doHas(list.Records, r) {
			continue
		}
		body := doRecord{Type: r.Type, Name: r.Name, Data: r.Data, TTL: 3600}
		if r.Type == "MX" {
			p := r.Priority
			body.Priority = &p
		}
		if err := s.doCall(ctx, token, http.MethodPost, "/v2/domains/"+domain+"/records", body, nil); err != nil {
			failed = append(failed, r)
		}
	}
	if len(failed) > 0 {
		return failed, refuse("DigitalOcean refused %d record(s) for %s; add them by hand.", len(failed), domain)
	}
	return nil, nil
}

// doHas says whether the zone already has the record, or a record of the same
// kind that must stay single (SPF, DMARC, DKIM at one selector).
func doHas(have []doRecord, r DNSRecord) bool {
	for _, h := range have {
		if !strings.EqualFold(h.Type, r.Type) || !strings.EqualFold(h.Name, r.Name) {
			continue
		}
		hd, rd := strings.ToLower(strings.TrimSuffix(h.Data, ".")), strings.ToLower(strings.TrimSuffix(r.Data, "."))
		switch {
		case hd == rd:
			return true
		case r.Type == "TXT" && strings.HasPrefix(rd, "v=spf1") && strings.HasPrefix(hd, "v=spf1"):
			return true
		case r.Type == "TXT" && strings.HasPrefix(rd, "v=dmarc1") && strings.HasPrefix(hd, "v=dmarc1"):
			return true
		case r.Type == "TXT" && strings.HasSuffix(r.Name, "._domainkey"):
			return true
		case r.Type == "CNAME": // one CNAME per name; the tracking check says if it points elsewhere
			return true
		}
	}
	return false
}

func (s *Service) doCall(ctx context.Context, token, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		raw, _ := json.Marshal(in)
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.doAPI+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return refuse("DigitalOcean refused the token (HTTP %d). Set a token with domain read and write.", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotFound {
		return refuse("DigitalOcean has no zone for this domain. Point the domain's nameservers at DigitalOcean and add it there first.")
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("digitalocean %s %s: HTTP %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}
