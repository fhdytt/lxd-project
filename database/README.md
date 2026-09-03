# Skema Database

Database PostgreSQL menyimpan seluruh state akademik & operasional sistem yaitu modul, ruangan, praktikan, sesi, dan environment.

## Diagram Relasi

```
modules ──┐
          │
rooms ────┼──▶ sessions ────▶ environments ────▶ praktikan
          │    (course_code)   (api_token_hash,   (persisten,
          │                     praktikan_id       upsert by npm)
          │                     nullable)

admins (terpisah, login web dashboard)
```

## Tabel

### `modules`
Jenis modul praktikum. Menentukan master container & profile LXD mana yang dipakai saat ingin digunakan.

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | UUID | Primary Key |
| `code` | VARCHAR, UNIQUE | `netbegin`, `netadmin` |
| `name` | VARCHAR | Nama tampilan, misal "Network Beginner" |
| `master_container` | VARCHAR | Nama master container LXD, misal `master-netbegin` |
| `lxd_profile` | VARCHAR | Nama profile LXD, misal `kursus-netbegin` |

### `rooms`
Tabel ini digunakan untuk mengidentifikasi ruangan kursus, dari ruangan, port, dan kapasitas container

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | UUID | Primary Key |
| `nama` | VARCHAR, UNIQUE | `f491`, `f492`, `f4111`, `f4112` |
| `port_prefix` | VARCHAR | `21`, `22`, `23`, `24` |
| `capacity` | INT | Jumlah container per sesi |

### `praktikan`
<div align="justify">
Tabel praktikan adalah tabel yang berisikan data mahasiswa, dimana data mahasiswa tidak akan hilang setelah kursus selesai karena disimpan di database dan tetap ada di minggu berikutnya. Teknisnya adalah ketika NPM mahasiswa belum ada maka sistem akan menambahkan data baru, namun jika NPM mahasiswa sudah ada maka sistem akan memperbarui data yang ada.
<div><br>

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | UUID | Primary Key |
| `npm` | VARCHAR, UNIQUE | Not Null |
| `nama` | VARCHAR | Not Null |

### `sessions`
Tabel sessions ini digunakan untuk mengidentifikasi setiap praktikum 

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | UUID | Primary Key |
| `course_code` | VARCHAR | ID kursus, contoh `1WADR261014L`. |
| `module_id` | UUID | FK → `modules` |
| `room_id` | UUID | FK → `rooms` |
| `meeting_number` | INT | Pertemuan ke-berapa |
| `session_date` | DATE | |
| `status` | ENUM | `scheduled`, `active`, `completed`, `cancelled` |

Tambahan Constraint: `UNIQUE (course_code, meeting_number)`, dimana satu course_code tidak boleh mempunyai 2 sesi dengan nomor pertemuan yang sama.

### `environments`
Tabel environments ini digunakan untuk setiap data pada setiap praktikan ketika menggunakan environments linux ini.

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | UUID | PK |
| `session_id` | UUID | FK → `sessions` |
| `praktikan_id` | UUID, nullable | FK → `praktikan`. **Nullable**, diisi melalui TUI setelah praktikan submit identitas |
| `container_name` | VARCHAR, UNIQUE | `f491-01` |
| `slot_number` | INT | 1, 2, 3, ... |
| `ssh_port` | INT | 2101, 2102, ... |
| `status` | ENUM | `provisioning`, `running`, `stopped`, `error`, `reset` |
| `has_clean_snapshot` | BOOLEAN | Ditandai `true` setelah snapshot `clean` dibuat |
| `api_token_hash` | TEXT | Hash SHA-256 dari token yang di-inject ke container |
| `identified_at` | TIMESTAMPTZ, nullable | Waktu praktikan submit nama/NPM |

Tambahan Constraint: `UNIQUE (session_id, slot_number)` dan `UNIQUE (session_id, ssh_port)`, hal ini mencegah 2 environment bentrok di slot/port yang sama dalam satu sesi.

### `admins` (untuk kebutuhan dashboard web)
Login web dashboard

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | UUID | PK |
| `username` | VARCHAR, UNIQUE | |
| `password_hash` | TEXT | |
| `full_name` | VARCHAR | |

## Dummy Data

`dummy-data.sql` menyertakan awalan data untuk `rooms` dan `modules`

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