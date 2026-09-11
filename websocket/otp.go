package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SessionID binds this OTP to the session it was minted for
// (checkLogin/serveLogin always mint against the caller's own, just-verified
// session). Without this, presenting a validly-minted OTP alongside an
// unrelated, made-up session_id at the /ws upgrade would let anyone who can
// log in once mint OTPs at will and pair each with an arbitrary session_id
// to register phantom, "logged in" clients with no real identity (see
// verifyOtp and ServeWS's no-existing-client branch).
type otpObj struct {
	Key       string
	SessionID string
	Created   time.Time
}

// otpsMap holds one-time passwords used to authorize a websocket upgrade.
// It's accessed both under the Manager's lock (minting, from serveLogin and
// checkLogin) and without it (verifying, from ServeWS; expiring, from the
// background sweep below) — those two call sites can never share the
// Manager's mutex, so this map needs its own independent synchronization
// instead of relying on the caller to hold one.
type otpsMap struct {
	mu   sync.Mutex
	data map[string]otpObj
}

// Factory function to create a new otps map
func newOtpsMap(ctx context.Context, expiryDuration time.Duration) *otpsMap {
	oMap := &otpsMap{data: make(map[string]otpObj)}

	// Go routine to check otps map for expired otps and delete them
	go oMap.checkOtps(ctx, expiryDuration)

	return oMap
}

func (oM *otpsMap) newOtp(sessionID string) otpObj {
	oM.mu.Lock()
	defer oM.mu.Unlock()

	oObj := otpObj{
		Key:       uuid.NewString(),
		SessionID: sessionID,
		Created:   time.Now(),
	}
	oM.data[oObj.Key] = oObj
	return oObj
}

// verifyOtp reports whether otp is a live, unused OTP minted for sessionID
// specifically — not merely a live, unused OTP. A caller who knows a valid
// otp but not the sessionID it was minted for (i.e. anyone but the session
// that requested it) must not be able to redeem it.
func (oM *otpsMap) verifyOtp(otp, sessionID string) bool {
	oM.mu.Lock()
	defer oM.mu.Unlock()

	oObj, ok := oM.data[otp]
	if !ok {
		return false // otp not found
	}
	delete(oM.data, otp) //Because ONE! time password is verified, delete it from map
	return oObj.SessionID == sessionID
}

// Function to check otps map for expired otps and delete them using a go routine ticker channel
func (oM *otpsMap) checkOtps(ctx context.Context, expiryDuration time.Duration) {
	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			oM.mu.Lock()
			for key, otp := range oM.data {
				if otp.Created.Add(expiryDuration).Before(time.Now()) {
					delete(oM.data, key)
				}
			}
			oM.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}
