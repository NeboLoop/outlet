package emailval

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/textproto"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeMX is a scripted in-process SMTP server.
type fakeMX struct {
	greeting string                   // "" never greets; "close" drops the connection
	mail     string                   // MAIL FROM reply; default "250 OK"
	rcpt     func(addr string) string // RCPT reply per recipient; "" never answers

	conns atomic.Int32
	mu    sync.Mutex
	rcpts []string
}

func (f *fakeMX) recipients() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.rcpts...)
}

func startFakeMX(t *testing.T, f *fakeMX) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			f.conns.Add(1)
			go f.serve(c)
		}
	}()
	return ln.Addr().String()
}

func (f *fakeMX) serve(c net.Conn) {
	defer c.Close()
	reply := func(s string) { io.WriteString(c, s+"\r\n") }
	switch f.greeting {
	case "close":
		return
	case "":
		io.Copy(io.Discard, c) // hold the connection open, silently
		return
	}
	reply(f.greeting)
	r := bufio.NewReader(c)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		cmd := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(cmd, "EHLO"):
			reply("250-fake.test\r\n250-SIZE 1000\r\n250 8BITMIME")
		case strings.HasPrefix(cmd, "HELO"), strings.HasPrefix(cmd, "RSET"):
			reply("250 OK")
		case strings.HasPrefix(cmd, "MAIL FROM"):
			if f.mail != "" {
				reply(f.mail)
			} else {
				reply("250 OK")
			}
		case strings.HasPrefix(cmd, "RCPT TO"):
			addr := line[strings.Index(line, "<")+1 : strings.LastIndex(line, ">")]
			f.mu.Lock()
			f.rcpts = append(f.rcpts, addr)
			f.mu.Unlock()
			resp := f.rcpt(addr)
			if resp == "" {
				io.Copy(io.Discard, r)
				return
			}
			reply(resp)
		case strings.HasPrefix(cmd, "QUIT"):
			reply("221 bye")
			return
		default:
			reply("500 unknown command")
		}
	}
}

// dialTo returns an SMTPDialer that connects to addr whatever MX host is requested.
func dialTo(addr string) SMTPDialer {
	return MockSMTPDialer{DialFunc: func(ctx context.Context, _ string) (*textproto.Conn, error) {
		var d net.Dialer
		c, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, err
		}
		return textproto.NewConn(c), nil
	}}
}

// rejectRandom accepts user@ and rejects every other mailbox with 5.1.1.
func rejectRandom(user string) func(string) string {
	return func(addr string) string {
		if strings.HasPrefix(addr, user+"@") {
			return "250 2.1.5 OK"
		}
		return "550 5.1.1 User unknown"
	}
}

func testOpts(dialer SMTPDialer) *Options {
	return &Options{
		Timeout:    300 * time.Millisecond,
		SMTPDialer: dialer,
		SMTP:       &SMTPConfig{HeloHostname: "probe.test", FromDomain: "probe.test", SkipRateLimiting: true, MaxRetries: 2},
	}
}

func probe(t *testing.T, domain, user string, opts *Options) ProbeResult {
	t.Helper()
	return probeMailbox(context.Background(), domain, user+"@"+domain, []string{"mx1." + domain + ".", "mx2." + domain + "."}, opts)
}

func boolPtrString(b *bool) string {
	if b == nil {
		return "nil"
	}
	if *b {
		return "true"
	}
	return "false"
}

