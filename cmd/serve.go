package cmd

import (
	"context"
	"crypto/tls"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/outlet-sh/outlet/app"
	"github.com/outlet-sh/outlet/internal/config"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/handler"
	publiclogic "github.com/outlet-sh/outlet/internal/logic/public"
	outletmcp "github.com/outlet-sh/outlet/internal/mcp"
	mcpoauth "github.com/outlet-sh/outlet/internal/mcp/oauth"
	"github.com/outlet-sh/outlet/internal/middleware"
	publicpages "github.com/outlet-sh/outlet/internal/public"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/webhook"
	"github.com/outlet-sh/outlet/internal/workers"
	outletsmtp "github.com/outlet-sh/outlet/internal/smtp"
	outletws "github.com/outlet-sh/outlet/internal/websocket"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"golang.org/x/crypto/acme/autocert"
)

// EmbeddedConfig holds the embedded configuration (set from main)
var EmbeddedConfig []byte

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Outlet.sh server",
	Long:  `Start the HTTP/HTTPS server for the Outlet.sh email platform.`,
	Run:   runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) {
	// Load .env file if it exists (silently ignore if not found)
	if err := godotenv.Load(); err != nil {
		log.Printf("Note: No .env file found, using environment variables")
	}

	var c config.Config

	// Use embedded config with env var expansion
	if err := conf.LoadFromYamlBytes([]byte(os.ExpandEnv(string(EmbeddedConfig))), &c); err != nil {
		fmt.Printf("Failed to load embedded config: %v\n", err)
		os.Exit(1)
	}

	// Validate required configuration
	if errors := c.Validate(); len(errors) > 0 {
		fmt.Println("Configuration errors:")
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
		os.Exit(1)
	}

	// Log warnings for optional config
	if warnings := c.ValidateAndWarn(); len(warnings) > 0 {
		fmt.Println("Configuration warnings:")
		for _, warn := range warnings {
			fmt.Printf("  - %s\n", warn)
		}
	}

	// Determine server host based on environment
	var srvHost = c.Host
	var serverPort = c.Port
	var useHTTPS = false

	// Check production mode from config or environment variable
	productionMode := c.App.ProductionMode
	if envProd := os.Getenv("PRODUCTION_MODE"); envProd == "true" || envProd == "1" {
		productionMode = true
	}

	// Behind a TLS-terminating proxy (a load balancer or an ingress), the
	// proxy owns HTTPS: serve plain HTTP on Port with the embedded SPA and
	// no Let's Encrypt listeners of our own.
	behindProxy := false
	if env := os.Getenv("BEHIND_PROXY"); env == "true" || env == "1" {
		behindProxy = true
	}

	if productionMode || behindProxy {
		if c.App.Domain == "" {
			fmt.Println("ERROR: App.Domain is required in production mode")
			os.Exit(1)
		}
		// Production mode - use domain name with HTTPS on standard port
		srvHost = c.App.Domain
		serverPort = 443
		useHTTPS = true
		fmt.Printf("Running in PRODUCTION mode - server.json will return https://%s\n", c.App.Domain)
	} else if serverPort == 443 || serverPort == 80 {
		// Fallback: check if running on standard HTTPS/HTTP ports
		if c.App.Domain == "" {
			fmt.Println("ERROR: App.Domain is required when using ports 80/443")
			os.Exit(1)
		}
		srvHost = c.App.Domain
		if serverPort == 443 {
			useHTTPS = true
		}
	} else {
		// Development mode - use localhost with port
		srvHost = "localhost"
		app.DevMode = true
		fmt.Printf("Running in DEVELOPMENT mode - server.json will return http://localhost:%d\n", serverPort)
	}

	fmt.Println("Server Host:", srvHost, "Port:", serverPort, "Use HTTPS:", useHTTPS)

	// Set server host for server.json
	app.SetServerHost(srvHost, serverPort, useHTTPS)

	// Set up SPA filesystem for static file serving (optional)
	spaFS, err := app.FileSystem()
	if err != nil {
		fmt.Printf("SPA not embedded: %v\n", err)
		fmt.Println("Frontend should be running separately (e.g., pnpm dev on port 5173)")
	}

	// Setup JSON error responses (must be before server creation)
	errorx.SetupErrorHandler()

	// Create go-zero server with optional embedded SPA
	var serverOpts []rest.RunOption
	if err == nil {
		serverOpts = append(serverOpts,
			rest.WithFileServer("/", http.FS(spaFS)),
			rest.WithNotFoundHandler(app.NotFoundHandler(spaFS)),
		)
	}
	server := rest.MustNewServer(c.RestConf, serverOpts...)
	defer server.Stop()

	// Initialize service context (DB, external services)
	ctx := svc.NewServiceContext(c)

	// Register generated API handlers
	handler.RegisterHandlers(server, ctx)

	// Root-level health check (for dosync/Docker health checks)
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/health",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		},
	})

	// Custom handlers for email tracking
	registerEmailTrackingRoutes(server, ctx)

	// Custom webhook handlers (need raw body access)
	registerWebhookRoutes(server, ctx)

	// MCP OAuth endpoints (well-known, DCR, authorize, token)
	registerMCPOAuthRoutes(server, ctx, c.App.BaseURL)

	// MCP (Model Context Protocol) server for AI integrations
	registerMCPRoutes(server, ctx, c.App.BaseURL)

	// Public pages (subscribe, confirm, unsubscribe, web view)
	registerPublicPageRoutes(server, ctx)

	// WebSocket endpoint for real-time updates
	registerWebSocketRoute(server, ctx)

	// Start email worker in background
	emailWorker := workers.StartEmailWorker(ctx)
	fmt.Println("Email worker started")

	// Start campaign scheduler in background
	campaignScheduler := workers.StartCampaignScheduler(ctx)
	fmt.Println("Campaign scheduler started")

	// Start domain verification worker in background
	domainVerificationWorker := workers.StartDomainVerificationWorker(ctx)
	fmt.Println("Domain verification worker started")

	// Start MCP session cleanup job (runs every hour, cleans sessions older than 30 days)
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		// Run cleanup immediately on startup
		if err := ctx.DB.CleanupOldMCPSessions(cleanupCtx); err != nil {
			fmt.Printf("[MCP] Initial session cleanup failed: %v\n", err)
		} else {
			fmt.Println("[MCP] Initial session cleanup completed")
		}

		for {
			select {
			case <-cleanupCtx.Done():
				fmt.Println("[MCP] Session cleanup job stopped")
				return
			case <-ticker.C:
				if err := ctx.DB.CleanupOldMCPSessions(cleanupCtx); err != nil {
					fmt.Printf("[MCP] Session cleanup failed: %v\n", err)
				} else {
					fmt.Println("[MCP] Session cleanup completed")
				}
			}
		}
	}()
	fmt.Println("[MCP] Session cleanup job started (runs every hour)")

	// Start SMTP ingress server if enabled
	var smtpServer *outletsmtp.Server
	if c.SMTP.IsEnabled() {
		smtpConfig := c.SMTP
		// Use app domain if SMTP domain not set
		if smtpConfig.Domain == "" {
			smtpConfig.Domain = c.App.Domain
		}
		smtpServer = outletsmtp.NewServer(ctx, smtpConfig)
		if err := smtpServer.Start(); err != nil {
			fmt.Printf("Warning: Failed to start SMTP server: %v\n", err)
		} else {
			fmt.Printf("SMTP ingress server started on port %d\n", smtpConfig.GetPort())
		}
	}

	// In development mode, and behind a TLS-terminating proxy, run only the
	// go-zero server, with graceful shutdown
	if app.DevMode || behindProxy {
		fmt.Printf("Starting go-zero backend server on %s:%d...\n", c.Host, c.Port)

		// Handle graceful shutdown in dev mode
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-quit
			fmt.Println("\nShutting down workers gracefully...")

			// Stop MCP cleanup job
			cleanupCancel()
			fmt.Println("MCP session cleanup job stopped")

			// Stop workers
			if emailWorker != nil {
				emailWorker.Stop()
				fmt.Println("Email worker stopped")
			}
			if campaignScheduler != nil {
				campaignScheduler.Stop()
				fmt.Println("Campaign scheduler stopped")
			}
			if domainVerificationWorker != nil {
				domainVerificationWorker.Stop()
				fmt.Println("Domain verification worker stopped")
			}
			if smtpServer != nil {
				smtpServer.Stop()
				fmt.Println("SMTP server stopped")
			}

			// Stop server
			server.Stop()
		}()

		server.Start()
		return
	}

	// Production mode: Start go-zero server in background, then HTTPS/HTTP servers
	go func() {
		fmt.Printf("Starting go-zero backend server on %s:%d...\n", c.Host, c.Port)
		server.Start()
	}()

	// Set up autocert for Let's Encrypt
	certManager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache("certs"),
		HostPolicy: autocert.HostWhitelist(c.App.Domain, "www."+c.App.Domain),
		Email:      "admin@" + c.App.Domain,
	}

	// Create reverse proxy to go-zero backend with connection pooling
	backendURL, _ := url.Parse(fmt.Sprintf("http://%s:%d", c.Host, c.Port))
	proxy := httputil.NewSingleHostReverseProxy(backendURL)

	// Modify director to preserve WebSocket headers
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		if req.Header.Get("Upgrade") != "" {
			req.Header.Set("Connection", "Upgrade")
		}
	}

	// Configure transport for optimal performance
	proxy.Transport = &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
		WriteBufferSize:     32 << 10,
		ReadBufferSize:      32 << 10,
	}

	// Add error handler for backend failures
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		fmt.Printf("Proxy error: %v\n", err)
		http.Error(w, "Backend temporarily unavailable", http.StatusBadGateway)
	}

	// HTTP handler for port 80 - ACME challenges and HTTPS redirect
	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/.well-known/acme-challenge/") {
			certManager.HTTPHandler(nil).ServeHTTP(w, r)
			return
		}
		host, _ := strings.CutPrefix(r.Host, "www.")
		newURL := fmt.Sprintf("https://%s%s", host, r.RequestURI)
		http.Redirect(w, r, newURL, http.StatusMovedPermanently)
	})

	// HTTPS handler for port 443
	baseHTTPSHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// www → non-www redirect
		if nonWWWHost, hadPrefix := strings.CutPrefix(r.Host, "www."); hadPrefix {
			newURL := fmt.Sprintf("https://%s%s", nonWWWHost, r.RequestURI)
			http.Redirect(w, r, newURL, http.StatusMovedPermanently)
			return
		}

		// Route health check to backend
		if r.URL.Path == "/health" {
			proxy.ServeHTTP(w, r)
			return
		}

		// Route API requests to backend
		if strings.HasPrefix(r.URL.Path, "/api/") {
			proxy.ServeHTTP(w, r)
			return
		}

		// Route SDK requests to backend
		if strings.HasPrefix(r.URL.Path, "/sdk/") {
			proxy.ServeHTTP(w, r)
			return
		}

		// Route webhook requests to backend
		if strings.HasPrefix(r.URL.Path, "/webhooks/") {
			proxy.ServeHTTP(w, r)
			return
		}

		// Route WebSocket requests to backend with proper upgrade handling
		if strings.HasPrefix(r.URL.Path, "/ws") {
			proxyWebSocket(w, r, c.Host, c.Port)
			return
		}

		// Route download requests to backend
		if strings.HasPrefix(r.URL.Path, "/downloads/") {
			proxy.ServeHTTP(w, r)
			return
		}

		// Route MCP requests to backend
		if strings.HasPrefix(r.URL.Path, "/mcp") {
			proxy.ServeHTTP(w, r)
			return
		}

		// Route public pages to backend (subscribe, confirm, unsubscribe, web view)
		if strings.HasPrefix(r.URL.Path, "/s/") ||
			strings.HasPrefix(r.URL.Path, "/u/") ||
			strings.HasPrefix(r.URL.Path, "/w/") ||
			strings.HasPrefix(r.URL.Path, "/confirm/") {
			proxy.ServeHTTP(w, r)
			return
		}

		// Route OAuth well-known endpoints to backend (for MCP OAuth discovery)
		if strings.HasPrefix(r.URL.Path, "/.well-known/oauth-") ||
			r.URL.Path == "/.well-known/openid-configuration" {
			proxy.ServeHTTP(w, r)
			return
		}

		// Route default OAuth endpoints to backend (MCP spec fallback paths)
		if r.URL.Path == "/register" || r.URL.Path == "/authorize" || r.URL.Path == "/token" {
			proxy.ServeHTTP(w, r)
			return
		}

		// Serve static files with SPA fallback
		if err == nil {
			app.SPAHandler(spaFS).ServeHTTP(w, r)
		} else {
			http.Error(w, "SPA not available", http.StatusServiceUnavailable)
		}
	})

	// Layer middlewares: Gzip → CacheControl → Handler
	httpsHandler := middleware.Gzip(middleware.CacheControl(baseHTTPSHandler))

	// Start HTTP server on port 80
	httpServer := &http.Server{
		Addr:         ":80",
		Handler:      httpHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		fmt.Println("Starting HTTP server on :80 for ACME challenges and HTTPS redirect...")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	// HTTPS handler
	combinedHandler := httpsHandler

	// TLS configuration for Let's Encrypt with HTTP/2 enabled
	tlsConfig := &tls.Config{
		GetCertificate: certManager.GetCertificate,
		MinVersion:     tls.VersionTLS12,
		NextProtos:     []string{"h2", "http/1.1", "acme-tls/1"},
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
	}

	// HTTPS server with HTTP/2 support
	httpsServer := &http.Server{
		Addr:              ":443",
		Handler:           combinedHandler,
		TLSConfig:         tlsConfig,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	fmt.Println("Starting HTTPS server on :443 with Let's Encrypt auto-certificate...")
	fmt.Println("HTTP/2 enabled")
	fmt.Println("Auto-redirect: www -> non-www, HTTP -> HTTPS")
	fmt.Println("API routes: /api/* (proxied to backend)")
	fmt.Println("Static SPA: /* (served directly from embedded FS)")

	// Start HTTPS server with TLS (ListenAndServeTLS auto-configures HTTP/2)
	go func() {
		if err := httpsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTPS server error: %v\n", err)
		}
	}()

	// Graceful shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down servers gracefully...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop workers first (allow them to finish current work)
	fmt.Println("Stopping workers...")

	// Stop MCP cleanup job
	cleanupCancel()
	fmt.Println("MCP session cleanup job stopped")

	if emailWorker != nil {
		emailWorker.Stop()
		fmt.Println("Email worker stopped")
	}
	if campaignScheduler != nil {
		campaignScheduler.Stop()
		fmt.Println("Campaign scheduler stopped")
	}
	if domainVerificationWorker != nil {
		domainVerificationWorker.Stop()
		fmt.Println("Domain verification worker stopped")
	}
	if smtpServer != nil {
		smtpServer.Stop()
		fmt.Println("SMTP server stopped")
	}

	// Shutdown HTTPS server
	if err := httpsServer.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("HTTPS server forced to shutdown: %v\n", err)
	} else {
		fmt.Println("HTTPS server stopped")
	}

	// Shutdown HTTP server (port 80)
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("HTTP server forced to shutdown: %v\n", err)
	} else {
		fmt.Println("HTTP server stopped")
	}

	fmt.Println("All servers shut down successfully")
}

