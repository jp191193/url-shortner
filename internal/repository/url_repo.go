package repository

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jay-ponkia/go-url-shortener/internal/db"
)

type URLRepo struct{}

func NewURLRepo() *URLRepo {
	return &URLRepo{}
}

type URLRecord struct {
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
}

func (r *URLRepo) Save(shortCode, originalURL, alias string) error {
	query := `INSERT INTO urls (short_code, original_url, alias, created_at) 
			  VALUES ($1, $2, $3, CURRENT_TIMESTAMP)`

	_, err := db.GetDB().Exec(query, shortCode, originalURL, alias)
	if err != nil {
		log.Printf("Error saving URL: %v", err)
		return err
	}

	log.Printf("Successfully saved short code: %s", shortCode)
	log.Printf("Successfully saved alias: %s", alias)
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

// GetAll returns all URL records from the database
func (r *URLRepo) GetAll() ([]URLRecord, error) {
	query := `SELECT short_code, original_url, created_at FROM urls ORDER BY created_at DESC`

	rows, err := db.GetDB().Query(query)
	if err != nil {
		log.Printf("Error querying all URLs: %v", err)
		return nil, err
	}
	defer rows.Close()

	var results []URLRecord
	for rows.Next() {
		var rec URLRecord
		if err := rows.Scan(&rec.ShortCode, &rec.OriginalURL, &rec.CreatedAt); err != nil {
			log.Printf("Error scanning URL row: %v", err)
			return nil, err
		}
		results = append(results, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
