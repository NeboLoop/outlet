// Package emailval provides layered email validation utilities.
// It performs syntax, domain/MX, disposable, role-based, and optional SMTP checks.
package emailval

import (
	"cmp"
	"context"
	"fmt"
	"time"
)

// Level represents a validation layer.
type Level int

const (
	LevelSyntax Level = iota + 1
	LevelDomain
	LevelDisposable
	LevelRole
	LevelSMTP
)

// Options configures email validation behavior.
type Options struct {
	// Timeout is the maximum duration for all network work (DNS, SMTP).
	// Defaults to 3 seconds if not set.
	Timeout time.Duration

	// CheckSMTP enables SMTP mailbox verification.
	CheckSMTP bool

	// CheckDisposable enables disposable/temporary domain detection.
	CheckDisposable bool

	// AllowRole allows role-based emails like info@, support@, admin@.
	AllowRole bool

	// DisposableProvider is an optional custom disposable domain provider.
	DisposableProvider DisposableProvider

	// SMTPDialer is an optional custom SMTP dialer (for testing or proxying).
	SMTPDialer SMTPDialer

	// SMTP configures the HELO hostname and MAIL FROM domain for SMTP checks.
	// Defaults to DefaultSMTPConfig() (localhost), which most MX servers reject.
	SMTP *SMTPConfig

	// Proxies are SOCKS5 URLs (socks5://user:pass@host:1080) of dedicated probe hosts.
	// Probes rotate across healthy proxies; empty means dial directly from this host.
	Proxies []string

	// CheckDomainAge looks up domain registration age over RDAP.
	CheckDomainAge bool
}

// Result is a structured report of validation.
type Result struct {
	Input      string
	Normalized string // lowercased, trimmed; Gmail leading periods removed

	SyntaxOK   bool
	DomainOK   bool // the domain has a mail host (MX, or A/AAAA as implicit MX)
	HasMX      bool // the domain publishes real MX records
	Disposable bool
	RoleBased  bool

	SMTPChecked     bool
	SMTPDeliverable *bool // nil = unknown, true/false if SMTP check done

	// FailedAt is the first failing level, if any.
	FailedAt *Level

	// Messages contains human readable reasons for failures or warnings.
	Messages []string

	// ZeroBounce-compatible verdict (see status.go).
	Status        string `json:"status"`
	SubStatus     string `json:"sub_status"`
	FreeEmail     bool   `json:"free_email"`
	DidYouMean    string `json:"did_you_mean"`
	Account       string `json:"account"`
	Domain        string `json:"domain"`
	DomainAgeDays int    `json:"domain_age_days"` // 0 = unknown
	SMTPProvider  string `json:"smtp_provider"`   // google, microsoft, yahoo, proofpoint, mimecast, ... or ""
	MXRecord      string `json:"mx_record"`       // preferred MX host
	CatchAll      bool   `json:"catch_all"`

	// Verdict inputs collected by the pipeline and read by setVerdict.
	invalidSub           string // syntax/typo/DNS invalid reason
	unknownSub           string // DNS gave no definitive answer
	dnmSub               string // first do_not_mail reason
	probeSub             string // SMTP probe sub-status
	aiAgent              bool
	alias                bool
	leadingPeriodRemoved bool
}

func (r *Result) addMessage(msg string, args ...any) {
	r.Messages = append(r.Messages, fmt.Sprintf(msg, args...))
}

// Valid returns true if no validation level failed.
func (r *Result) Valid() bool {
	return r.FailedAt == nil
}

