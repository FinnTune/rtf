package websocket

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"rtForum/database"
	"rtForum/utility"
	"strconv"
	"time"

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

var (
	ctx     = context.Background()
	manager = newManager(ctx)
)

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
				if client.connection != nil {
					client.connection.Close()
					client.connection = nil
				}

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
			//Check to see if client is already logged in
			m.Lock()
			defer m.Unlock()
			for client := range m.clients {
				if userInfo.Username == client.username {
					if client.loggedIn {
						log.Println("Client already logged in.")
						client.connection.Close()
						//Delete client from manage client list
						delete(m.clients, client)
						//Delete client from LoggedInList map
						delete(LoggedInList, client.username)
					}
				}
			}

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
				http.Error(w, err.Error(), http.StatusInternalServerError)
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
		// Find the client with the matching session ID
		m.Lock()
		defer m.Unlock()
		log.Println("Manager's clients: ", m.clients)
		for client := range m.clients {
			if client.sessionID == sessionCookie.Value {
				log.Println("Client found.")
				if client.loggedIn {
					// If the client is found, the user is logged in
					client.loggedIn = false
					delete(LoggedInList, client.username)
					log.Println("Logged in users: ", LoggedInList)
					log.Println("Session cookie found. User logged in.")
					client.connection.Close()
					client.connection = nil
					// m.removeClient(client)
					delete(m.clients, client)

					data, err := json.Marshal(LoggedInList)
					if err != nil {
						fmt.Printf("failed to marshal broadcast message error: %s", err)
						// return fmt.Errorf("failed to marshal broadcast message error: %s", err)
					}
					outgoingEvent := Event{
						Payload: json.RawMessage(data),
						Type:    UsersList,
					}

					for c := range manager.clients {
						c.egress <- outgoingEvent
					}

					// Send the login status to the client
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(UserLoginResponse{
						LoggedIn: client.loggedIn,
					})
					return
				}
			}
		}
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	manager.serveLogout(w, r)
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
	log.Println("ServeWS request body: ", r.Body)

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
	for c := range m.clients {
		if c.sessionID == sessionID {
			log.Println("Client already exists.")
			log.Println("ClientUName Debug: ", c.username)
			delete(LoggedInList, c.username)
			LoggedInList[c.username] = true
			c.connection = conn
			go c.readMessages()
			go c.writeMesssage()
			return
		}
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
	log.Printf("Registering user: %s", r.Body)

	//Decode request body to struct
	var user = RegUser{}
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		log.Printf("Error decoding request body: %s", err)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

func AllPostsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		log.Println("AllPostsHandler reached.")
		//Get all posts from database
		query := `SELECT * FROM post;`
		rows, err := database.ForumDB.Query(query)
		if err != nil {
			log.Printf("Error executing query: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			posts = append(posts, post)
		}
		//Encode posts slice to json and send to w
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(posts)
	}
}

func AddPost(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {

		// Decode request body to struct
		var requestBody struct {
			UserID     int    `json:"userID"`
			Title      string `json:"title"`
			Content    string `json:"content"`
			Author     string `json:"author"`
			Categories []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"categories"`
		}

		err := json.NewDecoder(r.Body).Decode(&requestBody)
		if err != nil {
			// Handle error
			http.Error(w, err.Error(), http.StatusBadRequest)
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
		post := DBPost{
			UserID:   requestBody.UserID, // Replace with actual UserID
			UserName: requestBody.Author, // Replace with actual UserName
			Title:    requestBody.Title,
			Content:  requestBody.Content,
		}

		// Insert the post into the post table
		insertPostQuery := "INSERT INTO post (user_id, title, content, author, created_at) VALUES (?, ?, ?, ?, ?)"
		result, err := database.ForumDB.Exec(insertPostQuery, post.UserID, post.Title, post.Content, post.UserName, createdAt)
		if err != nil {
			log.Fatal(err)
		}

		// Get the auto-generated post ID
		postID, err := result.LastInsertId()
		if err != nil {
			log.Fatal(err)
		}

		// Store the category relations in the category_relation table
		insertCategoryQuery := "INSERT INTO category_relation (category_id, post_id) VALUES (?, ?)"
		for _, category := range requestBody.Categories {
			_, err := database.ForumDB.Exec(insertCategoryQuery, category.ID, postID)
			if err != nil {
				log.Fatal(err)
			}
		}
		log.Println("Post added successfully: ", post)

		// Handle success
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Post Successful"))
	}
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

		args := make([]interface{}, len(categories.Categories))
		for i, v := range categories.Categories {
			args[i] = v
		}

		query := `SELECT DISTINCT post.id, post.user_id, post.title, post.content, post.author, post.created_at 
		FROM post 
		INNER JOIN category_relation ON post.id = category_relation.post_id 
		WHERE category_relation.category_id IN (
		SELECT id FROM category WHERE category_name IN (`

		for range categories.Categories {
			query += "?,"
		}
		// remove the last comma
		query = query[:len(query)-1] + "))"

		log.Printf("Executing query: %s", query)
		log.Printf("With arguments: %+v", args)

		rows, err := database.ForumDB.Query(query, args...)
		if err != nil {
			log.Printf("Error executing query: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Printf("Scanned post: %+v", post)
			posts = append(posts, post)
		}
		log.Printf("Processed %d posts", postCount)

		if err = rows.Err(); err != nil {
			log.Printf("Rows processing error: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
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

	log.Println("Adding comment...", comment)

	// Use your existing database connection to insert the comment
	_, err = database.ForumDB.Exec(`
	INSERT INTO comment (user_id, post_id, content, created_at) 
	VALUES ($1, $2, $3, $4)`,
		comment.UserID, comment.PostID, comment.Content, created)

	if err != nil {
		http.Error(w, "Failed to add comment", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
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

	rows, err := database.ForumDB.Query(`
	SELECT c.id, c.user_id, u.uname, c.post_id, c.content, c.created_at 
	FROM comment c 
	INNER JOIN user u ON c.user_id = u.id 
	WHERE c.post_id = $1`, postId)
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
	json.NewEncoder(w).Encode(comments)
}
