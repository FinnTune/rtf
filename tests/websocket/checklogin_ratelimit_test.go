package websocket_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"rtForum/utility"
	"rtForum/websocket"

	"golang.org/x/time/rate"
)

// TestCheckLoginHandler_RateLimited_RejectsWithoutReachingHandler proves
// checkLogin actually composes with utility.IPRateLimiter the way main.go's
// registration relies on: checkLogin needs no authentication to reach (any
// request carrying a session_id cookie, even a garbage value, passes its
// only early-return check) and takes the manager's full write lock for an
// O(n) scan of every connected client — previously the one endpoint in
// main.go with no rate limiter of any kind, unlike every other handler.
// The generic wrapping mechanism itself (burst/block/refill/Retry-After) is
// already exhaustively covered by tests/utility/ratelimit_test.go; this
// confirms it works correctly combined with this specific, real production
// handler, mirroring the equivalent CSRFProtect+AddPost end-to-end test in
// csrf_test.go.
func TestCheckLoginHandler_RateLimited_RejectsWithoutReachingHandler(t *testing.T) {
	websocket.ResetTestState()

	// A near-zero refill rate isolates this test to burst behavior alone —
	// matching tests/utility/ratelimit_test.go's own convention — so nothing
	// refills mid-test regardless of how long it takes to run.
	rl := utility.NewIPRateLimiter(rate.Every(time.Hour), 2)
	limited := rl.Limit(websocket.CheckLoginHandler)

	newCheckLoginRequest := func() *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/checkLogin", nil)
		req.RemoteAddr = "203.0.113.7:5555"
		// A garbage session_id is enough to reach checkLogin's locked scan
		// — it only short-circuits early when the cookie is absent
		// entirely.
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "not-a-real-session"})
		return req
	}

	for i := 0; i < 2; i++ {
		rr := httptest.NewRecorder()
		limited(rr, newCheckLoginRequest())
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: expected status %d within burst, got %d: %s", i+1, http.StatusOK, rr.Code, rr.Body.String())
		}
	}

	rr := httptest.NewRecorder()
	limited(rr, newCheckLoginRequest())
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d once the burst is exhausted, got %d: %s", http.StatusTooManyRequests, rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Retry-After"); got == "" {
		t.Fatal("expected a Retry-After header on the rate-limited response")
	}
	// Confirms this is genuinely IPRateLimiter's own rejection (the handler
	// never ran), not a coincidental 429 from somewhere inside checkLogin
	// itself.
	if body := rr.Body.String(); !strings.Contains(body, "Too many requests") {
		t.Fatalf("expected IPRateLimiter's own rejection body, got %q", body)
	}
}
