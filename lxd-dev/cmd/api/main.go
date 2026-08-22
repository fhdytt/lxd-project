package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lxd-dev/internal/config"
	"lxd-dev/internal/database"
	"lxd-dev/internal/handler"
	"lxd-dev/internal/repository"
)

// main sengaja dibuat setipis mungkin — cuma "merakit" komponen (wiring),
// tidak ada logic bisnis di sini. Logic sesungguhnya ada di masing-masing
// package internal/. Pola ini memudahkan testing tiap komponen secara
// terpisah tanpa harus menjalankan seluruh aplikasi.
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("aplikasi berhenti karena error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbPool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer dbPool.Close()

	slog.Info("koneksi database berhasil")

	envRepo := repository.NewEnvironmentRepository(dbPool)
	router := handler.NewRouter(envRepo)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,

		// Timeout eksplisit di semua sisi — mencegah satu koneksi lambat/macet
		// (misal TUI di container yang jaringannya bermasalah) menahan resource
		// server tanpa batas waktu. Penting untuk efisiensi resource saat
		// banyak container memanggil API bersamaan.
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("server berjalan", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err

	case <-ctx.Done():
		slog.Info("menerima sinyal shutdown, mematikan server dengan graceful...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		slog.Info("server berhasil dimatikan dengan bersih")
	}

	return nil
}