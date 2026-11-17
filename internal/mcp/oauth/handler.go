package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/errorx"
	"github.com/outlet-sh/outlet/internal/logic/auth"
	"github.com/outlet-sh/outlet/internal/mcp/mcpauth"
	"github.com/outlet-sh/outlet/internal/svc"
)

// Handler provides MCP OAuth endpoints
type Handler struct {
	svc     *svc.ServiceContext
	baseURL string
}

// NewHandler creates a new MCP OAuth handler
func NewHandler(svc *svc.ServiceContext, baseURL string) *Handler {
	return &Handler{
		svc:     svc,
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

// RegisterRoutes registers all OAuth routes on the given mux
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Well-known metadata endpoints
	mux.HandleFunc("GET /.well-known/oauth-protected-resource", h.HandleProtectedResourceMetadata)
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", h.HandleAuthServerMetadata)

	// Dynamic Client Registration
	mux.HandleFunc("POST /mcp/oauth/register", h.HandleClientRegistration)

	// OAuth endpoints
	mux.HandleFunc("GET /mcp/oauth/authorize", h.HandleAuthorize)
	mux.HandleFunc("POST /mcp/oauth/authorize", h.HandleAuthorizeSubmit)
	mux.HandleFunc("POST /mcp/oauth/authorize/code", h.HandleAuthorizeCode)
	mux.HandleFunc("GET /mcp/oauth/authorize/link", h.HandleAuthorizeLink)
	mux.HandleFunc("POST /mcp/oauth/authorize/link", h.HandleAuthorizeLinkSubmit)
	mux.HandleFunc("POST /mcp/oauth/token", h.HandleToken)
}

// setCORSHeaders sets CORS headers for OAuth endpoints (browser-based clients like ChatGPT)
func (h *Handler) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Max-Age", "86400")
}

