package mailpool

import (
	"bufio"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeSMTP is a tiny SMTP server on 127.0.0.1 that records what it is sent.
// No mail leaves the test.
type fakeSMTP struct {
	t    *testing.T
	ln   net.Listener
	port int

	mu       sync.Mutex
	authFail bool              // answer 535 to AUTH
	rcptCode map[string]string // recipient -> reply line, e.g. "550 5.1.1 no such user"
	messages []fakeMessage
}

type fakeMessage struct {
	MailFrom string
	Rcpt     []string
	Data     string
}

func newFakeSMTP(t *testing.T) *fakeSMTP {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSMTP{t: t, ln: ln, port: ln.Addr().(*net.TCPAddr).Port, rcptCode: map[string]string{}}
	t.Cleanup(func() { ln.Close() })
	go f.serve()
	return f
}

func (f *fakeSMTP) setAuthFail(v bool) {
	f.mu.Lock()
	f.authFail = v
	f.mu.Unlock()
}

func (f *fakeSMTP) reject(rcpt, line string) {
	f.mu.Lock()
	f.rcptCode[strings.ToLower(rcpt)] = line
	f.mu.Unlock()
}

func (f *fakeSMTP) sent() []fakeMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]fakeMessage(nil), f.messages...)
}

func (f *fakeSMTP) serve() {
	for {
		c, err := f.ln.Accept()
		if err != nil {
			return
		}
		go f.handle(c)
	}
}

func (f *fakeSMTP) handle(c net.Conn) {
	defer c.Close()
	r := bufio.NewReader(c)
	w := func(s string) { c.Write([]byte(s + "\r\n")) }
	w("220 fake.test ESMTP")
	var cur fakeMessage
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		cmd := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			w("250-fake.test")
			w("250 AUTH PLAIN")
		case strings.HasPrefix(cmd, "AUTH"):
			f.mu.Lock()
			fail := f.authFail
			f.mu.Unlock()
			if fail {
				w("535 5.7.8 authentication failed")
			} else {
				w("235 2.7.0 ok")
			}
		case strings.HasPrefix(cmd, "MAIL FROM:"):
			cur = fakeMessage{MailFrom: strings.Trim(line[len("MAIL FROM:"):], "<> ")}
			w("250 ok")
		case strings.HasPrefix(cmd, "RCPT TO:"):
			rcpt := strings.ToLower(strings.Trim(line[len("RCPT TO:"):], "<> "))
			f.mu.Lock()
			reply, refused := f.rcptCode[rcpt]
			f.mu.Unlock()
			if refused {
				w(reply)
				continue
			}
			cur.Rcpt = append(cur.Rcpt, rcpt)
			w("250 ok")
		case cmd == "DATA":
			w("354 go ahead")
			var b strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" {
					break
				}
				b.WriteString(l)
			}
			cur.Data = b.String()
			f.mu.Lock()
			f.messages = append(f.messages, cur)
			n := len(f.messages)
			f.mu.Unlock()
			w("250 queued as " + strconv.Itoa(n))
		case cmd == "RSET", cmd == "NOOP":
			w("250 ok")
		case cmd == "QUIT":
			w("221 bye")
			return
		default:
			w("502 not implemented")
		}
	}
}
