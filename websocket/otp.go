package websocket

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type otpObj struct {
	Key     string
	Created time.Time
}

// Map key of otpObj.Key to otpObj
type otpsMap map[string]otpObj

// Factory function to create a new otps map
func newOtpsMap(ctx context.Context, expiryDuration time.Duration) otpsMap {
	oMap := make(otpsMap)

	// Go routine to check otps map for expired otps and delete them
	go oMap.checkOtps(ctx, expiryDuration)

	return oMap
}

func (oM otpsMap) newOtp() otpObj {
	oObj := otpObj{
		Key:     uuid.NewString(),
		Created: time.Now(),
	}
	oM[oObj.Key] = oObj
	return oObj
}

func (oM otpsMap) verifyOtp(otp string) bool {
	if _, ok := oM[otp]; !ok {
		return false // otp not found
	}
	delete(oM, otp) //Because ONE! time password is verified, delete it from map
	return true
}

// Function to check otps map for expired otps and delete them using a go routine ticker channel
func (oM otpsMap) checkOtps(ctx context.Context, expiryDuration time.Duration) {
	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, otp := range oM {
				if otp.Created.Add(expiryDuration).Before(time.Now()) {
					delete(oM, otp.Key)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
