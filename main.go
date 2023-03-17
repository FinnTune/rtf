package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"rtForum/logfiles"
	"time"
)

// InitMessage prints a message when the server starts
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

func startFileServers() {
	log.Println("File Servers Started.")
	cssFS := http.FileServer(http.Dir("./frontend/css"))
	http.Handle("/css/", http.StripPrefix("/css/", cssFS))

	jsFS := http.FileServer(http.Dir("./frontend/js"))
	http.Handle("/js/", http.StripPrefix("/js/", jsFS))
	// imgFS := http.FileServer(http.Dir("./server/img"))
}

func startHandlers() {
	log.Println("Handlers Started.")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		println("Handling request.")
		http.ServeFile(w, r, "./frontend/index.html")
	})
}

func startServer() {
	// Declare and initialize server struct
	ser := &http.Server{
		Addr:    ":443",
		Handler: http.DefaultServeMux,
	}

	// localhost.crt and localhost.key files were created using the following CLI commands:
	// openssl req  -new  -newkey rsa:2048  -nodes  -keyout localhost.key  -out localhost.csr
	// openssl  x509  -req  -days 365  -in localhost.csr  -signkey localhost.key  -out localhost.crt
	log.Println("Server Started and listening on port 8080.")
	println("Listening on port 8080.")
	err := ser.ListenAndServeTLS("localhost.crt", "localhost.key")
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	//Checking if logfile exists.
	dir := "./logfiles/"
	filename := "forum.log"
	logfiles.CheckLog(dir, filename)

	// Open the file for appending and defer close
	logFile, err := os.OpenFile(dir+filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	// Set the output for the log package to the file
	log.SetOutput(logFile)
	log.Println("Main begun. Log file checked, opened, and set.")

	// Log to the file that new forum server has started with timestamp
	newForumBegunDate := time.Now()
	log.Printf("New Forum Begun: %s", newForumBegunDate)

	initMessage()

	// Run go routine to prompt quit server function.
	go quitServer()

	startFileServers()
	startHandlers()
	startServer()

}
