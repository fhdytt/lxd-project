# 8. Troubleshooting

Kumpulan masalah yang pernah ditemukan selama pengembangan, beserta gejala, penyebab, dan solusinya. Diurutkan per layer.

## 8.1 Layer Infrastruktur (LXD)

### 8.1.1 PuTTY: "Remote side unexpectedly closed network connection"

**Gejala:** percobaan SSH pertama ke container hasil clone gagal dengan error ini. Percobaan kedua langsung berhasil.

**Penyebab:** Ubuntu 22.04+/24.04 memakai **socket activation** untuk sshd — `ssh.service` dalam kondisi `inactive (dead)` sampai ada koneksi masuk ke `ssh.socket`, baru systemd meng-spawn service secara on-demand. Percobaan koneksi pertama kadang terputus sebelum service siap melakukan handshake SSH.

**Solusi:** nonaktifkan socket activation, paksa `ssh.service` selalu aktif — diterapkan di **master container**, sebelum di-stop, supaya ter-*carry* ke semua clone:
```bash
lxc exec <container> -- systemctl disable ssh.socket
lxc exec <container> -- systemctl enable ssh.service
lxc exec <container> -- systemctl start ssh.service
```

**Verifikasi:**
```bash
lxc exec <container> -- systemctl status ssh.service
# Harus: active (running) — bukan "inactive (dead)"
```

> Container yang sudah terlanjur di-clone **sebelum** fix ini diterapkan ke master perlu dihapus dan diprovisikan ulang agar konsisten.

### 8.1.2 Reset gagal: snapshot 'clean' tidak ditemukan

**Gejala:** `./kelola-lxd.sh reset <container>` keluar error snapshot tidak ditemukan.

**Penyebab:** container dibuat dengan versi `kelola-lxd.sh` sebelum fitur auto-snapshot ditambahkan ke script.

**Solusi:** hapus container lama (`stop`), provisioning ulang (`start`) dengan versi script terbaru.

## 8.2 Layer Bash Script (`kelola-lxd.sh`)

### 8.2.1 Variabel tidak terdefinisi di pesan akhir

**Gejala:** pesan akhir provisioning menampilkan nilai kosong untuk jumlah container.

**Penyebab:** script versi awal memanggil `$JUMLAH_SISWA` padahal variabel yang didefinisikan adalah `$JUMLAH_PRAKTIKAN`.

**Solusi:** sudah diperbaiki di versi terkini script.

### 8.2.2 Resource limit tidak ter-*apply*

**Gejala:** semua container tetap memakai profile `default`, bukan `praktikum-netbegin`/`praktikum-netadmin`.

**Penyebab:** `lxc copy` versi awal hanya menyertakan `--storage pool-lab`, tanpa `--profile`.

**Solusi:** tambahkan `--profile "$PROFILE"` di command `lxc copy`. Sudah diperbaiki di versi terkini.

### 8.2.3 `invalid input syntax for type uuid`

**Gejala:**
```
ERROR:  invalid input syntax for type uuid: "8898c8f0-410b-42d1-b875-052b5973b2bd
INSERT 0 1"
```

**Penyebab:** fungsi helper `psql_query()` yang memanggil `psql -tAc` ternyata tetap menyertakan baris status (`INSERT 0 1`) di output untuk statement `INSERT ... RETURNING`, bukan cuma nilai hasil query-nya saja. Akibatnya variabel seperti `$SESSION_ID` berisi 2 baris tergabung, bukan UUID murni.

**Solusi:** ambil baris pertama saja dari output:
```bash
psql_query() {
    psql -U "$DB_USER" -d "$DB_NAME" -h "$DB_HOST" -tAc "$1" | head -n 1
}
```

### 8.2.4 `duplicate key value violates unique constraint "environments_container_name_key"`

**Gejala:** error ini muncul berulang untuk tiap container saat `start`, walau container-nya sendiri tetap berhasil dibuat di LXD.

**Penyebab:** row lama di tabel `environments` untuk `container_name` yang sama masih ada di database (biasanya karena lupa menjalankan `stop` sebelum `start` ulang, atau `stop` sebelumnya gagal menghapus row karena alasan lain).

**Solusi:** ganti `INSERT` biasa menjadi **upsert** (`ON CONFLICT (container_name) DO UPDATE ...`), supaya idempotent — provisioning ulang tanpa `stop` dulu tetap menghasilkan state database yang benar, bukan error:
```sql
INSERT INTO environments (session_id, container_name, slot_number, ssh_port, status, api_token_hash)
VALUES ('$SESSION_ID', '$NAME', $SLOT, $PORT, 'running', '$TOKEN_HASH')
ON CONFLICT (container_name) DO UPDATE SET
    session_id = EXCLUDED.session_id,
    slot_number = EXCLUDED.slot_number,
    ssh_port = EXCLUDED.ssh_port,
    status = EXCLUDED.status,
    api_token_hash = EXCLUDED.api_token_hash,
    praktikan_id = NULL,
    identified_at = NULL,
    has_clean_snapshot = false
```
Sudah diperbaiki di versi terkini script. Untuk membersihkan sisa row yang sudah terlanjur duplikat: `DELETE FROM environments WHERE container_name LIKE '<ruang>-%';`