func TestProbeMailbox_Replies(t *testing.T) {
	tests := []struct {
		name        string
		mx          *fakeMX
		deliverable *bool
		catchAll    bool
		subStatus   string
	}{
		{
			name:        "accepted",
			mx:          &fakeMX{greeting: "220 fake ESMTP", rcpt: rejectRandom("alice")},
			deliverable: ptr(true),
		},
		{
			name: "mailbox not found multi-line 5.1.1",
			mx: &fakeMX{greeting: "220 fake ESMTP", rcpt: func(string) string {
				return "550-5.1.1 The email account that you tried to reach does not exist.\r\n550 5.1.1 https://support.example/NoSuchUser"
			}},
			deliverable: ptr(false),
			subStatus:   SubMailboxNotFound,
		},
		{
			name:        "mailbox not found by wording",
			mx:          &fakeMX{greeting: "220 fake ESMTP", rcpt: func(string) string { return "554 No such user here" }},
			deliverable: ptr(false),
			subStatus:   SubMailboxNotFound,
		},
		{
			name:        "quota 552",
			mx:          &fakeMX{greeting: "220 fake ESMTP", rcpt: func(string) string { return "552 5.2.2 Mailbox full" }},
			deliverable: ptr(false),
			subStatus:   SubMailboxQuotaExceeded,
		},
		{
			name:        "quota 452 4.2.2",
			mx:          &fakeMX{greeting: "220 fake ESMTP", rcpt: func(string) string { return "452 4.2.2 The email account is over quota" }},
			deliverable: ptr(false),
			subStatus:   SubMailboxQuotaExceeded,
		},
		{
			name:      "greylisted",
			mx:        &fakeMX{greeting: "220 fake ESMTP", rcpt: func(string) string { return "451 4.7.1 Greylisted, please try again later" }},
			subStatus: SubGreylisted,
		},
		{
			name:      "temporary error",
			mx:        &fakeMX{greeting: "220 fake ESMTP", rcpt: func(string) string { return "450 4.2.1 Mailbox busy" }},
			subStatus: SubMailServerTempError,
		},
		{
			name:      "antispam on rcpt",
			mx:        &fakeMX{greeting: "220 fake ESMTP", rcpt: func(string) string { return "550 5.7.1 Client host [1.2.3.4] blocked using Spamhaus" }},
			subStatus: SubAntispamSystem,
		},
		{
			name:      "antispam 5.7.x on MAIL FROM",
			mx:        &fakeMX{greeting: "220 fake ESMTP", mail: "550 5.7.0 Sender rejected", rcpt: rejectRandom("alice")},
			subStatus: SubAntispamSystem,
		},
		{
			name:      "antispam greeting",
			mx:        &fakeMX{greeting: "554 5.7.1 Your IP has poor reputation", rcpt: rejectRandom("alice")},
			subStatus: SubAntispamSystem,
		},
		{
			name:      "252 cannot verify",
			mx:        &fakeMX{greeting: "220 fake ESMTP", rcpt: func(string) string { return "252 2.1.5 Cannot VRFY user" }},
			subStatus: SubExceptionOccurred,
		},
		{
			name:      "forcible disconnect",
			mx:        &fakeMX{greeting: "close"},
			subStatus: SubForcibleDisconnect,
		},
		{
			name:      "no greeting",
			mx:        &fakeMX{greeting: ""},
			subStatus: SubMailServerNoResponse,
		},
		{
			name:      "rcpt timeout",
			mx:        &fakeMX{greeting: "220 fake ESMTP", rcpt: func(string) string { return "" }},
			subStatus: SubTimeoutExceeded,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := startFakeMX(t, tt.mx)
			domain := "replies" + string(rune('a'+i)) + ".test"
			res := probe(t, domain, "alice", testOpts(dialTo(addr)))
			if boolPtrString(res.Deliverable) != boolPtrString(tt.deliverable) || res.CatchAll != tt.catchAll || res.SubStatus != tt.subStatus {
				t.Errorf("got deliverable=%s catchAll=%v sub=%q, want deliverable=%s catchAll=%v sub=%q",
					boolPtrString(res.Deliverable), res.CatchAll, res.SubStatus,
					boolPtrString(tt.deliverable), tt.catchAll, tt.subStatus)
			}
		})
	}
}

func ptr(b bool) *bool { return &b }

