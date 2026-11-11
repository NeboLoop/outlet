package outbound

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/services/emailval"
)

// Do not contact: before outreach is queued for anyone, at import, at
// enrollment and before each follow-up, Outlet checks them against every
// source of "we already know them": its own suppression, the brand's own
// domains, the brand's customers, uploaded do-not-contact lists and an
// optional outside check (a CRM answering yes or no). The first that
// matches is the reason, and every skip says it.

// Exclusion is why someone gets no outreach: the source that matched and,
// in words, what it found.
type Exclusion struct {
	Source string `json:"source"` // suppressed, own_domain, customer, customer_domain, do_not_contact_list, outside_check
	Reason string `json:"reason"`
	// Retry says the outside check didn't answer: nobody is enrolled until
	// it does, but someone already in a campaign waits rather than stops.
	Retry bool `json:"retry,omitempty"`
}

// Text is the exclusion as one line, for skip reasons.
func (e Exclusion) Text() string { return "do not contact (" + e.Source + "): " + e.Reason }

// stop is what a stopped sequence records: the source only, so campaign
// results group by it.
func (e Exclusion) stop() string { return "do not contact: " + e.Source }

// The sources, in the order they are checked.
const (
	SourceSuppressed   = "suppressed"
	SourceOwnDomain    = "own_domain"
	SourceCustomer     = "customer"
	SourceCustomerDom  = "customer_domain"
	SourceList         = "do_not_contact_list"
	SourceOutsideCheck = "outside_check"
)

// dncCheck checks addresses for one brand, one batch at a time. An outside
// check that fails once fails the rest of its batch without being called
// again, so a CRM that is down costs one timeout, not one per prospect.
type dncCheck struct {
	s       *Service
	org     db.Organization
	own     []string
	outside bool  // call the outside check (when the brand has one)
	down    error // the outside check failed in this batch
}

// newDNC starts a batch of checks. outside false leaves out the outside
// check (import does: it runs again before anything is queued).
func (s *Service) newDNC(ctx context.Context, org db.Organization, outside bool) *dncCheck {
	c := &dncCheck{s: s, org: org, own: append([]string{}, s.protected...), outside: outside}
	if ds, err := s.db.ListOwnDomains(ctx, org.ID); err == nil {
		c.own = append(c.own, ds...)
	}
	return c
}

// check is the first exclusion that applies to an address, if any.
func (c *dncCheck) check(ctx context.Context, address string) (Exclusion, bool) {
	address = normEmail(address)
	domain := domainOf(address)
	s, orgID := c.s, c.org.ID
	if s.suppressed(ctx, orgID, address) {
		return Exclusion{Source: SourceSuppressed, Reason: "on the brand's suppression list, or bounced or complained before"}, true
	}
	if slices.ContainsFunc(c.own, func(d string) bool { return domain == d || strings.HasSuffix(domain, "."+d) }) {
		return Exclusion{Source: SourceOwnDomain, Reason: domain + " is the brand's own domain"}, true
	}
	nullOrg := nullStr(orgID)
	if found, err := s.db.FindCustomer(ctx, db.FindCustomerParams{OrgID: nullOrg, Email: address}); err != nil {
		return Exclusion{Source: SourceCustomer, Reason: "the customer check failed: " + err.Error(), Retry: true}, true
	} else if found != "" {
		return Exclusion{Source: SourceCustomer, Reason: "already known to the brand: " + found}, true
	}
	if c.org.DncCustomerDomains == 1 && !emailval.IsPersonalMailbox(domain) {
		if found, err := s.db.FindCustomerAtDomain(ctx, db.FindCustomerAtDomainParams{OrgID: nullOrg, Domain: domain}); err != nil {
			return Exclusion{Source: SourceCustomerDom, Reason: "the customer check failed: " + err.Error(), Retry: true}, true
		} else if found != "" {
			return Exclusion{Source: SourceCustomerDom, Reason: "a customer works at " + domain + " (" + found + ")"}, true
		}
	}
	if hit, err := s.db.FindDoNotContact(ctx, db.FindDoNotContactParams{OrgID: orgID, Email: address, Domain: domain}); err == nil {
		return Exclusion{Source: SourceList, Reason: fmt.Sprintf("%s %s is on the do-not-contact list %q", hit.Kind, hit.Value, hit.ListName)}, true
	}
	if c.outside && c.org.DncCheckUrl != "" {
		if c.down == nil {
			var ex Exclusion
			var skip bool
			ex, skip, c.down = c.ask(ctx, address, domain)
			if c.down == nil {
				return ex, skip
			}
		}
		return Exclusion{Source: SourceOutsideCheck, Reason: "the outside check didn't answer, so no outreach goes until it does (" + c.down.Error() + ")", Retry: true}, true
	}
	return Exclusion{}, false
}

