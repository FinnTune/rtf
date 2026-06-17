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
	uname VARCHAR(30) NOT NULL,
	email VARCHAR(30) NOT NULL,
	age VARCHAR(3) NOT NULL,
	gender VARCHAR(10) NOT NULL,
	pass TEXT NOT NULL,
	created_at VARCHAR(30) NOT NULL
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

CREATE TABLE message (
	id INTEGER NOT NULL PRIMARY KEY,
	from_user INTEGER NOT NULL,
	to_user INTEGER NOT NULL,
	is_read TINYINT(1) NOT NULL,
	txt TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	FOREIGN KEY(from_user) REFERENCES user(id),
	FOREIGN KEY(to_user) REFERENCES user(id)
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
		INSERT INTO user (id, fname, lname, uname, email, age, gender, pass, created_at) VALUES
		(1, 'Admin', 'User', 'admin', 'admin@example.com', '30', 'other', ?, datetime('now')),
		(2, 'Alice', 'Smith', 'alice', 'alice@example.com', '25', 'female', ?, datetime('now')),
		(42, 'Actual', 'User', 'actual_user', 'actual@example.com', '28', 'other', ?, datetime('now'));
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

	_, err = db.Exec(`
		INSERT INTO message (id, from_user, to_user, is_read, txt, created_at) VALUES
		(1, 1, 2, 0, 'hello alice', datetime('now')),
		(2, 2, 1, 0, 'hi admin', datetime('now'));
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
