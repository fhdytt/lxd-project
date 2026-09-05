# Infrastruktur LXD

## Storage & Network

| Komponen | Nilai | Catatan |
|---|---|---|
| Storage pool | `pool-lab` | Driver **ZFS** bersifat snapshot/clone berbasis copy-on-write, cepat & hemat disk |
| Network bridge | `lxdbr0` | Subnet `10.184.56.1/24` |
| LXD version | 5.21.6 LTS | |

## Profile LXD

Profile digunkan untuk mengalokasi resource per modul, memakai `limits.cpu.allowance` (CPU sharing) supaya container bisa berbagi core tanpa kelebihan beban

```bash
lxc profile create praktikum-netbegin
lxc profile device add praktikum-netbegin root disk path=/ pool=pool-lab
lxc profile device add praktikum-netbegin eth0 nic network=lxdbr0
lxc profile set praktikum-netbegin limits.cpu.allowance 20%
lxc profile set praktikum-netbegin limits.memory 256MB

lxc profile create praktikum-netadmin
lxc profile device add praktikum-netadmin root disk path=/ pool=pool-lab
lxc profile device add praktikum-netadmin eth0 nic network=lxdbr0
lxc profile set praktikum-netadmin limits.cpu.allowance 40%
lxc profile set praktikum-netadmin limits.memory 512MB
```

## Master Container

Master container adalah tamplate per modul, master ini harus selalu stopped keculai sedang di konfigurasi

### Setup umum

```bash
lxc start master-<modul>
lxc exec master-<modul> -- bash
apt update && apt upgrade -y
apt install -y openssh-server
```

### SSH socket activation

Terdapat sebuah bug do Ubuntu 22.04+/24.04 dimana socket activation untuk sshd, sehingga menyebabkan koneksi SSH pertama kadang gagal
```bash
lxc exec master-<modul> -- systemctl disable ssh.socket
lxc exec master-<modul> -- systemctl enable ssh.service
lxc exec master-<modul> -- systemctl start ssh.service
```

### SSH tanpa credential

Supaya praktikan tidak perlu memasukkan username/password SSH apapun sehingga praktikan langsung masuk ke `lxd-tui`, PAM stack untuk sshd diganti menjadikannya selalu meloloskan koneksi:
```bash
echo "auth     required pam_permit.so" > /etc/pam.d/sshd
echo "account  required pam_permit.so" >> /etc/pam.d/sshd
echo "session  required pam_permit.so" >> /etc/pam.d/sshd
```

Dan di `/etc/ssh/sshd_config.d/99-open-access.conf`:
```
PermitRootLogin yes
PasswordAuthentication no
KbdInteractiveAuthentication yes
AuthenticationMethods keyboard-interactive
```
**Disclaimer** 
wajib validasi sebelum restart dengan `sshd -t` harus tidak ada output sama sekali

### `ForceCommand` supaya TUI otomatis jalan saat SSH login

```bash
echo "Match User *" >> /etc/ssh/sshd_config
echo "ForceCommand /usr/local/bin/lxd-tui" >> /etc/ssh/sshd_config
```

### Selesai konfigurasi

```bash
exit
lxc stop master-<modul>
```

## Skema Penamaan & Port SSH

| Ruangan | Prefix Port |
|---|---|
| `ruang1` | `21` |
| `ruang2` | `22` |
| `ruang3` | `23` |
| `ruang4` | `24` |

Nama container: `<ruangan>-<nomor 2 digit>` (`ruangan-01`)
Port SSH: `<prefix><nomor>` (`2101`)
Dipetakan lewat **proxy device** LXD

### Urutan operasi pembuatan container

```bash
lxc copy "$MASTER_CONTAINER" "$NAME" --storage pool-lab --profile "$PROFILE"

lxc config set "$NAME" environment.PRAKTIKUM_API_URL "$API_URL"
lxc config set "$NAME" environment.PRAKTIKUM_API_TOKEN "$TOKEN"

lxc start "$NAME"
lxc config device add "$NAME" proxy22 proxy listen=tcp:0.0.0.0:"$PORT" connect=tcp:127.0.0.1:22
# ... tunggu Running, lalu buat snapshot...
lxc snapshot "$NAME" clean
```

## Reset Praktikan & Reset Sesi

Pada reset praktikan tetap murni **ZFS snapshot restore** Tapi di level `lxd-control` dengan melepas identitas praktikan, supaya praktikan yang menggunakan environment tersebut tidak tertolak saat verifikasi identitas karena data lama masih ada.

Selain reset praktikan, terdapat juga reset sesi dengan memindahkan environment ke sesi yang berbeda, tanpa harus menghapus atau membuat ulang container. Ini menjadikan terlaksananya berbagai sesi dalam 1 hari yang menggunakan ruangan sama.