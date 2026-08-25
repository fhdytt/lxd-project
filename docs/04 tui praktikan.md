# 4. TUI Praktikan

TUI (`praktikum-tui`) adalah dashboard terminal yang otomatis muncul saat praktikan SSH login ke environment-nya — menggantikan shell biasa untuk sesaat, menampilkan info sesi, meminta identifikasi (nama + NPM) sekali di awal, lalu melanjutkan ke shell normal.

## 4.1 Alur

```
Praktikan SSH login
        │
        ▼  (ForceCommand memaksa TUI jalan, bukan shell biasa)
┌───────────────────┐
│   Loading           │  ← fetch data dari API pakai token, spinner tampil
└─────────┬───────────┘
          ▼
┌───────────────────┐
│   Dashboard TUI      │
│  - Kode Kursus       │
│  - Modul             │
│  - Ruangan           │
│  - Pertemuan ke-     │
│  - Tanggal           │
│  - Status Env        │
│  - Status Identifikasi│
└─────────┬───────────┘
          │ tekan Enter
          ▼   (SELALU masuk sini, tidak ada jalan pintas walau
          │    already_identified sudah true — lihat 4.3a)
┌───────────────────┐
│  Input Nama          │
└─────────┬───────────┘
          ▼
┌───────────────────┐
│  Input NPM           │
└─────────┬───────────┘
          │ submit → POST ke API (spinner tampil)
          ▼
     berhasil?
          │
    ┌─────┴─────┐
    ▼            ▼
  ya           tidak (NPM tidak cocok / error lain)
    │            │
    ▼            ▼
┌────────────┐ ┌──────────────────┐
│ exec ke      │ │  screenError       │
│ $SHELL       │ │  [Enter] coba lagi  │
│ (proses TUI  │ │  [Ctrl+C] keluar   │
│  digantikan  │ └──────────────────┘
│  shell)      │
└────────────┘

Ctrl+C DI LAYAR MANAPUN → sesi langsung ditutup,
TIDAK PERNAH lanjut ke shell.
```

**Perilaku penting (diperbarui):**
- **Tidak ada lagi jalan pintas otomatis** — walau `already_identified: true` (environment sudah pernah diisi sebelumnya), praktikan **tetap wajib** mengisi nama+NPM. Backend memverifikasi kecocokannya, bukan sekadar mengizinkan lewat. Lihat § 4.3a.
- Kalau API tidak bisa dihubungi (env var belum ter-set, network bermasalah, dsb), TUI **tidak** melanjutkan ke shell.
- **Ctrl+C di layar manapun** (termasuk Dashboard) selalu menutup sesi dan **tidak pernah** lanjut ke shell — lihat § 4.5a untuk detail bug yang pernah terjadi dan fix-nya.

## 4.2 Struktur File

```
praktikum-tui/
├── go.mod          # dependency: bubbletea, bubbles, lipgloss
├── main.go         # state machine Bubble Tea, routing keyboard, exec ke shell
├── api.go          # APIClient — HTTP client ke Go backend
└── README.md       # panduan build & pasang
```

### `main.go`

Berisi model **Bubble Tea** dengan state machine:

| State | Fungsi |
|---|---|
| `screenLoading` | Fetch data environment dari API |
| `screenDashboard` | Tampilkan info sesi, tunggu Enter |
| `screenInputNama` | Input nama lengkap |
| `screenInputNPM` | Input NPM |
| `screenSubmitting` | Kirim data ke API |
| `screenError` | Tampilkan pesan error, tawarkan coba lagi |

### `api.go`

`APIClient` membungkus 2 pemanggilan HTTP:

```go
func (c *APIClient) FetchEnvInfo() (*EnvInfo, error)
func (c *APIClient) SubmitIdentity(nama, npm string) error
```

Struct `EnvInfo` merepresentasikan response dari `GET /api/v1/environments/me`:

```go
type EnvInfo struct {
    ContainerName     string `json:"container_name"`
    CourseCode        string `json:"course_code"`
    Module            string `json:"module"`
    Room              string `json:"room"`
    MeetingNumber     int    `json:"meeting_number"`
    SessionDate       string `json:"session_date"`
    Status            string `json:"status"`
    AlreadyIdentified bool   `json:"already_identified"`
}
```

