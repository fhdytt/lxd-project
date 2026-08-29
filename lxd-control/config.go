package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Config menampung konfigurasi lxd-control. Dibaca dari file .env di folder
// yang sama, atau dari environment variable biasa kalau sudah di-export.
//
// CATATAN: PGPASSWORD TIDAK DIBUTUHKAN LAGI di sini — kelola-lxd.sh sekarang
// murni eksekutor LXD, sudah tidak menyentuh PostgreSQL sama sekali. Semua
// akses database dilakukan lxd-control sendiri lewat DATABASE_URL.
type Config struct {
	DatabaseURL string
	ScriptPath  string // path ke kelola-lxd.sh
	APIURL      string // alamat praktikum-api, dari sudut pandang container (lewat lxdbr0)
}

func LoadConfig() (*Config, error) {
	loadDotEnv(".env")

	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		ScriptPath:  getEnvOrDefault("KELOLA_SCRIPT_PATH", "./kelola-lxd.sh"),
		APIURL:      getEnvOrDefault("PRAKTIKUM_API_URL", "http://10.184.56.1:8080"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL wajib di-set (cek file .env)")
	}
	if _, err := os.Stat(cfg.ScriptPath); err != nil {
		return nil, fmt.Errorf("kelola-lxd.sh tidak ditemukan di %q: %w", cfg.ScriptPath, err)
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