// handleProtectedResourceMetadata returns the protected resource metadata
// Per RFC 9728, authorization_servers points to the OAuth server (at root, not /mcp)
func (h *Handler) HandleProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	metadata := ProtectedResourceMetadata{
		Resource:               h.baseURL + "/mcp",
		AuthorizationServers:   []string{h.baseURL}, // Root URL, not /mcp
		ScopesSupported:        []string{"mcp:full"},
		BearerMethodsSupported: []string{"header"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	json.NewEncoder(w).Encode(metadata)
}

// handleAuthServerMetadata returns the authorization server metadata
// Uses root-level URLs for issuer and endpoints (matching working MCP OAuth servers)
func (h *Handler) HandleAuthServerMetadata(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	metadata := AuthorizationServerMetadata{
		Issuer:                            h.baseURL, // Root URL, not /mcp
		AuthorizationEndpoint:             h.baseURL + "/authorize",
		TokenEndpoint:                     h.baseURL + "/token",
		RegistrationEndpoint:              h.baseURL + "/register",
		ScopesSupported:                   []string{"mcp:full", "offline_access"},
		ResponseTypesSupported:            []string{"code"},
		ResponseModesSupported:            []string{"query"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post", "none"},
		CodeChallengeMethodsSupported:     []string{"S256"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	json.NewEncoder(w).Encode(metadata)
}

// HandleJWKS returns an empty JWKS (we use opaque tokens, not JWTs)
func (h *Handler) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	jwks := map[string]interface{}{
		"keys": []interface{}{},
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	json.NewEncoder(w).Encode(jwks)
}

// handleClientRegistration handles Dynamic Client Registration (DCR)
func (h *Handler) HandleClientRegistration(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req ClientRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if req.ClientName == "" {
		h.sendError(w, http.StatusBadRequest, ErrInvalidRequest, "client_name is required")
		return
	}
	if len(req.RedirectURIs) == 0 {
		h.sendError(w, http.StatusBadRequest, ErrInvalidRequest, "redirect_uris is required")
		return
	}
	for _, uri := range req.RedirectURIs {
		if !isAllowedRedirectURI(uri) {
			h.sendError(w, http.StatusBadRequest, ErrInvalidRequest, "redirect_uri must be https, http on localhost, or an app scheme: "+uri)
			return
		}
	}

	// Generate client credentials
	clientID := generateSecureToken(16)
	clientSecret := generateSecureToken(32)
	clientSecretHash := mcpauth.HashToken(clientSecret)

	// Set defaults
	if len(req.GrantTypes) == 0 {
		req.GrantTypes = []string{"authorization_code", "refresh_token"}
	}
	if len(req.ResponseTypes) == 0 {
		req.ResponseTypes = []string{"code"}
	}
	if req.TokenEndpointAuthMethod == "" {
		req.TokenEndpointAuthMethod = "client_secret_post"
	}
	if req.Scope == "" {
		req.Scope = "mcp:full"
	}

	// Store client in database
	isConfidential := int64(0)
	if req.TokenEndpointAuthMethod != "none" {
		isConfidential = 1
	}
	_, err := h.svc.DB.CreateMCPOAuthClient(r.Context(), db.CreateMCPOAuthClientParams{
		ID:               generateSecureToken(16),
		ClientID:         clientID,
		ClientSecretHash: clientSecretHash,
		Name:             req.ClientName,
		Description:      sql.NullString{String: "", Valid: false},
		RedirectUris:     strings.Join(req.RedirectURIs, ","),
		Scopes:           sql.NullString{String: req.Scope, Valid: req.Scope != ""},
		IsConfidential:   sql.NullInt64{Int64: isConfidential, Valid: true},
	})
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, ErrServerError, "Failed to register client")
		return
	}

	resp := ClientRegistrationResponse{
		ClientID:                clientID,
		ClientSecret:            clientSecret,
		ClientName:              req.ClientName,
		RedirectURIs:            req.RedirectURIs,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		GrantTypes:              req.GrantTypes,
		ResponseTypes:           req.ResponseTypes,
		Scope:                   req.Scope,
		ClientIDIssuedAt:        time.Now().Unix(),
		ClientSecretExpiresAt:   0, // Never expires
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// handleAuthorize shows the login/authorization page
func (h *Handler) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	req := AuthorizationRequest{
		ResponseType:        query.Get("response_type"),
		ClientID:            query.Get("client_id"),
		RedirectURI:         query.Get("redirect_uri"),
		Scope:               query.Get("scope"),
		State:               query.Get("state"),
		CodeChallenge:       query.Get("code_challenge"),
		CodeChallengeMethod: query.Get("code_challenge_method"),
	}

	client, ok := h.checkAuthorizeRequest(w, r, req)
	if !ok {
		return
	}

	// The checked request stays on the server from here on; the pages carry
	// only its handle. The cookie marks this browser, so the emailed link
	// finishes the sign-in only where it started.
	browser := h.browserSecret(w, r)
	handle := generateSecureToken(32)
	_ = h.svc.DB.DeleteExpiredMCPOAuthRequests(r.Context())
	if err := h.svc.DB.CreateMCPOAuthRequest(r.Context(), db.CreateMCPOAuthRequestParams{
		ID:                  generateSecureToken(16),
		RequestHash:         mcpauth.HashToken(handle),
		BrowserHash:         mcpauth.HashToken(browser),
		ClientID:            req.ClientID,
		RedirectUri:         req.RedirectURI,
		Scope:               req.Scope,
		State:               req.State,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		ExpiresAt:           time.Now().UTC().Add(auth.SignInTTL).Format(time.RFC3339),
	}); err != nil {
		h.sendError(w, http.StatusInternalServerError, ErrServerError, "Failed to start sign-in")
		return
	}

	h.renderPage(w, http.StatusOK, "email", pageData{ClientName: client.Name, Request: handle})
}

// browserCookie names the cookie that marks the browser a sign-in started
// in. Its value is a random secret; a sign-in in progress keeps its hash.
const browserCookie = "outlet_mcp_browser"

// browserSecret is this browser's sign-in secret: the one its cookie already
// holds, or a new one, set (again) for as long as a sign-in lasts.
func (h *Handler) browserSecret(w http.ResponseWriter, r *http.Request) string {
	secret := ""
	if c, err := r.Cookie(browserCookie); err == nil && len(c.Value) >= 32 {
		secret = c.Value
	} else {
		secret = generateSecureToken(32)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     browserCookie,
		Value:    secret,
		Path:     "/",
		MaxAge:   int(auth.SignInTTL.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil || strings.HasPrefix(h.baseURL, "https://"),
		SameSite: http.SameSiteLaxMode,
	})
	return secret
}

// sameBrowser reports whether the request comes from the browser the sign-in
// started in.
func sameBrowser(r *http.Request, pending db.McpOauthRequest) bool {
	c, err := r.Cookie(browserCookie)
	if err != nil || c.Value == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(mcpauth.HashToken(c.Value)), []byte(pending.BrowserHash)) == 1
}

// pendingRequest loads the sign-in in progress a page posted back, by its
// handle, and checks its OAuth request again exactly as the first page did:
// the client is still known and active, the redirect URI is still one it
// registered, and the PKCE challenge is S256. On a refusal it writes the
// page or error and returns false.
func (h *Handler) pendingRequest(w http.ResponseWriter, r *http.Request, handle string) (db.McpOauthRequest, db.McpOauthClient, bool) {
	pending, err := h.svc.DB.GetMCPOAuthRequest(r.Context(), mcpauth.HashToken(handle))
	if err != nil {
		h.renderPage(w, http.StatusBadRequest, "expired", pageData{})
		return db.McpOauthRequest{}, db.McpOauthClient{}, false
	}
	client, ok := h.checkAuthorizeRequest(w, r, AuthorizationRequest{
		ResponseType:        "code",
		ClientID:            pending.ClientID,
		RedirectURI:         pending.RedirectUri,
		Scope:               pending.Scope,
		State:               pending.State,
		CodeChallenge:       pending.CodeChallenge,
		CodeChallengeMethod: pending.CodeChallengeMethod,
	})
	return pending, client, ok
}

// checkAuthorizeRequest checks an authorization request: a code request
// from a known client, to one of the client's own redirect URIs, with an
// S256 PKCE challenge. The sign-in page and the sign-in form both run it, so
// the form can't be posted with values the page would refuse. On a refusal
// it writes the error and returns false.
func (h *Handler) checkAuthorizeRequest(w http.ResponseWriter, r *http.Request, req AuthorizationRequest) (db.McpOauthClient, bool) {
	if req.ResponseType != "code" {
		h.sendError(w, http.StatusBadRequest, ErrUnsupportedResponseType, "Only 'code' response type is supported")
		return db.McpOauthClient{}, false
	}

	client, err := h.svc.DB.GetMCPOAuthClientByClientID(r.Context(), req.ClientID)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, ErrInvalidClient, "Unknown client_id")
		return db.McpOauthClient{}, false
	}

	validRedirect := false
	for _, uri := range strings.Split(client.RedirectUris, ",") {
		if uri == req.RedirectURI {
			validRedirect = true
			break
		}
	}
	if !validRedirect || !isAllowedRedirectURI(req.RedirectURI) {
		h.sendError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid redirect_uri")
		return db.McpOauthClient{}, false
	}

	// PKCE is required for MCP.
	if req.CodeChallenge == "" {
		h.redirectError(w, r, req.RedirectURI, req.State, ErrInvalidRequest, "code_challenge is required")
		return db.McpOauthClient{}, false
	}
	if req.CodeChallengeMethod != "S256" {
		h.redirectError(w, r, req.RedirectURI, req.State, ErrInvalidRequest, "Only S256 code_challenge_method is supported")
		return db.McpOauthClient{}, false
	}
	return client, true
}

