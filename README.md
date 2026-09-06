# Sistem Manajemen Environment Linux

Sistem informasi manajemen environment linux berbasis container (LXD) untuk kebutuhan praktikum. Menyediakan environment Linux terisolasi per praktikan, dapat diakses lewat SSH, dengan identifikasi otomatis lewat dashboard terminal (TUI), serta dapat dikelola sepenuhnya melalui dashboard administrator

## Latar Belakang

Terdapat sebuah laboratorium harus menyediakan environment Linux untuk terselenggaranya praktikum dengan banyaknya mahasiswa secara bersamaan, kemudian muncul beberapa masalah:

- Konfigurasi dari sesi praktikum sebelumnya terbawa ke sesi berikutnya
- Sering kali Konflik saat menggunakan environment linux dengan praktikan lain
- Seorang admin harus melakukan setup dan reset secara manual

Dengan Sistem ini menghasilkan sebuah container LXD yang dikhususkan untuk 1 praktikan 1 environment, dengan membuat container dengan otomatis melalui template container, identifikasi + login akun Linux secara otomatis melalui dashboard terminal, mereset environment dengan acuan snapshot, dan kemampuan untuk mereset environment ke pertemuan berikutnya tanpa hapus-bikin ulang containernya. Kemudian terkait banyaknya praktikan dan beban container per-sesi yang berat, maka pada sistem ini memberlakukan limitasi resource dengan mengkaitkan setiap materi dengan profile masing-masing materi.

## Diagram Arsitektur

```
┌------------------------------------------------------------------┐
│                    1 Server Fisik (Ubuntu)                       │
│                                                                  │
│   ┌---------------------┐       ┌-----------------------------┐  │
│   │    Host OS          │       │   LXD (container layer)     │  │
│   │                     │       │                             │  │
│   │  - PostgreSQL       │       │  master-container (stopped) │  │
│   │  - lxd-api          | <---- |  master-container (stopped) │  │
│   │  - lxd-control      | <---- |  kelola-lxd.sh (subprocess) │  │
│   │                     │       │  ruang1-01 .. ruang1-XX     │  │
│   │                     │       │  ruang2-01 .. ruang2-XX     │  │
│   │                     │       │  ruang3-.. / ruang4-..      │  │
│   └---------------------┘       └-----------------------------┘  |
└------------------------------------------------------------------┘
```

Terdapat 2 alur penggunaan yaitu :

```
Praktikan (untrusted)                    Admin (trusted)
      │                                          │
      V                                          V
SSH tanpa credential (lxd-tui)           Berjalan di host (lxd-control)
      │                                          │
      │ HTTP + Bearer token                      ├──► PostgreSQL
      V                                          │
praktikum-api ----------> PostgreSQL             └──► scripting.sh ──► LXD
```

## Stack
- **Infrastruktur:** LXD (container), ZFS (storage, snapshot-based reset)
- **Backend:** Go, PostgreSQL
- **TUI:** Go + Bubble Tea

## Download ISO
Berikut ISO Ubuntu :
[Ubuntu Server](https://drive.google.com/drive/folders/1zK5kfMn1Zh2HF2cVfXMnEeplVNmxMHCf?usp=sharing)
