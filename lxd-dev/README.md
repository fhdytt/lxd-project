# 5. API Backend

`praktikum-api` adalah backend Go yang menjembatani `praktikum-tui` (container praktikan, *untrusted*) dengan PostgreSQL. **`lxd-control` (admin) TIDAK memakai API ini** — dia konek langsung ke database, lihat [Arsitektur § 1.2](01-arsitektur-sistem.md).

## 5.1 Struktur Project

```
praktikum-api/
├── go.mod
├── .env.example
├── README.md
├── cmd/api/main.go                     — entrypoint, HANYA wiring
└── internal/
    ├── config/config.go                — baca .env otomatis + environment variable
    ├── database/database.go            — connection pool PostgreSQL (pgx)
    ├── models/environment.go           — struct data (DTO)
    ├── repository/environment_repository.go  — semua query SQL
    ├── middleware/auth.go              — validasi Bearer token per environment
    └── handler/
        ├── environment_handler.go
        └── router.go
```

## 5.2 Keputusan Teknis

| Keputusan | Alasan |
|---|---|
| `net/http` bawaan Go 1.22, bukan framework | Routing modern Go 1.22 sudah cukup |
| `pgx` + `pgxpool` | Lebih cepat dari `database/sql` generik, connection pooling bawaan |
| SHA-256 untuk hash token, bukan bcrypt | Butuh lookup cepat & deterministik lewat query, beda kebutuhan dari password manusia |
| `log/slog` | Structured logging tanpa dependency pihak ketiga |
| Connection pool dibatasi (`MaxConns: 20`) | Cegah koneksi database tak terbatas saat banyak request bersamaan |
| Timeout eksplisit di HTTP server | Cegah koneksi macet menahan resource tanpa batas |
| Graceful shutdown | Selesaikan request berjalan dulu sebelum mati |

## 5.3 Konfigurasi Otomatis dari `.env`

**Perbaikan:** versi awal `config.Load()` cuma baca `os.Getenv`, jadi tiap sesi terminal baru harus `export $(cat .env | xargs)` manual — gampang lupa, bikin error `DATABASE_URL wajib di-set` berulang.

**Solusi:** `config.Load()` sekarang otomatis baca file `.env` di folder kerja **sebelum** membaca environment variable — env var yang sudah eksplisit di-set di shell/systemd tetap diprioritaskan, `.env` cuma fallback:

```go
func Load() (*Config, error) {
    loadDotEnv(".env")
    // ... baca os.Getenv seperti biasa ...
}
```

Sekarang cukup `go run ./cmd/api` langsung, tidak perlu `export` manual lagi (asalkan dijalankan dari folder yang ada `.env`-nya).

## 5.4 Otentikasi Token

```go
func HashToken(token string) string {
    sum := sha256.Sum256([]byte(token))
    return hex.EncodeToString(sum[:])
}
```

Middleware `Auth` hash token dari header `Authorization: Bearer <token>`, cari di `environments.api_token_hash`, sisipkan detail environment ke context request kalau ketemu.

## 5.5 Endpoint

### `GET /healthz` — tanpa otentikasi

### `GET /api/v1/environments/me`
```sql
SELECT e.id, e.container_name, s.course_code, m.code AS module,
       rm.nama AS room, s.meeting_number, s.session_date, e.status,
       (e.praktikan_id IS NOT NULL) AS already_identified
FROM environments e
JOIN sessions s ON s.id = e.session_id
JOIN rooms rm   ON rm.id = s.room_id
JOIN modules m  ON m.id = s.module_id
WHERE e.api_token_hash = $1
```

### `POST /api/v1/environments/me/identify`

Response sukses (baru diisi ATAU verifikasi nama+NPM cocok): `200`. Response kalau environment sudah pernah diisi dan **nama/NPM tidak cocok**: `403`.

## 5.6 Verifikasi Identitas

`IdentifyEnvironment()` mengunci baris environment (`FOR UPDATE OF e` — **bukan** `FOR UPDATE` polos, lihat § 5.7a) lalu bercabang:

