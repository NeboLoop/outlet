package app

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"
)

// Run `go generate ./...` before building the Go binary to refresh the embedded SPA build.
//go:generate sh -c "command -v pnpm >/dev/null 2>&1 || { echo 'pnpm not found. Please install pnpm first.' >&2; exit 1; }"
//go:generate pnpm run build
//go:generate sh -c "[ -d build ] || { echo 'Build failed - no build directory found' >&2; exit 1; }"

// Embed the entire build folder
//
//go:embed all:build
var buildFS embed.FS

var ServerHost string

// SetServerHost sets the server host for server.json
func SetServerHost(host string, port int, useHTTPS bool) {
	protocol := "http"
	if useHTTPS {
		protocol = "https"
	}

	// For standard ports, don't include port in URL
	if (useHTTPS && port == 443) || (!useHTTPS && port == 80) {
		ServerHost = fmt.Sprintf("%s://%s", protocol, host)
	} else {
		ServerHost = fmt.Sprintf("%s://%s:%d", protocol, host, port)
	}

	fmt.Printf("Server Host set to: %s\n", ServerHost)
}

// SPAHandler returns a handler that serves static files with SPA fallback
// Uses http.FileServer for regular files and http.ServeContent for index.html
// to avoid directory redirects while maintaining proper HTTP semantics
func SPAHandler(spaFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(spaFS))

	// serveFile serves a file with proper HTTP semantics (Content-Length, Last-Modified, etc)
	serveFile := func(w http.ResponseWriter, r *http.Request, path string) {
		file, err := spaFS.Open(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// Use http.ServeContent for proper headers and range request support
		http.ServeContent(w, r, stat.Name(), stat.ModTime(), file.(io.ReadSeeker))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// An unknown API path is a 404, never the app page: API clients must
		// not get HTML back.
		for _, prefix := range []string{"/api/", "/sdk/", "/mcp", "/webhooks/", "/network/"} {
			if strings.HasPrefix(r.URL.Path, prefix) {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Redirect trailing slash to non-trailing slash for SEO
		// /about/ → 301 → /about (canonical URL)
		if path != "" && strings.HasSuffix(path, "/") {
			http.Redirect(w, r, "/"+strings.TrimSuffix(path, "/"), http.StatusMovedPermanently)
			return
		}

		// Try exact path first
		file, err := spaFS.Open(path)
		if err == nil {
			stat, err := file.Stat()
			file.Close()

			if err == nil && !stat.IsDir() {
				// It's a file - serve via fileServer (handles all HTTP semantics)
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Try with .html extension for prerendered pages: /about → about.html
		if !strings.Contains(path, ".") {
			htmlPath := path + ".html"
			if _, err := spaFS.Open(htmlPath); err == nil {
				serveFile(w, r, htmlPath)
				return
			}
		}

		// File doesn't exist - check if it's a static asset (should 404) or SPA route
		if strings.Contains(path, ".") {
			http.NotFound(w, r)
			return
		}

		// SPA route - serve 200.html for client-side routing
		if _, err := spaFS.Open("200.html"); err == nil {
			serveFile(w, r, "200.html")
			return
		}

		// Final fallback to index.html
		serveFile(w, r, "index.html")
	})
}

// DevMode controls whether to use local filesystem instead of embedded FS
var DevMode bool

// FileSystem returns the SPA filesystem.
// In dev mode, it uses the local filesystem (app/build).
// In production, it uses the embedded filesystem.
func FileSystem() (fs.FS, error) {
	if DevMode {
		// Use local filesystem for hot reload during development
		localPath := "app/build"
		if _, err := os.Stat(localPath); err != nil {
			return nil, fmt.Errorf("dev mode: build directory not found at %s - run 'cd app && pnpm build' first", localPath)
		}
		fmt.Println("Using local filesystem for SPA (dev mode): app/build")
		return &serverJSONFS{os.DirFS(localPath)}, nil
	}

	// Production: use embedded filesystem
	sub, err := fs.Sub(buildFS, "build")
	if err != nil {
		return nil, err
	}
	return &serverJSONFS{sub}, nil
}

// serverJSONFS wraps the filesystem to intercept server.json
type serverJSONFS struct {
	fs.FS
}

// Open intercepts server.json and returns dynamic content
func (s *serverJSONFS) Open(name string) (fs.File, error) {
	// Handle both "server.json" and "/server.json" requests
	if name == "server.json" || name == "/server.json" {
		content := `{"server": "` + ServerHost + `"}`
		return &serverJSONFile{content: content}, nil
	}
	return s.FS.Open(name)
}

// serverJSONFile implements fs.File for dynamic server.json
type serverJSONFile struct {
	content string
	pos     int
}

func (f *serverJSONFile) Read(p []byte) (int, error) {
	if f.pos >= len(f.content) {
		return 0, io.EOF
	}
	n := copy(p, f.content[f.pos:])
	f.pos += n
	return n, nil
}

func (f *serverJSONFile) Close() error { return nil }

func (f *serverJSONFile) Stat() (fs.FileInfo, error) {
	return &serverJSONInfo{size: int64(len(f.content))}, nil
}

// serverJSONInfo implements fs.FileInfo
type serverJSONInfo struct {
	size int64
}

func (i *serverJSONInfo) Name() string       { return "server.json" }
func (i *serverJSONInfo) Size() int64        { return i.size }
func (i *serverJSONInfo) Mode() fs.FileMode  { return 0444 }
func (i *serverJSONInfo) ModTime() time.Time { return time.Now() }
func (i *serverJSONInfo) IsDir() bool        { return false }
func (i *serverJSONInfo) Sys() interface{}   { return nil }

// DiscoveredFunnel represents a funnel discovered from the embedded filesystem
type DiscoveredFunnel struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// DiscoverFunnels scans the embedded /lp/ directory to find all landing page funnels
func DiscoverFunnels() ([]DiscoveredFunnel, error) {
	var funnels []DiscoveredFunnel

	// Get the base filesystem
	var baseFS fs.FS
	var err error

	if DevMode {
		localPath := "app/build"
		if _, err := os.Stat(localPath); err != nil {
			return nil, fmt.Errorf("dev mode: build directory not found")
		}
		baseFS = os.DirFS(localPath)
	} else {
		baseFS, err = fs.Sub(buildFS, "build")
		if err != nil {
			return nil, err
		}
	}

	// Read the lp directory
	entries, err := fs.ReadDir(baseFS, "lp")
	if err != nil {
		// lp directory doesn't exist - return empty list
		return funnels, nil
	}

	for _, entry := range entries {
		name := entry.Name()
		// SvelteKit static adapter outputs .html files for prerendered pages
		// e.g., "ai-playbook.html" -> slug "ai-playbook"
		if strings.HasSuffix(name, ".html") {
			slug := strings.TrimSuffix(name, ".html")
			funnels = append(funnels, DiscoveredFunnel{
				Slug: slug,
				Name: slugToName(slug),
			})
		}
	}

	return funnels, nil
}

// slugToName converts a slug to a readable name
// e.g., "ai-playbook" -> "AI Playbook", "executive" -> "Executive"
func slugToName(slug string) string {
	words := strings.Split(slug, "-")
	for i, word := range words {
		if len(word) > 0 {
			// Handle common acronyms
			if word == "ai" {
				words[i] = "AI"
			} else {
				words[i] = strings.ToUpper(string(word[0])) + word[1:]
			}
		}
	}
	return strings.Join(words, " ")
}