## 8.3 Layer TUI & API

### 8.3.1 `Bad Match condition` di `sshd_config`

**Gejala:**
```
Unsupported Match attribute ForceCommand
/etc/ssh/sshd_config line 135: Bad Match condition
```

**Penyebab:** baris `Match User *` dan `ForceCommand ...` ke-gabung jadi **satu baris** saat ditulis lewat `echo` dengan `\n` di dalam satu string — padahal `ForceCommand` harus berada di baris **berikutnya**, sebagai bagian dari blok `Match`, bukan di baris yang sama.

**Solusi:** tulis sebagai 2 perintah `echo` terpisah:
```bash
echo "Match User *" >> /etc/ssh/sshd_config
echo "    ForceCommand /usr/local/bin/praktikum-tui" >> /etc/ssh/sshd_config
```
Selalu validasi dengan `sshd -t` sebelum `systemctl restart ssh` — kalau ada error syntax, restart akan gagal dengan `Job for ssh.service failed`.

### 8.3.2 Env var token tidak terbaca di sesi SSH

**Gejala:** TUI menampilkan `Error: PRAKTIKUM_API_URL atau PRAKTIKUM_API_TOKEN belum di-set`, padahal `lxc exec <container> -- env | grep PRAKTIKUM` menunjukkan env var itu **ada**.

**Penyebab (ditemukan 2 lapis):**