// registerEmailTrackingRoutes adds email open/click tracking endpoints
func registerEmailTrackingRoutes(server *rest.Server, ctx *svc.ServiceContext) {
	// Email open tracking - returns 1x1 transparent pixel
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/api/e/o/:token",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			token := r.PathValue("token")
			if token == "" {
				parts := strings.Split(r.URL.Path, "/")
				if len(parts) >= 4 {
					token = parts[len(parts)-1]
				}
			}

			// Track the open (fire and forget)
			go func(token string) {
				_ = ctx.Tracking.RecordOpen(context.Background(), token)
			}(token)

			// Return 1x1 transparent GIF
			w.Header().Set("Content-Type", "image/gif")
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			transparentGIF := []byte{
				0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00,
				0x80, 0x00, 0x00, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x21,
				0xf9, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, 0x2c, 0x00, 0x00,
				0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44,
				0x01, 0x00, 0x3b,
			}
			w.Write(transparentGIF)
		},
	})

	// Email click tracking - tracks click and redirects
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/api/e/c/:token",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			redirectUrl := r.URL.Query().Get("url")
			if redirectUrl == "" {
				http.Error(w, "Missing url parameter", http.StatusBadRequest)
				return
			}

			token := r.PathValue("token")
			if token == "" {
				parts := strings.Split(r.URL.Path, "/")
				if len(parts) >= 4 {
					token = parts[len(parts)-1]
				}
			}

			// Track the click (fire and forget)
			go func(token string) {
				_ = ctx.Tracking.RecordClick(context.Background(), token)
			}(token)

			http.Redirect(w, r, redirectUrl, http.StatusTemporaryRedirect)
		},
	})

	// Email unsubscribe
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/api/e/u/:token",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			token := r.PathValue("token")
			if token == "" {
				parts := strings.Split(r.URL.Path, "/")
				if len(parts) >= 4 {
					token = parts[len(parts)-1]
				}
			}

			if token == "" {
				http.Error(w, "Invalid unsubscribe link", http.StatusBadRequest)
				return
			}

			_ = ctx.Tracking.Unsubscribe(context.Background(), token)
			http.Redirect(w, r, "/unsubscribed", http.StatusTemporaryRedirect)
		},
	})

	// Confirm email (double opt-in)
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/api/confirm-email",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			token := r.URL.Query().Get("token")
			if token == "" {
				http.Redirect(w, r, "/confirm-expired", http.StatusTemporaryRedirect)
				return
			}

			logic := publiclogic.NewConfirmEmailLogic(r.Context(), ctx)
			resp, err := logic.ConfirmEmail(&types.ConfirmEmailRequest{Token: token})
			if err != nil {
				http.Redirect(w, r, "/confirm-expired", http.StatusTemporaryRedirect)
				return
			}

			if resp.Redirect != "" {
				http.Redirect(w, r, resp.Redirect, http.StatusTemporaryRedirect)
			} else if resp.Success {
				http.Redirect(w, r, "/thank-you?confirmed=1", http.StatusTemporaryRedirect)
			} else {
				http.Redirect(w, r, "/confirm-expired", http.StatusTemporaryRedirect)
			}
		},
	})

	// Serve downloadable files
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/downloads/:filename",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			filename := r.PathValue("filename")
			if filename == "" {
				parts := strings.Split(r.URL.Path, "/")
				if len(parts) >= 2 {
					filename = parts[len(parts)-1]
				}
			}

			// Sanitize filename
			filename = strings.ReplaceAll(filename, "..", "")
			filename = strings.ReplaceAll(filename, "/", "")
			filename = strings.ReplaceAll(filename, "\\", "")

			filepath := fmt.Sprintf("/data/downloads/%s", filename)
			http.ServeFile(w, r, filepath)
		},
	})
}