// isAllowedRedirectURI accepts https, http on a loopback host (native apps),
// and custom app schemes, but never script, data or file URLs.
func isAllowedRedirectURI(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return u.Host != ""
	case "http":
		host := u.Hostname()
		return host == "localhost" || host == "127.0.0.1" || host == "::1"
	case "javascript", "data", "vbscript", "file", "blob", "about":
		return false
	}
	return true
}

// HandleAuthorizeSubmit takes the email page: it emails the address a
// sign-in link and code, the same ones the app's emailed sign-in sends and
// under the same answer whether or not the address has an account, and shows
// the code page. Posted without an email it shows the email page again.
func (h *Handler) HandleAuthorizeSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.sendError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid form data")
		return
	}
	handle := r.PostFormValue("request")
	pending, client, ok := h.pendingRequest(w, r, handle)
	if !ok {
		return
	}
	page := pageData{ClientName: client.Name, Request: handle}
	if _, asked := r.PostForm["email"]; !asked {
		h.renderPage(w, http.StatusOK, "email", page)
		return
	}
	page.Email = strings.TrimSpace(r.PostFormValue("email"))
	if page.Email == "" {
		page.Error = "Enter your email."
		h.renderPage(w, http.StatusOK, "email", page)
		return
	}

	// The sign-in now lives as long as the code just sent.
	if n, err := h.svc.DB.SetMCPOAuthRequestEmail(r.Context(), db.SetMCPOAuthRequestEmailParams{
		ID:        pending.ID,
		Email:     page.Email,
		ExpiresAt: time.Now().UTC().Add(auth.SignInTTL).Format(time.RFC3339),
	}); err != nil || n != 1 {
		h.renderPage(w, http.StatusBadRequest, "expired", pageData{})
		return
	}
	link := func(token string) string { return h.signInLink(handle, token) }
	if err := auth.SendSignIn(r.Context(), h.svc, page.Email, link); err != nil {
		page.Error = "Sign-in failed. Try again."
		h.renderPage(w, http.StatusOK, "email", page)
		return
	}
	page.Notice = auth.SignInEmailSent
	h.renderPage(w, http.StatusOK, "code", page)
}

