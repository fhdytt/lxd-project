# API Backend

`lxd-dev` adalah backend yang berjalan dengan bahasa Go yang digunakan untuk menjembatani TUI, web dashboard, dan PostgreSQL.

## Struktur Project

```
lxd-dev/
├── go.mod
├── .env.example
├── README.md
├── cmd/
│   └── api/
│       └── main.go                     # entrypoint
└── internal/
    ├── config/
    │   └── config.go                   # baca konfigurasi dari environment variable
    ├── database/
    │   └── database.go                 # connection pool PostgreSQL (pgx)
    ├── models/
    │   └── environment.go              # struct data (DTO)
    ├── repository/
    │   └── environment_repository.go   # semua query SQL
    ├── middleware/
    │   └── auth.go                     # validasi Bearer token per environment
    └── handler/
        ├── environment_handler.go      # logic HTTP handler
        └── router.go                   # routing + logging middleware
```

**Alasan struktur ini:**

- **`cmd/` vs `internal/`** — konvensi standar komunitas Go. `cmd/` isinya entrypoint tipis untuk tiap binary yang mau dibangun (kalau nanti ada binary lain, misal CLI admin, tinggal tambah `cmd/cli/main.go`). `internal/` isinya kode privat aplikasi yang tidak boleh diimpor project Go lain di luar module ini.
- **Repository terpisah dari handler** — query SQL tidak bercampur dengan logic HTTP. Kalau nanti web dashboard butuh fungsi yang sama, tinggal panggil ulang method repository yang sama, tidak perlu duplikasi query.
- **Middleware terpisah** — validasi token dipakai di banyak endpoint, ditulis sekali, tinggal "dibungkus" ke handler mana saja yang butuh.

## 5.2 Keputusan Teknis & Alasannya

| Keputusan | Alasan |
|---|---|
| **`net/http` bawaan Go 1.22**, bukan Gin/Echo/Chi | Routing modern Go 1.22 (`mux.Handle("GET /path", ...)`) sudah cukup untuk kebutuhan endpoint sesederhana ini, tanpa overhead dependency framework |
| **`pgx` + `pgxpool`**, bukan `database/sql` + driver generik | Driver native yang lebih cepat, dengan connection pooling bawaan — penting karena backend ini dipanggil oleh ratusan TUI container bersamaan |
| **SHA-256 untuk hash token**, bukan bcrypt | Token API butuh lookup cepat & deterministik langsung lewat query database (`WHERE api_token_hash = $1`), beda kebutuhan dari password manusia yang butuh bcrypt (lambat, didesain anti brute-force offline) |
| **`log/slog`** (structured logging bawaan Go) | Tanpa dependency logging pihak ketiga |
| **Connection pool dibatasi** (`MaxConns: 20`) | Mencegah backend membuka koneksi database tanpa batas saat banyak request bersamaan |
| **Timeout eksplisit** di HTTP server (`ReadTimeout`, `WriteTimeout`, `IdleTimeout`) | Mencegah koneksi macet dari satu container menahan resource server tanpa batas waktu |
| **Graceful shutdown** (`signal.NotifyContext` + `server.Shutdown`) | Server menyelesaikan request yang sedang berjalan dulu (maksimal 10 detik) sebelum mati — penting untuk deployment yang butuh restart/update tanpa memutus request yang sedang diproses |

## 5.3 Otentikasi Token

Setiap environment (container) punya token unik yang di-generate saat provisioning. Alur otentikasi:

1. TUI mengirim `Authorization: Bearer <token>` di setiap request.
2. Middleware `Auth` meng-hash token itu dengan SHA-256, lalu query `environments` berdasarkan `api_token_hash`.
3. Kalau ketemu, detail environment disisipkan ke context request, dilanjutkan ke handler.
4. Kalau tidak ketemu, response `401 Unauthorized`.

```go
func HashToken(token string) string {
    sum := sha256.Sum256([]byte(token))
    return hex.EncodeToString(sum[:])
}
```

**Kenapa token perlu sama sekali?** Container praktikan bersifat *untrusted*. Tanpa otentikasi, satu container bisa memanipulasi data environment lain (sengaja atau karena bug). Token memastikan API tahu persis request datang dari environment mana.

## 5.4 Endpoint

### `GET /healthz`
Health check, tanpa otentikasi. Response: `ok` (200).