// Validate performs layered validation on an email.
// It never sends a message, only checks for deliverability.
// It never returns nil Result; on error you still get a partially filled Result.
//
// Pipeline: syntax (+ Gmail leading-period normalization) → typo → DNS (MX, null MX, implicit MX,
// unroutable) → provider/free/AI-agent facts → do-not-mail checks (disposable, trap, toxic, relay)
// → role → optional SMTP probe. setVerdict turns the collected facts into a ZeroBounce status.
func Validate(ctx context.Context, email string, opts *Options) (*Result, error) {
	if opts == nil {
		opts = &Options{}
	}

	res := &Result{
		Input:      email,
		Normalized: normalize(email),
	}
	defer setVerdict(res)

	if res.Normalized == "" {
		res.fail(LevelSyntax, SubFailedSyntaxCheck, "empty email")
		return res, nil
	}

	// Options.Timeout bounds DNS, RDAP and the disposable-list wait. The SMTP probe gets the
	// caller's context instead: probeMailbox applies Options.Timeout per SMTP session itself,
	// after its rate-limit wait, so DNS time never eats into the probe.
	probeCtx := ctx
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 1) Syntax check (Gmail ignores leading periods: normalize instead of failing).
	res.Normalized, res.leadingPeriodRemoved = stripLeadingPeriod(res.Normalized)
	local, domain, err := validateSyntax(res.Normalized)
	if err != nil {
		res.Account, res.Domain, _ = splitEmail(res.Normalized)
		res.fail(LevelSyntax, SubFailedSyntaxCheck, "syntax error: %v", err)
		return res, nil
	}
	res.SyntaxOK = true
	res.Account, res.Domain = local, domain
	res.FreeEmail = isFreeEmail(domain)
	if res.leadingPeriodRemoved {
		res.addMessage("leading period removed from local part")
	}

	// 2) Typo detection (judged after DNS: only a typo domain without MX fails).
	res.DidYouMean = suggestAddress(local, domain)

	// 3) DNS
	dns, err := lookupDomainDNS(ctx, domain)
	switch {
	case err != nil:
		res.fail(LevelDomain, "", "DNS lookup failed for %q: %v", domain, err)
		res.unknownSub = SubExceptionOccurred
		if ctx.Err() != nil {
			res.unknownSub = SubTimeoutExceeded
		}
		return res, nil
	case dns.NullMX:
		res.fail(LevelDomain, SubDoesNotAcceptMail, "domain %q publishes a null MX and accepts no mail", domain)
		return res, nil
	case len(dns.Hosts) == 0 && res.DidYouMean != "":
		res.fail(LevelDomain, SubPossibleTypo, "domain %q does not exist; did you mean %s?", domain, res.DidYouMean)
		return res, nil
	case len(dns.Hosts) == 0:
		res.fail(LevelDomain, SubNoDNSEntries, "no MX or A records for domain %q", domain)
		return res, nil
	}
	res.DomainOK = true
	res.HasMX = dns.MXFound
	res.MXRecord = dns.Hosts[0]
	if !dns.MXFound {
		res.addMessage("no MX records; using the domain's A/AAAA record as mail host")
	}
	switch {
	case res.DidYouMean != "" && !dns.MXFound:
		res.fail(LevelDomain, SubPossibleTypo, "domain %q has no MX; did you mean %s?", domain, res.DidYouMean)
		return res, nil
	case dns.Unroutable:
		res.fail(LevelDomain, SubUnroutableIPAddress, "mail hosts for %q resolve only to unroutable addresses", domain)
		return res, nil
	}
	res.SMTPProvider = detectProvider(dns.Hosts)
	res.aiAgent = isAIAgentMailbox(domain, dns.Hosts)
	res.alias = isPlainForwarder(dns.Hosts)

	// Domain age runs alongside the remaining checks and the SMTP probe, bounded by ctx.
	if opts.CheckDomainAge {
		ageCh := make(chan int, 1)
		go func() { ageCh <- domainAgeDays(ctx, domain) }()
		defer func() { res.DomainAgeDays = <-ageCh }()
	}

	// 4) Do-not-mail checks. All report at LevelDisposable: the mailbox may exist but must not be mailed.
	if opts.CheckDisposable {
		provider := opts.DisposableProvider
		if provider == nil {
			remoteDisposable.ensure(ctx)
			provider = DefaultDisposableProvider()
		}
		if reg := registrableDomain(domain); provider.IsDisposable(domain) || (reg != domain && provider.IsDisposable(reg)) {
			res.Disposable = true
			res.doNotMail(SubDisposable, "disposable or temporary domain")
		}
	}
	if isPossibleTrap(local, domain) {
		res.doNotMail(SubPossibleTrap, "address looks like a spam trap")
	}
	if isToxic(local, res.FreeEmail) {
		res.doNotMail(SubToxic, "machine-generated mailbox on a free provider")
	}
	if isPrivacyRelay(domain, dns.Hosts) {
		res.doNotMail(SubMXForward, "privacy relay / forwarding alias")
	}

	// 5) Role-based check
	if !opts.AllowRole && IsRoleBased(local) {
		res.RoleBased = true
		lvl := LevelRole
		if res.FailedAt == nil {
			res.FailedAt = &lvl
		}
		res.addMessage("role-based address: %s", local)
	}

	// 6) Optional SMTP check. Skipped for do_not_mail addresses: probing traps or relays only adds risk.
	if opts.CheckSMTP && res.dnmSub == "" {
		pr := probeMailbox(probeCtx, domain, res.Normalized, dns.Hosts, opts)
		res.SMTPChecked = true
		res.SMTPDeliverable = pr.Deliverable
		res.CatchAll = pr.CatchAll
		res.probeSub = pr.SubStatus
		switch {
		case pr.CatchAll:
			res.addMessage("domain accepts all addresses (catch-all)")
		case pr.Deliverable == nil:
			// Treat as unknown, do not hard-fail on transient SMTP errors.
			res.addMessage("SMTP check inconclusive: %s", cmp.Or(pr.SubStatus, SubExceptionOccurred))
		case !*pr.Deliverable:
			lvl := LevelSMTP
			if res.FailedAt == nil {
				res.FailedAt = &lvl
			}
			res.addMessage("SMTP server reports mailbox undeliverable")
		}
	}

	return res, nil
}

