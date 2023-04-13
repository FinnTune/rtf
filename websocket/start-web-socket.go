package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

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

func (m *Manager) loginHandler(w http.ResponseWriter, r *http.Request) {
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
	if req.Username == "admin" && req.Password == "123" {
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

func StartWebSocket(w http.ResponseWriter, r *http.Request) {
	// Start websocket and manager
	log.Println("Websocket started.")

	ctx := context.Background()

	manager := newManager(ctx)

	manager.loginHandler(w, r)

	manager.ServeWS(w, r)
}