### `GET /api/v1/environments/me`
Perlu header `Authorization: Bearer <token>`. Mengembalikan detail environment berdasarkan token.

```json
{
  "container_name": "f491-01",
  "course_code": "1WADR261014L",
  "module": "netbegin",
  "room": "f491",
  "meeting_number": 3,
  "session_date": "2026-08-18",
  "status": "running",
  "already_identified": false
}
```

Query yang dijalankan (JOIN `environments` + `sessions` + `rooms` + `modules`):
```sql
SELECT
    e.id, e.container_name, s.course_code, m.code AS module,
    rm.nama AS room, s.meeting_number, s.session_date, e.status,
    (e.praktikan_id IS NOT NULL) AS already_identified
FROM environments e
JOIN sessions s ON s.id = e.session_id
JOIN rooms rm   ON rm.id = s.room_id
JOIN modules m  ON m.id = s.module_id
WHERE e.api_token_hash = $1
```

### `POST /api/v1/environments/me/identify`
Perlu header `Authorization: Bearer <token>`. Body:
```json
{ "nama": "Budi Santoso", "npm": "2106123456" }
```

Response sukses (baru diisi ATAU verifikasi NPM cocok): `{"success": true}` (200).
Response kalau environment sudah pernah diisi dan NPM **tidak cocok**: `403 Forbidden` — lihat § 5.4a.

## 5.4a Verifikasi Identitas

**Latar belakang:** desain awal, kalau environment sudah pernah diisi (`praktikan_id` sudah ter-set), request `identify` berikutnya otomatis dianggap "konflik tidak fatal" (`409`) dan TUI tetap melanjutkan ke shell tanpa mengecek siapa yang login. Ini celah keamanan — siapapun yang tahu password root bisa langsung masuk ke environment orang lain tanpa verifikasi apapun.

**Perilaku sekarang** — `IdentifyEnvironment()` mengunci baris environment (`SELECT ... FOR UPDATE`) lalu bercabang:

```sql
SELECT p.npm
FROM environments e
LEFT JOIN praktikan p ON p.id = e.praktikan_id
WHERE e.id = $1
FOR UPDATE
```

- **`praktikan_id` masih NULL** (belum pernah diisi) → jalankan upsert + link seperti biasa (lihat query di bawah).
- **`praktikan_id` sudah terisi** → bandingkan `npm` yang di-submit dengan NPM yang sudah tercatat:
  - **Cocok** → dianggap berhasil (verifikasi), tidak ada data yang diubah.
  - **Tidak cocok** → kembalikan `ErrIdentityMismatch`, di-mapping handler ke `403 Forbidden`.

```sql
-- Hanya dijalankan kalau environment BELUM pernah diisi
INSERT INTO praktikan (npm, nama) VALUES ($1, $2)
ON CONFLICT (npm) DO UPDATE SET nama = EXCLUDED.nama, updated_at = now()
RETURNING id;

UPDATE environments SET praktikan_id = $1, identified_at = now()
WHERE id = $2 AND praktikan_id IS NULL;
```

Seluruh logic ini dijalankan dalam **satu transaksi** dengan row lock (`FOR UPDATE`), supaya aman dari race condition kalau ada 2 request bersamaan ke environment yang sama.

> **Belum terselesaikan:** apakah reset environment (lihat [Infrastruktur LXD § 2.7](02-infrastruktur-lxd.md#27-reset--recovery)) seharusnya juga meng-*unlink* `praktikan_id` di database, supaya environment yang direset bisa diklaim praktikan baru? Saat ini reset **tidak** menyentuh database, jadi `praktikan_id` yang lama tetap "nempel" walau isi container sudah bersih — konsisten dengan tujuan anti-pinjam-PC, tapi perlu didiskusikan lagi untuk kasus environment yang sengaja mau dipindah kepemilikan (misal praktikan pindah kelas). Lihat [Log Perkembangan](09-progress-log.md).

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

## 5.6 Yang Belum Diimplementasi

- Endpoint untuk web dashboard admin/asisten (list environment, kontrol start/stop/reset).
- Endpoint provisioning (integrasi langsung ke LXD API, menggantikan `kelola-lxd.sh`).
- Otentikasi admin (login web dashboard — tabel `admins` di database sudah ada, logic-nya belum ditulis).