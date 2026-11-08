package emailval

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// SMTPDialer defines how to connect to SMTP servers.
type SMTPDialer interface {
	DialSMTP(ctx context.Context, address string) (*textproto.Conn, error)
}

// DefaultSMTPDialer dials plain TCP with a timeout.
type DefaultSMTPDialer struct {
	Timeout time.Duration
}

// DialSMTP connects to an SMTP server.
func (d DefaultSMTPDialer) DialSMTP(ctx context.Context, address string) (*textproto.Conn, error) {
	timeout := d.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	return textproto.NewConn(conn), nil
}

// SMTPConfig provides configuration for SMTP verification.
// Use this to customize behavior and avoid blacklisting.
type SMTPConfig struct {
	// HeloHostname is the hostname to use in HELO/EHLO.
	// Should be a valid, resolvable hostname you control.
	// If empty, defaults to "localhost".
	HeloHostname string

	// FromDomain is the domain to use in MAIL FROM.
	// Should be a domain you control with valid SPF.
	// If empty, uses HeloHostname.
	FromDomain string

	// SkipRateLimiting disables the built-in rate limiter.
	// Only set this if you implement your own rate limiting.
	SkipRateLimiting bool

	// MaxRetries is the number of MX servers to try before giving up.
	// Defaults to 2.
	MaxRetries int
}

// DefaultSMTPConfig returns a safe default SMTP configuration.
func DefaultSMTPConfig() *SMTPConfig {
	return &SMTPConfig{
		HeloHostname: "localhost",
		FromDomain:   "localhost",
		MaxRetries:   2,
	}
}

// checkSMTPMailbox resolves the domain's MX hosts and reports whether the mailbox accepts mail.
// An inconclusive probe is returned as an error. probeMailbox is the probe itself.
func checkSMTPMailbox(ctx context.Context, domain, fullEmail string, opts *Options) (bool, error) {
	mxHosts, err := getMXHosts(ctx, domain)
	if err != nil || len(mxHosts) == 0 {
		return false, fmt.Errorf("no MX records for SMTP check: %w", err)
	}
	res := probeMailbox(ctx, domain, fullEmail, mxHosts, opts)
	if res.Deliverable == nil {
		return false, fmt.Errorf("SMTP probe inconclusive: %s", res.SubStatus)
	}
	return *res.Deliverable, nil
}

// smtpStage is the point in the SMTP conversation a reply answers.
type smtpStage int

const (
	stageGreeting smtpStage = iota
	stageHelo
	stageMail
	stageRcpt
)

// smtpReply is one (possibly multi-line) server reply.
type smtpReply struct {
	code int
	msg  string // lines joined with "\n", codes stripped
}

var enhancedCodeRe = regexp.MustCompile(`^([245]\.\d{1,3}\.\d{1,3})\b`)

// enhanced returns the RFC 3463 enhanced status code (e.g. "5.1.1"), or "".
func (r smtpReply) enhanced() string {
	for _, line := range strings.Split(r.msg, "\n") {
		if m := enhancedCodeRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			return m[1]
		}
	}
	return ""
}

// probeOutcome is the result of one SMTP session against one MX host.
type probeOutcome struct {
	result     ProbeResult
	blocked    bool  // a reply says the exit IP is blocked
	connFailed bool  // dial failure or forcible disconnect
	catchAll   *bool // catch-all verdict learned for the domain, nil when unknown
}

// smtpProbe runs one SMTP session: greeting, EHLO, MAIL FROM, RCPT target and, when
// testCatchAll is set and the target was accepted, RSET + RCPT for a random local part.
// ctx bounds all network I/O. No message is ever sent.
func smtpProbe(ctx context.Context, dialer SMTPDialer, addr, domain, target string, testCatchAll bool, cfg *SMTPConfig) probeOutcome {
	conn, err := dialer.DialSMTP(ctx, addr)
	if err != nil {
		return probeOutcome{result: ProbeResult{SubStatus: SubFailedSMTPConnection}, connFailed: true}
	}
	defer conn.Close()
	// Closing the connection unblocks any pending read once ctx is done.
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()

	fail := func(stage smtpStage, r smtpReply, err error) probeOutcome {
		if err != nil {
			res := classifyIOError(ctx, stage, err)
			return probeOutcome{result: res, connFailed: res.SubStatus == SubForcibleDisconnect}
		}
		res, blocked := classifyReply(stage, r)
		return probeOutcome{result: res, blocked: blocked}
	}

	heloHost := cfg.HeloHostname
	if heloHost == "" {
		heloHost = "localhost"
	}
	fromDomain := cfg.FromDomain
	if fromDomain == "" {
		fromDomain = heloHost
	}

	r, err := smtpCommand(conn, "")
	if err != nil || r.code != 220 {
		return fail(stageGreeting, r, err)
	}
	if r, err = smtpCommand(conn, "EHLO %s", heloHost); err != nil {
		return fail(stageHelo, r, err)
	}
	if r.code != 250 {
		if r, err = smtpCommand(conn, "HELO %s", heloHost); err != nil || r.code != 250 {
			return fail(stageHelo, r, err)
		}
	}
	if r, err = smtpCommand(conn, "MAIL FROM:<verify@%s>", fromDomain); err != nil || r.code != 250 {
		return fail(stageMail, r, err)
	}
	if r, err = smtpCommand(conn, "RCPT TO:<%s>", target); err != nil {
		return fail(stageRcpt, r, err)
	}
	res, blocked := classifyReply(stageRcpt, r)
	out := probeOutcome{result: res, blocked: blocked}
	defer conn.PrintfLine("QUIT")
	if !isAccepted(res) || !testCatchAll {
		return out
	}

	// Catch-all check on the same session. If anything here fails the target's
	// accept still stands; the domain's verdict is just left uncached.
	if r, err = smtpCommand(conn, "RSET"); err != nil || r.code != 250 {
		return out
	}
	if r, err = smtpCommand(conn, "MAIL FROM:<verify@%s>", fromDomain); err != nil || r.code != 250 {
		return out
	}
	if r, err = smtpCommand(conn, "RCPT TO:<%s@%s>", randomLocalPart(), domain); err != nil {
		return out
	}
	randRes, _ := classifyReply(stageRcpt, r)
	switch {
	case isAccepted(randRes):
		yes := true
		out.result.CatchAll = true
		out.catchAll = &yes
	case randRes.SubStatus == SubMailboxNotFound:
		no := false
		out.catchAll = &no
	}
	return out
}

