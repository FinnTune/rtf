package websocket

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"rtForum/database"
	"rtForum/utility"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	sqlite3 "github.com/mattn/go-sqlite3"
)

var (
	websocketUpgrader = websocket.Upgrader{
		CheckOrigin:     checkOrigin,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	ctx     = context.Background()
	manager = newManager(ctx)
)

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	log.Printf("Checking origin: %s", origin)
	allowedOrigin := "https://localhost:8443"
	if envOrigin := os.Getenv("ALLOWED_ORIGIN"); envOrigin != "" {
		allowedOrigin = envOrigin
	}
	return origin == allowedOrigin
}

func authenticatedClientFromRequest(r *http.Request) (*Client, error) {
	sessionCookie, err := r.Cookie("session_id")
	if err != nil {
		return nil, err
	}

	manager.Lock()
	defer manager.Unlock()
	for client := range manager.clients {
		if client.sessionID == sessionCookie.Value && client.loggedIn {
			if client.expired() {
				log.Println("Session expired for client:", client.username)
				client.closeConnection()
				delete(manager.clients, client)
				break
			}
			client.touch()
			return client, nil
		}
	}
	return nil, fmt.Errorf("no authenticated client found")
}

func (m *Manager) checkLogin(w http.ResponseWriter, r *http.Request) {
	// Get the session cookie from the request
	log.Println("Checking login status.")
	sessionCookie, err := r.Cookie("session_id")
	if err != nil {
		// If the cookie is not set, the user is not logged in
		log.Println("No session cookie found. User not logged in.")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(UserLoginResponse{
			LoggedIn: false,
		})
		return
	}

	// Find the client with the matching session ID
	m.Lock()
	defer m.Unlock()
	log.Println("Manager's clients: ", m.clients)
	for client := range m.clients {
		if client.sessionID == sessionCookie.Value {
			if client.expired() {
				log.Println("Session expired for client:", client.username)
				client.closeConnection()
				delete(m.clients, client)
				utility.ClearCookie(w)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(UserLoginResponse{
					LoggedIn: false,
				})
				return
			}
			client.touch()
			utility.RefreshCookie(w, client.sessionID)
			if !client.loggedIn {
				log.Println("Client found.")
				json.NewEncoder(w).Encode(UserLoginResponse{
					LoggedIn: client.loggedIn,
				})
				return
			} else if client.loggedIn {
				// If the client is found, the user is logged in
				log.Println("Session cookie found. User logged in")
				client.loggedIn = true
				client.closeConnection()

				// Otp
				//Create new OTP and store in manager otps map
				otp := m.otps.newOtp()

				// Send the login status to the client
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(UserLoginResponse{
					Username: client.username,
					Email:    client.email,
					Joined:   client.joined,
					LoggedIn: client.loggedIn,
					OTP:      otp.Key,
				})
				log.Println("OTP: ", otp.Key)
				return
			}
		}
	}

	// If no client was found with the matching session ID, the user is not logged in
	log.Println("No client found with matching session ID. User not logged in.")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		LoggedIn bool `json:"loggedIn"`
	}{
		LoggedIn: false,
	})
}

func CheckLoginHandler(w http.ResponseWriter, r *http.Request) {
	manager.checkLogin(w, r)
}