// registerWebhookRoutes adds webhook handlers that need raw body access
func registerWebhookRoutes(server *rest.Server, ctx *svc.ServiceContext) {
	// SES webhook (for bounces/complaints) - includes org ID in path
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/webhooks/ses/:orgId",
		Handler: webhook.SESHandler(ctx),
	})
}

// registerMCPOAuthRoutes adds MCP OAuth 2.1 endpoints for ChatGPT and other AI integrations
func registerMCPOAuthRoutes(server *rest.Server, ctx *svc.ServiceContext, baseURL string) {
	oauthHandler := mcpoauth.NewHandler(ctx, baseURL)

	// Well-known metadata endpoints
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/.well-known/oauth-protected-resource",
		Handler: http.HandlerFunc(oauthHandler.HandleProtectedResourceMetadata),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/.well-known/oauth-protected-resource/mcp",
		Handler: http.HandlerFunc(oauthHandler.HandleProtectedResourceMetadata),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/.well-known/oauth-authorization-server",
		Handler: http.HandlerFunc(oauthHandler.HandleAuthServerMetadata),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/.well-known/oauth-authorization-server/mcp",
		Handler: http.HandlerFunc(oauthHandler.HandleAuthServerMetadata),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/mcp/.well-known/oauth-protected-resource",
		Handler: http.HandlerFunc(oauthHandler.HandleProtectedResourceMetadata),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/mcp/.well-known/oauth-authorization-server",
		Handler: http.HandlerFunc(oauthHandler.HandleAuthServerMetadata),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/mcp/.well-known/openid-configuration",
		Handler: http.HandlerFunc(oauthHandler.HandleAuthServerMetadata),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/.well-known/openid-configuration",
		Handler: http.HandlerFunc(oauthHandler.HandleAuthServerMetadata),
	})

	// Default MCP spec fallback paths
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/register",
		Handler: http.HandlerFunc(oauthHandler.HandleClientRegistration),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodOptions,
		Path:    "/register",
		Handler: http.HandlerFunc(oauthHandler.HandleClientRegistration),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/authorize",
		Handler: http.HandlerFunc(oauthHandler.HandleAuthorize),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/authorize",
		Handler: http.HandlerFunc(oauthHandler.HandleAuthorizeSubmit),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/token",
		Handler: http.HandlerFunc(oauthHandler.HandleToken),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodOptions,
		Path:    "/token",
		Handler: http.HandlerFunc(oauthHandler.HandleToken),
	})

	// OPTIONS handlers for CORS preflight
	server.AddRoute(rest.Route{
		Method:  http.MethodOptions,
		Path:    "/.well-known/oauth-protected-resource",
		Handler: http.HandlerFunc(oauthHandler.HandleProtectedResourceMetadata),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodOptions,
		Path:    "/.well-known/oauth-authorization-server",
		Handler: http.HandlerFunc(oauthHandler.HandleAuthServerMetadata),
	})

	// JWKS endpoint
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/mcp/oauth/jwks",
		Handler: http.HandlerFunc(oauthHandler.HandleJWKS),
	})

	// Primary OAuth endpoints under /mcp/oauth
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/mcp/oauth/register",
		Handler: http.HandlerFunc(oauthHandler.HandleClientRegistration),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/mcp/oauth/authorize",
		Handler: http.HandlerFunc(oauthHandler.HandleAuthorize),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/mcp/oauth/authorize",
		Handler: http.HandlerFunc(oauthHandler.HandleAuthorizeSubmit),
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/mcp/oauth/token",
		Handler: http.HandlerFunc(oauthHandler.HandleToken),
	})
}

