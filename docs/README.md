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
Would you like to use LXD clustering? (yes/no) [default=no]: no
Do you want to configure a new storage pool? (yes/no) [default=yes]: yes
Name of the new storage pool [default=default]: pool-lxd
Name of the storage backend to use (ceph, dir, lvm, powerflex, powerstore, pure, btrfs, alletra, zfs) [default=zfs]: zfs
Create a new ZFS pool? (yes/no) [default=yes]: yes
Would you like to use an existing empty block device (e.g. a disk or partition)? (yes/no) [default=no]: no
Size in GiB of the new loop device (1GiB minimum) [default=7GiB]: 5GiB
Warning: Creating a bridge enables IPv4 and IPv6 forwarding on the host, which affects all interfaces.
Would you like to create a new local network bridge? (yes/no) [default=yes]: yes
What should the new bridge be called? [default=lxdbr0]: 
What IPv4 address should be used? (CIDR subnet notation, “auto” or “none”) [default=auto]: 
What IPv6 address should be used? (CIDR subnet notation, “auto” or “none”) [default=auto]:   
Would you like the LXD server to be available over the network? (yes/no) [default=no]: no
Would you like stale cached images to be updated automatically? (yes/no) [default=yes]: yes
Would you like a YAML "lxd init" preseed to be printed? (yes/no) [default=no]: no

## Cek Storage LXD
lxc storage list

## Cek Network LXD
lxc network list
```

## Profile LXD

```bash
lxc profile create nama-profile
lxc profile device add nama profile root disk path=/ pool=pool-lab
lxc profile device add nama-profile eth0 nic network=lxdbr0
lxc profile set nama-profile limits.cpu.allowance 40%
lxc profile set nama-profile limits.memory 512MB
```

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

## Master Container

```bash
lxc launch ubuntu:24.04 master-container --profile nama-profile
lxc start master-container
lxc exec master-container -- bash
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
lxc exec master-container -- passwd root
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

lxc start master-container 
lxc file push lxd-tui master-container/usr/local/bin/lxd-tui
lxc exec master-container -- chmod +x /usr/local/bin/lxd-tui

lxc exec master-container -- bash
```

Di dalam container:
```bash
echo "Match User *" >> /etc/ssh/sshd_config
echo "ForceCommand /usr/local/bin/nama-tui" >> /etc/ssh/sshd_config
sshd -t
systemctl restart ssh
exit
```

```bash
lxc stop master-container
```

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