package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL string
	ScriptPath  string
	APIURL      string
}

func LoadConfig() (*Config, error) {
	loadDotEnv(".env")

	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		ScriptPath:  getEnvOrDefault("KELOLA_SCRIPT_PATH", "./kelola-lxd.sh"),
		APIURL:      getEnvOrDefault("PRAKTIKUM_API_URL", "http://10.184.56.1:8080"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL wajib di-set")
	}
	if _, err := os.Stat(cfg.ScriptPath); err != nil {
		return nil, fmt.Errorf("file script tidak ditemukan di %q: %w", cfg.ScriptPath, err)
	}

	return cfg, nil
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if _, alreadySet := os.LookupEnv(key); !alreadySet {
			os.Setenv(key, value)
		}
	}
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
