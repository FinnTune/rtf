package websocket

import (
	"net/http"
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

// Struct to define a post
type Category struct {
	ID           int
	CategoryName string
	Description  string
	CreatedAt    string
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

// Struct to define a message
type Message struct {
	ID       int
	Username string
	// The username of the recipient of the message (if it is a private message)
	RecipientUsername string
	Content           string
	CreatedAt         string
	From              int       `json:"from"`
	Text              string    `json:"text"`
	ChatHistory       []Message `json:"chathistory"`
	To                int       `json:"to,omitempty"`
	Read              int       `json:"isread"`
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
}
