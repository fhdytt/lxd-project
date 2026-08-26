# API Backend

`lxd-dev` adalah backend yang berjalan dengan bahasa Go yang digunakan untuk menjembatani TUI, web dashboard, dan PostgreSQL.

## Struktur Project

```
lxd-dev/
├── go.mod
├── go.sum
├── .env.example
├── README.md
├── cmd/
│   └── api/
│       └── main.go                     # entrypoint
└── internal/
    ├── config/
    │   └── config.go                   # konfigurasi dari environment variable
    ├── database/
    │   └── database.go                 # connection pool PostgreSQL
    ├── models/
    │   └── environment.go              # struct data
    ├── repository/
    │   └── environment_repository.go   # semua query SQL
    ├── middleware/
    │   └── auth.go                     # validasi Bearer token per environment
    └── handler/
        ├── environment_handler.go      # logic HTTP handler
        └── router.go                   # routing + logging middleware
```

## Keputusan Teknis & Alasannya

| Keputusan | Alasan |
|---|---|
| **`net/http` bawaan Go 1.22**, bukan Gin/Echo/Chi | Routing modern Go 1.22 (`mux.Handle("GET /path", ...)`) sudah cukup untuk kebutuhan endpoint sesederhana ini, tanpa overhead dependency framework |
| **`pgx` + `pgxpool`**, bukan `database/sql` + driver generik | Driver native yang lebih cepat, dengan connection pooling bawaan — penting karena backend ini dipanggil oleh ratusan TUI container bersamaan |
| **SHA-256 untuk hash token**, bukan bcrypt | Token API butuh lookup cepat & deterministik langsung lewat query database (`WHERE api_token_hash = $1`), beda kebutuhan dari password manusia yang butuh bcrypt (lambat, didesain anti brute-force offline) |
| **`log/slog`** (structured logging bawaan Go) | Tanpa dependency logging pihak ketiga |
| **Connection pool dibatasi** (`MaxConns: 20`) | Mencegah backend membuka koneksi database tanpa batas saat banyak request bersamaan |
| **Timeout eksplisit** di HTTP server (`ReadTimeout`, `WriteTimeout`, `IdleTimeout`) | Mencegah koneksi macet dari satu container menahan resource server tanpa batas waktu |
| **Graceful shutdown** (`signal.NotifyContext` + `server.Shutdown`) | Server menyelesaikan request yang sedang berjalan dulu (maksimal 10 detik) sebelum mati — penting untuk deployment yang butuh restart/update tanpa memutus request yang sedang diproses |


## 5.5 Setup & Menjalankan

```bash
cp .env.example .env   # isi DATABASE_URL
go mod tidy
go run ./cmd/api
```

Sebagai systemd service (production):
```ini
# /etc/systemd/system/praktikum-api.service
[Unit]
Description=Praktikum API Backend
After=network.target postgresql.service

[Service]
Type=simple
User=praktikum
WorkingDirectory=/opt/praktikum-api
EnvironmentFile=/opt/praktikum-api/.env
ExecStart=/opt/praktikum-api/praktikum-api
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```