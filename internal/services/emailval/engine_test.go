package emailval

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeResolver answers from maps; unknown names are NXDOMAIN, names in fail return SERVFAIL.
type fakeResolver struct {
	mx   map[string][]string
	ips  map[string][]string
	fail map[string]bool
}

func (f fakeResolver) LookupMX(_ context.Context, name string) ([]*net.MX, error) {
	if f.fail[name] {
		return nil, &net.DNSError{Err: "server misbehaving", Name: name, IsTemporary: true}
	}
	hosts, ok := f.mx[name]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: name, IsNotFound: true}
	}
	out := make([]*net.MX, len(hosts))
	for i, h := range hosts {
		out[i] = &net.MX{Host: h, Pref: uint16(i * 10)}
	}
	return out, nil
}

func (f fakeResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	ips, ok := f.ips[host]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}
	out := make([]net.IPAddr, len(ips))
	for i, ip := range ips {
		out[i] = net.IPAddr{IP: net.ParseIP(ip)}
	}
	return out, nil
}

// useResolver swaps the package resolver and clears caches for one test.
func useResolver(t *testing.T, r dnsResolver) {
	t.Helper()
	old := resolver
	resolver = r
	domainDNSCache.reset()
	rdapCache.reset()
	t.Cleanup(func() {
		resolver = old
		domainDNSCache.reset()
		rdapCache.reset()
	})
}

var testDNS = fakeResolver{
	mx: map[string][]string{
		"gmail.com":        {"gmail-smtp-in.l.google.com."},
		"nullmx.test":      {"."},
		"private.test":     {"mx.private.test."},
		"company.test":     {"company-test.mail.protection.outlook.com."},
		"forwarded.test":   {"mx1.improvmx.com.", "mx2.improvmx.com."},
		"sl-custom.test":   {"mx1.simplelogin.co."},
		"duck.com":         {"inbound.duck.com."},
		"agentmail.to":     {"mx.agentmail.to."},
		"mailinator.com":   {"mail.mailinator.com."},
		"sub.trashmail.de": {"mx.trashmail.de."},
	},
	ips: map[string][]string{
		"gmail-smtp-in.l.google.com":               {"142.250.1.27"},
		"mx.private.test":                          {"10.0.0.5", "192.168.1.1"},
		"company-test.mail.protection.outlook.com": {"52.101.1.1"},
		"mx1.improvmx.com":                         {"5.5.5.5"},
		"mx2.improvmx.com":                         {"5.5.5.6"},
		"mx1.simplelogin.co":                       {"6.6.6.6"},
		"inbound.duck.com":                         {"7.7.7.7"},
		"mx.agentmail.to":                          {"8.8.4.4"},
		"mail.mailinator.com":                      {"9.9.9.9"},
		"mx.trashmail.de":                          {"9.9.9.10"},
		"a-only.test":                              {"93.184.216.34"},
		"loopback-a.test":                          {"127.0.0.1"},
		"gmial.com":                                {"1.2.3.4"}, // typo domain, parked web, no MX
	},
	fail: map[string]bool{"servfail.test": true},
}

