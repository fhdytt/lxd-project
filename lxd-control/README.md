# lxd-control

TUI administrator untuk mengelola environment praktikum — dijalankan di **host** (bukan di dalam container LXD), sebagai command standalone, bukan otomatis lewat SSH.

## Fitur v1

1. **Lihat Daftar Environment** — pilih ruangan, lihat semua container yang sedang ada (status, port, snapshot, siapa yang pakai)
2. **Provisioning Ruangan** — start (pilih modul) atau stop, tanpa perlu ingat syntax `kelola-lxd.sh` manual
3. **Reset Environment** — reset satu container tertentu, atau reset seluruh ruangan sekaligus

Semua aksi (start/stop/reset) dieksekusi dengan memanggil `kelola-lxd.sh` yang sudah ada sebagai subprocess — `lxd-control` tidak menduplikasi logic provisioning, cuma jadi antarmuka yang lebih nyaman di atasnya.

## Kenapa tidak lewat API backend (`praktikum-api`)?

`lxd-control` jalan di host yang sama dengan database dan `kelola-lxd.sh`, dijalankan oleh admin yang sudah punya akses penuh ke server itu — beda dengan TUI praktikan (`praktikum-tui`) yang jalan di dalam container *untrusted* dan wajib lewat API supaya bisa divalidasi. Untuk `lxd-control`, koneksi langsung ke PostgreSQL dan exec langsung ke `kelola-lxd.sh` lebih sederhana dan tidak perlu menunggu endpoint admin di API selesai dibuat (itu bagian dari web dashboard yang sengaja ditunda ke versi berikutnya).

## Setup

Buat file `.env` di folder yang sama dengan binary:
```dotenv
DATABASE_URL=postgres://praktikum_admin:password-kamu@localhost:5432/praktikum_db
PGPASSWORD=password-kamu
KELOLA_SCRIPT_PATH=./kelola-lxd.sh
```

**Catatan:** `PGPASSWORD` di sini **wajib diisi terpisah** dari yang ada di `DATABASE_URL`, karena `lxd-control` meneruskannya sebagai environment variable ke subprocess `kelola-lxd.sh` (yang membaca `PGPASSWORD` langsung, bukan `DATABASE_URL`). `KELOLA_SCRIPT_PATH` opsional, default `./kelola-lxd.sh` (relatif ke folder tempat `lxd-control` dijalankan).

## Build & Jalankan

```bash
go mod tidy
go build -o lxd-control .
./lxd-control
```

Pastikan `kelola-lxd.sh` ada di folder yang sama (atau sesuaikan `KELOLA_SCRIPT_PATH`), dan `chmod +x kelola-lxd.sh` sudah dilakukan.

## Struktur File

| File | Isi |
|---|---|
| `main.go` | State machine Bubble Tea — menu navigasi, alur provisioning/reset |
| `config.go` | Baca `.env`, validasi `DATABASE_URL`/`PGPASSWORD`/path script |
| `db.go` | Query PostgreSQL: daftar ruangan, modul, environment |
| `actions.go` | Eksekusi `kelola-lxd.sh` sebagai subprocess |

## Catatan Desain

- **Eksekusi blocking, bukan streaming** — output `kelola-lxd.sh` ditampilkan utuh setelah selesai (dengan spinner selama proses berjalan), bukan baris-per-baris live. Cukup untuk skala testing (provisioning 5 container selesai dalam hitungan detik). Kalau skala produksi jauh lebih besar dan prosesnya makan waktu lama, ini kandidat kuat untuk diubah ke streaming output.
- **Command standalone, bukan `ForceCommand` via SSH** — supaya kalau ada bug/crash di `lxd-control`, tidak sampai mengganggu akses SSH normal ke host (beda risiko dengan crash di TUI praktikan yang cuma berdampak ke 1 container).
- Tidak ada opsi hapus data test/`stop` yang mem-bypass konfirmasi — semua aksi destruktif (`stop`, `reset`, `reset-room`) selalu lewat layar konfirmasi eksplisit sebelum dieksekusi.

## Yang Belum Diimplementasi

- Statistik/riwayat kehadiran praktikan lintas sesi (tidak termasuk fitur v1 sesuai kesepakatan)
- Streaming output live untuk proses yang berjalan lama
- Manajemen `sessions`/`course_code` dari TUI (saat ini `course_code` masih di-generate otomatis oleh `kelola-lxd.sh` sebagai `TEST-<modul>`, sesuai catatan "sementara" di dokumentasi infra)