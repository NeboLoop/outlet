package middleware

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// RequestLog logs one line per request: method, path, status, client and
// duration. It never logs headers or bodies, which carry sign-in tokens, API
// keys and passwords. (go-zero's own log middleware dumps the whole request
// on a server error, so it is turned off in the config.)
func RequestLog(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
		next(sw, r)
		line := "[HTTP] %d - %s %s - %s - %s"
		args := []any{sw.code, r.Method, r.URL.Path, httpx.GetRemoteAddr(r), time.Since(start).Round(time.Millisecond)}
		if sw.code >= 500 {
			logx.WithContext(r.Context()).Errorf(line, args...)
		} else {
			logx.WithContext(r.Context()).Infof(line, args...)
		}
	}
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

// Flush keeps streaming responses (MCP, server-sent events) working.
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap lets http.ResponseController reach the connection (a download lifts
// its write deadline through it).
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Hijack keeps WebSocket upgrades working.
func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer can't be hijacked")
	}
	return h.Hijack()
}
