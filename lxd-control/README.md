# lxd-control

lxd-control merupakan dashboard terminal yang menjadi pusat control dari sistem lxd, dashboard terminal ini dijalankan langsung di terminal host oleh admin.

## Fitur Dashboard

1. **Lihat Daftar Environment** -> Melihat daftar environment
2. **Persiapan Sesi Ruangan** -> Start & stop sesi dalam 1 ruangan
3. **Reset Sesi Ruangan** -> Mereset sesi praktikum
4. **Reset Environment** -> Reset salah satu container atau seluruh ruangan
5. **Kelola Ruangan** -> Mengelola Ruangan
6. **Kelola Sesi** -> Mengelola Sesi

## Struktur File

| File | Isi |
|---|---|
| `main.go` | Entrypoint |
| `model.go` | State enum, struct model, styles, tipe pesan |
| `commands.go` | `Init()` + operasi async, query DB, mengelola LXD |
| `update.go` | `Update()`, `handleKey()`, logic transisi antar layar |
| `view.go` | Semua fungsi render tampilan |
| `db.go` | Query & mutasi PostgreSQL |
| `actions.go` | Pemanggilan script + generate token (`crypto/rand`) |
| `config.go` | Baca `.env` |

## Setup

Buat file `.env` dengan menyalin file `.env.example`
Pastikan script ada di folder yang sama :

## Build & Jalankan

```bash
go mod tidy
go build -o lxd-control .
./lxd-control
```

## Catatan Desain

- **Command standalone**
- **File script murni eksekutor** dialam file script terdapat 3 fungsi `provision`, `deprovision`, `reset`, masing-masing per satu container
- **Terhubung dengan database** pada terminal dashboard terkoneksi langsung dengan PostgreSQL dan file script sebagai subprocess, sehingga tidak perlu menunggu endpoint
- **Bulk Create**
