package repository

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/jay-ponkia/go-url-shortener/internal/db"
)

type URLRepo struct{}

func NewURLRepo() *URLRepo {
	return &URLRepo{}
}

func (r *URLRepo) Save(shortCode, originalURL string) error {
	query := `INSERT INTO urls (short_code, original_url, created_at) 
              VALUES ($1, $2, CURRENT_TIMESTAMP)`

	_, err := db.GetDB().Exec(query, shortCode, originalURL)
	if err != nil {
		log.Printf("Error saving URL: %v", err)
		return err
	}

	log.Printf("Successfully saved short code: %s", shortCode)
	return nil
}

func (r *URLRepo) Get(shortCode string) (string, error) {
	query := `SELECT original_url FROM urls WHERE short_code = $1`

	var originalURL string
	err := db.GetDB().QueryRow(query, shortCode).Scan(&originalURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("short code not found: %s", shortCode)
		}
		log.Printf("Error retrieving URL: %v", err)
		return "", err
	}

	return originalURL, nil
}
