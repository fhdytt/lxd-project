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

satu lab harus menyediakan environment Linux yang terisolasi untuk menjalankan banyak praktikan sekaligus, tanpa setup/reset manual satu per satu, dan tanpa konfigurasi antar praktikan saling mengganggu.

**Stack:**
- **Infrastruktur:** LXD (container), ZFS (storage, snapshot-based reset)
- **Backend:** Go, PostgreSQL
- **TUI:** Go + Bubble Tea
- **Deployment:** 1 server fisik on-premise, semua service native di host