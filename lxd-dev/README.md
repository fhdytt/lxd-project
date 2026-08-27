# 5. API Backend

`praktikum-api` adalah backend Go yang menjembatani TUI, web dashboard (rencana), dan PostgreSQL.

## 5.1 Struktur Project (Standard Go Project Layout)

```
praktikum-api/
├── go.mod
├── .env.example
├── README.md
├── cmd/
│   └── api/
│       └── main.go                     # entrypoint, HANYA wiring — tidak ada logic bisnis
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

Response sukses (baru diisi ATAU verifikasi nama+NPM cocok): `{"success": true}` (200).
Response kalau environment sudah pernah diisi dan nama/NPM **tidak cocok**: `403 Forbidden` — lihat § 5.4a.

## 5.4a Verifikasi Identitas

**Latar belakang:** desain awal, kalau environment sudah pernah diisi (`praktikan_id` sudah ter-set), request `identify` berikutnya otomatis dianggap "konflik tidak fatal" (`409`) dan TUI tetap melanjutkan ke shell tanpa mengecek siapa yang login. Ini celah keamanan — siapapun yang tahu password root bisa langsung masuk ke environment orang lain tanpa verifikasi apapun.

**Perilaku sekarang** — `IdentifyEnvironment()` mengunci baris environment (`SELECT ... FOR UPDATE OF e`) lalu bercabang:

```sql
SELECT p.npm, p.nama
FROM environments e
LEFT JOIN praktikan p ON p.id = e.praktikan_id
WHERE e.id = $1
FOR UPDATE OF e
```

- **`praktikan_id` masih NULL** (belum pernah diisi) → jalankan upsert + link seperti biasa (lihat query di bawah).
- **`praktikan_id` sudah terisi** → bandingkan **nama DAN NPM** yang di-submit dengan yang sudah tercatat:
  - **Keduanya cocok** → dianggap berhasil (verifikasi), tidak ada data yang diubah.
  - **Salah satu saja tidak cocok** → kembalikan `ErrIdentityMismatch`, di-mapping handler ke `403 Forbidden`.

```go
npmMatch := *existingNPM == npm
namaMatch := strings.EqualFold(strings.TrimSpace(*existingNama), strings.TrimSpace(nama))

if !npmMatch || !namaMatch {
    return ErrIdentityMismatch
}
```

Perbandingan nama dibuat **case-insensitive** dan **trim spasi** (`strings.EqualFold` + `strings.TrimSpace`) — supaya variasi kapitalisasi/spasi kecil antar sesi (misal "budi santoso" vs "Budi Santoso") tidak membuat praktikan asli malah terkunci dari environment-nya sendiri.

### 5.4b Bug: Verifikasi Sempat Hanya Mengecek NPM, Nama Diabaikan

**Gejala yang ditemukan:** login dengan NPM yang sudah terdaftar tapi nama **berbeda** tetap berhasil lolos (padahal seharusnya ditolak); sebaliknya, NPM berbeda dengan nama yang sudah benar tetap ditolak dengan benar.

**Penyebab:** implementasi awal § 5.4a hanya membandingkan `npm`, variabel `nama` yang di-submit sama sekali tidak diperiksa saat environment sudah pernah diisi. Query `checkQuery` pun awalnya cuma mengambil `p.npm`, tidak mengambil `p.nama` sama sekali.

**Dampak keamanan:** karena NPM biasanya bukan informasi rahasia di satu kelas (teman sekelas gampang tahu NPM satu sama lain), siapapun yang tahu NPM orang lain bisa masuk ke environment itu tinggal mengarang nama sembarang — tepat celah yang seharusnya ditutup oleh fitur verifikasi identitas ini (lihat [§ 4.3a](04-tui-praktikan.md#43a-verifikasi-identitas-wajib-anti-pinjam-pc)).

**Solusi:** `checkQuery` diubah mengambil `p.npm` **dan** `p.nama` sekaligus, dan validasi mensyaratkan **keduanya** cocok (`npmMatch && namaMatch`), bukan hanya NPM. Sudah diterapkan di kode saat ini (lihat query & snippet di atas).

| Nama | NPM | Hasil |
|---|---|---|
| Sama (case-insensitive) | Sama | ✅ Lolos |
| Beda | Sama | ❌ Ditolak *(sebelumnya ini yang jadi celah — sekarang sudah benar ditolak)* |
| Sama | Beda | ❌ Ditolak |
| Beda | Beda | ❌ Ditolak |

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

## 5.5a Testing & Troubleshooting dengan Postman

Testing lewat Postman berguna untuk **mengisolasi layer API** dari layer lain (TUI, SSH, LXD) — kalau ada masalah, kamu bisa pastikan dulu apakah API-nya sendiri sehat sebelum mencurigai TUI atau container.

### Menyiapkan token untuk testing

Endpoint `/environments/me` dan `/environments/me/identify` butuh token asli (bukan sembarang string) karena divalidasi lewat hash SHA-256 ke database. Ada 2 cara mendapatkannya:

**Cara 1 — Ambil dari container yang sudah diprovisioning** (paling gampang, tidak perlu akses database):
```bash
lxc exec <nama-container> -- cat /proc/1/environ | tr '\0' '\n' | grep PRAKTIKUM_API_TOKEN
```
Copy nilai setelah `=` itu ke Postman.

**Cara 2 — Generate token test manual, tanpa LXD sama sekali** (berguna untuk testing API secara terisolasi, misal saat mengembangkan endpoint baru tanpa mau bolak-balik provisioning container):
```bash
# Generate token & hash-nya
TOKEN=$(openssl rand -hex 32)
echo "Token (buat Postman): $TOKEN"
TOKEN_HASH=$(echo -n "$TOKEN" | sha256sum | awk '{print $1}')

