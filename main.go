package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"rtForum/backend"
	"rtForum/logfiles"
	"rtForum/websocket"
)

// initMessage prints a message when the server starts
func initMessage() {
	fmt.Printf("===============================================\n")
	fmt.Printf("Starting Realtime Forum\n")
	fmt.Printf("Server is running on port: " + "443\n")
	fmt.Printf("===============================================\n")
}

// quitServer prompts user to type 'x' and 'enter' to quit server.
func quitServer() {
	xpressed := ""
	fmt.Println("Type 'x' and 'enter' to quit server.")
	fmt.Scan(&xpressed)
	if xpressed == "x" {
		os.Exit(0)
	} else {
		quitServer()
	}
}

func startServer() {
	// Start file servers
	log.Println("File Servers Started.")
	cssFS := http.FileServer(http.Dir("./frontend/css"))
	http.Handle("/css/", http.StripPrefix("/css/", cssFS))

	jsFS := http.FileServer(http.Dir("./frontend/js"))
	http.Handle("/js/", http.StripPrefix("/js/", jsFS))

	imgFS := http.FileServer(http.Dir("./frontend/img"))
	http.Handle("/img/", http.StripPrefix("/img/", imgFS))

	// Start handlers
	log.Printf("Handlers Started.")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//get request URL and store in variable
		url := r.URL.Path
		log.Printf("Handling request \"%s\" and serving index.html", url)
		http.ServeFile(w, r, "./frontend/index.html")
	})
	http.HandleFunc("/login", backend.LoginHandler)
	http.HandleFunc("/register", backend.RegisterHandler)
	http.HandleFunc("/ws", websocket.StartWebSocket)

	// Declare and initialize server struct then listen and serve
	ser := &http.Server{
		Addr:    ":443", // Port 443 is used for HTTPS
		Handler: http.DefaultServeMux,
	}

	// localhost.crt and localhost.key files were created using the following CLI commands:
	// openssl req  -new  -newkey rsa:2048  -nodes  -keyout localhost.key  -out localhost.csr
	// openssl  x509  -req  -days 365  -in localhost.csr  -signkey localhost.key  -out localhost.crt
	log.Println("Server Started and listening on port 443.")
	err := ser.ListenAndServeTLS("localhost.crt", "localhost.key")
	if err != nil {
		log.Fatalf("ListenAndServeTLS error: %s", err)
	}

}

func main() {
	//Checking if logfile exists.
	dir := "./logfiles/"
	filename := "forum.log"
	logfiles.CheckLog(dir, filename)

	// Declare and open the log file for appending, defer close, and set for output
	logFile, err := os.OpenFile(dir+filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Log file could not be opened: %s", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// Log to the file that new forum server has started with timestamp
	log.Println("Main begun. Log file checked, opened, and set.")
	log.Println("New Forum Begun")

	// Initialize server start message, run go routine to prompt quit server function, and start server.
	initMessage()
	go quitServer()
	startServer()
}
