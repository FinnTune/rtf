package utility_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"rtForum/utility"

	"golang.org/x/time/rate"
)

func newTestLimiterHandler(rl *utility.IPRateLimiter) (http.HandlerFunc, *int) {
	calls := 0
	handler := rl.Limit(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	})
	return handler, &calls
}

func doRequest(handler http.HandlerFunc, remoteAddr string) int {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr.Code
}

func TestIPRateLimiter_AllowsUpToBurstThenBlocks(t *testing.T) {
	// A near-zero refill rate isolates this test to burst behavior alone:
	// once the burst is exhausted, nothing refills during the test.
	rl := utility.NewIPRateLimiter(rate.Every(time.Hour), 3)
	handler, _ := newTestLimiterHandler(rl)

	for i := 0; i < 3; i++ {
		if code := doRequest(handler, "1.2.3.4:5555"); code != http.StatusOK {
			t.Fatalf("request %d: expected 200 within burst, got %d", i+1, code)
		}
	}
	if code := doRequest(handler, "1.2.3.4:5555"); code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 once burst is exhausted, got %d", code)
	}
}

func TestIPRateLimiter_BlockedRequestNeverReachesHandler(t *testing.T) {
	rl := utility.NewIPRateLimiter(rate.Every(time.Hour), 1)
	handler, calls := newTestLimiterHandler(rl)

	doRequest(handler, "1.2.3.4:5555") // consumes the single token
	doRequest(handler, "1.2.3.4:5555") // should be rejected before reaching the handler
	if *calls != 1 {
		t.Fatalf("expected the wrapped handler to run exactly once, ran %d times", *calls)
	}
}

func TestIPRateLimiter_RefillsOverTime(t *testing.T) {
	rl := utility.NewIPRateLimiter(rate.Every(50*time.Millisecond), 1)
	handler, _ := newTestLimiterHandler(rl)

	if code := doRequest(handler, "1.2.3.4:5555"); code != http.StatusOK {
		t.Fatalf("expected the first request to succeed, got %d", code)
	}
	if code := doRequest(handler, "1.2.3.4:5555"); code != http.StatusTooManyRequests {
		t.Fatalf("expected the immediate second request to be rate-limited, got %d", code)
	}

	// Generous margin over the 50ms refill interval to keep this reliable
	// under load, matching the OTP expiry test's ~12x-margin convention.
	time.Sleep(600 * time.Millisecond)

	if code := doRequest(handler, "1.2.3.4:5555"); code != http.StatusOK {
		t.Fatalf("expected the token to have refilled after waiting, got %d", code)
	}
}

func TestIPRateLimiter_IsolatesPerIP(t *testing.T) {
	rl := utility.NewIPRateLimiter(rate.Every(time.Hour), 1)
	handler, _ := newTestLimiterHandler(rl)

	if code := doRequest(handler, "1.1.1.1:1111"); code != http.StatusOK {
		t.Fatalf("expected 1.1.1.1's first request to succeed, got %d", code)
	}
	if code := doRequest(handler, "1.1.1.1:1111"); code != http.StatusTooManyRequests {
		t.Fatalf("expected 1.1.1.1's second request to be blocked, got %d", code)
	}
	// A different IP's own bucket must be untouched by 1.1.1.1 exhausting theirs.
	if code := doRequest(handler, "2.2.2.2:2222"); code != http.StatusOK {
		t.Fatalf("expected 2.2.2.2's first request to succeed independently, got %d", code)
	}
}

func TestIPRateLimiter_TreatsMalformedRemoteAddrAsSingleSharedKey(t *testing.T) {
	// clientIP falls back to the raw RemoteAddr string when it has no parsable
	// host:port (net.SplitHostPort errors) — this locks in that the fallback
	// still keys consistently per distinct string, rather than e.g. every
	// malformed address colliding into one bucket or being ignored entirely.
	rl := utility.NewIPRateLimiter(rate.Every(time.Hour), 1)
	handler, _ := newTestLimiterHandler(rl)

	if code := doRequest(handler, "no-port-here"); code != http.StatusOK {
		t.Fatalf("expected the first malformed-address request to succeed, got %d", code)
	}
	if code := doRequest(handler, "no-port-here"); code != http.StatusTooManyRequests {
		t.Fatalf("expected a repeat request from the same malformed address to be blocked, got %d", code)
	}
	if code := doRequest(handler, "a-different-malformed-address"); code != http.StatusOK {
		t.Fatalf("expected a distinct malformed address to have its own bucket, got %d", code)
	}
}