// outsideCheckTimeout is how long the outside check may take to answer.
const outsideCheckTimeout = 5 * time.Second

// ask calls the brand's outside check: a POST of the address, signed with
// the brand's secret, answered with whether to leave the person alone.
func (c *dncCheck) ask(ctx context.Context, address, domain string) (Exclusion, bool, error) {
	secret, err := c.s.crypto.DecryptString(c.org.DncCheckSecretEnc)
	if err != nil {
		return Exclusion{}, false, errors.New("its secret can't be read; set it again")
	}
	body, _ := json.Marshal(map[string]any{"email": address, "domain": domain, "brand": c.org.Slug, "timestamp": c.s.now().UTC().Unix()})
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	ctx, cancel := context.WithTimeout(ctx, outsideCheckTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.org.DncCheckUrl, bytes.NewReader(body))
	if err != nil {
		return Exclusion{}, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Outlet-Signature", hex.EncodeToString(mac.Sum(nil)))
	resp, err := c.s.http.Do(req)
	if err != nil {
		return Exclusion{}, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Exclusion{}, false, fmt.Errorf("it answered %d", resp.StatusCode)
	}
	var ans struct {
		DoNotContact *bool  `json:"do_not_contact"`
		Reason       string `json:"reason"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&ans); err != nil || ans.DoNotContact == nil {
		return Exclusion{}, false, errors.New(`its answer isn't {"do_not_contact": true|false}`)
	}
	if !*ans.DoNotContact {
		return Exclusion{}, false, nil
	}
	reason := strings.TrimSpace(ans.Reason)
	if reason == "" {
		reason = "the outside check says not to contact them"
	}
	return Exclusion{Source: SourceOutsideCheck, Reason: reason}, true, nil
}

// DoNotContact checks one address against every source, the outside check
// included.
func (s *Service) DoNotContact(ctx context.Context, org db.Organization, address string) (Exclusion, bool) {
	return s.newDNC(ctx, org, true).check(ctx, address)
}

// Gate is the check the sequence engine runs before it queues any outreach
// step: what the person's stopped sequence records, or "" to go ahead. An
// outside check that doesn't answer lets the step be queued: it is checked
// again as it goes, and waits there until the check answers.
func (s *Service) Gate(ctx context.Context, org db.Organization, contact db.Contact) string {
	if ex, skip := s.DoNotContact(ctx, org, contact.Email); skip && !ex.Retry {
		return ex.stop()
	}
	return ""
}

// ---- settings ---------------------------------------------------------------

// DNCSettings are the brand's do-not-contact options. The secret is
// write-only: only whether it is set shows.
type DNCSettings struct {
	CustomerDomains bool   `json:"customer_domains"`
	CheckURL        string `json:"check_url"`
	CheckSecretSet  bool   `json:"check_secret_set"`
}

// DNCSettingsInput changes them. CustomerDomains and CheckURL left out
// keep theirs; an empty CheckURL turns the outside check off. CheckSecret
// is needed with a new CheckURL, and left out keeps the stored one.
type DNCSettingsInput struct {
	CustomerDomains *bool   `json:"customer_domains,omitempty"`
	CheckURL        *string `json:"check_url,omitempty"`
	CheckSecret     string  `json:"check_secret,omitempty"`
}

// DNCSettingsOf reads a brand's do-not-contact options.
func DNCSettingsOf(org db.Organization) DNCSettings {
	return DNCSettings{CustomerDomains: org.DncCustomerDomains == 1, CheckURL: org.DncCheckUrl, CheckSecretSet: len(org.DncCheckSecretEnc) > 0}
}

// UpdateDNCSettings changes the brand's do-not-contact options.
func (s *Service) UpdateDNCSettings(ctx context.Context, org db.Organization, in DNCSettingsInput) (DNCSettings, error) {
	if in.CustomerDomains != nil {
		org.DncCustomerDomains = boolInt(*in.CustomerDomains)
	}
	if in.CheckURL != nil {
		raw := strings.TrimSpace(*in.CheckURL)
		if raw == "" {
			org.DncCheckUrl, org.DncCheckSecretEnc = "", nil
		} else {
			u, err := url.Parse(raw)
			if err != nil || u.Host == "" || (u.Scheme != "https" && !(u.Scheme == "http" && isLoopback(u.Hostname()))) {
				return DNCSettingsOf(org), refuse("check_url must be an https:// address.")
			}
			if raw != org.DncCheckUrl && in.CheckSecret == "" {
				return DNCSettingsOf(org), refuse("A new check_url needs its check_secret: the shared secret the request is signed with.")
			}
			org.DncCheckUrl = raw
		}
	}
	if in.CheckSecret != "" {
		if org.DncCheckUrl == "" {
			return DNCSettingsOf(org), refuse("Set check_url with the secret.")
		}
		enc, err := s.crypto.EncryptString(in.CheckSecret)
		if err != nil {
			return DNCSettingsOf(org), err
		}
		org.DncCheckSecretEnc = enc
	}
	err := s.db.UpdateDoNotContactSettings(ctx, db.UpdateDoNotContactSettingsParams{
		DncCustomerDomains: org.DncCustomerDomains, DncCheckUrl: org.DncCheckUrl, DncCheckSecretEnc: org.DncCheckSecretEnc, ID: org.ID,
	})
	return DNCSettingsOf(org), err
}

// ---- uploaded lists ---------------------------------------------------------

// DNCList is one uploaded do-not-contact list.
type DNCList struct {
	Name      string `json:"name"`
	Emails    int64  `json:"emails"`
	Domains   int64  `json:"domains"`
	UpdatedAt string `json:"updated_at"`
}

// DNCAddResult counts what an upload added.
type DNCAddResult struct {
	Added      int               `json:"added"`
	Duplicates int               `json:"duplicates"`
	Refused    map[string]string `json:"refused,omitempty"`
}

const maxDNCBatch = 10000

// AddDoNotContact adds addresses and domains to a named list (made by the
// first entry). Each entry is an email address or a domain; a website
// address counts as its domain.
func (s *Service) AddDoNotContact(ctx context.Context, org db.Organization, list string, entries []string) (DNCAddResult, error) {
	list = strings.TrimSpace(list)
	if list == "" {
		return DNCAddResult{}, refuse("Name the list, such as \"Customers\" or \"Open deals\".")
	}
	if len(entries) == 0 || len(entries) > maxDNCBatch {
		return DNCAddResult{}, refuse("Add 1 to %d entries at a time.", maxDNCBatch)
	}
	res := DNCAddResult{Refused: map[string]string{}}
	for _, e := range entries {
		kind, value, ok := dncEntry(e)
		if !ok {
			if strings.TrimSpace(e) != "" {
				res.Refused[e] = "not an email address or a domain"
			}
			continue
		}
		n, err := s.db.AddDoNotContact(ctx, db.AddDoNotContactParams{OrgID: org.ID, ListName: list, Kind: kind, Value: value})
		switch {
		case err != nil:
			res.Refused[e] = err.Error()
		case n == 0:
			res.Duplicates++
		default:
			res.Added++
		}
	}
	if len(res.Refused) == 0 {
		res.Refused = nil
	}
	return res, nil
}

// dncEntry reads one entry: an email address, a domain, or a website.
func dncEntry(e string) (kind, value string, ok bool) {
	e = strings.ToLower(strings.Trim(strings.TrimSpace(e), `"'<>,;`))
	if strings.Contains(e, "@") {
		local, domain, _ := strings.Cut(e, "@")
		return "email", e, local != "" && strings.Contains(domain, ".") && !strings.ContainsAny(e, " /")
	}
	if strings.Contains(e, "://") {
		if u, err := url.Parse(e); err == nil {
			e = u.Hostname()
		}
	}
	e = strings.TrimPrefix(strings.TrimSuffix(strings.SplitN(e, "/", 2)[0], "."), "www.")
	return "domain", e, strings.Contains(e, ".") && !strings.ContainsAny(e, " :")
}

// DoNotContactLists lists the brand's uploaded lists.
func (s *Service) DoNotContactLists(ctx context.Context, org db.Organization) ([]DNCList, error) {
	rows, err := s.db.ListDoNotContactLists(ctx, org.ID)
	if err != nil {
		return nil, err
	}
	out := make([]DNCList, 0, len(rows))
	for _, r := range rows {
		out = append(out, DNCList{Name: r.ListName, Emails: r.Emails, Domains: r.Domains, UpdatedAt: r.UpdatedAt})
	}
	return out, nil
}

// DeleteDoNotContactList removes an uploaded list and everything on it.
func (s *Service) DeleteDoNotContactList(ctx context.Context, org db.Organization, list string) (int64, error) {
	n, err := s.db.DeleteDoNotContactList(ctx, db.DeleteDoNotContactListParams{OrgID: org.ID, ListName: strings.TrimSpace(list)})
	if err == nil && n == 0 {
		return 0, refuse("no do-not-contact list %q", list)
	}
	return n, err
}