1. **Env var LXD tidak diwariskan ke sesi SSH.** Env var yang di-set lewat `lxc config set <container> environment.X` hanya "menempel" ke proses **init container (PID 1)**. `lxc exec` secara eksplisit meneruskan env var itu ke proses yang dijalankannya, tapi proses yang dijalankan lewat SSH (lewat PAM) mendapat environment bersih dan **tidak** otomatis mewarisi env var dari PID 1.

   **Solusi:** TUI membaca env var lewat fungsi `getContainerEnv()` yang fallback membaca `/proc/1/environ` (env milik PID 1) kalau `os.Getenv()` kosong — lihat [TUI § 4.3](04-tui-praktikan.md#43-environment-variable).

2. **Urutan command salah di script provisioning.** LXD hanya menerapkan config `environment.*` ke PID 1 **pada saat container start**. Kalau `lxc config set environment.X` dijalankan **setelah** `lxc start`, container yang sudah nyala duluan tidak pernah membaca ulang config itu — sehingga bahkan fallback `/proc/1/environ` pun tetap kosong.

   **Solusi:** pindahkan `lxc config set environment.*` ke **sebelum** `lxc start` di `kelola-lxd.sh`. Lihat urutan yang benar di [Alur End-to-End § 6.1](06-alur-end-to-end.md#61-fase-provisioning-oleh-adminasisten).

**Verifikasi setelah fix:**
```bash
lxc exec <container> -- cat /proc/1/environ | tr '\0' '\n' | grep PRAKTIKUM
```

### 8.3.3 Koneksi ke PostgreSQL/API gagal dari browser tapi berhasil dari `curl` lokal

**Gejala:** `curl http://localhost:8080/healthz` di server berhasil, tapi browser dari laptop ke `http://<ip-vm>:8080` tidak bisa connect ("muter-muter"/timeout).

**Penyebab yang perlu dicek berurutan:**
1. Proses `go run`/binary API benar-benar masih hidup (cek `ss -tlnp | grep 8080`) — kalau terminal yang menjalankannya ditutup/di-Ctrl+C, server ikut mati.
2. Firewall (`ufw`) memblokir port itu dari luar (`sudo ufw status`, kalau perlu `sudo ufw allow 8080/tcp`).
3. Akses dari browser memakai IP yang salah/tidak reachable dari jaringan laptop (pastikan pakai IP yang sama dengan yang dipakai untuk SSH/WinSCP yang sudah terbukti jalan, bukan IP internal `lxdbr0`).

**Solusi:** ikuti urutan diagnosis dari dalam ke luar — test `curl localhost` di server dulu, baru `ufw`, baru akses dari luar. Detail langkah lengkap ada di riwayat pengerjaan project.

### 8.3.4 `FOR UPDATE cannot be applied to the nullable side of an outer join`

**Gejala:** TUI menampilkan `Terjadi Kesalahan: gagal mengirim data: internal server error` saat submit nama+NPM. Log API menunjukkan:
```
ERROR: FOR UPDATE cannot be applied to the nullable side of an outer join (SQLSTATE 0A000)
```

**Penyebab:** query verifikasi identitas (lihat [API § 5.4a](05-api-backend.md#54a-verifikasi-identitas)) memakai `LEFT JOIN praktikan p ON ...` supaya tetap dapat baris `environments` walau `praktikan_id` masih `NULL`, digabung dengan `FOR UPDATE` polos di akhir query. PostgreSQL **tidak mengizinkan** mengunci baris (`FOR UPDATE`) di sisi *nullable* dari outer join (di sini tabel `praktikan`), karena secara logis tidak jelas baris mana yang mau dikunci kalau sisi itu memang kosong.

**Solusi:** batasi eksplisit `FOR UPDATE` hanya ke tabel `environments`:
```sql
SELECT p.npm, p.nama
FROM environments e
LEFT JOIN praktikan p ON p.id = e.praktikan_id
WHERE e.id = $1
FOR UPDATE OF e
```
`FOR UPDATE OF e` (bukan `FOR UPDATE` saja) memberitahu PostgreSQL untuk hanya mengunci baris `environments`, membiarkan sisi `praktikan` yang nullable tidak ikut dikunci.

### 8.3.5 Verifikasi identitas hanya mengecek NPM, nama diabaikan

**Gejala:** login dengan NPM yang sudah terdaftar tapi **nama berbeda** tetap berhasil lolos ke shell — padahal seharusnya ditolak. Sebaliknya, NPM berbeda dengan nama yang sudah benar tetap ditolak dengan benar (perilaku ini sendiri sudah sesuai).

**Penyebab:** implementasi awal fitur verifikasi identitas ([§ 4.3a](04-tui-praktikan.md#43a-verifikasi-identitas-wajib-anti-pinjam-pc)) di sisi backend cuma membandingkan `npm` yang di-submit dengan yang sudah tercatat — variabel `nama` yang dikirim TUI sama sekali tidak diperiksa saat environment sudah pernah diisi. Query pengecekannya pun awalnya cuma mengambil `p.npm`, tidak ikut mengambil `p.nama`.

**Dampak keamanan:** NPM biasanya bukan informasi rahasia di satu kelas (teman sekelas gampang saling tahu NPM), jadi siapapun yang tahu NPM orang lain bisa masuk ke environment itu tinggal mengarang nama sembarang — persis celah yang seharusnya ditutup fitur ini. Detail skenario lengkap ada di [Alur End-to-End § 6.3 Skenario C](06-alur-end-to-end.md#skenario-c--tahu-npm-orang-lain-tapi-mengarang-nama-celah-yang-pernah-ada-sekarang-tertutup).

**Solusi:** query diubah mengambil `p.npm` **dan** `p.nama` sekaligus, validasi mensyaratkan **keduanya** cocok:
```go
npmMatch := *existingNPM == npm
namaMatch := strings.EqualFold(strings.TrimSpace(*existingNama), strings.TrimSpace(nama))

if !npmMatch || !namaMatch {
    return ErrIdentityMismatch
}
```
Perbandingan nama dibuat case-insensitive dan trim spasi, supaya variasi kapitalisasi kecil (misal "budi" vs "Budi") tidak membuat praktikan asli malah terkunci. Detail lengkap ada di [API § 5.4b](05-api-backend.md#54b-bug-verifikasi-sempat-hanya-mengecek-npm-nama-diabaikan).

## 8.4 Kesalahan Operasional Umum (Bukan Bug Kode)

### 8.4.1 Command tergabung jadi satu baris

**Gejala:** `go: 'go mod tidy' accepts no arguments` atau error serupa.

**Penyebab:** 2 command yang harusnya dijalankan terpisah (misal `go mod tidy` dan `go build -o nama .`) ter-copy/ter-paste jadi satu baris.

**Solusi:** selalu jalankan command Go satu per satu di baris terpisah.

### 8.4.2 Salah folder project

**Gejala:** `go build` gagal dengan `no Go files in <path>`, atau struktur file yang terlihat (`cmd/`, `internal/`) tidak sesuai dugaan.

**Penyebab:** project **API backend** (struktur `cmd/`+`internal/`) dan project **TUI** (file `main.go`/`api.go` langsung di root) tertukar atau tergabung dalam satu folder yang sama.

**Solusi:** selalu pakai folder terpisah untuk tiap project (misal `~/praktikum-api` dan `~/praktikum-tui`), dan `cd` ke folder yang benar sebelum menjalankan `go build`.

### 8.4.3 File tidak lengkap ter-upload

**Gejala:** `go mod tidy` gagal dengan `go.mod file not found`, padahal seharusnya sudah di-upload.

**Penyebab:** proses upload/copy-paste file dari sumber ke server terputus di tengah jalan, hanya sebagian file yang benar-benar sampai.

**Solusi:** selalu `ls -la` di folder tujuan setelah upload untuk memastikan **semua** file yang diharapkan benar-benar ada, sebelum melanjutkan ke command berikutnya.