func (m *Manager) serveLogin(w http.ResponseWriter, r *http.Request) {
	log.Println("Login handler reached.")
	//Check if user is already logged in
	// if utility.CheckCookieExist(w, r) {
	// 	log.Println("User already logged in.")
	// 	http.Redirect(w, r, "/", http.StatusSeeOther)

	type userLoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	//create struct to hold login request data
	var req userLoginRequest

	//Check if request is POST and decode request body into struct above
	if r.Method == http.MethodPost {
		log.Println("Login POST request received.")
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Error decoding request: %s", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := validateLogin(req.Username, req.Password); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		//Create instance of User struct to hold user info from database
		userInfo := User{}

		//Query database for user info, scan into struct, and check if password matches
		err := database.ForumDB.QueryRow("SELECT id, uname, email, pass, created_at FROM user WHERE uname = $1 OR email = $1", req.Username).Scan(&userInfo.ID, &userInfo.Username, &userInfo.Email, &userInfo.Password, &userInfo.Joined)
		if err != nil {
			log.Printf("Error querying database: %s", err)
			if err == sql.ErrNoRows {
				log.Printf("User not found: %+v\n", userInfo)
			}
		} else if utility.CheckPasswordHash(req.Password, userInfo.Password) {

			log.Printf("User found: %+v\n", userInfo)
			log.Println("Authentication condition reached.")
			log.Println("User Login list: ", LoggedInList)

			// Rotate the session cookie on every successful login so a
			// session_id observed/fixed before authentication can never be
			// reused to hijack the now-authenticated session.
			sessionID := utility.CreateCookie(w, r)

			//Check to see if client is already logged in
			m.Lock()
			defer m.Unlock()
			for client := range m.clients {
				if userInfo.Username == client.username {
					if client.loggedIn {
						log.Println("Client already logged in.")
						client.closeConnection()
						//Delete client from manage client list
						delete(m.clients, client)
						//Delete client from LoggedInList map
						LoggedInList.Remove(client.username)
					}
				}
			}

			// Bind the new session to the identity just verified against the
			// database. This Client is the single source of truth every
			// other handler (HTTP and websocket) trusts for "who is this
			// request/connection from" — it must never be set from
			// client-supplied data (see addUserInfo), only from a server-side
			// credential check like the one above.
			m.clients[newAuthenticatedClient(m, sessionID, userInfo.ID, userInfo.Username, userInfo.Email, userInfo.Joined)] = true

			//Create new OTP and store in manager otps map
			otp := m.otps.newOtp()

			resp := UserLoginResponse{
				OTP:      otp.Key,
				ID:       userInfo.ID,
				Username: userInfo.Username,
				Email:    userInfo.Email,
				Joined:   userInfo.Joined,
				LoggedIn: false,
			}

			//Marhsal response otp struct into JSON and write to 'w'.
			//Encode response to JSON using json.Encode or marhsalling. Difference???
			// err := json.NewEncoder(w).Encode(resp)
			data, err := json.Marshal(resp)
			if err != nil {
				log.Printf("Error marshalling response: %s", err)
				http.Error(w, "Failed to process login", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write(data)
			return
		}

	}
	w.WriteHeader(http.StatusUnauthorized)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	manager.serveLogin(w, r)
}

func (m *Manager) serveLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		log.Println("Logout POST request received.")
		// Always drop the cookie on logout, even if no matching in-memory
		// client is found below (e.g. it already expired server-side) -
		// logout must never leave a reusable session_id in the browser.
		utility.ClearCookie(w)
		//Check if user is logged in
		sessionCookie, err := r.Cookie("session_id")
		if err != nil {
			// If the cookie is not set, the user is not logged in
			log.Println("No session cookie found. User not logged in.")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(UserLoginResponse{
				LoggedIn: false,
			})
			return
		}
		// Find the client with the matching session ID. Looked up and
		// released under lock rather than held for the whole function —
		// m.removeClient below acquires the same lock itself, and holding it
		// across that call (or across the egress sends further down, which
		// can block on a slow/stuck client since egress is unbuffered) would
		// either deadlock or stall every other manager operation.
		m.Lock()
		var client *Client
		for c := range m.clients {
			if c.sessionID == sessionCookie.Value {
				client = c
				break
			}
		}
		m.Unlock()

		if client == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(UserLoginResponse{
				LoggedIn: false,
			})
			return
		}

		log.Println("Client found.")
		// If the client is found, the user is logged in
		client.loggedIn = false
		LoggedInList.Remove(client.username)
		m.removeClient(client)

		data, err := json.Marshal(LoggedInList.Snapshot())
		if err != nil {
			fmt.Printf("failed to marshal broadcast message error: %s", err)
			// return fmt.Errorf("failed to marshal broadcast message error: %s", err)
		}
		outgoingEvent := Event{
			Payload: json.RawMessage(data),
			Type:    UsersList,
		}

		log.Println("Logout and new users list sent")

		for _, recipient := range m.clientsSnapshot() {
			recipient.egress <- outgoingEvent
		}

		// Send the login status to the client
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(UserLoginResponse{
			LoggedIn: client.loggedIn,
		})
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	manager.serveLogout(w, r) // logout
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

	//JSON decode r.body into request struct
	log.Println("Decoding request body.")

	//Get cookie from request
	cookie, err := r.Cookie("session_id")
	if err != nil {
		log.Printf("Error getting cookie: %s", err)
		return
	}

	//Get cookie value and check if client already exists
	//If client exists, set connection to new connection and start client routines
	sessionID := cookie.Value
	log.Println("Session Id in ServeWS: ", sessionID)

	// m.clients is shared with every other handler (login, logout,
	// checkLogin, ...), all of which access it under m.Lock() — this lookup
	// must too, or it's an unsynchronized concurrent map access. The lock is
	// released before starting the client's goroutines / falling through to
	// m.addClient below, both of which acquire it themselves.
	m.Lock()
	var existing *Client
	for c := range m.clients {
		if c.sessionID == sessionID {
			if c.expired() {
				log.Println("Session expired; discarding stale client:", c.username)
				delete(m.clients, c)
			} else {
				existing = c
			}
			break
		}
	}
	m.Unlock()

	if existing != nil {
		log.Println("Client already exists.")
		log.Println("ClientUName Debug: ", existing.username)
		LoggedInList.Remove(existing.username)
		LoggedInList.Add(existing.username)
		existing.setConnection(conn)
		existing.touch()
		go existing.readMessages()
		go existing.writeMesssage()
		return
	}

	//If client does not exist, create new client,
	//set loggedIn to true, add client to manager,
	//and start client routines
	log.Println("Client does not exist.")
	//Create new client
	client := newClient(conn, m, sessionID)
	//Set client loggedIn to true
	client.loggedIn = true
	client.cookie = cookie

	//Add client to manager
	m.addClient(client)

	//Add user to LoggedInUsers struct
	// LoggedInUsers[client.username] = client
	//Add user to LoggedInList struct

	//Start client routines for reading and writing messages
	go client.readMessages()
	go client.writeMesssage()
}