// signInLink is the emailed link for a sign-in in progress: it opens
// HandleAuthorizeLink with the sign-in's handle and the link's token.
func (h *Handler) signInLink(handle, token string) string {
	return h.baseURL + "/mcp/oauth/authorize/link?request=" + url.QueryEscape(handle) + "&token=" + url.QueryEscape(token)
}

// HandleAuthorizeCode takes the code page: the 6-digit code from the email,
// checked as the app's emailed sign-in checks it (wrong codes count, and the
// fifth retires the email). The right code finishes the sign-in.
func (h *Handler) HandleAuthorizeCode(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.sendError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid form data")
		return
	}
	handle := r.PostFormValue("request")
	pending, client, ok := h.pendingRequest(w, r, handle)
	if !ok {
		return
	}
	page := pageData{ClientName: client.Name, Request: handle, Email: pending.Email}
	if pending.Email == "" {
		h.renderPage(w, http.StatusOK, "email", page)
		return
	}
	user, err := auth.SignInByEmail(r.Context(), h.svc, "", pending.Email, r.PostFormValue("code"))
	if err != nil {
		page.Error = signInError(err)
		h.renderPage(w, http.StatusOK, "code", page)
		return
	}
	h.finishSignIn(w, r, pending, client, user)
}

// HandleAuthorizeLink opens the emailed link. In the browser the sign-in
// started in, it asks to allow access, and allowing finishes the sign-in; the
// link is not used by just opening it, so a mail scanner that fetches it uses
// nothing. Anywhere else the client's redirect would land on the wrong
// device, so the page says to enter the code where the sign-in started, and
// the link and code stay unused.
func (h *Handler) HandleAuthorizeLink(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	handle := q.Get("request")
	pending, client, ok := h.pendingRequest(w, r, handle)
	if !ok {
		return
	}
	if !sameBrowser(r, pending) {
		h.renderPage(w, http.StatusOK, "elsewhere", pageData{ClientName: client.Name})
		return
	}
	h.renderPage(w, http.StatusOK, "link", pageData{ClientName: client.Name, Request: handle, Token: q.Get("token")})
}

// HandleAuthorizeLinkSubmit allows access from the emailed link, in the
// browser the sign-in started in, and finishes the sign-in.
func (h *Handler) HandleAuthorizeLinkSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.sendError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid form data")
		return
	}
	handle := r.PostFormValue("request")
	pending, client, ok := h.pendingRequest(w, r, handle)
	if !ok {
		return
	}
	if !sameBrowser(r, pending) {
		h.renderPage(w, http.StatusOK, "elsewhere", pageData{ClientName: client.Name})
		return
	}
	user, err := auth.SignInByEmail(r.Context(), h.svc, r.PostFormValue("token"), "", "")
	if err != nil {
		h.renderPage(w, http.StatusOK, "code", pageData{ClientName: client.Name, Request: handle, Email: pending.Email, Error: signInError(err)})
		return
	}
	h.finishSignIn(w, r, pending, client, user)
}

// signInError is the message for a refused emailed sign-in.
func signInError(err error) string {
	var ce *errorx.CodeError
	if errors.As(err, &ce) {
		return ce.Message
	}
	return "Sign-in failed. Try again."
}

