package svc

import (
	"context"
	"database/sql"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/outlet-sh/outlet/internal/config"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/events"
	"github.com/outlet-sh/outlet/internal/middleware"
	"github.com/outlet-sh/outlet/internal/services/crypto"
	"github.com/outlet-sh/outlet/internal/services/email"
	"github.com/outlet-sh/outlet/internal/services/outbound"
	"github.com/outlet-sh/outlet/internal/services/signalnet"
	"github.com/outlet-sh/outlet/internal/services/tracking"
	"github.com/outlet-sh/outlet/internal/services/validation"
	"github.com/outlet-sh/outlet/internal/services/webhook"
	"github.com/outlet-sh/outlet/internal/websocket"

	"github.com/zeromicro/go-zero/rest"
	_ "modernc.org/sqlite"
)

type ServiceContext struct {
	Config            config.Config
	DB                *db.Store
	Auth              rest.Middleware
	APIKeyAuth        rest.Middleware
	APIKeyMiddleware  *middleware.APIKeyMiddleware
	AuthRateLimit     rest.Middleware
	EmailService      *email.Service
	Sequences         *email.SequenceService
	CryptoService     *crypto.Service
	Tracking          *tracking.Service
	Events            *events.Subject
	WebhookDispatcher *webhook.Dispatcher
	WebSocketHub      *websocket.Hub
	Outbound          *outbound.Service

	// Signal Network hub (nil unless SignalNet.Hub is true; its routes then 404)
	NetworkHub  *signalnet.Hub
	NetworkAuth rest.Middleware
	// Email validation
	SignalNet  *signalnet.Client // nil when Signal Network is not configured
	Validation *validation.Service

	// UploadDir holds uploaded files waiting for a worker (CSV imports), next
	// to the database
	UploadDir string
}

func NewServiceContext(c config.Config) *ServiceContext {
	// Initialize SQLite database connection
	dbPath := c.Database.Path
	if dbPath == "" {
		dbPath = "./data/outlet.db"
	}

	// Ensure directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	// Open SQLite with WAL, foreign keys and a busy timeout. They go in the
	// DSN so the driver applies them to every pooled connection; a PRAGMA run
	// through conn.Exec reaches only whichever connection served it, and the
	// others then fail with SQLITE_BUSY instead of waiting.
	conn, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Test database connection
	if err := conn.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Printf("SQLite database connected: %s", dbPath)

	// Initialize Crypto service for credential encryption (must be before email service)
	var cryptoService *crypto.Service
	if c.Encryption.Key != "" {
		var err error
		cryptoService, err = crypto.NewService(c.Encryption.Key)
		if err != nil {
			log.Printf("Warning: Failed to initialize crypto service: %v", err)
		} else {
			log.Printf("Crypto service initialized successfully")
		}
	} else {
		log.Printf("Warning: ENCRYPTION_KEY not set - credential storage (Stripe, AI keys) will be disabled")
	}

	// Run database migrations (non-destructive - only applies pending
	// migrations). A migration that moves credentials encrypts them.
	log.Printf("Running database migrations...")
	var cipher migrations.Cipher
	if cryptoService != nil {
		cipher = cryptoService
	}
	if err := migrations.Run(conn, cipher); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}
	log.Printf("Database migrations completed")

	// Create database store
	store := db.NewStore(conn)

	// Initialize Email service (AWS SES preferred, SMTP fallback - config from platform_settings)
	emailService := email.NewService(store, cryptoService)
	emailService.SetLinkSecret(c.Auth.AccessSecret)
	if c.App.BaseURL != "" {
		emailService.SetBaseURL(c.App.BaseURL)
	}

	// Initialize rate limit middleware for auth endpoints
	authRateLimiter := middleware.NewRateLimitMiddleware(middleware.DefaultAuthRateLimitConfig())

	// Initialize API key middleware with cache
	apiKeyMiddleware := middleware.NewAPIKeyMiddleware(store)

	// Initialize Event bus for webhooks and automation
	eventSubject := events.NewSubject(
		events.WithBufferSize(1024),
		events.WithReplay(100),
	)
	log.Printf("Event bus initialized")
	// The store publishes contact and email events from the writes that make them
	store.SetEvents(eventSubject)

	// Initialize Tracking service (publishes opens and clicks on the event bus)
	trackingService := tracking.New(store, eventSubject)

	// Initialize and start Webhook Dispatcher for outbound webhook delivery
	webhookDispatcher := webhook.NewDispatcher(store.Queries, eventSubject)
	if err := webhookDispatcher.Start(context.Background()); err != nil {
		log.Printf("Warning: Failed to start webhook dispatcher: %v", err)
	} else {
		log.Printf("Webhook dispatcher started")
	}

	// Initialize WebSocket Hub for real-time updates
	wsHub := websocket.NewHub()
	go wsHub.Run()
	log.Printf("WebSocket hub initialized")

	var networkHub *signalnet.Hub
	if c.SignalNet.Hub {
		networkHub = signalnet.NewHub(store)
		log.Printf("Signal Network hub enabled at /network/v1")
	}
	// Email validation, with the Signal Network client when a hub is configured
	var signalNet *signalnet.Client
	if c.SignalNet.HubURL != "" {
		signalNet = &signalnet.Client{HubURL: c.SignalNet.HubURL, InstanceKey: c.SignalNet.InstanceKey}
	}
	validationService := validation.New(store, c.Validation, signalNet)
	sequences := email.NewSequenceServiceWithBaseURL(store, emailService, c.App.BaseURL)

	return &ServiceContext{
		Config:            c,
		DB:                store,
		UploadDir:         filepath.Join(dbDir, "uploads"),
		Auth:              middleware.NewAuthMiddleware(c.Auth.AccessSecret, store).Handle,
		APIKeyAuth:        apiKeyMiddleware.Handle,
		APIKeyMiddleware:  apiKeyMiddleware,
		AuthRateLimit:     authRateLimiter.Handle,
		EmailService:      emailService,
		Sequences:         sequences,
		CryptoService:     cryptoService,
		Tracking:          trackingService,
		Events:            eventSubject,
		WebhookDispatcher: webhookDispatcher,
		WebSocketHub:      wsHub,
		Outbound: outbound.New(store, cryptoService, outbound.Config{
			ProtectedDomains: strings.Split(c.Outbound.ProtectedDomains, ","),
			TrackingTarget:   publicHost(c.App.BaseURL),
			Mailer:           emailService,
			Sequences:        sequences,
			Verify: func(ctx context.Context, orgID, address string) (bool, string) {
				res, err := validationService.Validate(ctx, orgID, address, "outbound")
				if err != nil {
					return false, err.Error()
				}
				// Only a confirmed address is sent to; catch-all and unknown are held.
				if res.Status != "valid" {
					return false, strings.TrimSpace(res.Status + " " + res.SubStatus)
				}
				return true, "valid"
			},
		}),
		NetworkHub:  networkHub,
		NetworkAuth: middleware.NewNetworkAuthMiddleware(networkHub).Handle,
		SignalNet:   signalNet,
		Validation:  validationService,
	}
}

// publicHost is the host of the server's public URL.
func publicHost(baseURL string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}
