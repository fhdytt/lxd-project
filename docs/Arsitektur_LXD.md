# Arsitektur LXD

## Profile
```bash
## Buat Profile
lxc profile create nama-profile

## Menambahkan Rule Profile
lxc profile device add <rule>
## Contoh :
lxc profile device add nama-profile root disk path=/ pool=pool-lab
lxc profile device add nama-profile eth0 nic network=lxdbr0

## Mengedit rule profile
lxc profile edit nama-profile

## Cek Profile
lxc profile show nama-profile
```
## Storage
### dir (Directory)
Cara kerja: Menyimpan seluruh data container dan virtual machine sebagai folder dan file biasa di dalam sistem file host Ubuntu Anda (misalnya di /var/lib/lxd/storage-pools/...).
Kelebihan: Sangat kompatibel dengan semua jenis sistem file Linux (ext4, xfs, dll.) dan tidak memerlukan partisi khusus atau konfigurasi yang rumit.
Kekurangan: Tidak mendukung fitur snapshot instan yang efisien dan tidak ada fitur pembatasan kuota disk bawaan tingkat blok (kecuali menggunakan proyeksi file sistem lain).
Kegunaan: Paling cocok untuk testing, belajar (praktikum ringan), atau server dengan media penyimpanan sederhana di mana fitur snapshot cepat tidak terlalu dibutuhkan.

### btrfs
Cara kerja: Menggunakan sistem file Btrfs sebagai backend, sehingga memanfaatkan fitur bawaan Btrfs seperti subvolumes dan copy-on-write.
Kelebihan: Mendukung pembuatan snapshot dan clone container secara instan dalam hitungan milidetik serta sangat hemat ruang penyimpanan.
Kekurangan: Memerlukan partisi khusus atau formatted disk berformat Btrfs.
Kegunaan: Cocok untuk lingkungan development atau staging lokal di mana Anda sering membuat cadangan (snapshot) atau menduplikasi container dengan cepat.
zfs
Cara kerja: Menggunakan teknologi ZFS (Z File System) yang terkenal handal untuk pengelolaan storage tingkat enterprise.
Kelebihan: Sangat stabil untuk produksi, mendukung snapshot instan, compression data secara on-the-fly (menghemat kapasitas disk), deduplikasi, dan manajemen RAID tingkat lanjut.
Kekurangan: Cukup boros penggunaan RAM di server, dan disk/partisi harus dipersiapkan khusus untuk ZFS.
Kegunaan: Sangat disarankan untuk production server skala menengah hingga besar yang membutuhkan keandalan data tinggi dan manajemen volume tingkat lanjut.

### lvm (Logical Volume Manager)
Cara kerja: Menggunakan sistem LVM tradisional Linux untuk mengalokasikan logical volume sebagai blok penyimpanan bagi container.
Kelebihan: Performa baca/tulis yang stabil dan konsisten berbasis blok, serta mudah diintegrasikan jika infrastruktur server Anda sudah menggunakan LVM.
Kekurangan: Proses snapshot atau resize volume tidak sefleksibel Btrfs atau ZFS.
Kegunaan: Cocok untuk lingkungan server tradisional perusahaan yang standarnya sudah menggunakan LVM untuk partisi disk mereka.

## device
### Disk
Fungsi: Menghubungkan direktori atau partisi dari host (server utama) ke dalam container, atau memasang storage pool tambahan.
Contoh Penggunaan: Memasang folder dari host ke dalam container
```bash
lxc config device add nama-container data-folder disk source=/home/user/shared path=/mnt/shared
```

### proxy
Fungsi: Berfungsi sebagai penerus port (port forwarding). Mengarahkan trafik dari port tertentu di host ke container (atau sebaliknya). Sangat berguna jika container menjalankan web server dan ingin diakses dari luar melalui IP host.
Contoh Penggunaan: Meneruskan port 80 di host ke port 80 di container
```bash
lxc config device add nama-container http-proxy proxy listen=tcp:0.0.0.0:80 connect=tcp:127.0.0.1:80
```
### gpu
Fungsi: Memberikan akses kartu grafis (GPU) fisik dari host ke dalam container. Digunakan untuk keperluan komputasi berat seperti machine learning, rendering, atau transcoding video.

Contoh Penggunaan:
```bash
lxc config device add nama-container my-gpu gpu
```
### usb
Fungsi: Meneruskan perangkat USB fisik yang terhubung ke server host (seperti flashdisk, dongle, atau mikrokontroler) agar bisa dideteksi dan diakses di dalam container.
```bash
lxc config device add nama-container usb-device usb vendorid=1234 productid=5678
```

## Limitasi CPU
LXD menggunakan dua pendekatan utama untuk mengontrol CPU: pembatasan core fisik (pinning/CPU isolation) dan pembatasan kuota waktu (allowance).
Cara Kerja limits.cpu.allowance (Persentase):
Nilai seperti 40% menggunakan mekanisme CFS (Completely Fair Scheduler) Bandwidth di kernel Linux.
- Kernel membagi waktu prosesor menjadi siklus waktu tertentu (biasanya setiap 100 milidetik).
- Dengan setelan 40%, container Anda hanya diizinkan menggunakan jatah waktu CPU selama 40 milidetik dari setiap 100 milidetik. Sisa 60 milidetik berikutnya, proses container akan dipaksa "istirahat" (throttled).
Apa yang terjadi jika melebihi batas?
Container tidak akan crash atau mati. Proses yang berjalan di dalamnya hanya akan berjalan lebih lambat atau mengalami lag (seperti aplikasi yang mendadak patah-patah atau proses compiling yang memakan waktu lebih lama) karena harus antre menunggu jatah waktu CPU berikutnya tersedia
Contoh :
```bash
lxc profile set nama-profile limits.cpu.allowance 40%
```

## Limitasi memory
LXD mengontrol alokasi RAM menggunakan sub-sistem memori pada cgroups kernel Linux.

Cara Kerja limits.memory:
LXD menetapkan batas tegas (hard limit) pada memori fisik yang dapat digunakan oleh seluruh proses di dalam container tersebut.

Apa yang terjadi jika melebihi batas? (Mekanisme OOM Killer):
Berbeda dengan CPU yang hanya membuat lambat, jika penggunaan RAM container menyentuh batas maksimum (misalnya 512MB) dan aplikasi di dalamnya terus meminta memori tambahan:
1. Linux kernel akan langsung bertindak untuk melindungi RAM server utama (host) agar tidak habis total.
2. Kernel akan memicu mekanisme OOM (Out-Of-Memory) Killer.
3. Kernel akan mematikan (kill) proses terbesar atau proses utama yang memicu kelebihan memori di dalam container tersebut (biasanya layanan utama seperti database, web server, atau proses yang sedang berjalan).
4. Akibatnya, container mungkin akan mendadak terputus atau layanan di dalamnya berhenti (exit), sementara container itu sendiri statusnya di LXD tetap menyala (namun aplikasinya mati)
Contoh :
```bash
lxc profile set nama-profile limits.memory 512MB
```

## Edit Profile
Contoh isi dari edit profile
```yaml
name: test-profile
description: "Test Profile lxd"
config:
  limits.cpu.allowance: 50%
  limits.memory: 1GB
devices:
  eth0:
    network: lxdbr0
    type: nic
  root:
    path: /
    pool: default
    type: disk
used_by: []
project: default
```