// finishSignIn ends a sign-in in progress: it removes it, so it finishes
// once, and sends the browser back to the client with an authorization code
// for the request as it was checked and kept.
func (h *Handler) finishSignIn(w http.ResponseWriter, r *http.Request, pending db.McpOauthRequest, client db.McpOauthClient, user db.User) {
	if n, err := h.svc.DB.DeleteMCPOAuthRequest(r.Context(), pending.ID); err != nil || n != 1 {
		h.renderPage(w, http.StatusBadRequest, "expired", pageData{})
		return
	}

	// No scope requested means the server's default: full access, as this
	// person.
	scope := pending.Scope
	if strings.TrimSpace(scope) == "" {
		scope = mcpauth.ScopeFull
	}

	code := generateSecureToken(32)
	if _, err := h.svc.DB.CreateMCPOAuthCode(r.Context(), db.CreateMCPOAuthCodeParams{
		ID:                  generateSecureToken(16),
		ClientID:            client.ID,
		UserID:              user.ID,
		CodeHash:            mcpauth.HashToken(code),
		RedirectUri:         pending.RedirectUri,
		Scopes:              scope,
		CodeChallenge:       sql.NullString{String: pending.CodeChallenge, Valid: pending.CodeChallenge != ""},
		CodeChallengeMethod: sql.NullString{String: pending.CodeChallengeMethod, Valid: pending.CodeChallengeMethod != ""},
		ExpiresAt:           time.Now().Add(AuthCodeTTL).Format(time.RFC3339),
	}); err != nil {
		h.redirectError(w, r, pending.RedirectUri, pending.State, ErrServerError, "Failed to create authorization code")
		return
	}

	redirectURL, _ := url.Parse(pending.RedirectUri)
	q := redirectURL.Query()
	q.Set("code", code)
	if pending.State != "" {
		q.Set("state", pending.State)
	}
	redirectURL.RawQuery = q.Encode()

	// Show styled success page that auto-redirects
	h.renderPage(w, http.StatusOK, "success", pageData{ClientName: client.Name, RedirectURL: redirectURL.String()})
}

// handleToken handles token exchange and refresh
func (h *Handler) HandleToken(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.sendError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid form data")
		return
	}

	req := TokenRequest{
		GrantType:    r.FormValue("grant_type"),
		Code:         r.FormValue("code"),
		RedirectURI:  r.FormValue("redirect_uri"),
		RefreshToken: r.FormValue("refresh_token"),
		ClientID:     r.FormValue("client_id"),
		ClientSecret: r.FormValue("client_secret"),
		CodeVerifier: r.FormValue("code_verifier"),
	}

	// Support client_secret_basic: extract credentials from Authorization header
	if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Basic ") {
		if decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic ")); err == nil {
			if parts := strings.SplitN(string(decoded), ":", 2); len(parts) == 2 {
				// URL decode the parts (client_id and secret may be URL-encoded)
				if clientID, err := url.QueryUnescape(parts[0]); err == nil {
					req.ClientID = clientID
				}
				if clientSecret, err := url.QueryUnescape(parts[1]); err == nil {
					req.ClientSecret = clientSecret
				}
			}
		}
	}

	switch req.GrantType {
	case "authorization_code":
		h.handleAuthorizationCodeGrant(w, r, req)
	case "refresh_token":
		h.handleRefreshTokenGrant(w, r, req)
	default:
		h.sendError(w, http.StatusBadRequest, ErrUnsupportedGrantType, "Unsupported grant_type")
	}
}

