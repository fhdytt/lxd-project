# 4. TUI Praktikan

TUI (`praktikum-tui`) adalah dashboard terminal yang otomatis muncul saat praktikan SSH login — **tanpa perlu credential SSH apapun** — menampilkan info sesi, meminta identifikasi (nama + NPM), lalu meminta login sebagai akun Linux tertentu (dengan password akun itu), baru masuk ke shell sungguhan.

## 4.1 Alur Lengkap

```
Praktikan SSH login (TANPA diminta password/key apapun)
        │
        ▼  (PAM permit + ForceCommand memaksa TUI jalan)
┌───────────────────┐
│   Loading           │  ← fetch data dari API pakai token, spinner tampil
└─────────┬───────────┘
          ▼
┌───────────────────┐
│   Dashboard TUI      │
└─────────┬───────────┘
          │ Enter (SELALU, tidak ada jalan pintas)
          ▼
┌───────────────────┐      ┌───────────────────┐
│  Input Nama          │ ──▶ │  Input NPM           │
└───────────────────┘      └─────────┬───────────┘
                                       │ submit → POST /identify
                                       ▼
                                  berhasil?
                             ┌─────────┴─────────┐
                             ▼                    ▼
                     403 (nama/NPM         200 sukses
                     tidak cocok)                │
                             │                    ▼
                             ▼          ┌───────────────────┐
                       screenError      │  Pilih User Linux    │
                       (TIDAK exec)     │  (baca /etc/passwd    │
                                        │   langsung, TANPA API)│
                                        └─────────┬───────────┘
                                                   ▼
                                        ┌───────────────────┐
                                        │  Input Password       │
                                        │  akun Linux itu       │
                                        └─────────┬───────────┘
                                                   │ verifikasi via /etc/shadow
                                                   ▼
                                             cocok?
                                        ┌─────────┴─────────┐
                                        ▼                    ▼
                                  salah, ulangi        exec ke shell
                                  (TIDAK exec)          (login -f <user>)

Ctrl+C DI LAYAR MANAPUN → sesi langsung ditutup, TIDAK PERNAH lanjut ke shell.
```

## 4.2 Struktur File

```
praktikum-tui/
├── go.mod              — dependency: bubbletea, bubbles, lipgloss, GehirnInc/crypt (tidak dipakai lagi, lihat § 4.5), amoghe/go-crypt (cgo)
├── main.go              — entrypoint tipis: baca env var, jalankan TUI, exec ke login
├── model.go             — state enum, struct model, styles, tipe pesan
├── commands.go          — Init() + operasi async (fetch API, cek password)
├── update.go            — Update() + handleKey(), transisi antar layar
├── view.go               — semua fungsi render tampilan
├── api.go                — HTTP client ke praktikum-api
└── local_auth.go         — baca /etc/passwd & verifikasi /etc/shadow (LANGSUNG, tanpa API)
```

## 4.3 Tampilan: Responsive & Ringan

- **Spinner loading** (`bubbles/spinner`, tanpa dependency baru) saat fetch data dan submit.
- **Box responsive** — lebar menyesuaikan `tea.WindowSizeMsg`, diposisikan center layar.
- Skema warna konsisten via `lipgloss`.

Karena TUI murni rendering teks (bukan grafis), penambahan ini nyaris tidak menambah beban CPU/memory.

## 4.4 Verifikasi Identitas Wajib (Anti Pinjam-PC)

TUI **selalu** meminta nama+NPM di setiap login, apapun status `already_identified`-nya. Backend memverifikasi **nama DAN NPM sekaligus** — kalau environment sudah pernah diisi, keduanya harus cocok dengan yang tercatat, kalau salah satu saja beda → ditolak `403`. Ini penting terutama karena SSH-nya sendiri sudah tanpa gerbang otentikasi (lihat § 4.6) — satu-satunya lapisan keamanan ada di sini. Logic lengkap ada di [API Backend § 5.4a-5.4b](05-api-backend.md).

## 4.5 Pilih User Linux + Verifikasi Password (`local_auth.go`)

**Latar belakang:** modul `netbegin` mengharuskan praktikan praktik membuat user/group Linux — jadi setelah identifikasi, praktikan perlu bisa login sebagai `root` **atau** user yang sudah mereka buat sendiri, dengan password akun Linux sungguhan (bukan cuma identitas akademik).

### Alur

```go
listLocalUsers()       // baca /etc/passwd, filter UID>=1000 (+ root selalu ada)
                        // -> tampilkan sebagai menu pilihan (↑/↓ + Enter)
verifyLocalPassword()  // baca /etc/shadow, cocokkan hash password
```

### Riwayat perbaikan: panic karena format hash yescrypt

**Gejala:** TUI crash total (`TUI error: program was killed: program experienced a panic`) tepat setelah submit password, memutus seluruh sesi SSH.

