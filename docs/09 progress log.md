# 9. Log Perkembangan

Riwayat pengerjaan project dari awal sampai kondisi terkini, berurutan.

## 9.1 Timeline

| # | Tahap | Status |
|---|---|---|
| 1 | Audit & redesign infrastruktur LXD dari instalasi existing (storage `pool-lab`, network `lxdbr0` di-reuse) | ✅ Selesai |
| 2 | Setup profile LXD (`praktikum-netbegin`, `praktikum-netadmin`), master container, script `kelola-lxd.sh` dasar | ✅ Selesai |
| 3 | Fix bug SSH socket activation di master container | ✅ Selesai |
| 4 | Validasi reset via ZFS snapshot, tambah action `reset`/`reset-room` ke script | ✅ Selesai |
| 5 | Diskusi & keputusan arsitektur deployment (1 server fisik, semua service native di host) | ✅ Selesai |
| 6 | Desain skema database, termasuk revisi terminologi (`modules`, `course_code`, `rooms.nama`) | ✅ Selesai |
| 7 | Eksekusi `schema.sql` ke PostgreSQL di server | ✅ Selesai |
| 8 | Desain alur & kontrak API untuk TUI, konsep token otentikasi per environment | ✅ Selesai |
| 9 | Penulisan kode TUI (`praktikum-tui`) — Bubble Tea, state machine, API client | ✅ Selesai |
| 10 | Install Go & PostgreSQL di server Ubuntu | ✅ Selesai |
| 11 | Restrukturisasi & penulisan Go API backend (`praktikum-api`) dengan Standard Go Project Layout | ✅ Selesai |
| 12 | Build & testing API backend di server (`/healthz` berhasil diakses dari dalam & luar server) | ✅ Selesai |
| 13 | Patch `kelola-lxd.sh`: generate token, insert session/environment ke database (solusi sementara untuk testing, sebelum endpoint provisioning permanen di Go backend dibuat) | ✅ Selesai |
| 14 | Testing end-to-end manual: container → token → API `/environments/me` → API `/identify` → tersimpan di database | ✅ Selesai |
| 15 | Build & pasang TUI ke `master-netbegin`, setup `ForceCommand` | ✅ Selesai |
| 16 | Fix bug pembacaan env var (fallback `/proc/1/environ`) dan urutan `lxc config set` vs `lxc start` | ✅ Selesai |
| 17 | Testing SSH sungguhan end-to-end: login → dashboard TUI → identifikasi → shell | ✅ **Berhasil** |
| 17a | Perbaikan TUI: verifikasi identitas wajib (anti pinjam-PC), fix bug Ctrl+C lanjut ke shell, tampilan responsive + spinner (ringan, tanpa dependency baru) | ✅ Selesai |
| 17b | Fix bug database: `FOR UPDATE cannot be applied to the nullable side of an outer join` pada query verifikasi identitas | ✅ Selesai |
| 17c | Fix bug keamanan: verifikasi identitas sempat hanya mengecek NPM (nama diabaikan) — sekarang wajib nama DAN NPM cocok keduanya | ✅ Selesai |
| 18 | Pasang TUI ke `master-netadmin` | ⬜ Belum dimulai |
| 19 | Endpoint provisioning permanen di Go backend (menggantikan bagian database di `kelola-lxd.sh`) | ⬜ Belum dimulai |
| 20 | Web dashboard admin/asisten (HTML+HTMX+Tailwind) | ⬜ Belum dimulai |
| 21 | Firewall, resource reservation, storage terpisah, backup rutin (lihat [Arsitektur § 1.4](01-arsitektur-sistem.md#14-keputusan-deployment-kenapa-semua-di-1-server)) | ⬜ Belum dimulai |
| 22 | Keputusan: praktikan login sebagai `root` langsung (bukan user non-root+sudo), karena modul `netbegin` memang mengharuskan praktik membuat user/group yang butuh privilege tinggi | ✅ Diputuskan |

## 9.2 Keputusan Desain Kunci (Ringkasan)

Untuk konteks lebih lengkap tiap keputusan, lihat dokumen terkait yang ditautkan.

| Keputusan | Ringkasan Alasan | Detail |
|---|---|---|
| Reuse storage/network LXD existing, bukan init ulang | Menghindari mengganggu resource yang sudah terpakai | [Infra § 2.1](02-infrastruktur-lxd.md#21-kondisi-awal) |
| Clone langsung dari master container, bukan publish image | ZFS clone dari container sama cepatnya dengan clone dari image, tapi tidak perlu jaga sinkronisasi alias image | [Infra § 2.6](02-infrastruktur-lxd.md#26-provisioning-clone-langsung-dari-master) |
| Reset via ZFS snapshot restore, bukan delete+reclone | Lebih cepat, proxy device (port SSH) tidak perlu setup ulang | [Infra § 2.7](02-infrastruktur-lxd.md#27-reset--recovery) |
| Semua service (LXD, PostgreSQL, Go) native di 1 server fisik, bukan nested/multi-server | Hanya tersedia 1 server fisik; isolasi didapat dari boundary proses host vs container | [Arsitektur § 1.4](01-arsitektur-sistem.md#14-keputusan-deployment-kenapa-semua-di-1-server) |
| `praktikan` persisten lintas sesi (upsert by NPM), bukan row baru tiap sesi | Praktikan hadir 1x/minggu, NPM yang sama login lagi di pertemuan berikutnya | [Database § 3.2](03-skema-database.md#tabel-praktikan) |
| Tidak ada tabel `cohorts` terpisah | `course_code` dari staff sudah cukup sebagai string biasa; menghindari kebingungan istilah dengan `modules` | [Arsitektur § 1.2](01-arsitektur-sistem.md#12-terminologi-penting) |
| Token API pakai SHA-256, bukan bcrypt | Butuh lookup cepat & deterministik lewat query database, beda kebutuhan dari password manusia | [API § 5.3](05-api-backend.md#53-otentikasi-token) |
| Go backend pakai `net/http` bawaan + `pgx`, bukan framework pihak ketiga | Kebutuhan routing sederhana, `pgx` lebih cepat dari `database/sql` generik | [API § 5.2](05-api-backend.md#52-keputusan-teknis--alasannya) |
| Struktur Go project `cmd/`+`internal/` (Standard Go Project Layout) | Konvensi industri, memisahkan entrypoint dari logic bisnis privat | [API § 5.1](05-api-backend.md#51-struktur-project-standard-go-project-layout) |
| Praktikan login SSH sebagai `root` langsung | Modul `netbegin` mengharuskan praktik membuat user/group yang butuh privilege tinggi; container sudah terisolasi per praktikan | Diputuskan langsung dalam diskusi, belum ada dokumen tersendiri |
| Integrasi database di `kelola-lxd.sh` bersifat sementara | Sesuai arah project (provisioning akan dikontrol lewat web dashboard), tapi dibutuhkan sekarang untuk testing cepat TUI↔API tanpa menunggu endpoint Go selesai | [Infra § 2.8](02-infrastruktur-lxd.md#28-script-kelola-lxdsh) |

## 9.3 Catatan Penting untuk Pengembangan Lanjutan

- **Bagian "DB INTEGRATION (SEMENTARA)" di `kelola-lxd.sh`** akan dibuang begitu endpoint provisioning permanen di Go backend selesai dibuat. Jangan menganggap logic di situ sebagai desain final.
- **Perilaku `praktikan_id` setelah reset** — reset environment (via ZFS snapshot) **tidak** menyentuh database, jadi environment yang sudah pernah diisi tetap terkunci ke NPM yang sama walau isi container-nya sudah dikembalikan bersih. Ini konsisten dengan tujuan anti-pinjam-PC (§ 5.4a/6.3), tapi belum tentu perilaku yang diinginkan untuk semua kasus (misal praktikan resmi pindah kelas/ruangan dan environment itu perlu dialihkan ke praktikan lain). Perlu didiskusikan: apakah perlu endpoint/command khusus admin untuk "unlink" `praktikan_id` secara manual?
- **`master-netadmin` belum punya TUI terpasang** — kalau ada yang provisioning ruangan dengan modul `netadmin` sebelum ini dikerjakan, praktikan akan langsung dapat shell biasa tanpa lewat dashboard/identifikasi.
- **Firewall, resource reservation, storage terpisah, dan backup** (poin 21 di timeline) masih sebatas kesepakatan desain, **belum ada satupun yang diimplementasikan** di server. Ini penting sebelum sistem dipakai untuk praktikum sungguhan dengan data akademik asli.