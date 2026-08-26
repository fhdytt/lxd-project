# Sistem Manajemen Environment Linux

Sistem informasi manajemen environment linux berbasis container (LXD) untuk kebutuhan praktikum. Menyediakan environment Linux terisolasi per praktikan, dapat diakses lewat SSH, dengan identifikasi otomatis lewat dashboard terminal (TUI), dan dikelola lewat web dashboard.

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

## Ringkasan Cepat

**Masalah yang diselesaikan:** satu lab harus menyediakan environment Linux terisolasi untuk banyak praktikan sekaligus (4 ruangan, <50 praktikan/ruangan), tanpa asisten lab harus setup/reset manual satu per satu, dan tanpa konfigurasi antar praktikan saling mengganggu.

**Stack:**
- **Infrastruktur:** LXD (container), ZFS (storage, snapshot-based reset)
- **Backend:** Go (`net/http` + `pgx`), PostgreSQL
- **TUI:** Go + Bubble Tea
- **Web dashboard:** HTML + HTMX + Tailwind CSS (rencana, belum dikerjakan)
- **Deployment:** 1 server fisik on-premise, semua service native di host

**Status saat ini:** infrastruktur LXD, database, API backend, dan TUI sudah tervalidasi jalan end-to-end (SSH login → dashboard → identifikasi → tersimpan di database) untuk modul `netbegin`. Modul `netadmin` masih perlu dipasangi TUI. Web dashboard belum dikerjakan.