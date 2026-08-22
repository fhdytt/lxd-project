# 3. Skema Database

Database PostgreSQL menyimpan seluruh state akademik & operasional sistem: modul, ruangan, praktikan, sesi, dan environment. File skema lengkap ada di `schema.sql` pada root repository.

## 3.1 Diagram Relasi

```
modules ──┐
          │
rooms ────┼──▶ sessions ──▶ environments ──▶ praktikan
          │    (course_code)   (api_token_hash,   (persisten,
          │                     praktikan_id       upsert by npm)
          │                     nullable)

admins (terpisah, login web dashboard)
```

## 3.2 Tabel

### `modules`
Jenis modul teknis praktikum. Menentukan master container & profile LXD mana yang dipakai saat provisioning.

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | UUID | PK |
| `code` | VARCHAR, UNIQUE | `netbegin`, `netadmin` |
| `name` | VARCHAR | Nama tampilan, misal "Network Beginner" |
| `master_container` | VARCHAR | Nama master container LXD, misal `master-netbegin` |
| `lxd_profile` | VARCHAR | Nama profile LXD, misal `praktikum-netbegin` |

### `rooms`
Ruangan lab. **Hanya punya satu field identitas** (`nama`), tidak ada field `code` terpisah.

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | UUID | PK |
| `nama` | VARCHAR, UNIQUE | `f491`, `f492`, `f4111`, `f4112` |
| `port_prefix` | VARCHAR | `21`, `22`, `23`, `24` |
| `capacity` | INT | Jumlah slot/container per sesi (default 5) |

### `praktikan`
Data mahasiswa. **Persisten lintas sesi** — di-*upsert* berdasarkan `npm`, bukan dibuat baru tiap sesi, karena praktikan hadir 1x/minggu dan NPM yang sama login lagi di pertemuan berikutnya.

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | UUID | PK |
| `npm` | VARCHAR, UNIQUE | |
| `nama` | VARCHAR | |

### `sessions`
Satu baris = satu pertemuan praktikum.

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | UUID | PK |
| `course_code` | VARCHAR | ID kursus dari sistem staff, contoh `1WADR261014L`. **String biasa, bukan FK.** |
| `module_id` | UUID | FK → `modules` |
| `room_id` | UUID | FK → `rooms` |
| `meeting_number` | INT | Pertemuan ke-berapa |
| `session_date` | DATE | |
| `status` | ENUM | `scheduled`, `active`, `completed`, `cancelled` |

Constraint: `UNIQUE (course_code, meeting_number)` — satu course_code tidak boleh punya 2 sesi dengan nomor pertemuan yang sama.

### `environments`
Satu baris = satu container = satu slot dalam satu sesi.

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | UUID | PK |
| `session_id` | UUID | FK → `sessions` |
| `praktikan_id` | UUID, nullable | FK → `praktikan`. **Nullable** — diisi belakangan lewat TUI setelah praktikan submit identitas |
| `container_name` | VARCHAR, UNIQUE | `f491-01` |
| `slot_number` | INT | 1, 2, 3, ... |
| `ssh_port` | INT | 2101, 2102, ... |
| `status` | ENUM | `provisioning`, `running`, `stopped`, `error`, `reset` |
| `has_clean_snapshot` | BOOLEAN | Ditandai `true` setelah snapshot `clean` dibuat |
| `api_token_hash` | TEXT | Hash SHA-256 dari token yang di-inject ke container — lihat [API Backend](05-api-backend.md#41-otentikasi-token) |
| `identified_at` | TIMESTAMPTZ, nullable | Waktu praktikan submit nama/NPM |

Constraint: `UNIQUE (session_id, slot_number)` dan `UNIQUE (session_id, ssh_port)` — mencegah 2 environment bentrok di slot/port yang sama dalam satu sesi.

### `admins`
Login web dashboard. Saat ini **satu level akses saja** (belum ada role admin vs asisten terpisah).

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | UUID | PK |
| `username` | VARCHAR, UNIQUE | |
| `password_hash` | TEXT | |
| `full_name` | VARCHAR | |

## 3.3 Keputusan Desain & Alasannya

| Keputusan | Alasan |
|---|---|
| ID pakai **UUID**, bukan auto-increment integer | Lebih aman dipakai di URL/API publik — tidak gampang ditebak/diurut orang lain |
| **ENUM type** untuk `session_status` dan `environment_status` | Lebih strict daripada VARCHAR bebas, mencegah typo status di kode Go |
| **Trigger otomatis** `updated_at` di semua tabel relevan | Memudahkan debugging ("kapan terakhir status ini berubah") tanpa perlu set manual tiap update |
| Tidak ada tabel `cohorts` terpisah | `course_code` sudah cukup sebagai string biasa di `sessions` — lihat [Terminologi](01-arsitektur-sistem.md#12-terminologi-penting) |
| `rooms` hanya field `nama`, tanpa `code` | Nama ruangan (`f491`) sudah cukup jadi identitas tunggal, tidak perlu 2 field berbeda untuk hal yang sama |
| `praktikan_id` di `environments` **nullable** | Container di-*provision* duluan (dapat slot & port) sebelum tahu siapa pemakainya — baru di-*link* setelah TUI submit identitas |
| `api_token_hash` simpan **hash**, bukan token asli | Token asli hanya ada di 2 tempat: di dalam container (env var) dan sesaat saat digenerate. Kalau database bocor, token tidak bisa dipakai ulang |
| Token pakai **SHA-256**, bukan bcrypt | Token API butuh lookup cepat & deterministik langsung lewat query database (`WHERE api_token_hash = $1`) — beda kebutuhan dari password manusia yang butuh bcrypt (lambat, anti brute-force offline) |

## 3.4 Seed Data Dasar

`schema.sql` menyertakan seed data untuk `rooms` dan `modules`, sesuai yang dipakai `kelola-lxd.sh`:

```sql
INSERT INTO rooms (nama, port_prefix, capacity) VALUES
    ('f491',  '21', 5),
    ('f492',  '22', 5),
    ('f4111', '23', 5),
    ('f4112', '24', 5);

INSERT INTO modules (code, name, master_container, lxd_profile) VALUES
    ('netbegin', 'Network Beginner', 'master-netbegin', 'praktikum-netbegin'),
    ('netadmin', 'Network Admin',    'master-netadmin', 'praktikum-netadmin');
```