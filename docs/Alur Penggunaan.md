# Alur Penggunaan

Dokumen ini menjelaskan urutan penggunaan dari lxd ini

## Persiapan

```
Admin menjalankan lxd-control, pilih:
Persiapan Ruangan > f491 > Start > netbegin > pilih sesi
        │
        ▼
lxd-control mengambil detail modul (master_container, lxd_profile)
dan detail ruangan (port_prefix, capacity) dari database
        │
        ▼
Untuk tiap slot:
        │
        ▼
1. Pembuatan nama container & port (f491-01, port 2101, dst)
        │
        ▼
2. Sudah ada di database? → lewati
        │
        ▼
3. Generate token (crypto/rand) + hash SHA-256
        │
        ▼
4. Jalankan scripting.sh provision <master> <profile> <nama> <port> <api_url> <token>
        │                                      │
        │                                      ▼
        │                          lxc copy, lxc config set environment.*,
        │                          lxc start, lxc config device add proxy, lxc snapshot clean
        │
        ▼
5. Berhasil → INSERT/UPSERT baris `environments` (session_id, container_name,
             slot, port, api_token_hash) ke database
        │
        ▼
6. Tandai has_clean_snapshot = true
```

container siap diakses praktikan lewat SSH di port yang sudah dipetakan

## Praktikan Login pertama

```
Praktikan remote dengan ssh root@<ip-host> -p <port>
        │
        ▼
sshd menerima koneksi lalu PAM (pam_permit) meloloskan TANPA
password/key apapun
        │
        ▼
ForceCommand di sshd_config menjalankan /usr/local/bin/lxd-tui
        │
        ▼
Kemduian TUI akan getContainerEnv("PRAKTIKUM_API_URL" / "PRAKTIKUM_API_TOKEN")
        │
        ▼
GET /api/v1/environments/me -> praktikum-api
Header: Authorization: Bearer <token>
        │
        ▼
hash token (SHA-256) lalu mencari di `environments.api_token_hash`
JOIN sessions, rooms, modules, kemudian akan response course_code, module, room, meeting_number,
session_date, status, already_identified
        │
        ▼
TUI menampilkan Dashboard
        │
        ▼
Praktikan tekan Enter
        │
        ▼
lanjut ke penginputan Nama dan penginput NPM
        │
        ▼
POST /api/v1/environments/me/identify -> praktikum-api
Body: { "nama": "...", "npm": "..." }
        │
        ▼
API (transaksi, row locked FOR UPDATE OF e):
   praktikan_id environment ini masih NULL?
        ├── ya  → UPSERT praktikan (by npm)
        │         UPDATE environments SET praktikan_id=..., identified_at=now()
        └── tidak → bandingkan nama DAN npm yang sudah tercatat
                    ├── keduanya cocok → sukses, tidak ada perubahan data
                    └── salah satu tidak cocok → ErrIdentityMismatch (403)
        │
        ▼
    Response
        │
        ├── 403 gagal ──▶ TUI: screenError, maka tidak akan lanjut
        │
        ▼ 200 sukses
baca /etc/passwd di dalam container
kemudian tampilkan daftar user Linux
        │
        ▼
Praktikan pilih user
        │
        ▼
Masukan password akun Linux
        │
        ▼
TUI membaca /etc/shadow lalu akan verifikasi password
        │
        ├── salah → tampilkan error dan ulangi
        │
        ▼ benar
syscall.Exec("/bin/login", ["login", "-f", username])
```

## Praktikan Login jika sudah pernah login

Jika praktikan sudah melakukan login, maka nama + npm sudah tercata ke dalam database. Maka sistem hanya akan memverifikasi loginnya dengan membandingkan nama + npmnya, jika salah satu saja tidak cocok praktikan tersebut tidak bisa login ke dalam shell

## Reset Environment

```
lxd-control > Reset Environment > pilih ruangan atau 1 container
        │
        ▼
Untuk tiap container yang direset:
   1. scripting.sh reset <nama> → cek snapshot 'clean' ada, lanjut lxc restore
   2. Selanjutnya database akan mereset user dengan cara UPDATE environments
      SET praktikan_id = NULL, identified_at = NULL
      WHERE container_name = <nama>
```

## Ganti Sesi Ruangan

```
lxd-control > Ganti Sesi Ruangan > pilih ruangan
        │
        ▼
Sistem akan otomatis mendeteksi modul yang sedang digunakan
        │
        ▼
Sistem akan menampilkan daftar sesi scheduled/active untuk ruangan+modul
        │
        ▼
pilih sesi baru kemudian konfirmasi
        │
        ▼
Untuk setiap container akan :
   1. Reset dengan cara ZFS restore
   2. UnlinkPraktikan (melepas identitas praktikan)
   3. RepointSession dengan UPDATE environments SET session_id = <sesi baru>
```

## Penghapusan Environment

```
lxd-control > Persiapan Ruangan > pilih ruangan > Stop > konfirmasi
        │
        ▼
Untuk setiap container akan :
   1. Penghapusan container dengan cara lxc stop --force, lxc delete (dengan cara ini maka semua data di dalam container akan terhapus)
   2. Pelepasan identitas sesi dengan cara DELETE FROM environments
      WHERE container_name = <nama>
```

**Disclaimer** 
Kenapa disini ada reset, ganti sesi, dan penghapusan?
Jadi gini fungsi reset sendiri digunakan jika terdapat praktikan yang melakukan yang kegiatan yang merusak container sehingga tidak bisa mengikuti sesi kursus, maka container praktikan tersebut di reset untuk memperbaiki containernya tanpa menghapus sesi kursusnya. Lalu untuk ganti sesi digunakan ketika pada 1 hari terdapat lebih dari 1 sesi, maka pada sesi selanjutnya identitas sesi akan di lepas serta mereset container ke awal, menjadikan container tersebut bersih dan bisa digunakan pada sesi selanjutnya. Terakhir untuk penghapusan digunakan untuk mengakhiri kursus pada hari itu, supaya kinerja server tetap terjaga ketika sesi praktikum pada hari itu telah selesai.

## Diagram Komponen Saat Runtime

```
┌──────────────┐  SSH tanpa auth dengan port tertentu    ┌──────────────────────┐
│  Praktikan   │ ------------------------------------->  |     CONTAINER-LXD    |
│   PC-lab     │                                         │  ┌─────────────────┐ │
└──────────────┘                                         │  │    dashboard    │ │
                                                         │  │       TUI       │ │
                                                         │  └────────┬────────┘ │
                                                         └───────────┼──────────┘
                                                                     │ HTTP + Bearer token
                                                                     V
                                                          ┌─────────────────────┐
                                                          │    Dashboard-TUI    │
                                                          │  (host, port 8080)  │
                                                          └───────────┬─────────┘
                                                                      │ pgx pool
┌──────────────┐    jalankan langsung di terminal host                V
│    Admin     │ ---------------------------------------> ┌─────────────────────┐
│    (host)    │                                          │     PostgreSQL      │
└──────────────┘                                          │  (host, port 5432)  │
       │                                                  └─────────────────────┘
       │ lxd-control                                                 ^
       V                                                             │ pgx pool
   scripting.sh ----- lxc CLI ------> LXD ---------------------------┘
```