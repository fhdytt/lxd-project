# Troubleshooting

Kumpulan masalah yang pernah ditemukan selama pengembangan

## Layer Infrastruktur (LXD)

### PuTTY: "Remote side unexpectedly closed network connection"
**Penyebab:** Ubuntu 22.04+/24.04 socket activation untuk sshd. **Solusi:** `systemctl disable ssh.socket && systemctl enable ssh.service` di master

### Reset gagal: snapshot 'clean' tidak ditemukan
**Penyebab:** container dibuat sebelum fitur snapshot ada. **Solusi:** buat ulang container.

## Layer TUI & API

### `Bad Match condition` di `sshd_config`
**Penyebab:** baris `Match User *` dan `ForceCommand ...` tergabung jadi satu baris via heredoc/paste. **Solusi:** 2 perintah `echo` terpisah, validasi `sshd -t` sebelum restart.

### Env var token tidak terbaca di sesi SSH
**Penyebab (2 lapis):**
1. Env var LXD (`environment.*`) tidak diwariskan ke sesi SSH 
   **solusi:** TUI baca `/proc/1/environ` sebagai fallback.
2. Urutan command salah: `lxc config set environment.*` dijalankan **setelah** `lxc start` mengakibatkan LXD cuma menerapkan config itu ke PID 1 saat container **start**
   **Solusi:** urutan diperbaiki di `kelola-lxd.sh provision` dengan set env var dulu, baru start

### Koneksi API gagal dari browser tapi berhasil dari `curl` lokal
**Diagnosis berurutan:** proses masih hidup (`ss -tlnp`) → firewall (`ufw status`) → IP yang diakses benar.

### `FOR UPDATE cannot be applied to the nullable side of an outer join`
**Penyebab:** `FOR UPDATE` dipakai bareng `LEFT JOIN` tanpa spesifik tabel. 
**Solusi:** `FOR UPDATE OF e`

### Verifikasi identitas hanya mengecek NPM, nama diabaikan
**Penyebab:** query awal cuma ambil `p.npm`, `nama` tidak dibandingkan sama sekali mengakibatkan celah keamanan
**Solusi:** bandingkan nama dan NPM

### Reset tidak melepas identitas praktikan lama
**Gejala:** environment yang sudah direset tetap menolak praktikan baru lalu verifikasi identitas selalu gagal walau isi container sudah bersih.
**Penyebab:** reset cuma ZFS restore, tidak menyentuh `praktikan_id` di database
**Solusi:** setiap reset (di `lxd-control`) sekarang selalu diikuti `UnlinkPraktikan()` (`praktikan_id = NULL, identified_at = NULL`)

### TUI crash total (panic) setelah submit password Linux
**Penyebab:** Ubuntu 22.04+/24.04 default pakai hash **yescrypt** (`$y$...`), library Go murni pertama yang dipakai tidak mendukungnya dan panic.
**Solusi:** ganti ke binding cgo `crypt(3)` sistem (`amoghe/go-crypt`), plus `recover()` sebagai pengaman
**Konsekuensi:** server build butuh `gcc` (`build-essential`).

### Provisioning tidak "nempel" ke sesi yang sudah dibuat admin
**Gejala:** dashboard TUI menampilkan `course_code` auto-generate, bukan `course_code` asli yang sudah dibuat lewat `lxd-control`
**Penyebab:** desain awal `kelola-lxd.sh` selalu membuat sesi baru sendiri tiap provisioning, tidak pernah mencari sesi yang sudah ada
**Solusi:** desain ulang total, sekarang **wajib** memilih sesi yang sudah ada (dari daftar `scheduled`/`active`), tidak pernah membuat sesi otomatis lagi

## Kesalahan Operasional Umum

### Lupa `go mod tidy` setelah ada dependency baru
Setiap kali ada fitur baru yang menambah dependency (misal `bubbles/textinput`, `amoghe/go-crypt`), **wajib** `go mod tidy` sebelum `go build`, kalau tidak, muncul error `missing go.sum entry`.