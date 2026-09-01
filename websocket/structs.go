package websocket

import (
	"net/http"
	"sync"
	"time"
)

type RegUser struct {
	Fname  string `json:"fname"`
	Lname  string `json:"lname"`
	Uname  string `json:"uname"`
	Email  string `json:"email"`
	Age    string `json:"age"`
	Gender string `json:"gender"`
	Pass   string `json:"password"`
}

type AllPosts struct {
	Posts        []Post
	Categories   []Topic
	UserName     string
	Loggedin     bool
	ErrorMessage string
}
type SinglePost struct {
	Post         Post
	Comments     []Comment
	ErrorMessage string
	LoggedIn     bool
}

type User struct {
	ID             int
	Username       string
	Email          string
	Joined         string
	Password       string
	Role           string
	Session        string
	NumberComments int
	NumberPosts    int
}

type Topic struct {
	Id    int
	Topic string
}

type Post struct {
	PostId  int
	UserId  int
	Title   string
	Content string
	Author  string
	Created string
	// Empty string means "no image" — img_url is nullable in the schema,
	// but every SELECT that populates this struct uses COALESCE(img_url,
	// '') so callers never have to deal with a NULL/empty distinction.
	ImgURL string
	// Populated by attachReactionData, not by the SELECT * that scans the
	// rest of this struct — every post-listing/detail handler calls it
	// separately (see that function's doc comment for why: reactions are a
	// one-to-many table, and joining it directly into a paginated post
	// query invites duplicate-row/GROUP BY bugs). MyReaction is "none" for
	// an anonymous viewer or one who hasn't reacted, never zero-valued
	// empty string — always explicitly set.
	LikeCount    int
	DislikeCount int
	MyReaction   string
}

type DBPost struct {
	UserID   int
	UserName string
	Title    string `json:"title"`
	Content  string `json:"content"`
	Category string `json:"category"`
}

type Categories struct {
	Categories []string `json:"categories"`
}

type GetPost struct {
	ID       int      `json:"id"`
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Author   string   `json:"author"`
	Category Category `json:"category"`
}

type Alert struct {
	AlertMessage string
	AlertCode    int
	AllUserPosts []Post
	AllReactions []Post
	LoggedIn     bool
}

// Global maps to store all users, posts, comments, categories, and sessions
var LoggedInUsers = make(map[string]*Client)

// loggedInSet tracks which usernames are currently online. It's mutated
// concurrently from many goroutines — login/logout/checkLogin HTTP
// handlers, the user-connect event, and every connected client's own
// independent read/write loops on disconnect — so all access goes through
// these locked methods rather than a bare map.
type loggedInSet struct {
	mu    sync.Mutex
	users map[string]bool
}

func newLoggedInSet() *loggedInSet {
	return &loggedInSet{users: make(map[string]bool)}
}

func (s *loggedInSet) Add(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[username] = true
}

func (s *loggedInSet) Remove(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.users, username)
}

func (s *loggedInSet) Has(username string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.users[username]
}

// Snapshot returns a copy safe to marshal or range over without holding the
// lock for the duration (important since callers marshal it into a
// broadcast payload, and some also then range over the manager's clients to
// send it — neither should be done while holding this set's lock).
func (s *loggedInSet) Snapshot() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := make(map[string]bool, len(s.users))
	for k, v := range s.users {
		snapshot[k] = v
	}
	return snapshot
}

func (s *loggedInSet) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users = make(map[string]bool)
}

// LoggedInList is a list of all logged in users.
var LoggedInList = newLoggedInSet()

// Struct to define a session
type UserSession struct {
	Username string `json:"username"`
	UserID   int    `json:"id"`
	Email    string `json:"email"`
	Joined   string `json:"joined"`
	Cookie   *http.Cookie
}

// Struct to define a user account
// type User struct {
// 	ID        int
// 	Username  string
// 	Fname     string
// 	Lname     string
// 	Age       int
// 	Gender    string
// 	Password  string
// 	Email     string
// 	CreatedAt string
// 	Privilege int
// 	Send      chan []byte
// }

// Category mirrors the category table (id, category_name only — there is
// no description/created_at column).
type Category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Comment struct {
	Username  string `json:"username"`
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	PostID    int    `json:"post_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// type Post struct {
// 	ID             int
// 	UserID         int
// 	UserName       string
// 	Title          string
// 	Content        string
// 	CreatedAt      time.Time
// 	UpdatedAt      time.Time
// 	Date           string
// 	LikedNumber    int
// 	DislikedNumber int
// 	ImgUrl         string
// 	URL            string
// 	Approved       int
// 	Dummy          int
// 	IsEdited       bool
// }

type Reaction struct {
	ID        int
	UserID    int
	PostID    int
	IsLiked   int
	CreatedAt string
}

type Relation struct {
	ID         int
	CategoryID int
	PostID     int
}

// Message represents the structure of the message to be stored in the database
type Message struct {
	FromUser  int       `json:"from_user"`
	ToUser    int       `json:"to_user"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// Structs to define a hub (for websockets)
type ServerUser struct {
	Name      string `json:"name"`
	Username  string `json:"username"`
	Privilege int    `json:"privilege"`
}

type ServerMessage struct {
	Type        string           `json:"type"`
	Users       []ServerUser     `json:"users"`
	Categories  []ServerCategory `json:"categories"`
	Posts       []ServerPost     `json:"posts"`
	User        ServerUser       `json:"user"`
	Post        ServerPost       `json:"post"`
	Category    ServerCategory   `json:"category"`
	To          string           `json:"to"`
	From        string           `json:"from"`
	Text        string           `json:"text"`
	Username    string           `json:"username"`
	ChatHistory []Message        `json:"chathistory"`
}

type ServerCategory struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	ID           int    `json:"id"`
	CategoryName string `json:"categoryname"`
	Description  string `json:"description"`
	CreatedAt    string `json:"createdat"`
}
type ServerPost struct {
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	Author         string    `json:"author"`
	Date           string    `json:"date"`
	ID             int       `json:"id"`
	UserID         int       `json:"userid"`
	UserName       string    `json:"username"`
	CreatedAt      time.Time `json:"createdat"`
	UpdatedAt      time.Time `json:"updatedat"`
	LikedNumber    int       `json:"likednumber"`
	DislikedNumber int       `json:"dislikednumber"`
	ImgUrl         string    `json:"imgurl"`
	URL            string    `json:"url"`
	Approved       int       `json:"approved"`
	Dummy          int       `json:"dummy"`
	IsEdited       bool      `json:"isedited"`
}

// OTP response struct
type UserLoginResponse struct {
	OTP      string `json:"otp"`
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Joined   string `json:"joined"`
	LoggedIn bool   `json:"loggedIn"`
	// Role is UX-only — it tells the frontend whether to show admin-only
	// nav/views. It is never trusted as an authorization boundary: every
	// admin-gated endpoint (see RequireAdmin) re-verifies the role from the
	// database on each request regardless of what a client claims.
	Role string `json:"role"`
}

type ChatMessage struct {
	Id        int    `json:"id"`
	FromUser  string `json:"from"`
	ToUser    string `json:"to"`
	IsRead    bool   `json:"is_read"`
	Text      string `json:"message"`
	CreatedAt string `json:"created_at"`
	Limit     int    `json:"limit"`
	Offset    int    `json:"offset"`
}