// proxyWebSocket handles WebSocket upgrade and bidirectional proxying
func proxyWebSocket(w http.ResponseWriter, r *http.Request, backendHost string, backendPort int) {
	fmt.Printf("[WS Proxy] Incoming WebSocket request: %s\n", r.URL.String())

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		fmt.Println("[WS Proxy] ERROR: ResponseWriter does not support Hijack")
		http.Error(w, "WebSocket not supported", http.StatusInternalServerError)
		return
	}

	backendAddr := fmt.Sprintf("%s:%d", backendHost, backendPort)
	fmt.Printf("[WS Proxy] Dialing backend: %s\n", backendAddr)
	backendConn, err := net.Dial("tcp", backendAddr)
	if err != nil {
		fmt.Printf("[WS Proxy] ERROR: Failed to dial backend: %v\n", err)
		http.Error(w, "Backend unavailable", http.StatusBadGateway)
		return
	}
	defer backendConn.Close()

	clientConn, clientBuf, err := hijacker.Hijack()
	if err != nil {
		fmt.Printf("[WS Proxy] ERROR: Hijack failed: %v\n", err)
		http.Error(w, "Hijack failed", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()
	fmt.Println("[WS Proxy] Connection hijacked successfully")

	if err := r.Write(backendConn); err != nil {
		fmt.Printf("[WS Proxy] ERROR: Failed to forward request: %v\n", err)
		return
	}
	fmt.Println("[WS Proxy] Request forwarded to backend")

	if clientBuf.Reader.Buffered() > 0 {
		buffered := make([]byte, clientBuf.Reader.Buffered())
		clientBuf.Read(buffered)
		backendConn.Write(buffered)
	}

	done := make(chan struct{}, 2)
	go func() {
		io.Copy(backendConn, clientConn)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(clientConn, backendConn)
		done <- struct{}{}
	}()
	<-done
	fmt.Println("[WS Proxy] Connection closed")
}

// registerMCPRoutes adds MCP (Model Context Protocol) endpoints for AI integrations.
func registerMCPRoutes(server *rest.Server, ctx *svc.ServiceContext, baseURL string) {
	mcpHandler := outletmcp.NewHandler(ctx, baseURL)

	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/mcp",
		Handler: mcpHandler.ServeHTTP,
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/mcp",
		Handler: mcpHandler.ServeHTTP,
	})

	fmt.Println("MCP endpoints registered at /mcp (requires X-API-Key header)")
}