## 4.2a Tampilan: Responsive & Ringan

Beberapa peningkatan visual ditambahkan, dengan prinsip **tetap ringan** — karena TUI murni rendering teks di terminal (bukan grafis), penambahan ini nyaris tidak menambah beban CPU/memory dibanding versi polos:

- **Spinner loading** (`bubbles/spinner`, sub-package dari dependency yang sudah ada — tidak menambah dependency baru) ditampilkan saat fetch data dan saat submit identitas, memberi indikasi visual bahwa aplikasi sedang bekerja, bukan macet.
- **Box responsive** — lebar kotak dialog menyesuaikan ukuran terminal pengguna lewat `tea.WindowSizeMsg`, dengan batas wajar (30-60 karakter), dan diposisikan center layar. Ini membuat tampilan tetap rapi baik di terminal kecil (PuTTY default 80x24) maupun besar.
- Skema warna konsisten (aksen biru/cyan untuk elemen utama, abu-abu untuk label, merah untuk error) lewat `lipgloss`, tanpa dependency tambahan.

Tidak ada animasi berulang di luar spinner (yang sudah minimal — hanya berjalan saat benar-benar loading), dan tidak ada polling/refresh berkala yang membebani API atau CPU saat TUI dalam keadaan idle menunggu input pengguna.

## 4.3a Verifikasi Identitas Wajib (Anti Pinjam-PC)

**Latar belakang masalah:** desain awal, kalau environment sudah pernah diisi identitasnya, TUI langsung melewati layar input dan lanjut ke shell begitu `Enter` ditekan — tanpa mengecek siapa yang login. Ini celah keamanan: teman sekelas yang duduk di PC yang sama (atau tahu password root) bisa langsung masuk ke environment yang bukan miliknya tanpa hambatan apapun.

**Perilaku sekarang:** TUI **selalu** meminta nama+NPM di setiap login, apapun status `already_identified`-nya. Bedanya ada di sisi backend:

- **Environment belum pernah diisi** → nama+NPM yang di-submit didaftarkan sebagai pemilik baru (perilaku sama seperti sebelumnya).
- **Environment sudah pernah diisi** → NPM yang di-submit **wajib cocok** dengan NPM yang sudah tercatat sebelumnya. Kalau tidak cocok, request ditolak (`403 Forbidden`) dan TUI menampilkan pesan error yang jelas, **tidak** melanjutkan ke shell.

