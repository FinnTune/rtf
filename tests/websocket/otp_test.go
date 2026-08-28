package websocket_test

import (
	"sync"
	"testing"
	"time"

	"rtForum/websocket"
)

func TestNewOtpAndVerify(t *testing.T) {
	otps := websocket.NewTestOtps(5 * time.Second)
	defer otps.Close()

	key := otps.NewKey()
	if key == "" {
		t.Fatal("expected non-empty OTP key")
	}
	if !otps.Verify(key) {
		t.Fatal("expected OTP to verify successfully")
	}
}

func TestVerifyOtp_OneTimeUse(t *testing.T) {
	otps := websocket.NewTestOtps(5 * time.Second)
	defer otps.Close()

	key := otps.NewKey()

	if !otps.Verify(key) {
		t.Fatal("expected first verification to succeed")
	}
	if otps.Verify(key) {
		t.Fatal("expected OTP to be invalid after first use")
	}
}

func TestVerifyOtp_Invalid(t *testing.T) {
	otps := websocket.NewTestOtps(5 * time.Second)
	defer otps.Close()

	if otps.Verify("not-a-real-otp") {
		t.Fatal("expected unknown OTP to fail verification")
	}
}

func TestOtpExpiry(t *testing.T) {
	otps := websocket.NewTestOtps(50 * time.Millisecond)
	defer otps.Close()

	key := otps.NewKey()
	time.Sleep(600 * time.Millisecond)

	if otps.Verify(key) {
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
		go func() {
			defer wg.Done()
			key := otps.NewKey()
			otps.Verify(key)
		}()
	}
	wg.Wait()
}
