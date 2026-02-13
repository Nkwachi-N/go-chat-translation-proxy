package main

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port            string
	OllamaURL      string
	OllamaModel    string
	RateLimit       int
	RateLimitWindow time.Duration
	CacheTTL        time.Duration
}

func LoadConfig() Config {
	rateLimit, _ := strconv.Atoi(envOrDefault("RATE_LIMIT", "10"))
	rateLimitWindow, _ := time.ParseDuration(envOrDefault("RATE_LIMIT_WINDOW", "1m"))
	cacheTTL, _ := time.ParseDuration(envOrDefault("CACHE_TTL", "10m"))

	return Config{
		Port:            ":" + envOrDefault("PORT", "8080"),
		OllamaURL:       envOrDefault("OLLAMA_URL", "http://localhost:11434"),
		OllamaModel:     envOrDefault("OLLAMA_MODEL", "llama3.2"),
		RateLimit:        rateLimit,
		RateLimitWindow:  rateLimitWindow,
		CacheTTL:         cacheTTL,
	}
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
