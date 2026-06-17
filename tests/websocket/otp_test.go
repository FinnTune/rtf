package websocket_test

import (
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