// Catch manager and send to ServeWS
func WebsocketHandler(w http.ResponseWriter, r *http.Request) {
	manager.ServeWS(w, r)
}

func registerUser(w http.ResponseWriter, r *http.Request) {
	//Decode request body to struct
	var user = RegUser{}
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		log.Printf("Error decoding request body: %s", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validateRegistration(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//Hash password in user struct
	user.Pass = utility.HashPassword(user.Pass)

	//Insert user into database
	timeReg := time.Now().Format("2006-01-02 15:04:05")
	query := `INSERT INTO user (fname,lname,uname,email,age,gender,pass,created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?);`
	result, err := database.ForumDB.Exec(query,
		user.Fname,
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
		if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.Code == sqlite3.ErrConstraint {
			http.Error(w, "Username or email already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to register user", http.StatusInternalServerError)
		return
	}
	log.Printf("User registered result: %s", result)

	//Send message to w that registration was successful
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Registration successful."))
}

func RegistrationHandler(w http.ResponseWriter, r *http.Request) {
	registerUser(w, r)
}

// AllPostsHandler returns a page of posts, newest first. Callers may pass
// ?limit= and ?offset= query params; invalid or missing values fall back to
// sane defaults rather than erroring, since pagination is an optional
// refinement, not a required input. The total matching row count is
// reported via the X-Total-Count response header so the frontend can render
// Prev/Next controls without a second round trip.
func AllPostsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		log.Println("AllPostsHandler reached.")

		limit := defaultPostsPageSize
		if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
			limit = v
		}
		if limit > maxPostsPageSize {
			limit = maxPostsPageSize
		}

		offset := 0
		if v, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && v >= 0 {
			offset = v
		}

		var total int
		if err := database.ForumDB.QueryRow("SELECT COUNT(*) FROM post").Scan(&total); err != nil {
			log.Printf("Error counting posts: %s", err)
			http.Error(w, "Failed to load posts", http.StatusInternalServerError)
			return
		}

		//Get a page of posts from database, newest first. id DESC breaks ties
		//between posts created within the same second.
		query := `SELECT * FROM post ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?;`
		rows, err := database.ForumDB.Query(query, limit, offset)
		if err != nil {
			log.Printf("Error executing query: %s", err)
			http.Error(w, "Failed to load posts", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		//Create slice of posts
		posts := []Post{}

		//Iterate through rows and append to posts slice
		for rows.Next() {
			var post Post
			err = rows.Scan(&post.PostId, &post.UserId, &post.Title, &post.Content, &post.Author, &post.Created)
			if err != nil {
				log.Printf("Error scanning rows: %s", err)
				http.Error(w, "Failed to load posts", http.StatusInternalServerError)
				return
			}
			posts = append(posts, post)
		}
		//Encode posts slice to json and send to w
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Total-Count", strconv.Itoa(total))
		json.NewEncoder(w).Encode(posts)
	}
}

// GetPostHandler returns a single post by id, for deep-linking to a post
// via /posts/:id.
func GetPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id <= 0 {
		http.Error(w, "a valid id is required", http.StatusBadRequest)
		return
	}

	var post Post
	err = database.ForumDB.QueryRow("SELECT * FROM post WHERE id = ?", id).
		Scan(&post.PostId, &post.UserId, &post.Title, &post.Content, &post.Author, &post.Created)
	if err == sql.ErrNoRows {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error fetching post: %s", err)
		http.Error(w, "Failed to load post", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(post)
}

// GetCategoriesHandler returns every category from the database, so the
// frontend's filter checkboxes and post-creation category picker read from
// a single source of truth instead of duplicating a hardcoded list that can
// drift out of sync with the actual category table.
func GetCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := database.ForumDB.Query("SELECT id, category_name FROM category ORDER BY category_name ASC")
	if err != nil {
		log.Printf("Error querying categories: %s", err)
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	categories := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			log.Printf("Error scanning category: %s", err)
			http.Error(w, "Failed to load categories", http.StatusInternalServerError)
			return
		}
		categories = append(categories, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

// SearchPostsHandler returns posts whose title or content contains the
// query string (case-insensitive), newest first, capped at
// maxSearchResults.
func SearchPostsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query, err := validateSearchQuery(r.URL.Query().Get("q"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	likePattern := "%" + escapeLikePattern(query) + "%"

	rows, err := database.ForumDB.Query(
		`SELECT * FROM post WHERE title LIKE ? ESCAPE '\' OR content LIKE ? ESCAPE '\' ORDER BY created_at DESC, id DESC LIMIT ?`,
		likePattern, likePattern, maxSearchResults,
	)
	if err != nil {
		log.Printf("Error executing search query: %s", err)
		http.Error(w, "Failed to search posts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	posts := []Post{}
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.PostId, &post.UserId, &post.Title, &post.Content, &post.Author, &post.Created); err != nil {
			log.Printf("Error scanning rows: %s", err)
			http.Error(w, "Failed to search posts", http.StatusInternalServerError)
			return
		}
		posts = append(posts, post)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

func AddPost(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {

		// Decode request body to struct
		var requestBody struct {
			Title      string `json:"title"`
			Content    string `json:"content"`
			Categories []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"categories"`
		}

		err := json.NewDecoder(r.Body).Decode(&requestBody)
		if err != nil {
			// Handle error
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		title, content, err := validatePost(requestBody.Title, requestBody.Content)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		requestBody.Title = title
		requestBody.Content = content

		if len(requestBody.Categories) > maxCategoriesPerPost {
			http.Error(w, fmt.Sprintf("a post may have at most %d categories", maxCategoriesPerPost), http.StatusBadRequest)
			return
		}

		log.Println("Add Post Request body: ", requestBody)

		// Connect to the database when mySQL!!!
		// db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@/%s", dbUsername, dbPassword, dbName))
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// defer db.Close()

		// Store the post in the post table
		createdAt := time.Now().Format("2006-01-02 15:04:05")
		//createdAt := time.Now().Format("02-01-2006 15:04")

		// You can obtain the UserID and UserName from the authenticated user
		client, err := authenticatedClientFromRequest(r)
		if err != nil {
			utility.ClearCookie(w)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		post := DBPost{
			UserID:   client.userID,
			UserName: client.username,
			Title:    requestBody.Title,
			Content:  requestBody.Content,
		}

		// Insert the post into the post table
		insertPostQuery := "INSERT INTO post (user_id, title, content, author, created_at) VALUES (?, ?, ?, ?, ?)"
		result, err := database.ForumDB.Exec(insertPostQuery, post.UserID, post.Title, post.Content, post.UserName, createdAt)
		if err != nil {
			log.Printf("failed to insert post: %s", err)
			http.Error(w, "Failed to create post", http.StatusInternalServerError)
			return
		}

		// Get the auto-generated post ID
		postID, err := result.LastInsertId()
		if err != nil {
			log.Printf("failed to fetch inserted post id: %s", err)
			http.Error(w, "Failed to create post", http.StatusInternalServerError)
			return
		}

		// Store the category relations in the category_relation table
		insertCategoryQuery := "INSERT INTO category_relation (category_id, post_id) VALUES (?, ?)"
		for _, category := range requestBody.Categories {
			_, err := database.ForumDB.Exec(insertCategoryQuery, category.ID, postID)
			if err != nil {
				log.Printf("failed to insert post category relation: %s", err)
				http.Error(w, "Failed to assign category", http.StatusInternalServerError)
				return
			}
		}
		log.Println("Post added successfully: ", post)

		// Handle success
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Post Successful"))
	}
}

// EditPostHandler updates the title/content of a post. Only the post's
// author may edit it.
func EditPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestBody struct {
		ID         int    `json:"id"`
		Title      string `json:"title"`
		Content    string `json:"content"`
		Categories []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"categories"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	title, content, err := validatePost(requestBody.Title, requestBody.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(requestBody.Categories) > maxCategoriesPerPost {
		http.Error(w, fmt.Sprintf("a post may have at most %d categories", maxCategoriesPerPost), http.StatusBadRequest)
		return
	}

	client, err := authenticatedClientFromRequest(r)
	if err != nil {
		utility.ClearCookie(w)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var ownerID int
	err = database.ForumDB.QueryRow("SELECT user_id FROM post WHERE id = ?", requestBody.ID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("failed to look up post owner: %s", err)
		http.Error(w, "Failed to update post", http.StatusInternalServerError)
		return
	}
	if ownerID != client.userID {
		http.Error(w, "You can only edit your own posts", http.StatusForbidden)
		return
	}

	tx, err := database.ForumDB.Begin()
	if err != nil {
		log.Printf("failed to begin transaction: %s", err)
		http.Error(w, "Failed to update post", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec("UPDATE post SET title = ?, content = ? WHERE id = ?", title, content, requestBody.ID); err != nil {
		log.Printf("failed to update post: %s", err)
		http.Error(w, "Failed to update post", http.StatusInternalServerError)
		return
	}

	// Replace the post's category relations wholesale rather than diffing,
	// mirroring how AddPost assigns them on creation.
	if _, err := tx.Exec("DELETE FROM category_relation WHERE post_id = ?", requestBody.ID); err != nil {
		log.Printf("failed to clear post category relations: %s", err)
		http.Error(w, "Failed to update post", http.StatusInternalServerError)
		return
	}
	for _, category := range requestBody.Categories {
		if _, err := tx.Exec("INSERT INTO category_relation (category_id, post_id) VALUES (?, ?)", category.ID, requestBody.ID); err != nil {
			log.Printf("failed to insert post category relation: %s", err)
			http.Error(w, "Failed to update post", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("failed to commit post update: %s", err)
		http.Error(w, "Failed to update post", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"title": title, "content": content})
}

// GetPostCategoriesHandler returns the categories currently assigned to a
// post, for pre-checking the right boxes when opening the edit form.
func GetPostCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	postId, err := strconv.Atoi(r.URL.Query().Get("postId"))
	if err != nil || postId <= 0 {
		http.Error(w, "a valid postId is required", http.StatusBadRequest)
		return
	}

	rows, err := database.ForumDB.Query(`
	SELECT category.id, category.category_name
	FROM category
	INNER JOIN category_relation ON category.id = category_relation.category_id
	WHERE category_relation.post_id = ?
	ORDER BY category.category_name ASC`, postId)
	if err != nil {
		log.Printf("Error querying post categories: %s", err)
		http.Error(w, "Failed to load categories", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	categories := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			log.Printf("Error scanning category: %s", err)
			http.Error(w, "Failed to load categories", http.StatusInternalServerError)
			return
		}
		categories = append(categories, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

// DeletePostHandler removes a post along with its comments and category
// relations. Only the post's author may delete it.
func DeletePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestBody struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if requestBody.ID <= 0 {
		http.Error(w, "a valid id is required", http.StatusBadRequest)
		return
	}

	client, err := authenticatedClientFromRequest(r)
	if err != nil {
		utility.ClearCookie(w)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var ownerID int
	err = database.ForumDB.QueryRow("SELECT user_id FROM post WHERE id = ?", requestBody.ID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("failed to look up post owner: %s", err)
		http.Error(w, "Failed to delete post", http.StatusInternalServerError)
		return
	}
	if ownerID != client.userID {
		http.Error(w, "You can only delete your own posts", http.StatusForbidden)
		return
	}

	tx, err := database.ForumDB.Begin()
	if err != nil {
		log.Printf("failed to begin transaction: %s", err)
		http.Error(w, "Failed to delete post", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM comment WHERE post_id = ?", requestBody.ID); err != nil {
		log.Printf("failed to delete post comments: %s", err)
		http.Error(w, "Failed to delete post", http.StatusInternalServerError)
		return
	}
	if _, err := tx.Exec("DELETE FROM category_relation WHERE post_id = ?", requestBody.ID); err != nil {
		log.Printf("failed to delete post category relations: %s", err)
		http.Error(w, "Failed to delete post", http.StatusInternalServerError)
		return
	}
	if _, err := tx.Exec("DELETE FROM post WHERE id = ?", requestBody.ID); err != nil {
		log.Printf("failed to delete post: %s", err)
		http.Error(w, "Failed to delete post", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		log.Printf("failed to commit post deletion: %s", err)
		http.Error(w, "Failed to delete post", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Post deleted"))
}

func PostsByCategoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Get the category ID from the query string
		log.Println("GettingPostsByCategory...")
		var categories Categories

		// Decode the request body into the categories struct
		err := json.NewDecoder(r.Body).Decode(&categories)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		log.Println("PostsByCategoryHandler reached.")
		log.Printf("Categories: %+v", categories)

		if len(categories.Categories) > maxCategoriesPerReq {
			http.Error(w, fmt.Sprintf("at most %d categories may be requested", maxCategoriesPerReq), http.StatusBadRequest)
			return
		}

		if len(categories.Categories) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Total-Count", "0")
			json.NewEncoder(w).Encode([]Post{})
			return
		}

		limit := defaultPostsPageSize
		if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
			limit = v
		}
		if limit > maxPostsPageSize {
			limit = maxPostsPageSize
		}

		offset := 0
		if v, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && v >= 0 {
			offset = v
		}

		args := make([]interface{}, len(categories.Categories))
		for i, v := range categories.Categories {
			args[i] = v
		}

		placeholders := strings.Repeat("?,", len(categories.Categories))
		placeholders = placeholders[:len(placeholders)-1]

		whereClause := `INNER JOIN category_relation ON post.id = category_relation.post_id
		WHERE category_relation.category_id IN (
		SELECT id FROM category WHERE category_name IN (` + placeholders + `))`

		var total int
		countQuery := `SELECT COUNT(DISTINCT post.id) FROM post ` + whereClause
		if err := database.ForumDB.QueryRow(countQuery, args...).Scan(&total); err != nil {
			log.Printf("Error counting posts by category: %s", err)
			http.Error(w, "Failed to load posts", http.StatusInternalServerError)
			return
		}

		query := `SELECT DISTINCT post.id, post.user_id, post.title, post.content, post.author, post.created_at
		FROM post ` + whereClause + `
		ORDER BY post.created_at DESC, post.id DESC
		LIMIT ? OFFSET ?`
		queryArgs := append(append([]interface{}{}, args...), limit, offset)

		log.Printf("Executing query: %s", query)
		log.Printf("With arguments: %+v", queryArgs)

		rows, err := database.ForumDB.Query(query, queryArgs...)
		if err != nil {
			log.Printf("Error executing query: %s", err)
			http.Error(w, "Failed to load posts", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		posts := []Post{}
		postCount := 0
		for rows.Next() {
			postCount++
			var post Post
			err = rows.Scan(&post.PostId, &post.UserId, &post.Title, &post.Content, &post.Author, &post.Created)
			if err != nil {
				log.Printf("Error scanning rows: %s", err)
				http.Error(w, "Failed to load posts", http.StatusInternalServerError)
				return
			}
			log.Printf("Scanned post: %+v", post)
			posts = append(posts, post)
		}
		log.Printf("Processed %d posts", postCount)

		if err = rows.Err(); err != nil {
			log.Printf("Rows processing error: %s", err)
			http.Error(w, "Failed to load posts", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Total-Count", strconv.Itoa(total))
		json.NewEncoder(w).Encode(posts)
	}
}

func AddCommentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var comment Comment
	created := time.Now().Format("2006-01-02 15:04:05")
	err := json.NewDecoder(r.Body).Decode(&comment)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	content, err := validateComment(comment.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	comment.Content = content

	if comment.PostID <= 0 {
		http.Error(w, "a valid post_id is required", http.StatusBadRequest)
		return
	}

	client, err := authenticatedClientFromRequest(r)
	if err != nil {
		utility.ClearCookie(w)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	log.Println("Adding comment...", comment)

	// Use your existing database connection to insert the comment
	result, err := database.ForumDB.Exec(`
	INSERT INTO comment (user_id, post_id, content, created_at)
	VALUES ($1, $2, $3, $4)`,
		client.userID, comment.PostID, comment.Content, created)

	if err != nil {
		http.Error(w, "Failed to add comment", http.StatusInternalServerError)
		return
	}

	commentID, err := result.LastInsertId()
	if err != nil {
		log.Printf("failed to fetch inserted comment id: %s", err)
		http.Error(w, "Failed to add comment", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	comment.ID = int(commentID)
	comment.UserID = client.userID
	comment.Username = client.username
	comment.CreatedAt = created
	json.NewEncoder(w).Encode(comment)
}

func GetCommentsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Get post ID from the query parameters
	keys, ok := r.URL.Query()["postId"]

	if !ok || len(keys[0]) < 1 {
		http.Error(w, "Missing postId parameter", http.StatusBadRequest)
		return
	}

	postId, err := strconv.Atoi(keys[0])
	if err != nil {
		http.Error(w, "postId must be an integer", http.StatusBadRequest)
		return
	}

	limit := defaultCommentsPageSize
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
		limit = v
	}
	if limit > maxCommentsPageSize {
		limit = maxCommentsPageSize
	}

	offset := 0
	if v, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && v >= 0 {
		offset = v
	}

	var total int
	if err := database.ForumDB.QueryRow("SELECT COUNT(*) FROM comment WHERE post_id = ?", postId).Scan(&total); err != nil {
		http.Error(w, "Failed to query database", http.StatusInternalServerError)
		return
	}

	rows, err := database.ForumDB.Query(`
	SELECT c.id, c.user_id, u.uname, c.post_id, c.content, c.created_at
	FROM comment c
	INNER JOIN user u ON c.user_id = u.id
	WHERE c.post_id = $1
	ORDER BY c.created_at ASC, c.id ASC
	LIMIT $2 OFFSET $3`, postId, limit, offset)
	if err != nil {
		http.Error(w, "Failed to query database", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	log.Println("GetComments Rows: ", rows)

	comments := make([]Comment, 0)

	for rows.Next() {
		var comment Comment
		if err := rows.Scan(&comment.ID, &comment.UserID, &comment.Username, &comment.PostID, &comment.Content, &comment.CreatedAt); err != nil {
			http.Error(w, "Failed to scan row", http.StatusInternalServerError)
			return
		}

		comments = append(comments, comment)
	}
	log.Println("Comments sent: ", comments)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	json.NewEncoder(w).Encode(comments)
}

// EditCommentHandler updates a comment's content. Only the comment's author
// may edit it.
func EditCommentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestBody struct {
		ID      int    `json:"id"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	content, err := validateComment(requestBody.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client, err := authenticatedClientFromRequest(r)
	if err != nil {
		utility.ClearCookie(w)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var ownerID int
	err = database.ForumDB.QueryRow("SELECT user_id FROM comment WHERE id = ?", requestBody.ID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		http.Error(w, "Comment not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("failed to look up comment owner: %s", err)
		http.Error(w, "Failed to update comment", http.StatusInternalServerError)
		return
	}
	if ownerID != client.userID {
		http.Error(w, "You can only edit your own comments", http.StatusForbidden)
		return
	}

	if _, err := database.ForumDB.Exec("UPDATE comment SET content = ? WHERE id = ?", content, requestBody.ID); err != nil {
		log.Printf("failed to update comment: %s", err)
		http.Error(w, "Failed to update comment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"content": content})
}

// DeleteCommentHandler removes a comment. Only the comment's author may
// delete it.
func DeleteCommentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestBody struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if requestBody.ID <= 0 {
		http.Error(w, "a valid id is required", http.StatusBadRequest)
		return
	}

	client, err := authenticatedClientFromRequest(r)
	if err != nil {
		utility.ClearCookie(w)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var ownerID int
	err = database.ForumDB.QueryRow("SELECT user_id FROM comment WHERE id = ?", requestBody.ID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		http.Error(w, "Comment not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("failed to look up comment owner: %s", err)
		http.Error(w, "Failed to delete comment", http.StatusInternalServerError)
		return
	}
	if ownerID != client.userID {
		http.Error(w, "You can only delete your own comments", http.StatusForbidden)
		return
	}

	if _, err := database.ForumDB.Exec("DELETE FROM comment WHERE id = ?", requestBody.ID); err != nil {
		log.Printf("failed to delete comment: %s", err)
		http.Error(w, "Failed to delete comment", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Comment deleted"))
}
