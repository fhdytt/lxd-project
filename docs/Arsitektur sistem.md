# Arsitektur Sistem

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

**Disclaimer** 
Pada Container praktikan dapat diakses oleh siapa saja, jadi setiap aksinya perlu divalidasi lewat API dengan token per-environment. Namun untuk user Admin yang bertugas untuk mengontrol container sudah pasti trusted, menjadikan tidak perlu lapisan API tambahan serta langsung mengakses database dan LXD lebih sederhana dan tidak perlu menunggu endpoint API selesai dibuat.


## SSH Praktikan Tanpa Credential

Pada saat praktikan masuk menggunakan ssh cukup memasukkan username SSH apapun saat connect dengan begitu SSH berhasil terhubung dengan container langsung, kemudian praktikan akan masuk ke dashboard `lxd-tui`. Hal ini tercapai melalui konfigurasi PAM (`pam_permit`) di container, bukan melalui distribusi SSH key ke ratusan PC lab. Hal tersebut mengakibatkan praktikan wajib mengindetifikasi dengan nama + NPM, kemudian untuk mengakses shell praktikan harus memilih user linux yang tersedia