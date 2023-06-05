package websocket

import "net/http"

func Handler(w http.ResponseWriter, r *http.Request) {
	// Either servemux or error handling (basics)

	switch r.URL.Path {
	case "routes here":
		// http.HandleFunc("Handler here")
	default:
		// error 404
	}

}
