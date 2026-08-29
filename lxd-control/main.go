package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackc/pgx/v5/pgxpool"
)

// main sengaja setipis mungkin — cuma "merakit" komponen (baca config, buka
// koneksi database, jalankan program TUI). Semua logic sesungguhnya ada di
// file lain: model.go (state & tipe data), commands.go (operasi async),
// update.go (transisi state), view.go (rendering tampilan).
func main() {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error konfigurasi:", err)
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Gagal konek database:", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "Gagal ping database:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel(cfg, db), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "TUI error:", err)
		os.Exit(1)
	}
}