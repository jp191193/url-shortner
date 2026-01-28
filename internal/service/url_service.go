package service

import (
	"github.com/jay-ponkia/go-url-shortener/internal/repository"
	"github.com/jay-ponkia/go-url-shortener/internal/utils"
)

type URLService struct {
	repo *repository.URLRepo
}

func NewURLService(repo *repository.URLRepo) *URLService {
	return &URLService{repo: repo}
}

func (s *URLService) CreateShortURL(originalURL string) (string, error) {
	shortCode := utils.GenerateShortCode()
	err := s.repo.Save(shortCode, originalURL)
	if err != nil {
		return "", err
	}
	return shortCode, nil
}

func (s *URLService) GetURL(shortCode string) (string, error) {
	return s.repo.Get(shortCode)
}
