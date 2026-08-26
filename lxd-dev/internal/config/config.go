package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:        getEnvOrDefault("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("environment variable DATABASE_URL belum di set")
	}

	return cfg, nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}