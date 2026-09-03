package websocket

import (
	"log/slog"
	"net/http"
)

// CSRFProtect rejects state-changing requests whose Origin header doesn't
// match ALLOWED_ORIGIN - the same check already used to gate WebSocket
// upgrades (see checkOrigin). Browsers always send Origin on cross-origin
// requests and on same-origin POST/PUT/DELETE/PATCH requests, so a
// third-party page that auto-submits a form or fetch against this API using
// the victim's cookies gets rejected before the handler ever runs.
func CSRFProtect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkOrigin(r) {
			slog.Warn("rejecting request: origin check failed", "method", r.Method, "path", r.URL.Path)
			http.Error(w, "Invalid origin", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
