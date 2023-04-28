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

// Create Cookie for user logging in
func CreateCookie(w http.ResponseWriter, r *http.Request) {
	sessionID := uuid.Must(uuid.NewV4()).String()

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	//Different struct for user info???
	// sql.UpdateUser(data.User{Id: userId, Session: sessionToken}, true)
}

func CheckCookieExist(w http.ResponseWriter, r *http.Request) bool {
	_, err := r.Cookie("session_token")
	//The function returns the opposite of the comparison err != http.ErrNoCookie,
	//which means it returns true if the cookie exists and false otherwise.
	return err != http.ErrNoCookie
}
