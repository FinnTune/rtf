package websocket_test

import (
	"encoding/json"
	"testing"
	"time"

	"rtForum/websocket"

	"golang.org/x/time/rate"
)

// TestRouteEvent_DropsEventsOverRateLimit shrinks the per-client event rate
// limit to a tiny, effectively non-refilling budget and proves routeEvent
// drops events once it's exhausted: an unknown event type normally returns
// an error (see TestRouteEvent_UnknownType) - once the client is
// rate-limited, routeEvent short-circuits before even checking for a
// handler, so the call returns nil instead.
func TestRouteEvent_DropsEventsOverRateLimit(t *testing.T) {
	websocket.ResetTestState()
	// rate.Limit near zero means the bucket effectively never refills during
	// this test, so exactly `burst` events succeed and every one after that
	// is dropped - deterministic, no timing dependency.
	restore := websocket.SetEventRateLimitForTest(rate.Limit(0.0001), 3)
	defer restore()

	client := websocket.AddTestClient("s1", "admin", 1)
	payload := json.RawMessage(`{}`)

	for i := 0; i < 3; i++ {
		err := websocket.RouteEventForTest("unknown-event", payload, client)
		if err == nil {
			t.Fatalf("event %d: expected the unknown-event handler-lookup error while still under budget", i)
		}
	}

	if err := websocket.RouteEventForTest("unknown-event", payload, client); err != nil {
		t.Fatalf("expected the 4th event to be silently dropped by the rate limiter, got error: %v", err)
	}
}

// TestRouteEvent_AllowsBurstThenRefills proves the limiter isn't a one-shot
// budget: after the burst is spent and the clock is allowed to actually
// advance (a real, if tiny, refill rate), a subsequent event goes through
// the normal handler-lookup path again.
func TestRouteEvent_AllowsBurstThenRefills(t *testing.T) {
	websocket.ResetTestState()
	// A high, real refill rate (1000/s) means the single token consumed by
	// waiting for the burst to drain refills almost immediately, without an
	// actual t.Sleep in the test.
	restore := websocket.SetEventRateLimitForTest(rate.Limit(1000), 1)
	defer restore()

	client := websocket.AddTestClient("s1", "admin", 1)
	payload := json.RawMessage(`{}`)

	if err := websocket.RouteEventForTest("unknown-event", payload, client); err == nil {
		t.Fatal("expected the first event (within burst) to reach the handler-lookup error")
	}

	deadline := time.Now().Add(50 * time.Millisecond)
	for time.Now().Before(deadline) {
		if err := websocket.RouteEventForTest("unknown-event", payload, client); err != nil {
			return // refilled and reached the handler-lookup error again, as expected
		}
	}
	t.Fatal("expected the rate limiter to refill and eventually allow another event through")
}
