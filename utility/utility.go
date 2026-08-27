package utility

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes the password
func HashPassword(password string) string {
	byt, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		fmt.Println("Could not generate password", err.Error())
	}
	return string(byt)
}

// CheckPasswordHash compares the password and hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		log.Printf("Error comparing password and hash: %s", err.Error())
		return false
	}
	log.Println("Password and hash match.")
	return true
}

// SessionDuration is the idle timeout for a session: both the session_id
// cookie's Max-Age and the server-side session expiry (see Client.expired
// in the websocket package) are derived from this single value so the two
// never drift apart.
const SessionDuration = 24 * time.Hour

// CreateCookie issues a fresh session_id cookie and returns its value, so
// callers that need to bind the new session to a verified identity (see
// serveLogin) don't have to re-derive or guess it.
func CreateCookie(w http.ResponseWriter, r *http.Request) string {
	sessionID := uuid.Must(uuid.NewV4()).String()

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(SessionDuration),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	return sessionID
}

// RefreshCookie re-issues the session_id cookie with the same value but a
// renewed expiry, implementing the sliding half of the session's
// expiration/refresh policy: an active session stays alive, an idle one
// still expires after SessionDuration.
func RefreshCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(SessionDuration),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

// ClearCookie actively expires the session_id cookie in the browser. Used on
// logout and when a stale/expired session is detected server-side, so a
// dead session can never be silently reused.
func ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
}

func CheckCookieExist(w http.ResponseWriter, r *http.Request) bool {
	_, err := r.Cookie("session_id")
	//The function returns the opposite of the comparison err != http.ErrNoCookie,
	//which means it returns true if the cookie exists and false otherwise.
	return err != http.ErrNoCookie
}