Logic lengkap ada di `IdentifyEnvironment()`, lihat [API Backend § 5.4a](05-api-backend.md#54a-verifikasi-identitas).

## 4.3 Environment Variable

TUI membaca 2 nilai konfigurasi saat start:

| Variable | Contoh | Keterangan |
|---|---|---|
| `PRAKTIKUM_API_URL` | `http://10.184.56.1:8080` | Alamat Go backend, dari sudut pandang container (lewat gateway `lxdbr0`) |
| `PRAKTIKUM_API_TOKEN` | string hex 64 karakter | Token unik per environment, di-generate saat provisioning |

Nilai ini di-inject lewat `lxc config set <container> environment.KEY=value` saat provisioning (lihat [Infrastruktur LXD](02-infrastruktur-lxd.md#26-provisioning-clone-langsung-dari-master)).

### Kenapa tidak cukup `os.Getenv()` saja

Env var yang di-set lewat `lxc config set environment.*` hanya "menempel" pada proses **init container (PID 1)**. Proses yang dijalankan lewat SSH (lewat PAM) mendapat environment yang bersih dan **tidak otomatis mewarisi** env var itu — beda dengan `lxc exec` yang secara eksplisit meneruskannya.

Solusinya, TUI membaca env var lewat fungsi `getContainerEnv()` yang fallback ke `/proc/1/environ` kalau `os.Getenv()` kosong:

```go
func getContainerEnv(key string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    data, err := os.ReadFile("/proc/1/environ")
    if err != nil {
        return ""
    }
    for _, entry := range strings.Split(string(data), "\x00") {
        if strings.HasPrefix(entry, key+"=") {
            return strings.TrimPrefix(entry, key+"=")
        }
    }
    return ""
}
```

> Pendekatan ini mengasumsikan TUI dijalankan sebagai **root** (karena `/proc/1/environ` hanya bisa dibaca oleh root). Ini sesuai keputusan project: praktikan login sebagai root langsung (bukan user non-root dengan sudo) — lihat [Log Perkembangan](09-progress-log.md) untuk alasannya. Kalau nanti kebijakan ini berubah ke user non-root, mekanisme pembacaan konfigurasi ini **perlu diganti** (misal ke file konfigurasi per-container, bukan env var LXD).

## 4.4 Cara Kerja `ForceCommand`

TUI dipaksa jalan otomatis saat SSH login (menggantikan shell biasa) lewat `ForceCommand` di `sshd_config`, diterapkan di **master container**:

```bash
echo "Match User *" >> /etc/ssh/sshd_config
echo "    ForceCommand /usr/local/bin/praktikum-tui" >> /etc/ssh/sshd_config
systemctl restart ssh
```

> **Perhatian format:** baris `Match User *` dan `ForceCommand ...` harus jadi **2 baris terpisah** yang benar (bukan tergabung 1 baris), dan `ForceCommand` harus terindentasi sebagai bagian dari blok `Match`. Kalau di-generate lewat script, gunakan 2 perintah `echo` terpisah, bukan satu string dengan `\n` di dalamnya — lihat [Troubleshooting](08-troubleshooting.md#32-bad-match-condition-di-sshd_config).

Setelah baris ini ditambahkan, **wajib** divalidasi sebelum restart:
```bash
sshd -t   # harus tidak ada output sama sekali kalau valid
```

## 4.5 Exit Behavior

Setelah identifikasi/verifikasi berhasil, TUI melakukan `syscall.Exec()` ke `$SHELL`:

```go
func execShell() {
    shell := os.Getenv("SHELL")
    if shell == "" {
        shell = "/bin/bash"
    }
    env := os.Environ()
    syscall.Exec(shell, []string{shell, "-l"}, env)
}
```

Proses TUI **digantikan total** oleh shell (bukan tetap jalan di background menunggu) — lebih ringan dan lebih simpel secara resource.

### 4.5a Fix Bug: Ctrl+C Malah Lanjut ke Shell

**Gejala yang pernah terjadi:** menekan `Ctrl+C` di layar Dashboard (bukan di layar error) tidak menutup sesi seperti yang diharapkan — malah lanjut masuk ke shell root.

**Penyebab:** versi awal `main()` menentukan boleh-tidaknya lanjut ke shell dengan mengecek `state == screenError`. Padahal `Ctrl+C` memanggil `tea.Quit` dari **state apapun** (termasuk Dashboard), sehingga kondisi pengecekan itu tidak pernah kena, dan kode jatuh ke `execShell()` begitu saja.

**Solusi:** model sekarang punya field eksplisit `shouldContinueToShell bool`, default `false`. Field ini **hanya** diset `true` di satu titik: saat `identitySubmittedMsg` sukses (identifikasi/verifikasi benar-benar berhasil). Ctrl+C tidak pernah menyentuh flag ini, apapun state-nya:

```go
case identitySubmittedMsg:
    if msg.err != nil {
        // ... tampilkan error, flag TETAP false ...
        return m, nil
    }
    m.shouldContinueToShell = true   // satu-satunya tempat ini diset true
    return m, tea.Quit
```

```go
// main()
if !m.shouldContinueToShell {
    fmt.Println("Sesi ditutup.")
    os.Exit(0)
}
execShell()
```

Dengan ini, `Ctrl+C` di layar manapun konsisten menutup sesi, tidak pernah lanjut ke shell.

## 4.6 Build & Pasang

```bash
cd praktikum-tui
go mod tidy
go build -o praktikum-tui .

lxc start master-<modul>
lxc file push praktikum-tui master-<modul>/usr/local/bin/praktikum-tui
lxc exec master-<modul> -- chmod +x /usr/local/bin/praktikum-tui
# ... setup ForceCommand seperti 4.4 ...
lxc stop master-<modul>
```

Detail lengkap langkah operasional ada di [Panduan Operasional](07-panduan-operasional.md#pasang-tui-ke-master-container).

## 4.7 Status Implementasi

| Master | TUI terpasang? |
|---|---|
| `master-netbegin` | ✅ Sudah, tervalidasi end-to-end lewat SSH sungguhan |
| `master-netadmin` | ❌ Belum |