func TestProbeMailbox_CatchAllDetection(t *testing.T) {
	mx := &fakeMX{greeting: "220 fake ESMTP", rcpt: func(string) string { return "250 OK" }}
	opts := testOpts(dialTo(startFakeMX(t, mx)))

	res := probe(t, "catchall.test", "alice", opts)
	if !res.CatchAll || !isAccepted(res) {
		t.Fatalf("first probe = %+v, want catch-all accepted", res)
	}
	rcpts := mx.recipients()
	if mx.conns.Load() != 1 || len(rcpts) != 2 || strings.HasPrefix(rcpts[1], "alice@") {
		t.Fatalf("want target + random RCPT on one session, got %d conns, rcpts %v", mx.conns.Load(), rcpts)
	}

	// The cached verdict answers without touching the network.
	res = probe(t, "catchall.test", "bob", opts)
	if !res.CatchAll || !isAccepted(res) || mx.conns.Load() != 1 {
		t.Fatalf("cached probe = %+v with %d conns, want catch-all from cache", res, mx.conns.Load())
	}
}

func TestProbeMailbox_NotCatchAllCached(t *testing.T) {
	mx := &fakeMX{greeting: "220 fake ESMTP", rcpt: rejectRandom("alice")}
	opts := testOpts(dialTo(startFakeMX(t, mx)))

	if res := probe(t, "strict.test", "alice", opts); res.CatchAll || !isAccepted(res) {
		t.Fatalf("first probe = %+v, want accepted, not catch-all", res)
	}
	if res := probe(t, "strict.test", "alice", opts); res.CatchAll || !isAccepted(res) {
		t.Fatalf("second probe = %+v, want accepted, not catch-all", res)
	}
	// Second session skips the random RCPT: target, random, target.
	if rcpts := mx.recipients(); len(rcpts) != 3 {
		t.Fatalf("rcpts = %v, want 3", rcpts)
	}
}

func TestProbeMailbox_FailsOverToNextMX(t *testing.T) {
	mx := &fakeMX{greeting: "220 fake ESMTP", rcpt: rejectRandom("alice")}
	good := dialTo(startFakeMX(t, mx))
	var dialed []string
	dialer := MockSMTPDialer{DialFunc: func(ctx context.Context, address string) (*textproto.Conn, error) {
		dialed = append(dialed, address)
		if strings.HasPrefix(address, "mx1.") {
			return nil, errors.New("connection refused")
		}
		return good.DialSMTP(ctx, address)
	}}
	res := probe(t, "failover.test", "alice", testOpts(dialer))
	if !isAccepted(res) {
		t.Fatalf("res = %+v, want accepted via mx2", res)
	}
	if len(dialed) != 2 || dialed[1] != "mx2.failover.test:25" {
		t.Fatalf("dialed = %v", dialed)
	}
}

func TestProbeMailbox_DialFailure(t *testing.T) {
	dialer := MockSMTPDialer{DialFunc: func(context.Context, string) (*textproto.Conn, error) {
		return nil, errors.New("connection refused")
	}}
	if res := probe(t, "dialfail.test", "alice", testOpts(dialer)); res.SubStatus != SubFailedSMTPConnection || res.Deliverable != nil {
		t.Fatalf("res = %+v, want failed_smtp_connection", res)
	}
}

func TestProbeMailbox_YahooNotProbed(t *testing.T) {
	dialer := MockSMTPDialer{DialFunc: func(context.Context, string) (*textproto.Conn, error) {
		t.Fatal("yahoo MX must not be dialed")
		return nil, nil
	}}
	res := probeMailbox(context.Background(), "yahoo.com", "alice@yahoo.com", []string{"mta5.am0.yahoodns.net."}, testOpts(dialer))
	if !res.CatchAll || res.Deliverable != nil {
		t.Fatalf("res = %+v, want catch-all with unknown deliverability", res)
	}
}

