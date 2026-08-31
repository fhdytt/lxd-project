# Arsitektur Sistem

## Latar Belakang

Sebuah laboratorium harus menyediakan environment Linux untuk kebutuhan praktikum dengan banyak mahasiswa secara bersamaan. Dengan environment yang dikelola secara manual, muncul beberapa masalah:

- Konfigurasi dari praktikum sebelumnya bisa terbawa ke sesi berikutnya.
- Konflik penggunaan environment antar praktikan.
- harus melakukan setup dan pemulihan secara manual

Sistem ini dapat menyelesaikan permasalahannya dengan container LXD terisolasi per praktikan, menyediakan environment otomatis dari template (master container), identifikasi otomatis lewat dashboard terminal, dan reset environment berbasis snapshot, sehingga tidak perlu untuk memulai dari awal lagi hanya untuk meresetnya.

**Skala target:** 4 ruangan lab masing-masing kurang dari 50 praktikan bersamaan.

**Deployment:** sepenuhnya on-premise, di **1 server fisik** 

## Diagram Arsitektur

```
┌──────────────────────────────────────────────────────────────────┐
│                   1 Server Fisik (Ubuntu server)                 │
│                                                                  │
│   ┌──────────────────────────┐    ┌───────────────────────────┐  │
│   │   Host OS (native)       │    │   LXD (container layer)   │  │
│   │                          │    │                           │  │
│   │  - PostgreSQL            │    │  master-netbegin (stopped)│  │
│   │  - Go API backend        │◄───┤  master-netadmin (stopped)│  │
│   │  - Web dashboard         │    │  f491-01 .. f491-XX       │  │
│   │                          │    │  f492-01 .. f492-XX       │  │
│   │                          │    │  f4111-.. / f4112-..      │  │
│   │                          │    │  (tiap container jalankan │  │
│   │                          │    │   tui saat SSH)           │  │
│   └──────────────────────────┘    └───────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

Alur data singkat:

```
Praktikan --SSH--> Container (TUI)
                        │
                        │ HTTP + Bearer token
                        ▼
                  Go API Backend  <──── Web dashboard
                        │
                        ▼
                   PostgreSQL
```

## 1.3 Penjelasan Arsitektur
- LXD, PostgreSQL, dan Go backend semuanya berjalan **native di 1 OS host yang sama**, secara sejajar (**bukan** nested dengan kata lain LXD tidak dijalankan di dalam LXD, PostgreSQL/Go juga tidak dijalankan di dalam container LXD).
- Isolasi didapat dari hubungan **proses host vs container LXD** (container praktikan tetap terisolasi dari proses PostgreSQL/Go yang berjalan langsung di host).
- Go backend akses LXD melalui **Unix socket lokal**, PostgreSQL diakses lewat **localhost**, tidak perlu remote API/TLS antar mesin.
- Container praktikan (network `lxdbr0`) diblokir akses langsung ke port PostgreSQL (`5432`), hanya boleh akses port API Go (`8080`).
- PostgreSQL dan Go backend diberikan jatah CPU/memory terjamin (`systemd` `CPUWeight`/`MemoryHigh`) supaya tidak terganggu saat ratusan container praktikan aktif bersamaan.
- Data PostgreSQL idealnya di dataset ZFS/disk terpisah dari pool container (yang sering di-snapshot/restore), supaya I/O tidak saling mengganggu.
- Karena hanya 1 server fisik (single point of failure), wajib ada `pg_dump` terjadwal yang disimpan **di luar** server ini.