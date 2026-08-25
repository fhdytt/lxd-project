# Sistem Manajemen Environment Linux

Sistem informasi manajemen environment linux berbasis container (LXD) untuk kebutuhan praktikum. Menyediakan environment Linux terisolasi per praktikan, dapat diakses lewat SSH, dengan identifikasi otomatis lewat dashboard terminal (TUI), dan dikelola lewat web dashboard.

## Daftar Dokumentasi

| # | Dokumen | Isi |
|---|---|---|
| 1 | [Arsitektur Sistem](docs/01-arsitektur-sistem.md) | Latar belakang, diagram arsitektur keseluruhan, dan penjelasan arsitektur |
| 2 | [Infrastruktur LXD](docs/02-infrastruktur-lxd.md) | Storage, network, profile, master container, provisioning, reset/snapshot |
| 3 | [Skema Database](docs/03-skema-database.md) | Struktur tabel PostgreSQL, relasi, dan alasan tiap keputusan desain |
| 4 | [TUI Praktikan](docs/04-tui-praktikan.md) | Dashboard terminal yang dilihat praktikan saat SSH login |
| 5 | [API Backend](docs/05-api-backend.md) | Struktur Go backend, endpoint, keputusan teknis |
| 6 | [Alur End-to-End](docs/06-alur-end-to-end.md) | Urutan kejadian lengkap dari provisioning sampai praktikan selesai identifikasi |
| 7 | [Panduan Operasional](docs/07-panduan-operasional.md) | Cara pakai sehari-hari untuk admin/asisten lab |
| 8 | [Troubleshooting](docs/08-troubleshooting.md) | Kumpulan masalah yang pernah ditemukan beserta solusinya |
| 9 | [Log Perkembangan](docs/09-progress-log.md) | Riwayat pengerjaan project dari awal sampai kondisi terkini |

## Ringkasan Cepat

**Masalah yang diselesaikan:** satu lab harus menyediakan environment Linux terisolasi untuk banyak praktikan sekaligus (4 ruangan, <50 praktikan/ruangan), tanpa asisten lab harus setup/reset manual satu per satu, dan tanpa konfigurasi antar praktikan saling mengganggu.

**Stack:**
- **Infrastruktur:** LXD (container), ZFS (storage, snapshot-based reset)
- **Backend:** Go (`net/http` + `pgx`), PostgreSQL
- **TUI:** Go + Bubble Tea
- **Web dashboard:** HTML + HTMX + Tailwind CSS (rencana, belum dikerjakan)
- **Deployment:** 1 server fisik on-premise, semua service native di host

**Status saat ini:** infrastruktur LXD, database, API backend, dan TUI sudah tervalidasi jalan end-to-end (SSH login → dashboard → identifikasi → tersimpan di database) untuk modul `netbegin`. Modul `netadmin` masih perlu dipasangi TUI. Web dashboard belum dikerjakan.