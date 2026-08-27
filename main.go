package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"rtForum/database"
	"rtForum/logfiles"
	"rtForum/utility"
	"rtForum/websocket"
	"syscall"
	"time"

	"golang.org/x/time/rate"
)

// ASCI esacpe codes for colors
const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
)

// initMessage prints a message when the server starts
func initMessage() {
	port := getEnv("PORT", "8443")
	fmt.Printf(Cyan + "===============================================\n")
	fmt.Printf(Magenta + "Starting Realtime Forum\n")
	fmt.Printf(Magenta+"Server is running on port: "+Blue+"%s\n", port)
	fmt.Printf(Cyan + "===============================================\n")
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// buildServer registers all routes/handlers and returns the configured HTTP server.
func buildServer() *http.Server {
	log.Println("File Servers Started.")
	cssFS := http.FileServer(http.Dir("./frontend/css"))
	http.Handle("/css/", http.StripPrefix("/css/", cssFS))

	jsFS := http.FileServer(http.Dir("./frontend/js"))
	http.Handle("/js/", http.StripPrefix("/js/", jsFS))

	imgFS := http.FileServer(http.Dir("./frontend/img"))
	http.Handle("/img/", http.StripPrefix("/img/", imgFS))

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

	// authLimiter guards login/registration against brute-force and signup
	// spam: ~5 requests/minute per IP, with a small burst for legitimate
	// typo-and-retry flows.
	authLimiter := utility.NewIPRateLimiter(rate.Every(12*time.Second), 5)
	// writeLimiter guards authenticated write endpoints against automated
	// abuse while staying out of the way of normal posting/commenting.
	writeLimiter := utility.NewIPRateLimiter(rate.Every(2*time.Second), 10)
	// readLimiter guards unauthenticated, DB-backed read endpoints (post
	// listing/search/comments) against scraping and query-flood abuse.
	// Generous relative to writeLimiter since normal use (paging through
	// posts, searching, opening a post's comments) can legitimately fire
	// several of these per second.
	readLimiter := utility.NewIPRateLimiter(rate.Every(200*time.Millisecond), 20)

	http.HandleFunc("/checkLogin", websocket.CheckLoginHandler)
	http.HandleFunc("/getAllPosts", readLimiter.Limit(websocket.AllPostsHandler))
	http.HandleFunc("/getPost", readLimiter.Limit(websocket.GetPostHandler))
	http.HandleFunc("/getCategories", readLimiter.Limit(websocket.GetCategoriesHandler))
	http.HandleFunc("/logout", websocket.CSRFProtect(websocket.LogoutHandler))
	http.HandleFunc("/register", authLimiter.Limit(websocket.CSRFProtect(websocket.RegistrationHandler)))
	http.HandleFunc("/login", authLimiter.Limit(websocket.CSRFProtect(websocket.LoginHandler)))
	http.HandleFunc("/ws", websocket.WebsocketHandler)
	http.HandleFunc("/addPost", writeLimiter.Limit(websocket.CSRFProtect(websocket.AddPost)))
	http.HandleFunc("/editPost", writeLimiter.Limit(websocket.CSRFProtect(websocket.EditPostHandler)))
	http.HandleFunc("/deletePost", writeLimiter.Limit(websocket.CSRFProtect(websocket.DeletePostHandler)))
	http.HandleFunc("/getPostsByCategory", readLimiter.Limit(websocket.PostsByCategoryHandler))
	http.HandleFunc("/searchPosts", readLimiter.Limit(websocket.SearchPostsHandler))
	http.HandleFunc("/addcomment", writeLimiter.Limit(websocket.CSRFProtect(websocket.AddCommentHandler)))
	http.HandleFunc("/editComment", writeLimiter.Limit(websocket.CSRFProtect(websocket.EditCommentHandler)))
	http.HandleFunc("/deleteComment", writeLimiter.Limit(websocket.CSRFProtect(websocket.DeleteCommentHandler)))
	http.HandleFunc("/comments", readLimiter.Limit(websocket.GetCommentsHandler))

	port := getEnv("PORT", "8443")
	return &http.Server{
		Addr:    ":" + port,
		Handler: securityHeaders(http.DefaultServeMux),
	}
}

// securityHeaders sets a conservative set of security response headers on
// every response. The CSP has no external origins and no inline
// scripts/styles because the whole frontend is same-origin static assets
// and ES modules - it would need loosening if that ever changes.
func securityHeaders(next http.Handler) http.Handler {
	const csp = "default-src 'self'; " +
		"script-src 'self'; " +
		"style-src 'self'; " +
		"img-src 'self'; " +
		"connect-src 'self'; " +
		"frame-ancestors 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		h.Set("Content-Security-Policy", csp)
		next.ServeHTTP(w, r)
	})
}

// runServer opens the database, starts the HTTP server, and blocks until ctx
// is cancelled (SIGINT/SIGTERM), then shuts the server down gracefully.
func runServer(ctx context.Context) {
	//Open database
	database.ForumDB = database.OpenDB()
	defer func() {
		database.ForumDB.Close()
		log.Println("Database closed.")
	}()

	ser := buildServer()

	// localhost.crt and localhost.key files were created using the following CLI commands:
	// openssl req  -new  -newkey rsa:2048  -nodes  -keyout localhost.key  -out localhost.csr
	// openssl  x509  -req  -days 365  -in localhost.csr  -signkey localhost.key  -out localhost.crt
	tlsCert := getEnv("TLS_CERT", "localhost.crt")
	tlsKey := getEnv("TLS_KEY", "localhost.key")

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Server Started and listening on port %s.", ser.Addr)
		err := ser.ListenAndServeTLS(tlsCert, tlsKey)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			log.Fatalf("ListenAndServeTLS error: %s", err)
		}
	case <-ctx.Done():
		fmt.Println(Red + "Shutting down server... (Ctrl+C again to force)")
		log.Println("Shutdown signal received. Shutting down server.")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := ser.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error during server shutdown: %s", err)
		}
		<-serverErr
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

	initMessage()

	// Ctrl+C (SIGINT) or a `docker stop`/systemd stop (SIGTERM) trigger a graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runServer(ctx)

	log.Println("Server stopped.")
}
