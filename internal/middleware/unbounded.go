package middleware

import (
	"net/http"
	"time"
)

// Unbounded lifts the server's write deadline for this one response, so a
// download of any size finishes: a download has no time limit. Its route must
// also have no go-zero timeout (rest.WithTimeout(0)), which would otherwise
// buffer the whole response and cut it off.
func Unbounded(w http.ResponseWriter) error {
	return http.NewResponseController(w).SetWriteDeadline(time.Time{})
}