func TestValidate_Engine(t *testing.T) {
	useResolver(t, testDNS)
	opts := &Options{CheckDisposable: true, DisposableProvider: NewStaticDisposableProvider([]string{"mailinator.com", "trashmail.de"})}

	tests := []struct {
		email, status, sub string
		wantMX             string
		wantProvider       string
		wantDidYouMean     string
		wantFree, wantOK   bool
	}{
		{email: "jane@gmail.com", status: StatusValid, wantMX: "gmail-smtp-in.l.google.com", wantProvider: "google", wantFree: true, wantOK: true},
		{email: ".jane@gmail.com", status: StatusValid, sub: SubLeadingPeriodRemoved, wantMX: "gmail-smtp-in.l.google.com", wantProvider: "google", wantFree: true, wantOK: true},
		{email: "jane@nullmx.test", status: StatusInvalid, sub: SubDoesNotAcceptMail},
		{email: "jane@nxdomain.test", status: StatusInvalid, sub: SubNoDNSEntries},
		{email: "jane@private.test", status: StatusInvalid, sub: SubUnroutableIPAddress, wantMX: "mx.private.test"},
		{email: "jane@loopback-a.test", status: StatusInvalid, sub: SubUnroutableIPAddress, wantMX: "loopback-a.test"},
		{email: "jane@a-only.test", status: StatusValid, wantMX: "a-only.test", wantOK: true},
		{email: "jane@servfail.test", status: StatusUnknown, sub: SubExceptionOccurred},
		{email: "jane@gmial.com", status: StatusInvalid, sub: SubPossibleTypo, wantDidYouMean: "jane@gmail.com", wantMX: "gmial.com"},
		{email: "jane@gmaill.com", status: StatusInvalid, sub: SubPossibleTypo, wantDidYouMean: "jane@gmail.com"},
		{email: "bob@company.test", status: StatusValid, wantMX: "company-test.mail.protection.outlook.com", wantProvider: "microsoft", wantOK: true},
		{email: "bob@forwarded.test", status: StatusValid, sub: SubAliasAddress, wantMX: "mx1.improvmx.com", wantProvider: "improvmx", wantOK: true},
		{email: "bob@sl-custom.test", status: StatusDoNotMail, sub: SubMXForward, wantMX: "mx1.simplelogin.co", wantProvider: "simplelogin"},
		{email: "x1y2@duck.com", status: StatusDoNotMail, sub: SubMXForward, wantMX: "inbound.duck.com"},
		{email: "bot@agentmail.to", status: StatusValid, sub: SubAIAgentMailbox, wantMX: "mx.agentmail.to", wantProvider: "agentmail", wantOK: true},
		{email: "x@mailinator.com", status: StatusDoNotMail, sub: SubDisposable, wantMX: "mail.mailinator.com"},
		{email: "x@sub.trashmail.de", status: StatusDoNotMail, sub: SubDisposable, wantMX: "mx.trashmail.de"},
		{email: "spamtrap@company.test", status: StatusDoNotMail, sub: SubPossibleTrap, wantMX: "company-test.mail.protection.outlook.com", wantProvider: "microsoft"},
		{email: "3f9a2b7c1d8e4f6a0b5c9d2e@gmail.com", status: StatusDoNotMail, sub: SubToxic, wantMX: "gmail-smtp-in.l.google.com", wantProvider: "google", wantFree: true},
		{email: "info@company.test", status: StatusDoNotMail, sub: SubRoleBased, wantMX: "company-test.mail.protection.outlook.com", wantProvider: "microsoft"},
		{email: "bad..dots@gmail.com", status: StatusInvalid, sub: SubFailedSyntaxCheck},
	}
	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			r, err := Validate(context.Background(), tt.email, opts)
			if err != nil {
				t.Fatal(err)
			}
			if r.Status != tt.status || r.SubStatus != tt.sub {
				t.Errorf("verdict = %s/%s, want %s/%s (messages %v)", r.Status, r.SubStatus, tt.status, tt.sub, r.Messages)
			}
			if r.MXRecord != tt.wantMX {
				t.Errorf("MXRecord = %q, want %q", r.MXRecord, tt.wantMX)
			}
			if r.SMTPProvider != tt.wantProvider {
				t.Errorf("SMTPProvider = %q, want %q", r.SMTPProvider, tt.wantProvider)
			}
			if r.DidYouMean != tt.wantDidYouMean {
				t.Errorf("DidYouMean = %q, want %q", r.DidYouMean, tt.wantDidYouMean)
			}
			if r.FreeEmail != tt.wantFree {
				t.Errorf("FreeEmail = %v, want %v", r.FreeEmail, tt.wantFree)
			}
			if r.Valid() != tt.wantOK {
				t.Errorf("Valid() = %v, want %v", r.Valid(), tt.wantOK)
			}
		})
	}
}

