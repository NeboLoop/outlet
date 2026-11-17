package oauth

import (
	"html/template"
	"net/http"
)

// pageData is what the sign-in pages show. html/template escapes every
// field for where it lands: text, attribute or script.
type pageData struct {
	ClientName string
	// Request is the handle of the sign-in in progress (mcp_oauth_requests).
	Request string
	Email   string
	// Token is the emailed link's token, on the page that confirms it.
	Token       string
	Error       string
	Notice      string
	RedirectURL string
}

var pages = template.Must(template.New("pages").Parse(`
{{define "head"}}<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Sign in to Outlet</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f3f4f6; min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 16px; }
        .container { background: white; padding: 32px; border-radius: 12px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); width: 100%; max-width: 400px; }
        .center { text-align: center; }
        h1 { font-size: 24px; margin-bottom: 8px; color: #111827; }
        .subtitle { color: #6b7280; margin-bottom: 24px; line-height: 1.5; }
        .client-name { font-weight: 600; color: #111827; }
        .error { color: #dc2626; background: #fef2f2; padding: 12px; border-radius: 6px; margin-bottom: 16px; }
        .notice { color: #374151; background: #eff6ff; padding: 12px; border-radius: 6px; margin-bottom: 16px; line-height: 1.5; }
        label { display: block; font-size: 14px; font-weight: 500; color: #374151; margin-bottom: 6px; }
        input[type="email"], input[type="text"] { width: 100%; padding: 10px 12px; border: 1px solid #d1d5db; border-radius: 6px; font-size: 16px; margin-bottom: 16px; }
        input.code { font-size: 24px; letter-spacing: 6px; text-align: center; }
        input:focus { outline: none; border-color: #2563eb; box-shadow: 0 0 0 3px rgba(37,99,235,0.1); }
        button { width: 100%; padding: 12px; background: #2563eb; color: white; border: none; border-radius: 6px; font-size: 16px; font-weight: 500; cursor: pointer; }
        button:hover { background: #1d4ed8; }
        button.secondary { background: none; color: #2563eb; padding: 8px; font-size: 14px; }
        button.secondary:hover { background: #f3f4f6; }
        .more { margin-top: 16px; }
        .scope { background: #f3f4f6; padding: 12px; border-radius: 6px; margin-bottom: 16px; font-size: 14px; color: #4b5563; }
        .success-icon { width: 64px; height: 64px; background: #10b981; border-radius: 50%; display: flex; align-items: center; justify-content: center; margin: 0 auto 24px; }
        .success-icon svg { width: 32px; height: 32px; color: white; }
        .redirect-note { font-size: 14px; color: #9ca3af; margin-top: 16px; }
        .spinner { width: 20px; height: 20px; border: 2px solid #e5e7eb; border-top: 2px solid #2563eb; border-radius: 50%; animation: spin 1s linear infinite; display: inline-block; vertical-align: middle; margin-right: 8px; }
        @keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }
    </style>
</head>
<body>
    <div class="container">{{end}}

{{define "foot"}}    </div>
</body>
</html>{{end}}

{{define "error"}}{{if .Error}}<div class="error">{{.Error}}</div>{{end}}{{end}}

{{define "scope"}}<div class="scope"><strong>Requested access:</strong> Full MCP access to manage your SaaS</div>{{end}}

{{define "email"}}{{template "head" .}}
        <h1>Sign in to Outlet</h1>
        <p class="subtitle"><span class="client-name">{{.ClientName}}</span> wants to access your account</p>
        {{template "error" .}}
        {{template "scope" .}}
        <form method="POST" action="/authorize">
            <input type="hidden" name="request" value="{{.Request}}">
            <label for="email">Email</label>
            <input type="email" id="email" name="email" value="{{.Email}}" autocomplete="email" required autofocus>
            <button type="submit">Email me a sign-in code</button>
        </form>
{{template "foot"}}{{end}}

{{define "code"}}{{template "head" .}}
        <h1>Check your email</h1>
        <p class="subtitle">Sign in to connect <span class="client-name">{{.ClientName}}</span></p>
        {{if .Notice}}<div class="notice">{{.Notice}} ({{.Email}}). Enter the 6-digit code from it here, or open its link on this device.</div>{{end}}
        {{template "error" .}}
        <form method="POST" action="/mcp/oauth/authorize/code">
            <input type="hidden" name="request" value="{{.Request}}">
            <label for="code">Sign-in code</label>
            <input type="text" class="code" id="code" name="code" inputmode="numeric" autocomplete="one-time-code" pattern="[0-9]{6}" maxlength="6" required autofocus>
            <button type="submit">Sign in and authorize</button>
        </form>
        <form method="POST" action="/authorize" class="more">
            <input type="hidden" name="request" value="{{.Request}}">
            <input type="hidden" name="email" value="{{.Email}}">
            <button type="submit" class="secondary">Send a new code</button>
        </form>
        <form method="POST" action="/authorize">
            <input type="hidden" name="request" value="{{.Request}}">
            <button type="submit" class="secondary">Use a different email</button>
        </form>
{{template "foot"}}{{end}}

{{define "link"}}{{template "head" .}}
        <h1>Allow access</h1>
        <p class="subtitle"><span class="client-name">{{.ClientName}}</span> wants to access your Outlet account</p>
        {{template "scope" .}}
        <form method="POST" action="/mcp/oauth/authorize/link">
            <input type="hidden" name="request" value="{{.Request}}">
            <input type="hidden" name="token" value="{{.Token}}">
            <button type="submit">Allow and continue</button>
        </form>
{{template "foot"}}{{end}}

{{define "elsewhere"}}{{template "head" .}}
        <h1>Finish where you started</h1>
        <p class="subtitle">You started connecting <span class="client-name">{{.ClientName}}</span> on another device or browser, and the sign-in has to finish there.</p>
        <div class="notice">Go back to the Outlet sign-in page on that device and enter the 6-digit code from this email. The code works once and expires 15 minutes after it was sent.</div>
{{template "foot"}}{{end}}

{{define "expired"}}{{template "head" .}}
        <h1>This sign-in has expired</h1>
        <p class="subtitle">It was used already or is more than 15 minutes old. Go back to the app you were connecting to Outlet and connect again.</p>
{{template "foot"}}{{end}}

{{define "success"}}{{template "head" .}}
        <div class="center">
            <div class="success-icon">
                <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
                </svg>
            </div>
            <h1>Authentication Successful</h1>
            <p class="subtitle">You've successfully signed in to <span class="client-name">{{.ClientName}}</span>.</p>
            <p class="redirect-note"><span class="spinner"></span>Redirecting back to the application...</p>
        </div>
    <script>
        setTimeout(function() {
            window.location.href = {{.RedirectURL}};
        }, 1500);
    </script>
{{template "foot"}}{{end}}
`))

// renderPage writes one of the sign-in pages. The pages carry sign-in
// handles and links, so they are never cached, and they are never framed,
// so no other site can overlay their buttons.
func (h *Handler) renderPage(w http.ResponseWriter, status int, name string, d pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Frame-Options", "DENY")
	w.WriteHeader(status)
	_ = pages.ExecuteTemplate(w, name, d)
}
