# 2. Infrastruktur LXD

## 2.1 Kondisi Awal

Sebelum project ini di-redesign, sudah ada instalasi LXD yang pernah dipakai untuk eksperimen sebelumnya:

```bash
lxc storage list   # pool-lab (zfs) sudah ada
lxc network list   # lxdbr0 sudah ada, subnet 10.184.56.1/24
lxc profile list    # hanya ada profile "default"
lxc list            # ada 1 container sisa: master-netbegin (stopped, masih bersih)
lxc image list      # ubuntu 24.04 sudah ter-cache lokal
```

**Keputusan:** storage pool (`pool-lab`, ZFS) dan network bridge (`lxdbr0`) **di-reuse**, tidak dibuat ulang dari nol — menghindari risiko mengganggu resource yang sudah terpakai.

## 2.2 Storage & Network

| Komponen | Nilai | Catatan |
|---|---|---|
| Storage pool | `pool-lab` | Driver **ZFS** — dipilih karena snapshot/clone berbasis copy-on-write, jauh lebih cepat & hemat disk dibanding driver `dir` |
| Network bridge | `lxdbr0` | Subnet `10.184.56.1/24` |
| LXD version | 5.21.6 LTS | |

## 2.3 Profile LXD

Dua profile dibuat untuk membedakan alokasi resource per jenis modul. Karena VM testing awal hanya 2 core/4GB, dipakai `limits.cpu.allowance` (persentase waktu CPU / CPU sharing), **bukan** `limits.cpu` (reserved core penuh), supaya container bisa berbagi core tanpa oversubscribe.

```bash
# netbegin — ringan, cuma user/group/sftp
lxc profile create praktikum-netbegin
lxc profile device add praktikum-netbegin root disk path=/ pool=pool-lab
lxc profile device add praktikum-netbegin eth0 nic network=lxdbr0
lxc profile set praktikum-netbegin limits.cpu.allowance 20%
lxc profile set praktikum-netbegin limits.memory 256MB

# netadmin — lebih berat, nmap & tools security
lxc profile create praktikum-netadmin
lxc profile device add praktikum-netadmin root disk path=/ pool=pool-lab
lxc profile device add praktikum-netadmin eth0 nic network=lxdbr0
lxc profile set praktikum-netadmin limits.cpu.allowance 40%
lxc profile set praktikum-netadmin limits.memory 512MB
```

> **Catatan skalabilitas:** angka di atas hanya untuk skala uji coba (2 core/4GB, 5 container/ruang). Saat pindah ke server produksi dengan spek lebih besar, jalankan `lxc profile set` ulang dengan nilai baru — tidak perlu rebuild profile dari nol.

## 2.4 Master Container

Master container adalah "golden template" per modul. Selalu **stopped** kecuali sedang dikonfigurasi — tidak pernah dipakai langsung oleh praktikan, hanya jadi sumber clone.

### Konvensi penamaan

```
master-<kode-modul>
```
Contoh: `master-netbegin`, `master-netadmin`.

### Langkah setup umum

```bash
lxc start master-<modul>
lxc exec master-<modul> -- bash
```

Di dalam container:
```bash
apt update && apt upgrade -y
apt install -y openssh-server
# setup user/group dasar sesuai kebutuhan modul
```

### Fix penting: SSH socket activation