func TestValidate_AccountDomainAndNormalization(t *testing.T) {
	useResolver(t, testDNS)
	r, _ := Validate(context.Background(), "  ..Jane.Doe@GMail.com ", nil)
	if r.Normalized != "jane.doe@gmail.com" || r.Account != "jane.doe" || r.Domain != "gmail.com" {
		t.Errorf("got normalized=%q account=%q domain=%q", r.Normalized, r.Account, r.Domain)
	}
	// Leading period stays a syntax error outside Gmail.
	r, _ = Validate(context.Background(), ".jane@company.test", nil)
	if r.Status != StatusInvalid || r.SubStatus != SubFailedSyntaxCheck {
		t.Errorf("non-gmail leading period = %s/%s", r.Status, r.SubStatus)
	}
}

func TestValidate_TypoWithMXOnlyHints(t *testing.T) {
	useResolver(t, fakeResolver{
		mx:  map[string][]string{"hotmial.com": {"mx.hotmial.com."}},
		ips: map[string][]string{"mx.hotmial.com": {"1.2.3.4"}},
	})
	r, _ := Validate(context.Background(), "sam@hotmial.com", nil)
	if r.Status != StatusValid || r.DidYouMean != "sam@hotmail.com" {
		t.Errorf("got %s/%s did_you_mean=%q", r.Status, r.SubStatus, r.DidYouMean)
	}
}

func TestLookupDomainDNS_CachesDefinitiveOnly(t *testing.T) {
	calls := 0
	r := countingResolver{fakeResolver: testDNS, calls: &calls}
	useResolver(t, r)
	ctx := context.Background()
	for range 3 {
		if _, err := lookupDomainDNS(ctx, "gmail.com"); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Errorf("MX lookups for cached domain = %d, want 1", calls)
	}
	calls = 0
	for range 2 {
		if _, err := lookupDomainDNS(ctx, "servfail.test"); err == nil {
			t.Fatal("want error")
		}
	}
	if calls != 2 {
		t.Errorf("transient failures should not be cached: lookups = %d, want 2", calls)
	}
}

type countingResolver struct {
	fakeResolver
	calls *int
}

func (c countingResolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	*c.calls++
	return c.fakeResolver.LookupMX(ctx, name)
}

func TestIsRoutableIP(t *testing.T) {
	tests := map[string]bool{
		"8.8.8.8": true, "142.250.1.27": true, "2607:f8b0::1": true,
		"10.1.2.3": false, "172.16.0.1": false, "192.168.0.1": false, "127.0.0.1": false, "0.0.0.0": false,
		"169.254.1.1": false, "100.64.0.1": false, "192.0.2.1": false, "198.18.0.1": false, "240.0.0.1": false,
		"::1": false, "fc00::1": false, "fe80::1": false, "2001:db8::1": false, "::ffff:10.0.0.1": false,
	}
	for ip, want := range tests {
		if got := isRoutableIP(net.ParseIP(ip)); got != want {
			t.Errorf("isRoutableIP(%s) = %v, want %v", ip, got, want)
		}
	}
}

