# Sistem Manajemen Environment Linux

Sistem informasi manajemen environment linux berbasis container (LXD) untuk kebutuhan laboratorium. Menyediakan environment Linux terisolasi, diakses lewat SSH tanpa perlu login, menggunakan identifikasi & login akun Linux lewat dashboard terminal, serta dapat dikelola sepenuhnya melalui dashboard administrator.

## Daftar Dokumentasi

| # | Dokumen | Isi |
|---|---|---|
| 1 | [Arsitektur Sistem](docs/01-arsitektur-sistem.md) | Latar belakang, terminologi, diagram arsitektur keseluruhan, keputusan deployment |
| 2 | [Infrastruktur LXD](docs/02-infrastruktur-lxd.md) | Storage, network, profile, master container, `kelola-lxd.sh` sebagai eksekutor murni |
| 3 | [Skema Database](docs/03-skema-database.md) | Struktur tabel PostgreSQL, relasi, dan alasan tiap keputusan desain |
| 4 | [TUI Praktikan](docs/04-tui-praktikan.md) | Dashboard terminal yang dilihat praktikan saat SSH login, verifikasi identitas, login akun Linux |
| 5 | [API Backend](docs/05-api-backend.md) | Struktur Go backend, endpoint, keputusan teknis |
| 6 | [Alur End-to-End](docs/06-alur-end-to-end.md) | Urutan kejadian lengkap dari provisioning sampai praktikan selesai login |
| 7 | [Panduan Operasional](docs/07-panduan-operasional.md) | Cara pakai sehari-hari lewat `lxd-control` untuk admin/asisten lab |
| 8 | [Troubleshooting](docs/08-troubleshooting.md) | Kumpulan masalah yang pernah ditemukan beserta solusinya |
| 9 | [Log Perkembangan](docs/09-progress-log.md) | Riwayat pengerjaan project dari awal sampai kondisi terkini |
| 10 | [lxd-control (Admin TUI)](docs/10-lxd-control.md) | Struktur, fitur, dan alur kerja TUI administrator |

## Ringkasan Cepat

satu lab harus menyediakan environment Linux yang terisolasi untuk menjalankan banyak praktikan sekaligus, tanpa setup/reset manual satu per satu, dan tanpa konfigurasi antar praktikan saling mengganggu.

**Stack:**
- **Infrastruktur:** LXD (container), ZFS (storage, snapshot-based reset)
- **Backend:** Go, PostgreSQL
- **TUI:** Go + Bubble Tea
- **Deployment:** 1 server fisik on-premise, semua service native di host

## Download ISO
Berikut ISO Ubuntu :
[Ubuntu Server](https://drive.google.com/drive/folders/1zK5kfMn1Zh2HF2cVfXMnEeplVNmxMHCf?usp=sharing)
