package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	connstr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	var err error
	DB, err = sql.Open("postgres", connstr)
	if err != nil {
		log.Fatal(err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal(err)
	}

	log.Println(" DB connected")

	createTable()
}
func createTable() error {
	query := `
    CREATE TABLE IF NOT EXISTS url (
        id           TEXT PRIMARY KEY,
        original_url TEXT NOT NULL,
        new_url      TEXT UNIQUE NOT NULL,
        created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    CREATE INDEX IF NOT EXISTS idx_url_id ON url(id);
    `
	_, err := DB.Exec(query)
	if err != nil {
		return fmt.Errorf("createTable: %w", err)
	}
	return nil
}
func insertLink(u Url) error {

	query := `
	INSERT INTO url(id, original_url, new_url, created_at)
	VALUES ($1,$2,$3,$4)
	ON CONFLICT (id) DO NOTHING
	`

	_, err := DB.Exec(
		query,
		u.ID,
		u.OURL,
		u.NURL,
		u.DATE,
	)

	return err
}
func getLink(id string) (Url, error) {

	var u Url

	query := `
	SELECT id, original_url, new_url, created_at
	FROM url
	WHERE id=$1
	`

	err := DB.QueryRow(query, id).
		Scan(&u.ID, &u.OURL, &u.NURL, &u.DATE)

	return u, err
}