**Penyebab:** Ubuntu 22.04+/24.04 default pakai algoritma hash **yescrypt** (`$y$...`) di `/etc/shadow`. Library Go murni pertama yang dipakai (`GehirnInc/crypt`) tidak mendukung yescrypt dan panic saat menemukan format tak dikenal.

**Solusi:** ganti ke `amoghe/go-crypt` — binding **cgo** ke `crypt(3)` bawaan sistem operasi, otomatis mendukung algoritma apapun yang dipakai sistem (termasuk yescrypt), karena memanggil implementasi asli OS, bukan reimplementasi di Go. Tetap ditambah `recover()` sebagai pengaman terakhir supaya error tak terduga apapun tidak pernah membuat seluruh TUI crash.

> **Konsekuensi build:** karena pakai cgo, server yang build `praktikum-tui` **wajib** punya `gcc` (`sudo apt install build-essential` kalau belum ada).

### Kenapa "login -f", bukan "su"?

```go
syscall.Exec("/bin/login", []string{"login", "-f", username}, env)
```

`login -f` (force, tanpa password lagi — password sudah diverifikasi manual oleh TUI) mencatat sesi rapi di `utmp`/`wtmp` seperti login normal. TUI selalu jalan sebagai `root` (lihat § 4.6), jadi valid dijalankan untuk user manapun.

## 4.6 SSH Tanpa Credential (PAM Permit)

**Latar belakang:** distribusi SSH key ke ratusan PC lab (40 PC × 4 ruangan) dinilai tidak praktis. Solusinya: matikan otentikasi SSH sepenuhnya di level PAM, siapapun connect ke port itu langsung sampai ke TUI.

```bash
# /etc/pam.d/sshd
auth     required pam_permit.so
account  required pam_permit.so
session  required pam_permit.so
```

```
# /etc/ssh/sshd_config.d/99-open-access.conf
PermitRootLogin yes
PasswordAuthentication no
KbdInteractiveAuthentication yes
AuthenticationMethods keyboard-interactive
```

**Trade-off yang disadari dan diterima:** port SSH container jadi benar-benar terbuka tanpa gerbang apapun di level SSH. Keamanan sesungguhnya sepenuhnya bertumpu pada TUI (§ 4.4 dan § 4.5) — harus tahu NPM+nama yang cocok, dan password akun Linux yang benar.

**Konsekuensi UX:** praktikan wajib login pakai `user@` eksplisit (`ssh root@<ip> -p <port>`), karena tanpa itu klien SSH pakai username default OS lokal mereka yang kemungkinan tidak ada di container. Bisa disederhanakan lagi lewat config `~/.ssh/config` di tiap PC lab (`Host <ip>` → `User root`), disiapkan sekali saat imaging PC.

## 4.7 Fix Bug: Ctrl+C Malah Lanjut ke Shell

**Gejala:** menekan `Ctrl+C` di layar Dashboard (bukan layar error) tidak menutup sesi — malah lanjut masuk shell.

**Penyebab:** versi awal `main()` menentukan boleh-tidaknya lanjut ke shell dengan mengecek `state == screenError`, padahal `Ctrl+C` memanggil `tea.Quit` dari state manapun.

**Solusi:** field eksplisit `shouldContinueToShell bool`, default `false`, hanya diset `true` di satu titik: password akun Linux **benar-benar** terverifikasi cocok.

## 4.8 Environment Variable & Fallback `/proc/1/environ`

| Variable | Keterangan |
|---|---|
| `PRAKTIKUM_API_URL` | Alamat `praktikum-api`, di-inject `lxd-control` saat provisioning |
| `PRAKTIKUM_API_TOKEN` | Token unik per environment |

Env var LXD (`lxc config set environment.*`) hanya menempel ke PID 1 container, **tidak** diwariskan otomatis ke sesi SSH. TUI fallback baca `/proc/1/environ` kalau `os.Getenv()` kosong — detail lengkap di [Troubleshooting § 8.3.2](08-troubleshooting.md).

## 4.9 Build & Pasang

```bash
cd praktikum-tui
go mod tidy   # butuh gcc terpasang, lihat § 4.5
go build -o praktikum-tui .

lxc start master-<modul>
lxc file push praktikum-tui master-<modul>/usr/local/bin/praktikum-tui
lxc exec master-<modul> -- chmod +x /usr/local/bin/praktikum-tui
# ... setup PAM permit (§ 4.6) dan ForceCommand (Infrastruktur § 2.4) ...
lxc stop master-<modul>
```

Setelah master diupdate, **provisioning ulang** ruangan terkait lewat `lxd-control` supaya container baru membawa TUI versi terbaru (container lama tidak otomatis ter-update).

## 4.10 Status Implementasi

| Master | TUI terpasang? |
|---|---|
| `master-netbegin` | ✅ Lengkap, tervalidasi end-to-end (SSH tanpa auth → identifikasi → pilih user → login) |
| `master-netadmin` | ❌ Belum, perlu langkah yang sama |