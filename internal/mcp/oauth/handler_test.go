package oauth

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsAllowedRedirectURI(t *testing.T) {
	for uri, want := range map[string]bool{
		"https://claude.ai/api/mcp/auth_callback": true,
		"http://localhost:6274/callback":          true,
		"http://127.0.0.1:33418/callback":         true,
		"cursor://anysphere.cursor-mcp/callback":  true,
		"http://evil.example/callback":            false,
		"javascript:alert(1)":                     false,
		"JavaScript:alert(1)":                     false,
		"data:text/html,<script>":                 false,
		"https:///nohost":                         false,
		"":                                        false,
	} {
		if got := isAllowedRedirectURI(uri); got != want {
			t.Errorf("%q: got %v, want %v", uri, got, want)
		}
	}
}

func TestPagesEscapeInput(t *testing.T) {
	h := &Handler{}
	attack := `"><script>alert(1)</script>`

	rr := httptest.NewRecorder()
	h.renderLoginPage(rr, AuthorizationRequest{State: attack, Scope: attack, ClientID: attack}, attack, attack)
	if strings.Contains(rr.Body.String(), "<script>alert(1)") {
		t.Error("login page renders unescaped input")
	}

	rr = httptest.NewRecorder()
	h.renderSuccessPage(rr, "https://app.example/cb?state="+attack, attack)
	if strings.Contains(rr.Body.String(), "<script>alert(1)") {
		t.Error("success page renders unescaped input")
	}

	// The redirect keeps its query string: a client reads code and state.
	rr = httptest.NewRecorder()
	h.renderSuccessPage(rr, "https://claude.ai/api/mcp/auth_callback?code=c1&state=s1", "Claude")
	if !strings.Contains(rr.Body.String(), `window.location.href = "https://claude.ai/api/mcp/auth_callback?code=c1\u0026state=s1";`) {
		t.Errorf("success page redirect is mangled:\n%s", rr.Body.String())
	}
}
