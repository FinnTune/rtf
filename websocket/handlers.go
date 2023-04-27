package websocket

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"rtForum/database"
	"rtForum/utility"
	"time"

	"github.com/gorilla/websocket"
)

var (
	websocketUpgrader = websocket.Upgrader{
		CheckOrigin:     checkOrigin,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	log.Printf("Checking origin: %s", origin)
	switch origin {
	case "https://localhost":
		return true
	default:
		return false
	}
}

var (
	ctx     = context.Background()
	manager = newManager(ctx)
)

func (m *Manager) serveLogin(w http.ResponseWriter, r *http.Request) {
	log.Println("Login handler reached.")
	//Check if user is already logged in
	// if utility.CheckCookieExist(w, r) {
	// 	log.Println("User already logged in.")
	// 	http.Redirect(w, r, "/", http.StatusSeeOther)
	type userLoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var req userLoginRequest
	if r.Method == http.MethodPost {
		log.Println("Login POST request received.")
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Error decoding request: %s", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		userInfo := User{}

		err := database.ForumDB.QueryRow("SELECT id, uname, email, pass, created_at FROM user WHERE uname = $1 OR email = $1", req.Username).Scan(&userInfo.ID, &userInfo.Username, &userInfo.Email, &userInfo.Password, &userInfo.Joined)
		if err != nil {
			log.Printf("Error querying database: %s", err)
			if err == sql.ErrNoRows {
				log.Printf("User not found: %+v\n", userInfo)
			}
		} else if utility.CheckPasswordHash(req.Password, userInfo.Password) {
			log.Printf("User found: %+v\n", userInfo)
			log.Println("Authentication condition reached.")
			type userLoginResponse struct {
				OTP string `json:"otp"`
			}
			otp := m.otps.newOtp()

			resp := userLoginResponse{
				OTP: otp.Key,
			}

			//Encode response to JSON using json.Encode or marhsalling. Difference???
			// err := json.NewEncoder(w).Encode(resp)
			data, err := json.Marshal(resp)
			if err != nil {
				log.Printf("Error marshalling response: %s", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write(data)
			return
		}

	}
	w.WriteHeader(http.StatusUnauthorized)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	manager.serveLogin(w, r)
}

// Serve websocket, upgrade incoming requests, and begin client routines for reading and writing messages
func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {

	//Check if otp is valid
	otp := r.URL.Query().Get("otp")
	if otp == "" {
		w.WriteHeader(http.StatusUnauthorized)
		log.Println("OTP is empty.")
		return
	}

	if !m.otps.verifyOtp(otp) {
		// w.WriteHeader(http.StatusUnauthorized)
		log.Println("OTP is invalid.")
		return
	}

	//Upgrade request to websocket if otp is valid
	log.Println("Serving websocket.")
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := newClient(conn, m)

	m.addClient(client)

	//Start client routines for reading and writing messages
	go client.readMessages()
	go client.writeMesssage()
}

// Catch manager and send to ServeWS
func WebsocketHandler(w http.ResponseWriter, r *http.Request) {
	manager.ServeWS(w, r)
}

func registerUser(w http.ResponseWriter, r *http.Request) {
	log.Printf("Registering user: %s", r.Body)

	//Decode request body to struct
	var user = RegUser{}
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		log.Printf("Error decoding request body: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//Hash password in user struct
	user.Pass = utility.HashPassword(user.Pass)

	//Insert user into database
	timeReg := time.Now().Format("2006-01-02 15:04:05")
	query := `INSERT INTO user (fname,lname,uname,email,age,gender,pass,created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?);`
	result, err := database.ForumDB.Exec(query,
		user.Fname,
		user.Lname,
		user.Uname,
		user.Email,
		user.Age,
		user.Gender,
		user.Pass,
		timeReg,
	)
	if err != nil {
		log.Printf("Error executing user query: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("User registered result: %s", result)

	//Send message to w that registration was successful
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Registration successful."))
}

func RegistrationHandler(w http.ResponseWriter, r *http.Request) {
	registerUser(w, r)
}
