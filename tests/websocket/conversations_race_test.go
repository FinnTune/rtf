package websocket_test

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"rtForum/tests/testutil"
	"rtForum/websocket"
)

// TestOpenDirectChat_ConcurrentFirstMessageIsRaceSafe exercises the exact
// scenario resolveOrCreateDirectConversation's comment calls out: two users'
// very first message to each other arriving at (effectively) the same time,
// from both directions, must still collapse onto a single conversation row
// rather than each request "winning" a check-then-create race.
func TestOpenDirectChat_ConcurrentFirstMessageIsRaceSafe(t *testing.T) {
	websocket.ResetTestState()
	db := testutil.UseForumDB(t)
	// database/sql's default unlimited pool would hand concurrent goroutines
	// separate connections — and a bare ":memory:" DSN (no shared cache) gives
	// each connection its own isolated database, which would silently defeat
	// this test (every goroutine "wins" its own private race). Forcing a
	// single shared connection is what actually lets the goroutines below
	// interleave against one real database, the way concurrent requests
	// through the production connection pool do.
	db.SetMaxOpenConns(1)

	// admin (1) and actual_user (42) is the base seed's only pair with no
	// pre-existing direct conversation (see TestOpenDirectChat_CreatesAndReturnsConversation).
	const n = 10
	var wg sync.WaitGroup
	results := make([]int, n)
	errs := make([]error, n)

	// Client handles are created up front, sequentially — AddTestClient
	// writes directly into the manager's client map with no locking of its
	// own, so creating them concurrently would race the test harness itself
	// rather than the production code under test.
	clients := make([]*websocket.TestClientHandle, n)
	targets := make([]string, n)
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			clients[i] = websocket.AddTestClient(fmt.Sprintf("s%d", i), "admin", 1)
			targets[i] = "actual_user"
		} else {
			clients[i] = websocket.AddTestClient(fmt.Sprintf("s%d", i), "actual_user", 42)
			targets[i] = "admin"
		}
	}

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			payload, _ := json.Marshal(websocket.OpenDirectChatRequest{Username: targets[i]})
			if err := websocket.OpenDirectChatForTest(payload, clients[i]); err != nil {
				errs[i] = err
				return
			}
			eventType, eventPayload, ok := clients[i].WaitEvent(2 * time.Second)
			if !ok {
				errs[i] = fmt.Errorf("timed out waiting for chat-opened")
				return
			}
			if eventType != websocket.ChatOpened {
				errs[i] = fmt.Errorf("expected chat-opened, got %q", eventType)
				return
			}
			var info websocket.ConversationInfo
			if err := json.Unmarshal(eventPayload, &info); err != nil {
				errs[i] = err
				return
			}
			results[i] = info.ConversationID
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d failed: %v", i, err)
		}
	}

	first := results[0]
	if first == 0 {
		t.Fatal("expected a non-zero conversation id")
	}
	for i, id := range results {
		if id != first {
			t.Fatalf("expected every concurrent open to resolve to the same conversation, goroutine %d got %d, want %d", i, id, first)
		}
	}

	var convCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation WHERE direct_pair_key = ?`, "1-42").Scan(&convCount); err != nil {
		t.Fatalf("failed to count conversations: %v", err)
	}
	if convCount != 1 {
		t.Fatalf("expected exactly 1 conversation row for the pair, got %d — the race was not safe", convCount)
	}

	var memberCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversation_member WHERE conversation_id = ?`, first).Scan(&memberCount); err != nil {
		t.Fatalf("failed to count conversation members: %v", err)
	}
	if memberCount != 2 {
		t.Fatalf("expected exactly 2 conversation_member rows, got %d", memberCount)
	}
}