// registerPublicPageRoutes adds public pages for subscribe, confirm, unsubscribe, and web view.
func registerPublicPageRoutes(server *rest.Server, ctx *svc.ServiceContext) {
	publicHandler, err := publicpages.NewHandler(ctx)
	if err != nil {
		fmt.Printf("Warning: Failed to initialize public pages handler: %v\n", err)
		return
	}

	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/s/:slug",
		Handler: publicHandler.HandleSubscribe,
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/s/:slug",
		Handler: publicHandler.HandleSubscribe,
	})

	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/confirm/:token",
		Handler: publicHandler.HandleConfirm,
	})

	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/u/:token",
		Handler: publicHandler.HandleUnsubscribe,
	})
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/u/:token",
		Handler: publicHandler.HandleUnsubscribe,
	})

	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/w/:token",
		Handler: publicHandler.HandleWebView,
	})

	fmt.Println("Public pages registered: /s/:slug, /confirm/:token, /u/:token, /w/:token")
}

// registerWebSocketRoute adds the WebSocket endpoint for real-time updates
func registerWebSocketRoute(server *rest.Server, ctx *svc.ServiceContext) {
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/ws",
		Handler: outletws.Handler(ctx.WebSocketHub),
	})
	fmt.Println("WebSocket endpoint registered at /ws")
}
