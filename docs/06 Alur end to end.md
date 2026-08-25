# 6. Alur End-to-End

Dokumen ini menjelaskan urutan kejadian lengkap dari environment diprovisioning sampai praktikan selesai diidentifikasi — menghubungkan semua layer (LXD, database, API, TUI) yang dijelaskan terpisah di dokumen lain.

## 6.1 Fase Provisioning (oleh Admin/Asisten)

```
Admin/asisten menjalankan:
  ./kelola-lxd.sh start f491 netbegin
        │
        ▼
1. Script cari module_id (modules) & room_id (rooms) di database
        │
        ▼
2. Script buat 1 baris baru di `sessions`
   (course_code, module_id, room_id, meeting_number, session_date)
        │
        ▼
   Untuk tiap slot (1..5):
        │
        ▼
3. lxc copy master-netbegin <ruang>-<nomor> --profile praktikum-netbegin
        │
        ▼
4. Generate token: openssl rand -hex 32
   Hash token: sha256sum
        │
        ▼
5. lxc config set <container> environment.PRAKTIKUM_API_URL   <-- SEBELUM start!
   lxc config set <container> environment.PRAKTIKUM_API_TOKEN <-- SEBELUM start!
        │
        ▼
6. INSERT/UPSERT ke `environments`
   (session_id, container_name, slot_number, ssh_port, api_token_hash)
        │
        ▼
7. lxc start <container>
   lxc config device add <container> proxy22 proxy listen=... connect=127.0.0.1:22
        │
        ▼
8. Tunggu container Running, lalu:
   lxc snapshot <container> clean
   UPDATE environments SET has_clean_snapshot = true
```

