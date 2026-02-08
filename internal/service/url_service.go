package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jay-ponkia/go-url-shortener/internal/cache"
	"github.com/jay-ponkia/go-url-shortener/internal/queue"
	"github.com/jay-ponkia/go-url-shortener/internal/repository"
	"github.com/jay-ponkia/go-url-shortener/internal/utils"
)

type URLService struct {
	repo       *repository.URLRepo
	cache      *cache.RedisCache
	cacheTTL   time.Duration
	clickQueue *queue.ClickQueue
}

func NewURLService(repo *repository.URLRepo, cache *cache.RedisCache, cacheTTL time.Duration) *URLService {
	return &URLService{
		repo:       repo,
		cache:      cache,
		cacheTTL:   cacheTTL,
		clickQueue: nil,
	}
}

// SetClickQueue sets the click queue for async click processing
func (s *URLService) SetClickQueue(q *queue.ClickQueue) {
	s.clickQueue = q
}

func (s *URLService) CreateShortURL(originalURL string, alias string) (string, error) {
	shortCode := ""
	if alias == "" {
		shortCode = utils.GenerateShortCode()
	} else {
		shortCode = alias
	}

	err := s.repo.Save(shortCode, originalURL, alias)
	if err != nil {
		return "", err
	}

	// Cache the new URL
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cacheKey := shortCode
	if alias != "" {
		cacheKey = alias
	}
	if err := s.cache.Set(ctx, cacheKey, originalURL, s.cacheTTL); err != nil {
		log.Printf("Warning: Failed to cache short code %s: %v", cacheKey, err)
		// Don't return error, cache is optional
	}

	// Invalidate the all_urls cache since we added a new URL
	if err := s.cache.Delete(ctx, "all_urls"); err != nil {
		log.Printf("Warning: Failed to invalidate all_urls cache: %v", err)
		// Don't return error, cache invalidation is optional
	}

	return shortCode, nil
}

func (s *URLService) GetURL(shortCode string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to get from cache first
	cachedURL, err := s.cache.Get(ctx, shortCode)
	if err == nil {
		fmt.Println("Cache hit for short code: %s", shortCode)
		// Enqueue click asynchronously (non-blocking)
		go s.enqueueClick(shortCode)
		return cachedURL, nil
	}

	// Cache miss or error, fetch from database
	originalURL, err := s.repo.Get(shortCode)
	if err != nil {
		return "", err
	}

	// Try to cache the result
	if err := s.cache.Set(ctx, shortCode, originalURL, s.cacheTTL); err != nil {
		log.Printf("Warning: Failed to cache short code %s: %v", shortCode, err)
		// Don't return error, cache is optional
	}

	// Enqueue click asynchronously (non-blocking)
	go s.enqueueClick(shortCode)

	return originalURL, nil
}

// enqueueClick safely enqueues a click event
func (s *URLService) enqueueClick(shortCode string) {
	if s.clickQueue == nil {
		return
	}

	if err := s.clickQueue.EnqueueClick(shortCode); err != nil {
		log.Printf("Error enqueuing click for %s: %v", shortCode, err)
	}
}

func (s *URLService) ResetAllURLsCache() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.cache.Delete(ctx, "all_urls")
}

// GetAllURLs returns all URL records using cache-first strategy.
// It returns the records, a source string ("cache" or "db"), and error.
func (s *URLService) GetAllURLs() ([]repository.URLRecord, string, error) {
	// _ = s.ResetAllURLsCache()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cacheKey := "all_urls"

	// Try cache first
	cached, err := s.cache.Get(ctx, cacheKey)
	if err == nil {
		fmt.Println("Cache hit for all_urls")
		var records []repository.URLRecord
		if err := json.Unmarshal([]byte(cached), &records); err == nil {
			return records, "cache", nil
		}
		// if unmarshalling fails, fall through to DB
		log.Printf("Warning: failed to unmarshal cached all_urls: %v", err)
	}

	// Fetch from DB
	records, err := s.repo.GetAll()
	if err != nil {
		return nil, "db", err
	}

	// Cache the result (best-effort)
	if b, err := json.Marshal(records); err == nil {
		if err := s.cache.Set(ctx, cacheKey, string(b), s.cacheTTL); err != nil {
			log.Printf("Warning: failed to set cache for all_urls: %v", err)
		}
	}

	return records, "db", nil
}
