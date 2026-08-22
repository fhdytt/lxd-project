package config

import (
	"fmt"
	"os"
)

// Config menampung seluruh konfigurasi aplikasi yang diambil dari environment
// variable. Sengaja tidak pakai library eksternal (viper dkk) karena kebutuhan
// konfigurasi kita sederhana — cukup os.Getenv, lebih ringan dan lebih cepat start.
type Config struct {
	// Port HTTP server akan listen, contoh: "8080"
	Port string

	// DatabaseURL adalah connection string PostgreSQL, format:
	// postgres://user:password@host:port/dbname
	DatabaseURL string
}

// Load membaca konfigurasi dari environment variable dan memvalidasi bahwa
// semua yang wajib ada sudah di-set. Aplikasi sengaja gagal start (fail-fast)
// kalau ada yang kurang, daripada jalan dengan konfigurasi tidak lengkap.
func Load() (*Config, error) {
	cfg := &Config{
		Port:        getEnvOrDefault("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("environment variable DATABASE_URL wajib di-set")
	}

	return cfg, nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}