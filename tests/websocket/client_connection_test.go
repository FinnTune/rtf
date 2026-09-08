package websocket_test

import (
	"sync"
	"testing"

	"rtForum/websocket"
)

// TestClientConnection_ConcurrentAccess exercises a single client's
// connection field from many goroutines at once — the same shape of access
// readMessages and writeMesssage's independent cleanup paths have in
// production, which race on a connection drop since both goroutines' I/O
// tends to fail around the same time. No real *websocket.Conn is needed:
// concurrent reads/writes racing on a field is unsafe in Go regardless of
// whether the value itself ever changes, so exercising get/set/close with
// nil is a valid (and panic-free) way to prove the accessors are properly
// synchronized. Only useful run with -race.
func TestClientConnection_ConcurrentAccess(t *testing.T) {
	websocket.ResetTestState()
	client := websocket.AddTestClient("s1", "admin", 1)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.ClearConnectionForTest()
			client.CloseConnectionForTest()
			client.HasConnectionForTest()
		}()
	}
	wg.Wait()
}

// TestClientLastSeen_ConcurrentAccess exercises touch()/expired() from many
// goroutines at once — the same shape of access authenticatedClientFromRequest
// has in production now that it only takes a read lock (RLock) on the
// manager for the common case: touch() itself has to be safe to call
// concurrently with no lock of its own, relying entirely on lastSeen's
// atomic.Int64 storage. Only useful run with -race.
func TestClientLastSeen_ConcurrentAccess(t *testing.T) {
	websocket.ResetTestState()
	client := websocket.AddTestClient("s1", "admin", 1)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.TouchForTest()
			client.IsExpiredForTest()
		}()
	}
	wg.Wait()
}
