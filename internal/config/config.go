package config

import (
	"fmt"
	"os"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	BaseURL    string
	Port       string
	RedisAddr  string
	CacheTTL   string
}

func LoadConfig() *Config {
	return &Config{
		DBHost:     getEnv("DB_HOST", ""),
		DBPort:     getEnv("DB_PORT", ""),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "url_shortner"),
		BaseURL:    getEnv("BASE_URL", "http://localhost:8080"),
		Port:       getEnv("PORT", ":8080"),
		RedisAddr:  getEnv("REDIS_ADDR", "localhost:6379"),
		CacheTTL:   getEnv("CACHE_TTL", "3600"), // default to 1 hour
	}
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
