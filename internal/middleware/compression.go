package middleware

import (
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// Gzip wraps an HTTP handler with gzip compression
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip compression if client doesn't support it
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Skip compression for API routes, SDK routes, MCP (SSE streaming), OAuth, and well-known endpoints
		// API/SDK routes return small JSON - compression overhead not worth it and can cause Content-Length issues
		// SSE requires unbuffered streaming which gzip breaks
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/") ||
			strings.HasPrefix(path, "/sdk/") ||
			strings.HasPrefix(path, "/mcp") ||
			strings.HasPrefix(path, "/.well-known/") ||
			path == "/register" ||
			path == "/authorize" ||
			path == "/token" {
			next.ServeHTTP(w, r)
			return
		}

		// Skip compression for already compressed content
		contentType := w.Header().Get("Content-Type")
		if strings.Contains(contentType, "image/") ||
			strings.Contains(contentType, "video/") ||
			strings.Contains(contentType, "application/pdf") {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length") // Length will change after compression

		gz := gzip.NewWriter(w)
		defer gz.Close()

		gzw := &gzipResponseWriter{Writer: gz, ResponseWriter: w}
		next.ServeHTTP(gzw, r)
	})
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
	statusCode int
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// Unwrap lets http.ResponseController reach the connection.
func (w *gzipResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// ContentType sets correct MIME types based on file extension using Go's mime package
func ContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get file extension
		ext := filepath.Ext(r.URL.Path)

		// Use Go's mime package to detect content type
		if ext != "" {
			if contentType := mime.TypeByExtension(ext); contentType != "" {
				w.Header().Set("Content-Type", contentType)
			}
		}

		next.ServeHTTP(w, r)
	})
}

// CacheControl adds cache headers for static assets and disables caching for dynamic routes
func CacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// No cache for dynamic routes - API, webhooks, WebSocket, downloads, OAuth, MCP
		if strings.HasPrefix(path, "/api/") ||
			strings.HasPrefix(path, "/webhooks/") ||
			strings.HasPrefix(path, "/ws/") ||
			strings.HasPrefix(path, "/downloads/") ||
			strings.HasPrefix(path, "/.well-known/") ||
			strings.HasPrefix(path, "/mcp") ||
			path == "/register" ||
			path == "/authorize" ||
			path == "/token" {
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			next.ServeHTTP(w, r)
			return
		}

		// Immutable cache for hashed assets (SvelteKit _app/immutable/) - safe because filenames change on rebuild
		if strings.Contains(path, "/_app/immutable/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else if strings.HasSuffix(path, ".html") || path == "/" {
			// No cache for HTML to ensure fresh content
			w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		} else {
			// 24 hour cache for all other static assets (images, fonts, non-hashed JS/CSS)
			w.Header().Set("Cache-Control", "public, max-age=86400")
		}

		next.ServeHTTP(w, r)
	})
}