func (h *Handler) handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request, req TokenRequest) {
	// Validate required fields
	if req.Code == "" {
		h.sendError(w, http.StatusBadRequest, ErrInvalidRequest, "code is required")
		return
	}
	if req.ClientID == "" {
		h.sendError(w, http.StatusBadRequest, ErrInvalidRequest, "client_id is required")
		return
	}

	// Get authorization code
	codeHash := mcpauth.HashToken(req.Code)
	authCode, err := h.svc.DB.GetMCPOAuthCodeByHash(r.Context(), codeHash)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, ErrInvalidGrant, "Invalid authorization code")
		return
	}

	// Verify client
	if authCode.OauthClientID != req.ClientID {
		h.sendError(w, http.StatusBadRequest, ErrInvalidClient, "Client mismatch")
		return
	}

	// Verify redirect_uri matches
	if authCode.RedirectUri != req.RedirectURI {
		h.sendError(w, http.StatusBadRequest, ErrInvalidRequest, "redirect_uri mismatch")
		return
	}

	// Verify PKCE
	if authCode.CodeChallenge.Valid {
		if req.CodeVerifier == "" {
			h.sendError(w, http.StatusBadRequest, ErrInvalidRequest, "code_verifier is required")
			return
		}
		if !verifyPKCE(req.CodeVerifier, authCode.CodeChallenge.String, authCode.CodeChallengeMethod.String) {
			h.sendError(w, http.StatusBadRequest, ErrInvalidGrant, "Invalid code_verifier")
			return
		}
	}

	// Mark code as used
	if err := h.svc.DB.MarkMCPOAuthCodeUsed(r.Context(), authCode.ID); err != nil {
		h.sendError(w, http.StatusInternalServerError, ErrServerError, "Failed to process authorization code")
		return
	}

	// A disabled account gets no tokens.
	if _, err := access.Active(r.Context(), h.svc.DB, authCode.UserID); err != nil {
		h.sendError(w, http.StatusBadRequest, ErrInvalidGrant, "Account is not active")
		return
	}

	// Generate tokens
	accessToken := generateSecureToken(32)
	refreshToken := generateSecureToken(32)
	accessTokenHash := mcpauth.HashToken(accessToken)
	refreshTokenHash := mcpauth.HashToken(refreshToken)

	expiresAt := time.Now().Add(AccessTokenTTL)
	refreshExpiresAt := time.Now().Add(RefreshTokenTTL)

	// Store tokens
	_, err = h.svc.DB.CreateMCPOAuthToken(r.Context(), db.CreateMCPOAuthTokenParams{
		ID:               generateSecureToken(16),
		ClientID:         authCode.ClientID,
		UserID:           authCode.UserID,
		AccessTokenHash:  accessTokenHash,
		RefreshTokenHash: sql.NullString{String: refreshTokenHash, Valid: true},
		Scopes:           authCode.Scopes,
		ExpiresAt:        expiresAt.Format(time.RFC3339),
		RefreshExpiresAt: sql.NullString{String: refreshExpiresAt.Format(time.RFC3339), Valid: true},
	})
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, ErrServerError, "Failed to create tokens")
		return
	}

	resp := TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(AccessTokenTTL.Seconds()),
		RefreshToken: refreshToken,
		Scope:        authCode.Scopes,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request, req TokenRequest) {
	if req.RefreshToken == "" {
		h.sendError(w, http.StatusBadRequest, ErrInvalidRequest, "refresh_token is required")
		return
	}

	// Get existing token by refresh token hash
	refreshTokenHash := mcpauth.HashToken(req.RefreshToken)
	oldToken, err := h.svc.DB.GetMCPOAuthTokenByRefreshHash(r.Context(), sql.NullString{String: refreshTokenHash, Valid: true})
	if err != nil {
		h.sendError(w, http.StatusBadRequest, ErrInvalidGrant, "Invalid refresh token")
		return
	}

	// Revoke old token
	if err := h.svc.DB.RevokeMCPOAuthToken(r.Context(), oldToken.ID); err != nil {
		h.sendError(w, http.StatusInternalServerError, ErrServerError, "Failed to refresh token")
		return
	}

	// A disabled account gets no new tokens.
	if _, err := access.Active(r.Context(), h.svc.DB, oldToken.UserID); err != nil {
		h.sendError(w, http.StatusBadRequest, ErrInvalidGrant, "Account is not active")
		return
	}

	// Generate new tokens
	accessToken := generateSecureToken(32)
	refreshToken := generateSecureToken(32)
	accessTokenHash := mcpauth.HashToken(accessToken)
	newRefreshTokenHash := mcpauth.HashToken(refreshToken)

	expiresAt := time.Now().Add(AccessTokenTTL)
	refreshExpiresAt := time.Now().Add(RefreshTokenTTL)

	// Store new tokens
	_, err = h.svc.DB.CreateMCPOAuthToken(r.Context(), db.CreateMCPOAuthTokenParams{
		ID:               generateSecureToken(16),
		ClientID:         oldToken.ClientID,
		UserID:           oldToken.UserID,
		AccessTokenHash:  accessTokenHash,
		RefreshTokenHash: sql.NullString{String: newRefreshTokenHash, Valid: true},
		Scopes:           oldToken.Scopes,
		ExpiresAt:        expiresAt.Format(time.RFC3339),
		RefreshExpiresAt: sql.NullString{String: refreshExpiresAt.Format(time.RFC3339), Valid: true},
	})
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, ErrServerError, "Failed to create tokens")
		return
	}

	resp := TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(AccessTokenTTL.Seconds()),
		RefreshToken: refreshToken,
		Scope:        oldToken.Scopes,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) sendError(w http.ResponseWriter, status int, errCode, errDesc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:            errCode,
		ErrorDescription: errDesc,
	})
}

func (h *Handler) redirectError(w http.ResponseWriter, r *http.Request, redirectURI, state, errCode, errDesc string) {
	u, _ := url.Parse(redirectURI)
	q := u.Query()
	q.Set("error", errCode)
	q.Set("error_description", errDesc)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func generateSecureToken(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func verifyPKCE(verifier, challenge, method string) bool {
	if method != "S256" {
		return false
	}

	h := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(h[:])

	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}