```sql
SELECT p.npm, p.nama
FROM environments e
LEFT JOIN praktikan p ON p.id = e.praktikan_id
WHERE e.id = $1
FOR UPDATE OF e
```

- `praktikan_id` masih `NULL` → upsert praktikan + link seperti biasa.
- `praktikan_id` sudah terisi → bandingkan **nama DAN NPM**:
  ```go
  npmMatch := *existingNPM == npm
  namaMatch := strings.EqualFold(strings.TrimSpace(*existingNama), strings.TrimSpace(nama))
  if !npmMatch || !namaMatch {
      return ErrIdentityMismatch  // -> 403
  }
  ```

Perbandingan nama **case-insensitive** + trim spasi, supaya variasi kapitalisasi kecil tidak mengunci praktikan asli dari environment-nya sendiri.

### 5.7a Riwayat Bug: `FOR UPDATE` + `LEFT JOIN`

**Gejala:** `500 internal server error` saat submit identifikasi. Log: `FOR UPDATE cannot be applied to the nullable side of an outer join (SQLSTATE 0A000)`.

**Penyebab:** PostgreSQL tidak mengizinkan `FOR UPDATE` menyasar sisi *nullable* dari `LEFT JOIN` (di sini tabel `praktikan`, karena `praktikan_id` bisa `NULL`).

**Solusi:** `FOR UPDATE OF e` — kunci eksplisit hanya baris `environments`.

### 5.7b Riwayat Bug: Verifikasi Sempat Hanya Cek NPM

**Gejala:** login dengan NPM terdaftar tapi nama **berbeda** tetap lolos.

**Penyebab:** implementasi awal cuma membandingkan `npm`, `nama` diabaikan — celah keamanan (siapa saja yang tahu NPM orang lain bisa masuk, tinggal karang nama).

**Solusi:** bandingkan **nama DAN NPM sekaligus**, sudah diterapkan di kode saat ini (lihat § 5.6). Detail skenario lengkap ada di [Alur End-to-End § 6.3](06-alur-end-to-end.md).

## 5.8 Setup & Menjalankan

```bash
cp .env.example .env   # isi DATABASE_URL
go mod tidy
go run ./cmd/api        # otomatis baca .env, tidak perlu export manual
```

## 5.9 Testing & Troubleshooting dengan Postman

Testing lewat Postman mengisolasi layer API dari layer lain (TUI, SSH, LXD).

**Siapkan token test:**
```bash
lxc exec <nama-container> -- cat /proc/1/environ | tr '\0' '\n' | grep PRAKTIKUM_API_TOKEN
```

**Tabel diagnosis status code:**

| Status | Arti | Yang perlu dicek |
|---|---|---|
| `401` | Token tidak dikenali | Header `Authorization: Bearer <token>` (ada spasi setelah Bearer), atau token memang tidak ada di `environments.api_token_hash` |
| `403` | Nama/NPM tidak cocok | Bukan bug — lihat § 5.6. Cek data tercatat lewat `SELECT p.nama, p.npm FROM praktikan p JOIN environments e ON e.praktikan_id=p.id WHERE e.container_name='...'` |
| `500` | Error internal | **Selalu cek log terminal server** (`slog.Error` mencatat detail lengkap) — Postman sengaja tidak menampilkan detail internal demi keamanan |
| Timeout/connection refused | Server tidak reachable | `ss -tlnp \| grep 8080`, cek `ufw status` |

## 5.10 Yang Belum Diimplementasi

- Endpoint untuk web dashboard admin (sengaja ditunda — kontrol admin sekarang lewat `lxd-control`, bukan lewat API, lihat [Arsitektur § 1.5](01-arsitektur-sistem.md)).
- Endpoint provisioning lewat API (juga tidak dibutuhkan lagi — provisioning sekarang orkestrasi langsung di `lxd-control`).
- Otentikasi admin (tabel `admins` di skema sudah ada, tapi tidak dipakai selama kontrol admin lewat `lxd-control`, bukan web).