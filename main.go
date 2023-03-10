package main

import (
	"log"
	"net/http"
	"os"
	"rtForum/logfiles"
	"time"
)

func startFileServers() {
	log.Println("File Servers Started.")
	cssFS := http.FileServer(http.Dir("./server/css"))
	http.Handle("/css/", http.StripPrefix("/css/", cssFS))

	jsFS := http.FileServer(http.Dir("./server/js"))
	http.Handle("/js/", http.StripPrefix("/js/", jsFS))
	// imgFS := http.FileServer(http.Dir("./server/img"))
}

func startHandlers() {
	log.Println("Handlers Started.")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		println("Handling request.")
		http.ServeFile(w, r, "./server/index.html")
	})
}

func startServer() {
	log.Println("Server Started and listening on port 8080.")
	println("Listening on port 8080.")
	http.ListenAndServe(":8080", nil)
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

	startFileServers()
	startHandlers()
	startServer()

}