Ubuntu 22.04+/24.04 memakai **socket activation** untuk sshd — `ssh.service` baru di-*spawn* saat ada koneksi masuk ke `ssh.socket`, bukan selalu aktif. Ini menyebabkan percobaan SSH pertama ke container hasil clone kadang gagal (lihat [Troubleshooting](08-troubleshooting.md#31-ssh-socket-activation)). Fix diterapkan di **setiap master**:

```bash
lxc exec master-<modul> -- systemctl disable ssh.socket
lxc exec master-<modul> -- systemctl enable ssh.service
lxc exec master-<modul> -- systemctl start ssh.service
lxc exec master-<modul> -- systemctl status ssh.service   # pastikan "active (running)"
```

### Selesai konfigurasi

```bash
exit
lxc stop master-<modul>
```

### Status per modul

| Master | Status |
|---|---|
| `master-netbegin` | Selesai: openssh-server, user/group, SFTP, fix socket activation, TUI terpasang. Tervalidasi SFTP (WinSCP) dan SSH+TUI end-to-end. |
| `master-netadmin` | Setup dasar selesai (openssh, update, fix socket activation). Tools security (nmap, dll) menunggu daftar resmi dari senior lab. TUI **belum** dipasang. |

> **Catatan capability tools network/security:** LXD unprivileged container (default) umumnya tetap bisa menjalankan tools seperti `nmap`, `tcpdump`, `iptables` untuk keperluan di dalam namespace container itu sendiri. Yang tidak didukung secara default (dan sengaja tidak dilonggarkan) adalah operasi yang butuh akses ke luar namespace container (capture traffic container lain, modifikasi firewall host) — karena akan melanggar isolasi antar praktikan.

## 2.5 Skema Penamaan & Port SSH

| Ruangan | Prefix Port |
|---|---|
| `f491` | `21` |
| `f492` | `22` |
| `f4111` | `23` |
| `f4112` | `24` |

Nama container: `<ruangan>-<nomor 2 digit>`, contoh `f491-01`, `f491-02`, dst.
Port SSH eksternal: `<prefix><nomor>`, contoh `f491-01` → port `2101`.

Port dipetakan lewat **proxy device** LXD:
```bash
lxc config device add <nama-container> proxy22 proxy listen=tcp:0.0.0.0:<port> connect=tcp:127.0.0.1:22
```

## 2.6 Provisioning: Clone Langsung dari Master

Dipilih clone langsung dari master container (bukan publish ke image dulu), karena:
- `pool-lab` berbasis ZFS → `lxc copy` dari container stopped berjalan sebagai **ZFS clone**, cepat & hemat disk.
- Master bisa diupdate kapan saja tanpa perlu re-publish image.
- Sesuai alur kerja: clone di depan sesi, hapus/reset di akhir sesi.

```bash
lxc copy master-<modul> <nama-container> --storage pool-lab --profile praktikum-<modul>
lxc start <nama-container>
lxc config device add <nama-container> proxy22 proxy listen=tcp:0.0.0.0:<port> connect=tcp:127.0.0.1:22
lxc snapshot <nama-container> clean
```

Snapshot `clean` dibuat **segera setelah container running**, sebagai baseline untuk reset (lihat 2.7).

> **PENTING — urutan `lxc config set environment.*` vs `lxc start`:** kalau container butuh env var yang di-inject lewat `lxc config set <container> environment.KEY=value` (dipakai untuk token API, lihat [API Backend](05-api-backend.md) & [Alur End-to-End](06-alur-end-to-end.md)), config itu **HARUS di-set sebelum `lxc start`**. LXD hanya menerapkan `environment.*` ke proses init (PID 1) pada saat container start — kalau di-set setelah container sudah nyala, PID 1 yang sudah jalan duluan tidak pernah membaca ulang config itu. Detail masalah ini ada di [Troubleshooting](08-troubleshooting.md#33-env-var-token-tidak-terbaca-di-sesi-ssh).

## 2.7 Reset & Recovery

Dua level reset, keduanya berbasis **ZFS snapshot restore**:

```bash
# Reset 1 praktikan/container tertentu
./kelola-lxd.sh reset f491-03

# Reset seluruh container dalam satu ruangan
./kelola-lxd.sh reset-room f491
```

Snapshot `clean` dibuat otomatis sesaat setelah container pertama kali provisioning dan running, sehingga `restore` selalu mengembalikan container ke kondisi persis seperti baru selesai di-clone dari master.

**Kenapa snapshot restore, bukan delete+reclone?** Snapshot restore lebih cepat dan **proxy device tetap nempel** (tidak perlu setup ulang port SSH), karena container-nya tidak dihapus, hanya di-restore state-nya.

## 2.8 Script `kelola-lxd.sh`

Script bash untuk mengelola siklus hidup container per ruangan. Lihat isi lengkap script dan penjelasan tiap bagian di [Panduan Operasional](07-panduan-operasional.md#kelola-lxdsh).

Action yang didukung:

| Action | Fungsi |
|---|---|
| `start <ruang> <modul>` | Provisioning container baru untuk 1 ruangan |
| `stop <ruang>` | Hapus semua container di 1 ruangan |
| `reset <nama-container>` | Reset 1 container ke snapshot `clean` |
| `reset-room <ruang>` | Reset semua container di 1 ruangan |

> Script ini saat ini juga terintegrasi dengan database (generate token, insert session/environment) untuk keperluan testing TUI↔API. Bagian ini bersifat **sementara** dan akan digantikan endpoint provisioning permanen di Go backend — lihat [Log Perkembangan](09-progress-log.md).