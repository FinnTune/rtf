package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"rtForum/database"
	"rtForum/logfiles"
	"rtForum/utility"
	"rtForum/websocket"
	"strings"
	"syscall"
	"time"

	"golang.org/x/time/rate"
)

// distDir holds the React frontend's production build (webapp/, built by
// the Dockerfile's node stage — see buildServer's catch-all handler below).
const distDir = "./webapp/dist"

// uploadsDir holds user-uploaded post images — bind-mounted the same way
// ./database is in docker-compose.yml, so uploads survive a container
// recreate. Served back under /uploads/, matching websocket.postImageURLPrefix.
const uploadsDir = "./uploads"

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
	slog.Info("file servers started")
	fileServer := http.FileServer(http.Dir(distDir))

	slog.Info("handlers started")
	// Serves the React app's build output. A request for a real built file
	// (e.g. /assets/index-abc123.js, /img/favglobe.ico) gets that file
	// directly, with long-lived caching for content-hashed assets; anything
	// else (e.g. /posts/5, /users/alice) falls back to index.html so
	// react-router owns client-side routing.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Path
		slog.Debug("handling request", "path", url)

		//Check if cookie exists and create if not
		if utility.CheckCookieExist(w, r) {
			slog.Debug("cookie exists")
		} else {
			slog.Debug("cookie does not exist, creating cookie")
			utility.CreateCookie(w, r)
		}

		cleanPath := path.Clean(url)
		if isRegularFile(filepath.Join(distDir, cleanPath)) {
			if strings.HasPrefix(cleanPath, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fileServer.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
	})

	// Serves uploaded post images by their (random, unguessable) filename.
	// noDirListing keeps a bare directory request from returning Go's
	// default file listing — there's never a reason to enumerate this dir.
	uploadsFileServer := http.FileServer(http.Dir(uploadsDir))
	http.Handle("/uploads/", http.StripPrefix("/uploads/", noDirListing(uploadsFileServer, uploadsDir)))

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

	http.HandleFunc("/healthz", healthzHandler)
	http.HandleFunc("/checkLogin", websocket.CheckLoginHandler)
	http.HandleFunc("/getAllPosts", readLimiter.Limit(websocket.AllPostsHandler))
	http.HandleFunc("/getPostsByAuthor", readLimiter.Limit(websocket.GetPostsByAuthorHandler))
	http.HandleFunc("/getPost", readLimiter.Limit(websocket.GetPostHandler))
	http.HandleFunc("/getCategories", readLimiter.Limit(websocket.GetCategoriesHandler))
	http.HandleFunc("/getPostCategories", readLimiter.Limit(websocket.GetPostCategoriesHandler))
	http.HandleFunc("/createCategory", writeLimiter.Limit(websocket.CSRFProtect(websocket.RequireAdmin(websocket.CreateCategoryHandler))))
	http.HandleFunc("/editCategory", writeLimiter.Limit(websocket.CSRFProtect(websocket.RequireAdmin(websocket.EditCategoryHandler))))
	http.HandleFunc("/deleteCategory", writeLimiter.Limit(websocket.CSRFProtect(websocket.RequireAdmin(websocket.DeleteCategoryHandler))))
	http.HandleFunc("/listUsers", readLimiter.Limit(websocket.RequireAdmin(websocket.ListUsersHandler)))
	http.HandleFunc("/setUserBanned", writeLimiter.Limit(websocket.CSRFProtect(websocket.RequireAdmin(websocket.SetUserBannedHandler))))
	http.HandleFunc("/logout", websocket.CSRFProtect(websocket.LogoutHandler))
	http.HandleFunc("/register", authLimiter.Limit(websocket.CSRFProtect(websocket.RegistrationHandler)))
	http.HandleFunc("/login", authLimiter.Limit(websocket.CSRFProtect(websocket.LoginHandler)))
	http.HandleFunc("/ws", websocket.WebsocketHandler)
	http.HandleFunc("/addPost", writeLimiter.Limit(websocket.CSRFProtect(websocket.AddPost)))
	http.HandleFunc("/editPost", writeLimiter.Limit(websocket.CSRFProtect(websocket.EditPostHandler)))
	http.HandleFunc("/deletePost", writeLimiter.Limit(websocket.CSRFProtect(websocket.DeletePostHandler)))
	http.HandleFunc("/reactToPost", writeLimiter.Limit(websocket.CSRFProtect(websocket.ReactToPostHandler)))
	http.HandleFunc("/uploadPostImage", writeLimiter.Limit(websocket.CSRFProtect(websocket.UploadPostImageHandler)))
	http.HandleFunc("/getPostsByCategory", readLimiter.Limit(websocket.PostsByCategoryHandler))
	http.HandleFunc("/searchPosts", readLimiter.Limit(websocket.SearchPostsHandler))
	http.HandleFunc("/searchMessages", readLimiter.Limit(websocket.SearchMessagesHandler))
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

