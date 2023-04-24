package database

import (
	"database/sql"
	"log"

	//sqlite3
	_ "github.com/mattn/go-sqlite3"
)

var ForumDB *sql.DB

func OpenDB() *sql.DB {
	dataBase, err := sql.Open("sqlite3", "./database/forum.db")
	if err != nil {
		log.Fatalf("Error opening database: %s", err)
	}
	log.Println("Database opened successfully.")
	return dataBase
}
