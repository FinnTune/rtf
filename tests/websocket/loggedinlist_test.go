package websocket_test

import (
	"fmt"
	"sync"
	"testing"

	"rtForum/websocket"
)

// TestLoggedInList_ConcurrentAccess exercises the online-users set from many
// goroutines at once — the same shape of access it gets in production from
// concurrent login/logout HTTP handlers and every connected client's own
// independent read/write goroutines. Only useful run with -race: the
// underlying map used to have no synchronization of its own.
func TestLoggedInList_ConcurrentAccess(t *testing.T) {
	websocket.ResetTestState()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			username := fmt.Sprintf("user%d", i%5) // deliberate overlap across goroutines
			websocket.SetLoggedInList(username)
			websocket.IsInLoggedInList(username)
		}(i)
	}
	wg.Wait()
}
