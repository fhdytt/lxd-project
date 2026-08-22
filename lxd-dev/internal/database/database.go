package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool membuat connection pool ke PostgreSQL menggunakan pgx.
//
// Connection pooling penting di sini karena backend ini akan dipanggil oleh
// banyak container TUI secara bersamaan (bisa ratusan saat jam praktikum
// ramai) — tanpa pooling, tiap request akan buka-tutup koneksi baru yang
// mahal dari sisi resource dan latency.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("gagal parse DATABASE_URL: %w", err)
	}

	// Batas pool disesuaikan untuk skala menengah (4 ruangan x ~50 praktikan).
	// Angka ini bisa dituning lagi setelah observasi beban produksi nyata.
	poolConfig.MaxConns = 20
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat connection pool: %w", err)
	}

	// Pastikan koneksi beneran hidup sebelum aplikasi dianggap siap menerima trafik.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("gagal ping database: %w", err)
	}

	return pool, nil
}