> **Kenapa langkah 5 (set env var) harus SEBELUM langkah 7 (start)?** LXD hanya menerapkan config `environment.*` ke proses init (PID 1) pada saat container **start**. Kalau di-set setelah container sudah nyala, PID 1 yang sudah jalan duluan tidak pernah membaca ulang config itu. Ini pernah jadi bug nyata di project ini — lihat [Troubleshooting](08-troubleshooting.md#33-env-var-token-tidak-terbaca-di-sesi-ssh).

Setelah fase ini selesai, container siap diakses praktikan lewat SSH di port yang sudah dipetakan.

## 6.2 Fase Praktikan Login (Pertama Kali)

```
Praktikan: ssh root@<ip-host> -p <port>
        │
        ▼
sshd menerima koneksi, cek ForceCommand di sshd_config
        │
        ▼
sshd menjalankan /usr/local/bin/praktikum-tui
(BUKAN shell biasa)
        │
        ▼
TUI: getContainerEnv("PRAKTIKUM_API_URL")
     getContainerEnv("PRAKTIKUM_API_TOKEN")
     (baca os.Getenv, fallback ke /proc/1/environ)
        │
        ▼
TUI --GET /api/v1/environments/me--> API Backend
     Header: Authorization: Bearer <token>
        │
        ▼
API: hash token (SHA-256) → cari di `environments.api_token_hash`
     JOIN sessions, rooms, modules
     → response: course_code, module, room, meeting_number,
                 session_date, status, already_identified=false
        │
        ▼
TUI menampilkan Dashboard
        │
        ▼
Praktikan tekan Enter
        │
        ▼
SELALU lanjut ke: Input Nama → Input NPM
(tidak ada jalan pintas berdasarkan already_identified)
        │
        ▼
TUI --POST /api/v1/environments/me/identify--> API Backend
     Body: { "nama": "...", "npm": "..." }
        │
        ▼
API (dalam 1 transaksi database, row locked FOR UPDATE):
   praktikan_id environment ini masih NULL?
     ├── ya  → UPSERT praktikan (by npm)
     │         UPDATE environments SET praktikan_id=..., identified_at=now()
     └── tidak → bandingkan npm submit vs npm yang sudah tercatat
                  ├── cocok    → sukses, tidak ada perubahan data
                  └── tidak cocok → ErrIdentityMismatch (403 Forbidden)
        │
        ▼
Response: { "success": true }  ATAU  403 + pesan error
        │
        ├── 200 sukses ──────────────────────────┐
        │                                          ▼
        └── 403 gagal ──▶ TUI: screenError    TUI: syscall.Exec ke $SHELL
                          [Enter] coba lagi     (proses TUI digantikan shell)
                          [Ctrl+C] keluar
                          (TIDAK exec ke shell)
```

## 6.3 Fase Praktikan Login (Sesi Berikutnya) — Verifikasi, Bukan Auto-Skip

> **Perubahan penting:** desain awal, kalau environment sudah pernah diisi, TUI otomatis melewati layar input begitu `Enter` ditekan. Ini diganti karena jadi celah keamanan (siapapun bisa login ke environment orang lain tanpa verifikasi). Perilaku sekarang: **selalu** diminta isi nama+NPM, backend yang memutuskan lolos atau tidak. Detail alasan & fix ada di [TUI § 4.3a](04-tui-praktikan.md#43a-verifikasi-identitas-wajib-anti-pinjam-pc) dan [API § 5.4a](05-api-backend.md#54a-verifikasi-identitas).

### Skenario A — NPM yang sama login lagi (kondisi normal)

```
Praktikan (pemilik asli) SSH login ke environment yang SAMA
        │
        ▼
TUI fetch /environments/me → already_identified = true (sekadar info di dashboard)
        │
        ▼
Praktikan tekan Enter → TETAP diminta isi Nama + NPM
        │
        ▼
Submit npm yang SAMA dengan yang tercatat sebelumnya
        │
        ▼
API: praktikan_id sudah ada → bandingkan npm → COCOK
        │
        ▼
Response 200 { "success": true } (tidak ada data yang diubah)
        │
        ▼
TUI: exec ke $SHELL — praktikan lanjut kerja normal
```

### Skenario B — Orang lain mencoba login ke environment yang bukan miliknya

```
Orang lain (misal teman sekelas, tahu password root) SSH login
ke environment yang SUDAH dimiliki praktikan lain
        │
        ▼
TUI fetch /environments/me → already_identified = true
        │
        ▼
Tekan Enter → diminta isi Nama + NPM
        │
        ▼
Submit NPM milik DIA SENDIRI (bukan NPM pemilik environment ini)
        │
        ▼
API: praktikan_id sudah ada → bandingkan npm → TIDAK COCOK
        │
        ▼
Response 403 Forbidden — ErrIdentityMismatch
        │
        ▼
TUI: screenError, pesan "Environment ini sudah terdaftar
     atas nama praktikan lain" — TIDAK exec ke shell
```

> **Catatan tentang reset:** ini berlaku selama environment yang sama belum di-reset (`reset`/`reset-room`) atau dihapus (`stop`). Reset environment **tidak** menyentuh database — row `environments` dan `praktikan_id` tetap sama seperti sebelum reset (hanya isi container yang dikembalikan lewat ZFS snapshot). Artinya environment yang sudah pernah diisi **tetap terkunci** ke NPM yang sama walau sudah direset, konsisten dengan tujuan anti-pinjam-PC. **Ini belum tentu perilaku yang diinginkan untuk semua kasus** (misal praktikan pindah kelas/ruangan) — lihat catatan terbuka di [Log Perkembangan](09-progress-log.md).

## 6.4 Fase Reset Environment

```
Asisten: ./kelola-lxd.sh reset f491-03
        │
        ▼
Cek: apakah container f491-03 punya snapshot "clean"?
        │
        ├── tidak ada → Error, keluar
        │
        └── ada
              │
              ▼
        lxc restore f491-03 clean
              │
              ▼
        Container kembali ke kondisi persis
        seperti baru selesai di-clone dari master
        (proxy device TETAP ada, tidak perlu setup ulang)
```

Reset **tidak** menyentuh database — row `environments` dan relasi `praktikan_id` tetap sama seperti sebelum reset. Ini artinya kalau container sudah pernah diidentifikasi lalu direset, `already_identified` akan **tetap `true`** meski isi container-nya sudah dikembalikan bersih.

## 6.5 Fase Penghapusan Environment (Akhir Sesi/Semester)

```
Asisten: ./kelola-lxd.sh stop f491
        │
        ▼
Untuk tiap container di ruangan itu:
   lxc stop <container> --force
   lxc delete <container>
   DELETE FROM environments WHERE container_name = <container>
```

**Kenapa row `environments` harus dihapus di sini?** Karena `container_name` bersifat `UNIQUE` di database — kalau tidak dihapus, provisioning berikutnya dengan nama container yang sama (`f491-01` dst) akan gagal insert karena bentrok. Row `praktikan` **tidak** ikut terhapus (tetap persisten untuk sesi berikutnya).

## 6.6 Diagram Komponen Saat Runtime

```
┌──────────────┐  SSH (port 21xx-24xx)   ┌─────────────────────┐
│  Praktikan    │ ───────────────────────▶│  Container LXD        │
│  (laptop)     │                          │  ┌─────────────────┐│
└──────────────┘                          │  │ praktikum-tui    ││
                                            │  │ (ForceCommand)   ││
                                            │  └────────┬─────────┘│
                                            └───────────┼──────────┘
                                                         │ HTTP + Bearer token
                                                         │ (lewat lxdbr0 gateway)
                                                         ▼
                                            ┌─────────────────────┐
                                            │  Go API Backend      │
                                            │  (host, port 8080)   │
                                            └───────────┬──────────┘
                                                         │ pgx pool
                                                         ▼
                                            ┌─────────────────────┐
                                            │  PostgreSQL           │
                                            │  (host, port 5432)    │
                                            └─────────────────────┘
```