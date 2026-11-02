package email

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// SMTPPool manages a pool of SMTP connections for efficient email sending
type SMTPPool struct {
	host     string
	port     int
	user     string
	pass     string
	useTLS   bool
	poolSize int

	conns    chan *smtpConn
	mu       sync.RWMutex
	closed   bool
	created  int
	maxIdle  time.Duration
}

type smtpConn struct {
	client    *smtp.Client
	createdAt time.Time
	lastUsed  time.Time
}

// SMTPPoolConfig configures the SMTP connection pool
type SMTPPoolConfig struct {
	Host     string
	Port     int
	User     string
	Pass     string
	UseTLS   bool
	PoolSize int
	MaxIdle  time.Duration
}

// DefaultSMTPPoolConfig returns sensible defaults
func DefaultSMTPPoolConfig() SMTPPoolConfig {
	return SMTPPoolConfig{
		PoolSize: 10,
		MaxIdle:  5 * time.Minute,
		UseTLS:   true,
	}
}

// NewSMTPPool creates a new SMTP connection pool
func NewSMTPPool(config SMTPPoolConfig) *SMTPPool {
	if config.PoolSize <= 0 {
		config.PoolSize = 10
	}
	if config.MaxIdle <= 0 {
		config.MaxIdle = 5 * time.Minute
	}

	pool := &SMTPPool{
		host:     config.Host,
		port:     config.Port,
		user:     config.User,
		pass:     config.Pass,
		useTLS:   config.UseTLS,
		poolSize: config.PoolSize,
		conns:    make(chan *smtpConn, config.PoolSize),
		maxIdle:  config.MaxIdle,
	}

	// Start idle connection cleanup
	go pool.cleanupLoop()

	return pool
}

// Get retrieves a connection from the pool or creates a new one
func (p *SMTPPool) Get() (*smtp.Client, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("pool is closed")
	}
	p.mu.RUnlock()

	// Try to get from pool (non-blocking)
	select {
	case conn := <-p.conns:
		// Check if connection is still valid
		if time.Since(conn.lastUsed) > p.maxIdle {
			conn.client.Close()
			return p.createConnection()
		}
		// Test connection with NOOP
		if err := conn.client.Noop(); err != nil {
			conn.client.Close()
			return p.createConnection()
		}
		conn.lastUsed = time.Now()
		return conn.client, nil
	default:
		// Pool empty, create new connection
		return p.createConnection()
	}
}

// Put returns a connection to the pool
func (p *SMTPPool) Put(client *smtp.Client) {
	if client == nil {
		return
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		client.Close()
		return
	}
	p.mu.RUnlock()

	conn := &smtpConn{
		client:   client,
		lastUsed: time.Now(),
	}

	// Try to return to pool (non-blocking)
	select {
	case p.conns <- conn:
		// Successfully returned
	default:
		// Pool full, close connection
		client.Close()
	}
}

// Close closes all connections in the pool
func (p *SMTPPool) Close() {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()

	close(p.conns)
	for conn := range p.conns {
		conn.client.Close()
	}
}

// AccountError is the SMTP server refusing the sending account: it could
// not be reached, refused the sign-in, or refused the sender address. It is
// never a fault of the message or its recipient.
type AccountError struct{ Err error }

func (e *AccountError) Error() string { return e.Err.Error() }
func (e *AccountError) Unwrap() error { return e.Err }

// createConnection creates a new SMTP connection; every failure is an
// *AccountError.
func (p *SMTPPool) createConnection() (*smtp.Client, error) {
	client, err := p.dial()
	if err != nil {
		return nil, &AccountError{Err: err}
	}
	return client, nil
}

// dial connects, upgrades to TLS when asked, and signs in.
func (p *SMTPPool) dial() (*smtp.Client, error) {
	addr := fmt.Sprintf("%s:%d", p.host, p.port)

	var client *smtp.Client
	var err error

	if p.useTLS && p.port == 465 {
		// Implicit TLS (port 465)
		tlsConfig := &tls.Config{
			ServerName: p.host,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("TLS dial failed: %w", err)
		}
		client, err = smtp.NewClient(conn, p.host)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("SMTP client creation failed: %w", err)
		}
	} else {
		// Plain connection or STARTTLS
		conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
		if err != nil {
			return nil, fmt.Errorf("dial failed: %w", err)
		}
		client, err = smtp.NewClient(conn, p.host)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("SMTP client creation failed: %w", err)
		}

		// STARTTLS if supported and requested
		if p.useTLS {
			if ok, _ := client.Extension("STARTTLS"); ok {
				tlsConfig := &tls.Config{
					ServerName: p.host,
				}
				if err := client.StartTLS(tlsConfig); err != nil {
					client.Close()
					return nil, fmt.Errorf("STARTTLS failed: %w", err)
				}
			}
		}
	}

	// Authenticate
	if p.user != "" && p.pass != "" {
		auth := smtp.PlainAuth("", p.user, p.pass, p.host)
		if err = client.Auth(auth); err != nil {
			client.Close()
			return nil, fmt.Errorf("auth failed: %w", err)
		}
	}

	p.mu.Lock()
	p.created++
	p.mu.Unlock()

	return client, nil
}

// cleanupLoop periodically removes idle connections
func (p *SMTPPool) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.RLock()
		if p.closed {
			p.mu.RUnlock()
			return
		}
		p.mu.RUnlock()

		// Drain and re-add valid connections
		var valid []*smtpConn
		for {
			select {
			case conn := <-p.conns:
				if time.Since(conn.lastUsed) < p.maxIdle {
					valid = append(valid, conn)
				} else {
					conn.client.Close()
				}
			default:
				goto done
			}
		}
	done:
		for _, conn := range valid {
			select {
			case p.conns <- conn:
			default:
				conn.client.Close()
			}
		}
	}
}

// Stats returns pool statistics
func (p *SMTPPool) Stats() (pooled, created int) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.conns), p.created
}

// SendWithPool sends an email using a pooled connection
func (p *SMTPPool) SendWithPool(from string, to []string, msg []byte) error {
	client, err := p.Get()
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}

	// Reset the connection for new message
	if err := client.Reset(); err != nil {
		// Connection bad, close and retry with new connection
		client.Close()
		client, err = p.createConnection()
		if err != nil {
			return fmt.Errorf("failed to create new connection: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(from); err != nil {
		p.Put(client)
		return &AccountError{Err: fmt.Errorf("MAIL FROM failed: %w", err)}
	}

	// Set recipients
	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			client.Reset()
			p.Put(client)
			return fmt.Errorf("RCPT TO failed for %s: %w", addr, err)
		}
	}

	// Send data
	w, err := client.Data()
	if err != nil {
		client.Reset()
		p.Put(client)
		return fmt.Errorf("DATA failed: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		w.Close()
		client.Reset()
		p.Put(client)
		return fmt.Errorf("write failed: %w", err)
	}

	if err := w.Close(); err != nil {
		client.Reset()
		p.Put(client)
		return fmt.Errorf("close data writer failed: %w", err)
	}

	// Return connection to pool
	p.Put(client)

	logx.Debugf("Email sent via pooled connection to %v", to)
	return nil
}