# Pastikan sudah ada module_id, room_id, dan 1 sesi (lihat 07-panduan-operasional.md kalau belum ada)
# Lalu insert manual 1 baris environment test:
psql -U praktikum_admin -d praktikum_db -h localhost -c "
    INSERT INTO environments (session_id, container_name, slot_number, ssh_port, status, api_token_hash)
    VALUES ('<UUID-session-yang-sudah-ada>', 'test-postman-01', 99, 9999, 'running', '$TOKEN_HASH');
"
```
Simpan nilai `$TOKEN` yang tercetak — itu yang dipakai di Postman (database hanya menyimpan hash-nya, token asli tidak bisa diambil ulang dari database).

### Setup di Postman

Buat **Environment** baru di Postman dengan 2 variable:

| Variable | Value |
|---|---|
| `base_url` | `http://10.10.10.9:8080` (sesuaikan IP server kamu) |
| `token` | Token dari salah satu cara di atas |

### Koleksi request dasar

**1. Health check** — pastikan server hidup, tanpa perlu token:
```
GET {{base_url}}/healthz
```
Expected: status `200`, body `ok`.

**2. Ambil detail environment:**
```
GET {{base_url}}/api/v1/environments/me
Headers:
  Authorization: Bearer {{token}}
```
Expected: status `200`, body JSON berisi `container_name`, `course_code`, `module`, `room`, dst.

**3. Submit identifikasi:**
```
POST {{base_url}}/api/v1/environments/me/identify
Headers:
  Authorization: Bearer {{token}}
  Content-Type: application/json
Body (raw JSON):
  {
    "nama": "Budi Santoso",
    "npm": "2106123456"
  }
```
Expected (pertama kali submit): status `200`, body `{"success": true}`.
Expected (submit ulang dengan nama/NPM beda dari yang tercatat): status `403`.

### Tabel diagnosis status code

| Status | Arti | Yang perlu dicek |
|---|---|---|
| `200` | Sukses | — |
| `400` | Body request tidak valid (misal `nama`/`npm` kosong) | Cek format JSON di body, pastikan `Content-Type: application/json` ter-set |
| `401` | Token tidak dikenali | Header `Authorization` salah format (harus persis `Bearer <token>`, ada spasi setelah `Bearer`), atau token memang tidak ada di database (`api_token_hash` tidak match) — cek ulang dengan `SELECT container_name FROM environments WHERE api_token_hash = '<hash-token>';` |
| `403` | Nama/NPM tidak cocok dengan yang sudah tercatat | Ini bukan bug — lihat [§ 5.4a](#54a-verifikasi-identitas). Cek data yang sudah tercatat: `SELECT p.nama, p.npm FROM praktikan p JOIN environments e ON e.praktikan_id = p.id WHERE e.container_name = '<nama-container>';` |
| `404` (di endpoint yang tidak terdaftar) | Salah path/method | Cek lagi endpoint di [§ 5.4](#54-endpoint) — perhatikan method (`GET`/`POST`) harus sesuai |
| `500` | Error internal (bug di kode/query SQL) | **Jangan berhenti di Postman** — cek log terminal server (`go run ./cmd/api`), errornya selalu di-log lewat `slog.Error` dengan detail lengkap. Postman cuma menampilkan pesan generik "internal server error" demi keamanan (tidak membocorkan detail internal ke luar) |
| Tidak ada response / timeout | Server tidak reachable | Cek `ss -tlnp \| grep 8080` di server, cek firewall (`ufw status`) — lihat [Troubleshooting § 8.3.3](08-troubleshooting.md#833-koneksi-ke-postgresqlapi-gagal-dari-browser-tapi-berhasil-dari-curl-lokal) |

> **Kenapa error `500` tidak menampilkan detail di Postman?** Ini sengaja — pesan error internal (termasuk potongan query SQL, dsb) tidak boleh bocor ke response API yang bisa diakses pihak luar. Detail sesungguhnya selalu ada di log server, bukan di response HTTP. Jadi kalau ketemu `500`, langkah pertama **selalu** cek terminal API, bukan cuma mengandalkan Postman.

### Membersihkan data test

Kalau pakai Cara 2 (insert manual), jangan lupa hapus row test-nya setelah selesai supaya tidak mengotori data:
```bash
psql -U praktikum_admin -d praktikum_db -h localhost -c "DELETE FROM environments WHERE container_name = 'test-postman-01';"
```

## 5.6 Yang Belum Diimplementasi

- Endpoint untuk web dashboard admin/asisten (list environment, kontrol start/stop/reset).
- Endpoint provisioning (integrasi langsung ke LXD API, menggantikan `kelola-lxd.sh`).
- Otentikasi admin (login web dashboard — tabel `admins` di database sudah ada, logic-nya belum ditulis).