package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"rtForum/database"
	"rtForum/logfiles"
	"rtForum/utility"
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
quitPrompt:
	xpressed := ""
	fmt.Println("Type 'x' and 'enter' to quit server.")
	fmt.Scan(&xpressed)
	if xpressed == "x" {
		log.Println("Server stopped.")
		os.Exit(0)
	} else {
		goto quitPrompt // Go back to quitPrompt if anything but 'x' and 'enter' is pressed.
	}
}

// Opens database, starts file servers, starts handlers, and starts server
func startServer() {
	//Open database
	database.ForumDB = database.OpenDB()
	defer func() {
		database.ForumDB.Close()
		log.Println("Database closed.")
	}()
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
	//Serve index.html for all root requests to comply with Single Page Application (SPA) design
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//get request URL and store in variable to be used in log message
		url := r.URL.Path
		log.Printf("Handling request \"%s\" and serving index.html", url)

		//Check if cookie exists and create if not
		if utility.CheckCookieExist(w, r) {
			log.Println("Cookie exists.")
		} else {
			log.Println("Cookie does not exist. Creating cookie.")
			utility.CreateCookie(w, r)
		}
		http.ServeFile(w, r, "./frontend/index.html")
	})
	http.HandleFunc("/checkLogin", websocket.CheckLoginHandler)
	http.HandleFunc("/logout", websocket.LogoutHandler)
	http.HandleFunc("/register", websocket.RegistrationHandler)
	http.HandleFunc("/login", websocket.LoginHandler)
	http.HandleFunc("/ws", websocket.WebsocketHandler)

	// Declare and initialize server struct then listen and serve
	ser := &http.Server{
		Addr:    ":443", // Port 443 is used for HTTPS
		Handler: http.DefaultServeMux,
	}

	// localhost.crt and localhost.key files were created using the following CLI commands:
	// openssl req  -new  -newkey rsa:2048  -nodes  -keyout localhost.key  -out localhost.csr
	// openssl  x509  -req  -days 365  -in localhost.csr  -signkey localhost.key  -out localhost.crt
	log.Printf("Server Started and listening on port %s.", ser.Addr)
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

	// Declare and open the log file for appending, defer close, and set for output, set flags for log file lines.
	logFile, err := os.OpenFile(dir+filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Log file could not be opened: %s", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Log to the file that new forum server has started with timestamp
	log.Println("Main begun. Log file checked, opened, and set.")
	log.Println("New Forum Begun")

	// Initialize server start message, run go routine to prompt quit server function, and start server.
	initMessage()
	go quitServer()
	startServer()
}
