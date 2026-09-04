package websocket_test

import (
	"rtForum/websocket"
	"testing"
)

func TestSweepExpiredClients_RemovesExpiredClient(t *testing.T) {
	websocket.ResetTestState()
	handle := websocket.AddTestClient("session-expired", "stale_user", 1)
	handle.ExpireForTest()

	websocket.SweepExpiredClientsForTest()

	if !handle.IsRemovedFromManager() {
		t.Fatal("expected sweep to remove the expired client")
	}
}

func TestSweepExpiredClients_LeavesActiveClientAlone(t *testing.T) {
	websocket.ResetTestState()
	handle := websocket.AddTestClient("session-active", "active_user", 2)

	websocket.SweepExpiredClientsForTest()

	if handle.IsRemovedFromManager() {
		t.Fatal("expected sweep to leave a non-expired client in place")
	}
}
