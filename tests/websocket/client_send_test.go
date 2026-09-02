package websocket_test

import (
	"encoding/json"
	"testing"
	"time"

	"rtForum/tests/testutil"
	"rtForum/websocket"
)

// TestSend_TimesOutOnStuckRecipientWithoutBlockingOthers proves the fix for
// a real, repeatedly-observed bug: every broadcast (addUserInfo, logout,
// chat delivery, ...) sends to each recipient's unbuffered egress channel,
// which only that client's own writeMesssage goroutine ever drains. A
// client whose connection drops without a clean close (a bare network
// drop — no read/write error ever surfaces) is never removed from
// manager.clients, so its egress channel is left with nothing reading it.
// Before Client.send()'s timeout existed, the very next broadcast that
// reached such a client would block forever, silently stalling whichever
// client's own event triggered it.
func TestSend_TimesOutOnStuckRecipientWithoutBlockingOthers(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	restore := websocket.SetSendTimeoutForTest(20 * time.Millisecond)
	defer restore()

	stuck := websocket.AddTestClient("s-stuck", "stuckuser", 1)
	live := websocket.AddTestClient("s-live", "liveuser", 2)
	_ = stuck

	payload, _ := json.Marshal(websocket.UserSession{})

	// AddTestClient's egress channel has a small (4-slot) buffer, standing
	// in for production's real writeMesssage goroutine briefly being busy —
	// send() should tolerate that without timing out. Saturate it by
	// broadcasting 4 times while only ever draining the live recipient, so
	// stuck's copies pile up unread.
	for i := 0; i < 4; i++ {
		if err := websocket.AddUserInfoForTest(payload, live); err != nil {
			t.Fatalf("priming broadcast %d failed: %v", i, err)
		}
		if _, _, ok := live.WaitEvent(time.Second); !ok {
			t.Fatalf("expected priming broadcast %d to reach the live recipient", i)
		}
	}

	// stuck's buffer is now full and nothing will ever drain it — exactly
	// the "dead client" case. The next broadcast's send to it has nowhere
	// to go and must hit the (shrunk) timeout rather than block this call
	// indefinitely.
	done := make(chan error, 1)
	go func() { done <- websocket.AddUserInfoForTest(payload, live) }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("addUserInfo failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("addUserInfo hung — send() did not time out on the stuck recipient")
	}

	// The live recipient must still receive its own copy, unaffected by the
	// other recipient being stuck.
	if _, _, ok := live.WaitEvent(time.Second); !ok {
		t.Fatal("expected the live recipient to still receive the broadcast despite the stuck one")
	}
}

// TestSend_ParallelBroadcastDoesNotCompoundDelay proves broadcastTo fans out
// concurrently: a broadcast reaching several stuck recipients at once still
// finishes in about one sendTimeout, not sendTimeout multiplied by however
// many recipients happen to be stuck. A sequential loop of send() calls
// would still be correct (each call is individually bounded) but would let
// stale connections accumulate into a real, growing user-facing delay —
// exactly what was observed live: multiple leftover stale clients caused
// several-second delays before this fix.
func TestSend_ParallelBroadcastDoesNotCompoundDelay(t *testing.T) {
	websocket.ResetTestState()
	testutil.UseForumDB(t)
	const timeout = 200 * time.Millisecond
	restore := websocket.SetSendTimeoutForTest(timeout)
	defer restore()

	const numStuck = 5
	for i := 0; i < numStuck; i++ {
		websocket.AddTestClient("s-stuck-"+string(rune('a'+i)), "stuckuser"+string(rune('a'+i)), 100+i)
	}
	live := websocket.AddTestClient("s-live", "liveuser", 2)

	payload, _ := json.Marshal(websocket.UserSession{})

	// Saturate every stuck client's 4-slot buffer the same way as the test
	// above, draining only live each round.
	for i := 0; i < 4; i++ {
		if err := websocket.AddUserInfoForTest(payload, live); err != nil {
			t.Fatalf("priming broadcast %d failed: %v", i, err)
		}
		if _, _, ok := live.WaitEvent(time.Second); !ok {
			t.Fatalf("expected priming broadcast %d to reach the live recipient", i)
		}
	}

	start := time.Now()
	if err := websocket.AddUserInfoForTest(payload, live); err != nil {
		t.Fatalf("addUserInfo failed: %v", err)
	}
	elapsed := time.Since(start)

	// Sequential delivery to 5 stuck clients would take ~5*timeout (1s);
	// parallel delivery should take about one timeout period regardless of
	// how many recipients are stuck. Generous slack for scheduling jitter.
	maxExpected := timeout * 2
	if elapsed > maxExpected {
		t.Fatalf("broadcast to %d stuck recipients took %s, want under %s (parallel, not compounding per recipient)",
			numStuck, elapsed, maxExpected)
	}

	if _, _, ok := live.WaitEvent(time.Second); !ok {
		t.Fatal("expected the live recipient to still receive the broadcast despite several stuck recipients")
	}
}
