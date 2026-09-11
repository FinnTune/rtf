package websocket_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"rtForum/websocket"
)

func TestNewOtpAndVerify(t *testing.T) {
	otps := websocket.NewTestOtps(5 * time.Second)
	defer otps.Close()

	key := otps.NewKey("session-a")
	if key == "" {
		t.Fatal("expected non-empty OTP key")
	}
	if !otps.Verify(key, "session-a") {
		t.Fatal("expected OTP to verify successfully against the session it was minted for")
	}
}

// TestVerifyOtp_WrongSessionRejected is the core regression test for the
// session-binding fix: an otp minted for one session must not verify
// against a different one, even though the otp itself is valid and unused.
// Before this fix, otpObj carried no session identity at all, so any
// authenticated user could mint an otp for their own session and redeem it
// here with an arbitrary, made-up session_id — see ServeWS's
// no-existing-client branch, which would register a fresh Client with
// userID 0 and an empty username for whatever session_id was presented.
func TestVerifyOtp_WrongSessionRejected(t *testing.T) {
	otps := websocket.NewTestOtps(5 * time.Second)
	defer otps.Close()

	key := otps.NewKey("session-a")
	if otps.Verify(key, "session-b") {
		t.Fatal("expected an otp minted for session-a to be rejected when presented with session-b")
	}
	// Also confirm it's genuinely consumed (one-time use) rather than left
	// usable by the failed cross-session attempt above.
	if otps.Verify(key, "session-a") {
		t.Fatal("expected the otp to have been consumed by the earlier verify attempt, even though that attempt failed")
	}
}

func TestVerifyOtp_OneTimeUse(t *testing.T) {
	otps := websocket.NewTestOtps(5 * time.Second)
	defer otps.Close()

	key := otps.NewKey("session-a")

	if !otps.Verify(key, "session-a") {
		t.Fatal("expected first verification to succeed")
	}
	if otps.Verify(key, "session-a") {
		t.Fatal("expected OTP to be invalid after first use")
	}
}

func TestVerifyOtp_Invalid(t *testing.T) {
	otps := websocket.NewTestOtps(5 * time.Second)
	defer otps.Close()

	if otps.Verify("not-a-real-otp", "session-a") {
		t.Fatal("expected unknown OTP to fail verification")
	}
}

func TestOtpExpiry(t *testing.T) {
	otps := websocket.NewTestOtps(50 * time.Millisecond)
	defer otps.Close()

	key := otps.NewKey("session-a")
	time.Sleep(600 * time.Millisecond)

	if otps.Verify(key, "session-a") {
		t.Fatal("expected expired OTP to fail verification")
	}
}

// TestOtp_ConcurrentAccess exercises minting, verifying, and (via the short
// expiry) the background sweep all racing against each other on the same
// map. It's only useful run with -race — the otp map used to be a bare
// map[string]otpObj with no synchronization of its own, mutated from
// serveLogin/checkLogin (under the Manager's lock) and from ServeWS plus the
// background expiry sweep (both lock-free), which is a data race regardless
// of whether this test happens to observe a corrupted value.
func TestOtp_ConcurrentAccess(t *testing.T) {
	otps := websocket.NewTestOtps(20 * time.Millisecond)
	defer otps.Close()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sessionID := fmt.Sprintf("session-%d", i)
			key := otps.NewKey(sessionID)
			otps.Verify(key, sessionID)
		}(i)
	}
	wg.Wait()
}
