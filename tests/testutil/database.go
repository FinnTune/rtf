package testutil

import (
	"database/sql"
	"rtForum/database"
	"rtForum/utility"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

const forumSchema = `
PRAGMA foreign_keys = ON;

CREATE TABLE user (
	id INTEGER NOT NULL PRIMARY KEY,
	fname VARCHAR(30) NOT NULL,
	lname VARCHAR(30) NOT NULL,
	uname VARCHAR(30) NOT NULL UNIQUE,
	email VARCHAR(30) NOT NULL UNIQUE,
	age VARCHAR(3) NOT NULL,
	gender VARCHAR(10) NOT NULL,
	pass TEXT NOT NULL,
	created_at VARCHAR(30) NOT NULL,
	role VARCHAR(10) NOT NULL DEFAULT 'user'
);

CREATE TABLE category (
	id INTEGER NOT NULL PRIMARY KEY,
	category_name VARCHAR(30) NOT NULL
);

CREATE TABLE post (
	id INTEGER NOT NULL PRIMARY KEY,
	user_id INTEGER NOT NULL,
	title VARCHAR(30) NOT NULL,
	content VARCHAR(150) NOT NULL,
	author VARCHAR(30) NOT NULL,
	created_at DATETIME NOT NULL,
	img_url VARCHAR(200),
	FOREIGN KEY(user_id) REFERENCES user(id)
);

CREATE TABLE comment (
	id INTEGER NOT NULL PRIMARY KEY,
	user_id INTEGER NOT NULL,
	post_id INTEGER NOT NULL,
	content VARCHAR(150) NOT NULL,
	created_at DATETIME NOT NULL,
	FOREIGN KEY(user_id) REFERENCES user(id),
	FOREIGN KEY(post_id) REFERENCES post(id)
);

CREATE TABLE category_relation (
	id INTEGER NOT NULL PRIMARY KEY,
	category_id INTEGER NOT NULL,
	post_id INTEGER NOT NULL,
	FOREIGN KEY(category_id) REFERENCES category(id),
	FOREIGN KEY(post_id) REFERENCES post(id)
);

CREATE TABLE user_post_reaction (
	id INTEGER NOT NULL PRIMARY KEY,
	user_id INTEGER NOT NULL,
	post_id INTEGER NOT NULL,
	is_liked TINYINT(1) NOT NULL,
	created_at DATETIME NOT NULL,
	UNIQUE(user_id, post_id),
	FOREIGN KEY(user_id) REFERENCES user(id),
	FOREIGN KEY(post_id) REFERENCES post(id)
);

CREATE TABLE conversation (
	id INTEGER NOT NULL PRIMARY KEY,
	is_group TINYINT(1) NOT NULL DEFAULT 0,
	name VARCHAR(50),
	direct_pair_key VARCHAR(21),
	created_at DATETIME NOT NULL,
	UNIQUE(direct_pair_key)
);

CREATE TABLE conversation_member (
	id INTEGER NOT NULL PRIMARY KEY,
	conversation_id INTEGER NOT NULL,
	user_id INTEGER NOT NULL,
	joined_at DATETIME NOT NULL,
	UNIQUE(conversation_id, user_id),
	FOREIGN KEY(conversation_id) REFERENCES conversation(id),
	FOREIGN KEY(user_id) REFERENCES user(id)
);

CREATE TABLE message (
	id INTEGER NOT NULL PRIMARY KEY,
	conversation_id INTEGER NOT NULL,
	sender_id INTEGER NOT NULL,
	txt TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	FOREIGN KEY(conversation_id) REFERENCES conversation(id),
	FOREIGN KEY(sender_id) REFERENCES user(id)
);
`

// SetupForumDB creates a seeded in-memory SQLite database for tests.
func SetupForumDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	if _, err = db.Exec(forumSchema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	hashedPass := utility.HashPassword("secret123")
	_, err = db.Exec(`
		INSERT INTO user (id, fname, lname, uname, email, age, gender, pass, created_at, role) VALUES
		(1, 'Admin', 'User', 'admin', 'admin@example.com', '30', 'other', ?, datetime('now'), 'admin'),
		(2, 'Alice', 'Smith', 'alice', 'alice@example.com', '25', 'female', ?, datetime('now'), 'user'),
		(42, 'Actual', 'User', 'actual_user', 'actual@example.com', '28', 'other', ?, datetime('now'), 'user');
	`, hashedPass, hashedPass, hashedPass)
	if err != nil {
		t.Fatalf("failed to seed users: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO category (id, category_name) VALUES
		(1, 'Cuisine'), (2, 'Places'), (5, 'Code');
	`)
	if err != nil {
		t.Fatalf("failed to seed categories: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO post (id, user_id, title, content, author, created_at) VALUES
		(1, 42, 'seed', 'seed content', 'actual_user', datetime('now')),
		(2, 1, 'Asian Food', 'Thai Khun Mom', 'admin', datetime('now')),
		(3, 1, 'Best Sushi', 'Fina Fisken', 'admin', datetime('now'));
	`)
	if err != nil {
		t.Fatalf("failed to seed posts: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO category_relation (id, category_id, post_id) VALUES
		(1, 1, 2), (2, 1, 3), (3, 5, 1);
	`)
	if err != nil {
		t.Fatalf("failed to seed category relations: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO comment (id, user_id, post_id, content, created_at) VALUES
		(1, 42, 1, 'existing comment', datetime('now'));
	`)
	if err != nil {
		t.Fatalf("failed to seed comments: %v", err)
	}

	// A single seeded direct conversation between admin (1) and alice (2),
	// with one message each way.
	_, err = db.Exec(`
		INSERT INTO conversation (id, is_group, direct_pair_key, created_at) VALUES
		(1, 0, '1-2', datetime('now'));

		INSERT INTO conversation_member (id, conversation_id, user_id, joined_at) VALUES
		(1, 1, 1, datetime('now')),
		(2, 1, 2, datetime('now'));

		INSERT INTO message (id, conversation_id, sender_id, txt, created_at) VALUES
		(1, 1, 1, 'hello alice', datetime('now')),
		(2, 1, 2, 'hi admin', datetime('now'));
	`)
	if err != nil {
		t.Fatalf("failed to seed messages: %v", err)
	}

	return db
}

// UseForumDB installs an in-memory database as the global forum DB for a test.
func UseForumDB(t *testing.T) *sql.DB {
	t.Helper()
	db := SetupForumDB(t)
	database.ForumDB = db
	t.Cleanup(func() {
		db.Close()
		database.ForumDB = nil
	})
	return db
}