// healthzHandler backs the Dockerfile's HEALTHCHECK. It pings the database
// rather than just returning 200 unconditionally, so a wedged/unreachable DB
// connection (the actual failure mode that would leave the process alive but
// unable to serve real requests) marks the container unhealthy too.
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := database.ForumDB.Ping(); err != nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

// isRegularFile reports whether path names an existing, non-directory file
// under distDir — used by the catch-all handler to tell a real built asset
// apart from a client-side route it should fall back to index.html for.
func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// noDirListing wraps a file server so a request for a directory 404s
// instead of returning Go's default directory listing. Used for /uploads/,
// which only ever needs to serve individual files by their random filename.
func noDirListing(next http.Handler, root string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		if isDir(filepath.Join(root, path.Clean(r.URL.Path))) {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
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
	dbPath := getEnv("DB_PATH", "./database/forum.db")
	database.ForumDB = database.OpenDB(dbPath)
	defer func() {
		database.ForumDB.Close()
		slog.Info("database closed")
	}()

	ser := buildServer()

	// localhost.crt and localhost.key files were created using the following CLI commands:
	// openssl req  -new  -newkey rsa:2048  -nodes  -keyout localhost.key  -out localhost.csr
	// openssl  x509  -req  -days 365  -in localhost.csr  -signkey localhost.key  -out localhost.crt
	tlsCert := getEnv("TLS_CERT", "localhost.crt")
	tlsKey := getEnv("TLS_KEY", "localhost.key")

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("server started", "addr", ser.Addr)
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
			slog.Error("listen and serve TLS failed", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		fmt.Println(Red + "Shutting down server... (Ctrl+C again to force)")
		slog.Info("shutdown signal received, shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := ser.Shutdown(shutdownCtx); err != nil {
			slog.Error("error during server shutdown", "error", err)
		}
		<-serverErr
	}
}

func main() {
	//Checking if logfile exists.
	dir := "./logfiles/"
	filename := "forum.log"
	logfiles.CheckLog(dir, filename)

	// Declare and open the log file for appending, defer close, and point the
	// default slog logger at it. Every slog.Info/Warn/Error/Debug call
	// anywhere in the process (this package and every package it imports)
	// goes through this one default logger, so pointing it at the file here
	// is what actually makes the rest of the app's structured logging land
	// in forum.log instead of stderr.
	logFile, err := os.OpenFile(dir+filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("log file could not be opened", "error", err)
		os.Exit(1)
	}
	defer logFile.Close()
	slog.SetDefault(slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{AddSource: true})))

	// Log to the file that new forum server has started with timestamp
	slog.Info("main begun: log file checked, opened, and set")
	slog.Info("new forum begun")

	initMessage()

	// Ctrl+C (SIGINT) or a `docker stop`/systemd stop (SIGTERM) trigger a graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runServer(ctx)

	slog.Info("server stopped")
}