// fail records a failing level with its invalid sub-status (empty when the outcome is unknown).
func (r *Result) fail(lvl Level, invalidSub, msg string, args ...any) {
	if r.FailedAt == nil {
		r.FailedAt = &lvl
	}
	if r.invalidSub == "" {
		r.invalidSub = invalidSub
	}
	r.addMessage(msg, args...)
}

// doNotMail records a do_not_mail reason; the first one wins the sub-status.
func (r *Result) doNotMail(sub, msg string) {
	lvl := LevelDisposable
	if r.FailedAt == nil {
		r.FailedAt = &lvl
	}
	if r.dnmSub == "" {
		r.dnmSub = sub
	}
	r.addMessage("%s", msg)
}

// setVerdict fills Status and SubStatus from the layer results. Precedence:
// syntax/typo/DNS invalid → do_not_mail (disposable, trap, toxic, relay) → SMTP invalid
// → role (catch-all variant) → catch-all → unknown → valid (+ valid sub-statuses).
func setVerdict(r *Result) {
	r.Status, r.SubStatus = verdict(r)
}

func verdict(r *Result) (status, sub string) {
	failedAt := func(l Level) bool { return r.FailedAt != nil && *r.FailedAt == l }
	switch {
	case r.invalidSub != "":
		return StatusInvalid, r.invalidSub
	case failedAt(LevelSyntax):
		return StatusInvalid, SubFailedSyntaxCheck
	case failedAt(LevelDomain) && r.unknownSub == "":
		return StatusInvalid, SubNoDNSEntries
	case r.dnmSub != "":
		return StatusDoNotMail, r.dnmSub
	case r.Disposable:
		return StatusDoNotMail, SubDisposable
	case r.SMTPDeliverable != nil && !*r.SMTPDeliverable:
		return StatusInvalid, cmp.Or(r.probeSub, SubMailboxNotFound)
	case r.RoleBased && r.CatchAll:
		return StatusDoNotMail, SubRoleBasedCatchAll
	case r.RoleBased:
		return StatusDoNotMail, SubRoleBased
	case r.CatchAll && r.aiAgent:
		return StatusCatchAll, SubAIAgentMailbox
	case r.CatchAll:
		return StatusCatchAll, ""
	case r.unknownSub != "":
		return StatusUnknown, r.unknownSub
	case r.SMTPChecked && r.SMTPDeliverable == nil:
		return StatusUnknown, cmp.Or(r.probeSub, SubExceptionOccurred)
	case r.aiAgent:
		return StatusValid, SubAIAgentMailbox
	case r.alias:
		return StatusValid, SubAliasAddress
	case r.leadingPeriodRemoved:
		return StatusValid, SubLeadingPeriodRemoved
	}
	return StatusValid, ""
}
