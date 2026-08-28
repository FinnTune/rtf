package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type otpObj struct {
	Key     string
	Created time.Time
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

func (oM *otpsMap) newOtp() otpObj {
	oM.mu.Lock()
	defer oM.mu.Unlock()

	oObj := otpObj{
		Key:     uuid.NewString(),
		Created: time.Now(),
	}
	oM.data[oObj.Key] = oObj
	return oObj
}

func (oM *otpsMap) verifyOtp(otp string) bool {
	oM.mu.Lock()
	defer oM.mu.Unlock()

	if _, ok := oM.data[otp]; !ok {
		return false // otp not found
	}
	delete(oM.data, otp) //Because ONE! time password is verified, delete it from map
	return true
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
