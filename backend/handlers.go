package backend

import (
	"net/http"
)

// func (m *Manager) loginHandler(w http.ResponseWriter, r *http.Request) {
// 	type userLoginRequest struct {
// 		Username string `json:"username"`
// 		Password string `json:"password"`
// 	}
// 	var req userLoginRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return
// 	}

// 	//Check if user is valid by hardcoding username and password
// 	if req.Username == "admin" && req.Password == "123" {
// 		type userLoginResponse struct {
// 			OTP string `json:"otp"`
// 		}
// 		otp := m.otps.newOtp()

// 		resp := userLoginResponse{
// 			OTP: otp.Key,
// 		}

// 		//Encode response to JSON using json.Encode or marhsalling. Difference???
// 		// err := json.NewEncoder(w).Encode(resp)
// 		data, err := json.Marshal(resp)
// 		if err != nil {
// 			log.Printf("Error marshalling response: %s", err)
// 			http.Error(w, err.Error(), http.StatusInternalServerError)
// 			return
// 		}

// 		w.WriteHeader(http.StatusOK)
// 		w.Write(data)
// 		return
// 	}
// 	w.WriteHeader(http.StatusUnauthorized)
// }

func LoginHandler(w http.ResponseWriter, r *http.Request) {

}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {

}