func TestSuggestDomain(t *testing.T) {
	tests := map[string]string{
		"gmial.com":     "gmail.com",
		"gmai.com":      "gmail.com",
		"gamil.com":     "gmail.com",
		"gmail.con":     "gmail.com",
		"gmail.co":      "gmail.com",
		"gmail.com.com": "gmail.com",
		"yaho.com":      "yahoo.com",
		"yahooo.com":    "yahoo.com",
		"hotmial.com":   "hotmail.com",
		"hotmal.com":    "hotmail.com",
		"outlok.com":    "outlook.com",
		"iclod.com":     "icloud.com",
		"comcast.nte":   "comcast.net",
		"comcats.net":   "comcast.net",
		"verizon.ent":   "verizon.net",
		"aol.co":        "aol.com",
		// No suggestion: exact popular domains, unrelated domains, short-domain neighbours.
		"gmail.com":    "",
		"mail.com":     "",
		"outlook.de":   "",
		"acme.io":      "",
		"example.com":  "",
		"qx.com":       "",
		"mycompany.co": "",
		"now.com":      "",
	}
	for in, want := range tests {
		if got := suggestDomain(in); got != want {
			t.Errorf("suggestDomain(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEditDistance(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"gmail.com", "gmail.com", 0},
		{"gmial.com", "gmail.com", 1}, // transposition
		{"gmai.com", "gmail.com", 1},
		{"gmaill.com", "gmail.com", 1},
		{"gnail.con", "gmail.com", 2},
		{"", "abc", 3},
	}
	for _, tt := range tests {
		if got := editDistance(tt.a, tt.b); got != tt.want {
			t.Errorf("editDistance(%q,%q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		hosts []string
		want  string
	}{
		{[]string{"aspmx.l.google.com"}, "google"},
		{[]string{"acme-com.mail.protection.outlook.com"}, "microsoft"},
		{[]string{"mx-aol.mail.gm0.yahoodns.net"}, "aol"},
		{[]string{"mta5.am0.yahoodns.net"}, "yahoo"},
		{[]string{"mx01.mail.icloud.com"}, "apple"},
		{[]string{"mx.zoho.com"}, "zoho"},
		{[]string{"mx0a-001.pphosted.com"}, "proofpoint"},
		{[]string{"us-smtp-inbound-1.mimecast.com"}, "mimecast"},
		{[]string{"d123.ess.barracudanetworks.com"}, "barracuda"},
		{[]string{"mx1.hc123.iphmx.com"}, "cisco"},
		{[]string{"in1-smtp.messagingengine.com"}, "fastmail"},
		{[]string{"mail.protonmail.ch"}, "protonmail"},
		{[]string{"mx00.gmx.net"}, "gmx"},
		{[]string{"mx00.ionos.com"}, "ionos"},
		{[]string{"mx.yandex.net"}, "yandex"},
		{[]string{"mxa.mailgun.org"}, "mailgun"},
		{[]string{"inbound-smtp.us-east-1.amazonaws.com"}, "amazon_ses"},
		{[]string{"route1.mx.cloudflare.net"}, "cloudflare"},
		{[]string{"mail.example.com", "backup.pphosted.com"}, "proofpoint"},
		{[]string{"mail.example.com"}, ""},
		{[]string{"notgoogle.com"}, ""},
		{nil, ""},
	}
	for _, tt := range tests {
		if got := detectProvider(tt.hosts); got != tt.want {
			t.Errorf("detectProvider(%v) = %q, want %q", tt.hosts, got, tt.want)
		}
	}
}

func TestIsFreeEmail(t *testing.T) {
	for d, want := range map[string]bool{
		"gmail.com": true, "yahoo.co.uk": true, "proton.me": true, "qq.com": true, "web.de": true,
		"comcast.net": false, "acme.com": false, "gmail.co": false,
	} {
		if got := isFreeEmail(d); got != want {
			t.Errorf("isFreeEmail(%q) = %v, want %v", d, got, want)
		}
	}
}

func TestForwardAndAIAgent(t *testing.T) {
	if !isPrivacyRelay("privaterelay.appleid.com", nil) || !isPrivacyRelay("mozmail.com", nil) ||
		!isPrivacyRelay("jane.anonaddy.com", nil) || !isPrivacyRelay("custom.test", []string{"mx2.simplelogin.co"}) {
		t.Error("privacy relay not detected")
	}
	if isPrivacyRelay("gmail.com", []string{"gmail-smtp-in.l.google.com"}) {
		t.Error("gmail flagged as relay")
	}
	if !isPlainForwarder([]string{"route2.mx.cloudflare.net"}) || !isPlainForwarder([]string{"mx1.forwardemail.net"}) {
		t.Error("plain forwarder not detected")
	}
	if isPlainForwarder([]string{"aspmx.l.google.com", "mx1.improvmx.com"}) {
		t.Error("forwarder detected from a backup MX")
	}
	if !isAIAgentMailbox("agentmail.to", nil) || !isAIAgentMailbox("bots.test", []string{"mx.agentmail.to"}) {
		t.Error("AI agent mailbox not detected")
	}
}

func TestTrapAndToxic(t *testing.T) {
	traps := map[[2]string]bool{
		{"spamtrap", "acme.com"}:     true,
		{"my-honeypot", "acme.com"}:  true,
		{"trap+x", "acme.com"}:       true,
		{"jane", "honeypot.example"}: true,
		{"abuse-report", "acme.com"}: true,
		{"jane", "acme.com"}:         false,
		{"trapper", "acme.com"}:      false,
		{"spamalot.fan", "acme.com"}: false,
	}
	for in, want := range traps {
		if got := isPossibleTrap(in[0], in[1]); got != want {
			t.Errorf("isPossibleTrap(%q,%q) = %v, want %v", in[0], in[1], got, want)
		}
	}
	toxic := []struct {
		local string
		free  bool
		want  bool
	}{
		{"3f9a2b7c1d8e4f6a0b5c9d2e", true, true},
		{"0f8fad5b-d9cb-469f-a165-70867728950e", true, true},
		{"3f9a2b7c1d8e4f6a0b5c9d2e", false, false}, // custom domains may use hashed addresses
		{"deadbeefdeadbeefdeadbeef", true, false},  // no digits
		{"123456789012345678901234", true, false},  // no letters
		{"3f9a2b7c1d8e", true, false},              // too short
		{"jane.doe1985", true, false},
	}
	for _, tt := range toxic {
		if got := isToxic(tt.local, tt.free); got != tt.want {
			t.Errorf("isToxic(%q,%v) = %v, want %v", tt.local, tt.free, got, tt.want)
		}
	}
}

func TestStripLeadingPeriod(t *testing.T) {
	tests := []struct {
		in, want string
		removed  bool
	}{
		{".jane@gmail.com", "jane@gmail.com", true},
		{"..jane@googlemail.com", "jane@googlemail.com", true},
		{"jane@gmail.com", "jane@gmail.com", false},
		{".jane@yahoo.com", ".jane@yahoo.com", false},
		{".@gmail.com", ".@gmail.com", false},
	}
	for _, tt := range tests {
		got, removed := stripLeadingPeriod(tt.in)
		if got != tt.want || removed != tt.removed {
			t.Errorf("stripLeadingPeriod(%q) = %q,%v want %q,%v", tt.in, got, removed, tt.want, tt.removed)
		}
	}
}

func TestValidateSyntax_NewRules(t *testing.T) {
	for _, bad := range []string{
		"a,b@example.com", "a(b)@example.com", "jane@-example.com", "jane@example-.com",
		"jane@exa_mple.com", "jane@example..com",
		fmt.Sprintf("%065d@example.com", 1),
	} {
		if _, _, err := validateSyntax(bad); err == nil {
			t.Errorf("validateSyntax(%q) = nil error, want error", bad)
		}
	}
	for _, good := range []string{"o'brien@example.com", "a!b#c$d@example.com", "jane@xn--bcher-kva.de", "jöhn@münchen.de"} {
		if _, _, err := validateSyntax(good); err != nil {
			t.Errorf("validateSyntax(%q) = %v, want nil", good, err)
		}
	}
}

func TestRegistrableDomain(t *testing.T) {
	for in, want := range map[string]string{
		"example.com":         "example.com",
		"mail.example.com":    "example.com",
		"a.b.example.com":     "example.com",
		"example.co.uk":       "example.co.uk",
		"mail.example.co.uk":  "example.co.uk",
		"shop.example.com.au": "example.com.au",
		"co.uk":               "co.uk",
		"localhost":           "localhost",
	} {
		if got := registrableDomain(in); got != want {
			t.Errorf("registrableDomain(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestVerdictPrecedence(t *testing.T) {
	yes, no := true, false
	lvl := func(l Level) *Level { return &l }
	tests := []struct {
		name   string
		r      Result
		status string
		sub    string
	}{
		{"syntax beats everything", Result{invalidSub: SubFailedSyntaxCheck, dnmSub: SubDisposable, RoleBased: true}, StatusInvalid, SubFailedSyntaxCheck},
		{"typo invalid", Result{invalidSub: SubPossibleTypo}, StatusInvalid, SubPossibleTypo},
		{"legacy syntax level", Result{FailedAt: lvl(LevelSyntax)}, StatusInvalid, SubFailedSyntaxCheck},
		{"legacy domain level", Result{FailedAt: lvl(LevelDomain)}, StatusInvalid, SubNoDNSEntries},
		{"do_not_mail beats smtp invalid", Result{dnmSub: SubPossibleTrap, SMTPDeliverable: &no}, StatusDoNotMail, SubPossibleTrap},
		{"disposable flag", Result{Disposable: true}, StatusDoNotMail, SubDisposable},
		{"smtp invalid beats role", Result{RoleBased: true, SMTPChecked: true, SMTPDeliverable: &no, probeSub: SubMailboxQuotaExceeded}, StatusInvalid, SubMailboxQuotaExceeded},
		{"smtp invalid default sub", Result{SMTPChecked: true, SMTPDeliverable: &no}, StatusInvalid, SubMailboxNotFound},
		{"role catch-all", Result{RoleBased: true, CatchAll: true, SMTPChecked: true, SMTPDeliverable: &yes}, StatusDoNotMail, SubRoleBasedCatchAll},
		{"role", Result{RoleBased: true, SMTPChecked: true, SMTPDeliverable: &yes}, StatusDoNotMail, SubRoleBased},
		{"catch-all beats unknown", Result{CatchAll: true, SMTPChecked: true}, StatusCatchAll, ""},
		{"ai agent catch-all", Result{CatchAll: true, aiAgent: true, SMTPChecked: true, SMTPDeliverable: &yes}, StatusCatchAll, SubAIAgentMailbox},
		{"dns unknown", Result{FailedAt: lvl(LevelDomain), unknownSub: SubTimeoutExceeded}, StatusUnknown, SubTimeoutExceeded},
		{"smtp unknown", Result{SMTPChecked: true, probeSub: SubGreylisted}, StatusUnknown, SubGreylisted},
		{"smtp unknown default sub", Result{SMTPChecked: true}, StatusUnknown, SubExceptionOccurred},
		{"unknown beats alias", Result{SMTPChecked: true, alias: true}, StatusUnknown, SubExceptionOccurred},
		{"ai agent", Result{aiAgent: true, alias: true}, StatusValid, SubAIAgentMailbox},
		{"alias", Result{alias: true, SMTPChecked: true, SMTPDeliverable: &yes}, StatusValid, SubAliasAddress},
		{"leading period", Result{leadingPeriodRemoved: true}, StatusValid, SubLeadingPeriodRemoved},
		{"plain valid", Result{SMTPChecked: true, SMTPDeliverable: &yes}, StatusValid, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setVerdict(&tt.r)
			if tt.r.Status != tt.status || tt.r.SubStatus != tt.sub {
				t.Errorf("verdict = %s/%s, want %s/%s", tt.r.Status, tt.r.SubStatus, tt.status, tt.sub)
			}
		})
	}
}

func TestDomainAgeDays_RDAP(t *testing.T) {
	registered := time.Now().AddDate(0, 0, -400).UTC().Format(time.RFC3339)
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		switch r.URL.Path {
		case "/domain/example.co.uk":
			fmt.Fprintf(w, `{"events":[{"eventAction":"last changed","eventDate":"2024-01-01T00:00:00Z"},{"eventAction":"registration","eventDate":%q}]}`, registered)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	oldURL := rdapBaseURL
	rdapBaseURL = srv.URL + "/domain/"
	rdapCache.reset()
	t.Cleanup(func() { rdapBaseURL = oldURL; rdapCache.reset() })

	ctx := context.Background()
	if got := domainAgeDays(ctx, "mail.example.co.uk"); got < 399 || got > 401 {
		t.Errorf("domainAgeDays = %d, want ~400", got)
	}
	_ = domainAgeDays(ctx, "example.co.uk")
	if hits != 1 {
		t.Errorf("RDAP hits = %d, want 1 (cached)", hits)
	}
	if got := domainAgeDays(ctx, "unknown.test"); got != 0 {
		t.Errorf("domainAgeDays on 404 = %d, want 0", got)
	}
}

func TestValidate_DomainAgeOption(t *testing.T) {
	useResolver(t, testDNS)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"events":[{"eventAction":"registration","eventDate":"2004-04-01T00:00:00Z"}]}`)
	}))
	defer srv.Close()
	oldURL := rdapBaseURL
	rdapBaseURL = srv.URL + "/domain/"
	t.Cleanup(func() { rdapBaseURL = oldURL })

	r, _ := Validate(context.Background(), "jane@gmail.com", &Options{CheckDomainAge: true})
	if r.DomainAgeDays < 365*20 {
		t.Errorf("DomainAgeDays = %d, want > 20 years", r.DomainAgeDays)
	}
	r, _ = Validate(context.Background(), "jane@gmail.com", nil)
	if r.DomainAgeDays != 0 {
		t.Errorf("DomainAgeDays without option = %d, want 0", r.DomainAgeDays)
	}
}

func TestRemoteDisposableList(t *testing.T) {
	fail := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail {
			http.Error(w, "down", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, "# comment\n\nfreshthrowaway.example\nAnotherTemp.example\n")
	}))
	defer srv.Close()

	oldURL, oldList := disposableListURL, remoteDisposable
	disposableListURL = srv.URL
	remoteDisposable = &remoteDisposableList{}
	t.Cleanup(func() { disposableListURL, remoteDisposable = oldURL, oldList })

	p := DefaultDisposableProvider()
	if p.IsDisposable("freshthrowaway.example") {
		t.Fatal("remote domain known before download")
	}
	remoteDisposable.ensure(context.Background()) // first load waits
	if !p.IsDisposable("freshthrowaway.example") || !p.IsDisposable("anothertemp.example") {
		t.Error("remote domains not loaded")
	}
	if !p.IsDisposable("mailinator.com") || p.IsDisposable("gmail.com") {
		t.Error("builtin fallback broken")
	}

	// A failed refresh keeps the previous list.
	fail = true
	remoteDisposable.mu.Lock()
	remoteDisposable.fetchedAt = time.Now().Add(-48 * time.Hour)
	remoteDisposable.mu.Unlock()
	remoteDisposable.ensure(context.Background())
	deadline := time.Now().Add(2 * time.Second)
	for {
		remoteDisposable.mu.Lock()
		loading := remoteDisposable.loading
		remoteDisposable.mu.Unlock()
		if loading == nil || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !p.IsDisposable("freshthrowaway.example") {
		t.Error("failed refresh dropped the previous list")
	}
}

func TestRemoteDisposableList_FirstLoadBoundedByContext(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer srv.Close()
	defer close(release)

	oldURL, oldList := disposableListURL, remoteDisposable
	disposableListURL = srv.URL
	remoteDisposable = &remoteDisposableList{}
	t.Cleanup(func() { disposableListURL, remoteDisposable = oldURL, oldList })

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	remoteDisposable.ensure(ctx)
	if time.Since(start) > time.Second {
		t.Error("ensure blocked past the context deadline")
	}
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Error("expected the context to have expired")
	}
}
