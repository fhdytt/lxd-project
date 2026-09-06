# lxd-control

TUI administrator merupakan pusat kontrol utama dari sistem ini. Dapat dijalankan langsung di terminal host bukan lewat container

## Fitur

1. **Lihat Daftar Environment** -> pilih ruangan, lihat semua container
2. **Persiapan Ruangan** -> start (pilih modul + sesi yang sudah dibuat) atau stop
3. **Ganti Sesi Ruangan** ->pindah sesi tanpa hapus dan bikin ulang container 
4. **Reset Environment** -> reset 1 container atau seluruh ruangan
5. **Kelola Ruangan** -> CRUD ruangan
6. **Kelola Sesi** -> CRUD sesi

## Struktur File

| File | Isi |
|---|---|
| `main.go` | Entrypoint |
| `model.go` | State enum, struct model, styles, tipe pesan |
| `commands.go` | `Init()` + operasi async (query DB, orkestrasi LXD) |
| `update.go` | `Update()`, `handleKey()`, logic transisi antar layar |
| `view.go` | Semua fungsi render tampilan |
| `db.go` | Query & mutasi PostgreSQL |
| `actions.go` | Pemanggilan `scripting.sh` + generate token (`crypto/rand`) |
| `config.go` | Baca `.env` |

## Setup

Buat file `.env` dengan cara menduplikasi file `.env.example`.

Pastikan `kelola-lxd.sh` ada di folder yang sama (atau sesuaikan `KELOLA_SCRIPT_PATH`):
```bash
cp ../kelola-lxd.sh .
chmod +x kelola-lxd.sh
```

## Build & Jalankan

```bash
go mod tidy
go build -o lxd-control .
./lxd-control
```

## Catatan Desain

- **Command standalone** kalau `lxd-control` crash, tidak sampai mengganggu akses SSH ke host itu sendiri
- **Reset selalu melepas identitas praktikan** environment yang direset tetap terkunci ke praktikan
- **Semua aksi penghapusan & pelepasan**  wajib lewat layar konfirmasi eksplisit.