func TestProbeMailbox_RateLimitWaitDoesNotConsumeTimeout(t *testing.T) {
	mx := &fakeMX{greeting: "220 fake ESMTP", rcpt: rejectRandom("alice")}
	opts := testOpts(dialTo(startFakeMX(t, mx)))
	opts.SMTP.SkipRateLimiting = false
	opts.Timeout = 300 * time.Millisecond // shorter than the 1s Google spacing
	mxHosts := []string{"aspmx.l.google.com."}

	start := time.Now()
	for _, domain := range []string{"ratelimit-a.test", "ratelimit-b.test"} {
		res := probeMailbox(context.Background(), domain, "alice@"+domain, mxHosts, opts)
		if !isAccepted(res) {
			t.Fatalf("%s: res = %+v, want accepted after waiting", domain, res)
		}
	}
	if elapsed := time.Since(start); elapsed < probeIntervalLargeProvider-50*time.Millisecond {
		t.Fatalf("two google probes took %v, want >= %v", elapsed, probeIntervalLargeProvider)
	}

	// A cancelled context stops the wait.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if res := probeMailbox(ctx, "ratelimit-c.test", "alice@ratelimit-c.test", mxHosts, opts); res.SubStatus != SubTimeoutExceeded {
		t.Fatalf("cancelled wait res = %+v, want timeout_exceeded", res)
	}
}

func TestProbeProvider(t *testing.T) {
	tests := map[string]string{
		"aspmx.l.google.com.":                     "google",
		"alt1.aspmx.l.google.com":                 "google",
		"example-com.mail.protection.outlook.com": "microsoft",
		"mx-aol.mail.gm0.yahoodns.net.":           "yahoo",
		"mx.example.com.":                         "mx.example.com",
		"notgoogle.com":                           "notgoogle.com",
	}
	for host, want := range tests {
		if got := probeProvider(host); got != want {
			t.Errorf("probeProvider(%q) = %q, want %q", host, got, want)
		}
	}
}

// startFakeSOCKS runs a no-auth SOCKS5 server that connects every request to target.
func startFakeSOCKS(t *testing.T, target string) (string, *atomic.Int32) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	var conns atomic.Int32
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			conns.Add(1)
			go serveSOCKS(c, target)
		}
	}()
	return "socks5://user:pass@" + ln.Addr().String(), &conns
}