// smtpCommand sends a command (an empty format only reads, for the greeting) and reads the reply.
func smtpCommand(conn *textproto.Conn, format string, args ...any) (smtpReply, error) {
	if format != "" {
		if err := conn.PrintfLine(format, args...); err != nil {
			return smtpReply{}, err
		}
	}
	code, msg, err := conn.ReadResponse(0)
	if err != nil {
		return smtpReply{}, err
	}
	return smtpReply{code: code, msg: msg}, nil
}

// randomLocalPart returns a local part no real mailbox will have.
func randomLocalPart() string {
	return strings.ToLower(rand.Text())
}

func isAccepted(r ProbeResult) bool {
	return r.Deliverable != nil && *r.Deliverable
}

// classifyIOError maps an I/O failure on an open session to a sub-status.
func classifyIOError(ctx context.Context, stage smtpStage, err error) ProbeResult {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded) && stage == stageGreeting:
		return ProbeResult{SubStatus: SubMailServerNoResponse}
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return ProbeResult{SubStatus: SubTimeoutExceeded}
	case ctx.Err() != nil:
		return ProbeResult{SubStatus: SubExceptionOccurred}
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF),
		errors.Is(err, syscall.ECONNRESET), errors.Is(err, syscall.EPIPE), errors.Is(err, net.ErrClosed):
		return ProbeResult{SubStatus: SubForcibleDisconnect}
	}
	return ProbeResult{SubStatus: SubExceptionOccurred}
}

var (
	notFoundPhrases = []string{"user unknown", "unknown user", "no such user", "does not exist", "mailbox unavailable",
		"user not found", "recipient not found", "invalid recipient", "no mailbox"}
	quotaPhrases  = []string{"quota", "mailbox full", "mailbox is full", "insufficient storage"}
	blockPhrases  = []string{"block", "blacklist", "spamhaus", " listed", "reputation", "rbl", "dnsbl"}
	policyIPWords = []string{"your ip", "client ip", "connecting ip", "ip address", "client host", "sending ip"}
)

// classifyReply maps a non-success reply (or any RCPT reply) to a probe result.
// blocked reports a block signal against the exit IP.
func classifyReply(stage smtpStage, r smtpReply) (res ProbeResult, blocked bool) {
	text := strings.ToLower(r.msg)
	enh := r.enhanced()
	no := false
	blockWording := containsAny(text, blockPhrases) ||
		(strings.Contains(text, "policy") && containsAny(text, policyIPWords))

	switch {
	case stage == stageRcpt && (r.code == 250 || r.code == 251):
		yes := true
		return ProbeResult{Deliverable: &yes}, false
	case r.code < 400:
		// 252 (cannot verify) and stray 2xx/3xx replies prove nothing.
		return ProbeResult{SubStatus: SubExceptionOccurred}, false
	case stage == stageRcpt && (enh == "5.1.1" || enh == "5.1.10" || enh == "5.1.0"):
		return ProbeResult{Deliverable: &no, SubStatus: SubMailboxNotFound}, false
	case stage == stageRcpt && (r.code == 552 || enh == "5.2.2" || enh == "4.2.2" || containsAny(text, quotaPhrases)):
		return ProbeResult{Deliverable: &no, SubStatus: SubMailboxQuotaExceeded}, false
	case blockWording:
		return ProbeResult{SubStatus: SubAntispamSystem}, true
	case stage == stageRcpt && r.code >= 500 && containsAny(text, notFoundPhrases):
		return ProbeResult{Deliverable: &no, SubStatus: SubMailboxNotFound}, false
	case (r.code == 450 || r.code == 451 || enh == "4.7.1") && isGreylisted(text):
		return ProbeResult{SubStatus: SubGreylisted}, false
	case strings.HasPrefix(enh, "5.7."):
		// 5.7.x before RCPT rejects the connecting host, not a mailbox.
		return ProbeResult{SubStatus: SubAntispamSystem}, stage != stageRcpt
	case stage == stageRcpt && (r.code == 550 || r.code == 551 || r.code == 553):
		// Without block or quota signals, a permanent RCPT rejection means the mailbox is gone.
		return ProbeResult{Deliverable: &no, SubStatus: SubMailboxNotFound}, false
	case r.code < 500:
		return ProbeResult{SubStatus: SubMailServerTempError}, false
	case stage == stageGreeting && r.code == 554:
		// 554 greeting: the server refuses to talk to this host.
		return ProbeResult{SubStatus: SubAntispamSystem}, false
	}
	return ProbeResult{SubStatus: SubExceptionOccurred}, false
}

// isGreylisted reports whether reply text reads like greylisting.
func isGreylisted(text string) bool {
	return containsAny(strings.ToLower(text), []string{"greylist", "graylist", "try again", "temporarily", "retry later", "please retry"})
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
