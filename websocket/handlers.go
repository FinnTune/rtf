package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"rtForum/database"
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
	type userLoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var req userLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//Check if user is valid by hardcoding username and password
	//Harcoded username and password???
	if req.Username == "admin" && req.Password == "123" {
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
	// CREATE TABLE user (
	// 	id INTEGER NOT NULL PRIMARY KEY,
	// 	username VARCHAR(30) NOT NULL,
	// 	passwrd VARCHAR(100) NOT NULL,
	// 	email VARCHAR(30) NOT NULL,
	// 	fname VARCHAR(30) NOT NULL,
	// 	lname VARCHAR(30) NOT NULL,
	// 	age INTEGER NOT NULL,
	// 	gender VARCHAR(10) NOT NULL,
	// 	created_at DATETIME NOT NULL
	//    );
	// type RegUser struct {
	// 	fname  string `json:"fname"`
	// 	lname  string `json:"lname"`
	// 	uname  string `json:"uname"`
	// 	email  string `json:"email"`
	// 	age    int    `json:"age"`
	// 	gender string `json:"gender"`
	// 	pass   string `json:"pass"`
	// }

	var user = RegUser{}
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		log.Printf("Error decoding request body: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("User: %s", user)
	log.Printf("User: %T", user.Age)

	timeReg := time.Now().Format("2006-01-02 15:04:05")
	query := `INSERT INTO user (fname,lname,uname,email,age,gender,pass,created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?);`
	result, err := database.ForumDB.Exec(query, user.Fname,
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
	log.Printf("DbResult: %s", result)
	w.WriteHeader(http.StatusOK)
	//Send message to w that registration was successful
	w.Write([]byte("Registration successful."))
}

func RegistrationHandler(w http.ResponseWriter, r *http.Request) {
	registerUser(w, r)
}
