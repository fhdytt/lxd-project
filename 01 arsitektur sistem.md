# 1. Arsitektur Sistem

## 1.1 Latar Belakang

Satu laboratorium harus menyediakan environment Linux praktikum untuk banyak mahasiswa secara bersamaan. Dengan environment yang dikelola manual, muncul beberapa masalah:

- Konfigurasi dari praktikum sebelumnya bisa terbawa ke sesi berikutnya.
- Konflik penggunaan environment antar praktikan.
- Asisten lab harus melakukan setup dan pemulihan (reset) secara manual, satu per satu.

Sistem ini menyelesaikannya dengan container LXD terisolasi per praktikan, provisioning otomatis dari template (master container), identifikasi otomatis lewat dashboard terminal, dan reset environment berbasis snapshot.

**Skala target:** 4 ruangan lab (`f491`, `f492`, `f4111`, `f4112`), masing-masing kurang dari 50 praktikan bersamaan (potensi hingga ~200 container aktif total).

**Deployment:** sepenuhnya on-premise, di **1 server fisik** (bukan cloud, bukan multi-server) — lihat bagian 1.4.

## 1.2 Terminologi Penting

Istilah-istilah berikut sempat berubah makna selama proses desain. Versi final (yang dipakai di kode dan dokumentasi) ada di kolom kanan.

| Istilah | Arti yang dipakai SEKARANG | Catatan |
|---|---|---|
| **Modul** (tabel `modules`) | Jenis modul teknis praktikum: `netbegin`, `netadmin`. Menentukan master container & profile LXD mana yang dipakai saat provisioning. | Sebelumnya disebut "course" — di-rename supaya tidak tertukar dengan `course_code`. |
| **course_code** (field di `sessions`) | ID resmi kursus dari sistem staff kampus, contoh: `1WADR261014L`. String bebas, bukan foreign key ke tabel manapun. | Sebelumnya sempat dinamai `cohort_code` dengan tabel `cohorts` terpisah — keduanya sudah dihapus. Tidak ada lagi konsep "cohort" di sistem ini. |
| **Ruangan** (tabel `rooms`, field `nama`) | Nama ruangan lab, contoh: `f491`. | Tabel ini hanya punya satu field identitas (`nama`), tidak ada field `code` terpisah. |
| **Master container** | Container LXD yang menjadi "template" per modul (`master-netbegin`, `master-netadmin`). Selalu dalam kondisi *stopped* kecuali sedang dikonfigurasi. Tidak pernah dipakai langsung oleh praktikan. | |
| **Environment** (tabel `environments`) | Satu container hasil clone dari master, dipakai oleh satu praktikan dalam satu sesi. Satu baris tabel = satu container. | |
| **Sesi** (tabel `sessions`) | Satu pertemuan praktikum: kombinasi `course_code` + modul + ruangan + nomor pertemuan + tanggal. | |
| **Praktikan** (tabel `praktikan`) | Data mahasiswa, **persisten lintas sesi** — di-*upsert* berdasarkan NPM, bukan dibuat baru tiap sesi, karena praktikan hadir 1x/minggu dan NPM yang sama login lagi di pertemuan berikutnya. | |
| **TUI** | Program terminal (`praktikum-tui`) yang otomatis muncul saat praktikan SSH login, sebelum mereka sampai ke shell. | |
| **Control plane / API backend** | Program Go (`praktikum-api`) yang menjadi jembatan antara TUI, web dashboard, dan database. | |

## 1.3 Diagram Arsitektur

```
┌───────────────────────────────────────────────────────────────────┐
│                      1 Server Fisik (Ubuntu)                      │
│                                                                   │
│   ┌──────────────────────────┐    ┌────────────────────────────┐  │
│   │   Host OS (native)       │    │   LXD (container layer)    │  │
│   │                          │    │                            │  │
│   │  - PostgreSQL            │    │  master-netbegin (stopped) │  │
│   │  - Go API backend        │◄───┤  master-netadmin (stopped) │  │
│   │    (praktikum-api)       │    │                            │  │
│   │  - Web dashboard         │    │  f491-01 .. f491-XX        │  │
│   │    (rencana)             │    │  f492-01 .. f492-XX        │  │
│   │                          │    │  f4111-.. / f4112-..       │  │
│   │                          │    │  (tiap container jalankan  │  │
│   │                          │    │   praktikum-tui saat SSH)  │  │
│   └──────────────────────────┘    └────────────────────────────┘  │
└───────────────────────────────────────────────────────────────────┘
```

Alur data singkat:

```
Praktikan --SSH--> Container (TUI jalan otomatis)
                        │
                        │ HTTP + Bearer token
                        ▼
                  Go API Backend  <──── Web dashboard (rencana)
                        │
                        ▼
                   PostgreSQL
```

## 1.4 Keputusan Deployment: Kenapa Semua di 1 Server?

Awalnya dipertimbangkan memisah LXD (host container, untrusted) dari web/database server (data sensitif) demi isolasi keamanan dan menghindari resource contention. Namun karena **hanya tersedia 1 server fisik**, pemisahan fisik itu tidak memungkinkan.

**Solusi yang dipakai — pemisahan logis, bukan fisik:**

- LXD, PostgreSQL, dan Go backend semuanya jalan **native di 1 OS host yang sama**, sejajar (**bukan** nested — LXD tidak dijalankan di dalam LXD, PostgreSQL/Go juga tidak dijalankan di dalam container LXD terpisah).
- Isolasi didapat dari boundary **proses host vs container LXD** (container praktikan tetap dalam namespace terisolasi dari proses PostgreSQL/Go yang jalan langsung di host).
- Go backend akses LXD lewat **Unix socket lokal**, PostgreSQL diakses lewat **localhost**, tidak perlu remote API/TLS antar mesin.

**Yang perlu diterapkan supaya isolasi logis ini efektif** (lihat detail langkah di [Panduan Operasional](07-panduan-operasional.md)):

1. **Firewall** — container praktikan (network `lxdbr0`) diblokir akses langsung ke port PostgreSQL (`5432`), hanya boleh akses port API Go (`8080`).
2. **Resource reservation** — PostgreSQL dan Go backend diberi jatah CPU/memory terjamin (`systemd` `CPUWeight`/`MemoryHigh`) supaya tidak terganggu saat ratusan container praktikan aktif bersamaan.
3. **Storage terpisah** — data PostgreSQL idealnya di dataset ZFS/disk terpisah dari pool container (yang sering di-snapshot/restore), supaya I/O tidak saling mengganggu.
4. **Backup rutin** — karena hanya 1 server fisik (single point of failure), wajib ada `pg_dump` terjadwal yang disimpan **di luar** server ini.

> **Status implementasi poin 1-4 di atas:** belum dikerjakan (masih tahap desain/kesepakatan). Lihat [Log Perkembangan](09-progress-log.md) untuk status terkini.

## 1.5 Kapan Perlu Pisah Server?

Kalau ke depannya:
- Jumlah praktikan/container concurrent naik signifikan dari estimasi awal,
- Ada kebutuhan compliance/keamanan lebih ketat,
- Tersedia resource server tambahan,

...maka layak dipertimbangkan memisah LXD host dari server aplikasi (Web+DB), dengan Go backend mengakses LXD lewat **remote API over HTTPS** (`lxc remote add`) alih-alih Unix socket lokal.
