# Setup Lengkap

## Clone Repository

```bash
git clone https://github.com/fhdytt/lxd-project.git
cd lxd-project
```

## LXD, Storage & Network

```bash
sudo snap install lxd
sudo lxd init
```

Saat wizard, storage backend **zfs** (bikin pool baru kalau belum ada), network bridge baru (biarkan LXD generate subnet otomatis), trust password boleh di-skip

> Kalau sudah ada instalasi LXD lama silahkan cek dulu `lxc storage list` dan `lxc network list` gunakan kembali pool ZFS dan bridge yang sudah ada, jangan init ulang

## Profile LXD

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

> Ganti `pool-lab`/`lxdbr0` sesuai nama pool/network dan resource sesuaikan spek server 

## Go & PostgreSQL

```bash
# Go
cd /tmp
wget https://go.dev/dl/go1.22.5.linux-amd64.tar.gz  
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version

# PostgreSQL
sudo apt update && sudo apt install -y postgresql postgresql-contrib
sudo systemctl enable --now postgresql

# gcc untuk build lxd-tui supaya verifikasi password Linux pakai cgo
sudo apt install -y build-essential
```

## Database

```bash
sudo -u postgres psql -c "CREATE USER (user) WITH PASSWORD (password);"
sudo -u postgres psql -c "CREATE DATABASE (database_name) OWNER (user);"

cd ~/lxd-project
psql -U (user) -d (database_name) -h localhost -f schema.sql
psql -U (user) -d (database_name) -h localhost -c '\dt'
```

Harus muncul tabel: `modules`, `rooms`, `praktikan`, `sessions`, `environments`, `admins`

## Master Container

Silahkan Ulangi untuk `netbegin` dan `netadmin`:

```bash
lxc launch ubuntu:24.04 master-netbegin --profile praktikum-netbegin
lxc start master-netbegin
lxc exec master-netbegin -- bash
```

Di dalam container:
```bash
apt update && apt upgrade -y
apt install -y openssh-server

# Fix SSH socket activation
systemctl disable ssh.socket
systemctl enable ssh.service
systemctl start ssh.service

# SSH tanpa credential untuk praktikan langsung masuk ke TUI
echo "auth     required pam_permit.so" > /etc/pam.d/sshd
echo "account  required pam_permit.so" >> /etc/pam.d/sshd
echo "session  required pam_permit.so" >> /etc/pam.d/sshd

echo "PermitRootLogin yes" > /etc/ssh/sshd_config.d/99-open-access.conf
echo "PasswordAuthentication no" >> /etc/ssh/sshd_config.d/99-open-access.conf
echo "KbdInteractiveAuthentication yes" >> /etc/ssh/sshd_config.d/99-open-access.conf
echo "AuthenticationMethods keyboard-interactive" >> /etc/ssh/sshd_config.d/99-open-access.conf

# pastikan valid sebelum restart
sshd -t
systemctl restart ssh

exit
```

Set password `root` di dalam container:
```bash
lxc exec master-netbegin -- passwd root
```

## Build & Jalankan `lxd-dev`

```bash
cd ~/lxd-project/lxd-dev
cp .env.example .env
```

Edit `.env`, isi `DATABASE_URL` (encode karakter `@` di password jadi `%40` kalau ada):
```dotenv
PORT=8080
DATABASE_URL=postgres://(user):(password)@localhost:5432/(database_name)
```

```bash
go mod tidy
go build -o lxd-dev ./cmd/api
./lxd-dev
```

Untuk memastikan API berjalan: `curl http://localhost:8080/healthz` harus balas `ok`. Biarkan proses ini tetap jalan

## 0.8 Build & Pasang `lxd-tui`

Di terminal baru:
```bash
cd ~/lxd-project/lxd-tui
go mod tidy    
go build -o lxd-tui .

lxc start master-netbegin 
lxc file push lxd-tui master-netbegin/usr/local/bin/lxd-tui
lxc exec master-netbegin -- chmod +x /usr/local/bin/lxd-tui

lxc exec master-netbegin -- bash
```

Di dalam container:
```bash
echo "Match User *" >> /etc/ssh/sshd_config
echo "ForceCommand /usr/local/bin/praktikum-tui" >> /etc/ssh/sshd_config
sshd -t
systemctl restart ssh
exit
```

```bash
lxc stop master-netbegin
```

Ulangi untuk `master-netadmin`.

## Build & Setup `lxd-control`

```bash
cd ~/lxd-project/lxd-control
cp ../kelola-lxd.sh .
chmod +x kelola-lxd.sh
```

Buat `.env`:
```bash
echo "DATABASE_URL=postgres://(user):(password)@localhost:5432/(database_name)" > .env
```

```bash
go mod tidy
go build -o lxd-control .
```