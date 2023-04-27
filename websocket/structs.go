package websocket

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

type Comment struct {
	Id           int
	Content      string
	Creator      string
	CreatorId    int
	PostId       int
	Likes        int
	Dislikes     int
	CreationDate string
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
	PostId   int
	UserId   int
	TopicId  int
	Creator  string
	Topic    string
	Label    string
	Content  string
	Created  string
	Likes    int
	Dislikes int
}

type Alert struct {
	AlertMessage string
	AlertCode    int
	AllUserPosts []Post
	AllReactions []Post
	LoggedIn     bool
}