func serveSOCKS(c net.Conn, target string) {
	defer c.Close()
	buf := make([]byte, 256)
	if _, err := io.ReadFull(c, buf[:2]); err != nil {
		return
	}
	if _, err := io.ReadFull(c, buf[:buf[1]]); err != nil {
		return
	}
	c.Write([]byte{5, 0}) // no auth required
	if _, err := io.ReadFull(c, buf[:4]); err != nil {
		return
	}
	var addrLen int
	switch buf[3] {
	case 1:
		addrLen = 4
	case 4:
		addrLen = 16
	case 3:
		if _, err := io.ReadFull(c, buf[:1]); err != nil {
			return
		}
		addrLen = int(buf[0])
	}
	if _, err := io.ReadFull(c, buf[:addrLen+2]); err != nil {
		return
	}
	up, err := net.Dial("tcp", target)
	if err != nil {
		c.Write([]byte{5, 5, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}
	defer up.Close()
	c.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
	go io.Copy(up, c)
	io.Copy(c, up)
}

func TestProbeMailbox_ProxyQuarantinedOnBlock(t *testing.T) {
	mx := &fakeMX{greeting: "554 5.7.1 Service unavailable; Client host blocked using zen.spamhaus.org", rcpt: rejectRandom("alice")}
	proxyURL, proxyConns := startFakeSOCKS(t, startFakeMX(t, mx))
	opts := testOpts(nil)
	opts.Proxies = []string{proxyURL}

	if res := probe(t, "blocked-a.test", "alice", opts); res.SubStatus != SubAntispamSystem {
		t.Fatalf("first probe = %+v, want antispam_system", res)
	}
	if proxyConns.Load() != 1 {
		t.Fatalf("proxy conns = %d, want 1", proxyConns.Load())
	}
	// Every proxy is quarantined now: fail without dialing.
	if res := probe(t, "blocked-b.test", "alice", opts); res.SubStatus != SubFailedSMTPConnection {
		t.Fatalf("second probe = %+v, want failed_smtp_connection", res)
	}
	if proxyConns.Load() != 1 {
		t.Fatalf("quarantined proxy was dialed: %d conns", proxyConns.Load())
	}
}

func TestProbeMailbox_ProxyRotation(t *testing.T) {
	mx := &fakeMX{greeting: "220 fake ESMTP", rcpt: rejectRandom("alice")}
	mxAddr := startFakeMX(t, mx)
	urlA, connsA := startFakeSOCKS(t, mxAddr)
	urlB, connsB := startFakeSOCKS(t, mxAddr)
	opts := testOpts(nil)
	opts.Proxies = []string{urlA, urlB}

	for _, d := range []string{"rot-a.test", "rot-b.test", "rot-c.test", "rot-d.test"} {
		if res := probe(t, d, "alice", opts); !isAccepted(res) {
			t.Fatalf("%s: res = %+v, want accepted through proxy", d, res)
		}
	}
	if connsA.Load() != 2 || connsB.Load() != 2 {
		t.Fatalf("proxy conns = %d/%d, want 2/2", connsA.Load(), connsB.Load())
	}
}

func TestProxyHealth_RepeatedConnFailures(t *testing.T) {
	h := &proxyHealth{}
	const u = "socks5://flaky.test:1080"
	for i := 0; i < proxyFailureThreshold-1; i++ {
		h.report(u, probeOutcome{connFailed: true})
	}
	if _, ok := h.next([]string{u}); !ok {
		t.Fatal("proxy quarantined before threshold")
	}
	h.report(u, probeOutcome{connFailed: true})
	if _, ok := h.next([]string{u}); ok {
		t.Fatal("proxy not quarantined after repeated connection failures")
	}
}

func TestClassifyReply(t *testing.T) {
	tests := []struct {
		stage   smtpStage
		reply   smtpReply
		sub     string
		blocked bool
	}{
		{stageRcpt, smtpReply{550, "5.1.10 RESOLVER.ADR.RecipientNotFound"}, SubMailboxNotFound, false},
		{stageRcpt, smtpReply{550, "Requested action not taken: mailbox unavailable"}, SubMailboxNotFound, false},
		{stageRcpt, smtpReply{550, "5.7.1 Service unavailable, Client host [1.2.3.4] blocked using Spamhaus"}, SubAntispamSystem, true},
		{stageRcpt, smtpReply{550, "5.7.1 Relaying denied"}, SubAntispamSystem, false},
		{stageMail, smtpReply{550, "5.7.1 Rejected by policy"}, SubAntispamSystem, true},
		{stageGreeting, smtpReply{421, "4.7.0 Try again later, closing connection. (IP reputation)"}, SubAntispamSystem, true},
		{stageGreeting, smtpReply{421, "Too many connections"}, SubMailServerTempError, false},
		{stageRcpt, smtpReply{451, "4.7.1 Please try again later"}, SubGreylisted, false},
		{stageRcpt, smtpReply{452, "4.5.3 Too many recipients"}, SubMailServerTempError, false},
	}
	for _, tt := range tests {
		res, blocked := classifyReply(tt.stage, tt.reply)
		if res.SubStatus != tt.sub || blocked != tt.blocked {
			t.Errorf("classifyReply(%d, %d %q) = %q blocked=%v, want %q blocked=%v",
				tt.stage, tt.reply.code, tt.reply.msg, res.SubStatus, blocked, tt.sub, tt.blocked)
		}
	}
}
