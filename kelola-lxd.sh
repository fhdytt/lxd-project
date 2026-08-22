#!/bin/bash
# ============================================================
# kelola-lxd.sh
#
# CATATAN: Bagian integrasi database di script ini (generate
# token, insert session/environment) bersifat SEMENTARA untuk
# keperluan testing TUI <-> API end-to-end. Nantinya seluruh
# logic ini akan dipindah & ditulis ulang sebagai endpoint
# provisioning di Go backend (praktikum-api), yang bisa dipanggil
# langsung dari web dashboard. Bagian yang ditandai
# "--- DB INTEGRATION (SEMENTARA) ---" adalah yang akan dibuang.
# ============================================================

usage() {
    echo "Penggunaan: $0 [start|stop|reset|reset-room] [ruang] [kursus/nama-container]"
    echo "Daftar Ruang: f491, f492, f4111, f4112"
    echo "Daftar Kursus: netbegin, netadmin"
    echo ""
    echo "Contoh mulai        : $0 start f491 netbegin"
    echo "Contoh selesai      : $0 stop f491"
    echo "Contoh reset 1 user : $0 reset f491-03"
    echo "Contoh reset ruangan: $0 reset-room f491"
    exit 1
}

# --- DB INTEGRATION (SEMENTARA): konfigurasi koneksi database ---
DB_USER="lxd_admin"
DB_NAME="lxd_db"
DB_HOST="localhost"
API_URL="http://10.184.56.1:8080"   # alamat Go backend dari sudut pandang container (lewat lxdbr0)

if [ -z "$PGPASSWORD" ]; then
    echo "Error: environment variable PGPASSWORD belum di-set."
    echo "Jalankan dulu: export PGPASSWORD='password-postgres-kamu'"
    exit 1
fi

psql_query() {
    # head -n1 penting: psql -tAc kadang tetap menyertakan baris status
    # (misal "INSERT 0 1") setelah nilai hasil query, terutama untuk
    # INSERT ... RETURNING. Ambil baris pertama saja supaya variabel
    # yang menampung hasilnya (misal SESSION_ID) benar-benar bersih.
    psql -U "$DB_USER" -d "$DB_NAME" -h "$DB_HOST" -tAc "$1" | head -n 1
}
# --- END DB INTEGRATION CONFIG ---

ACTION=$1
RUANG=$2
KURSUS=$3

if [ -z "$ACTION" ]; then
    usage
fi

case "$RUANG" in
    "f491"|"f491"*)   PORT_PREFIX="21" ;;
    "f492"|"f492"*)   PORT_PREFIX="22" ;;
    "f4111"|"f4111"*) PORT_PREFIX="23" ;;
    "f4112"|"f4112"*) PORT_PREFIX="24" ;;
    *)
        echo "Error: Ruangan tidak terdaftar! Pilih antara: f491, f492, f4111, atau f4112."
        exit 1
        ;;
esac

JUMLAH_PRAKTIKAN=3

# START (MEMBUAT DAN MENYALAKAN KONTAINER)
if [ "$ACTION" == "start" ]; then
    if [ -z "$KURSUS" ]; then
        echo "Error: Tentukan kursus saat start."
        exit 1
    fi

    MASTER_CONTAINER="master-$KURSUS"
    PROFILE="praktikum-$KURSUS"

    if ! lxc info "$MASTER_CONTAINER" > /dev/null 2>&1; then
        echo "Error: Kontainer master '$MASTER_CONTAINER' tidak ditemukan!"
        exit 1
    fi

    if ! lxc profile show "$PROFILE" > /dev/null 2>&1; then
        echo "Error: Profile '$PROFILE' belum dibuat!"
        exit 1
    fi

    # --- DB INTEGRATION (SEMENTARA): cari module_id & room_id, buat sesi baru ---
    MODULE_ID=$(psql_query "SELECT id FROM modules WHERE code='$KURSUS'")
    if [ -z "$MODULE_ID" ]; then
        echo "Error: modul '$KURSUS' tidak ditemukan di database (tabel modules)."
        exit 1
    fi

    ROOM_ID=$(psql_query "SELECT id FROM rooms WHERE nama='$RUANG'")
    if [ -z "$ROOM_ID" ]; then
        echo "Error: ruangan '$RUANG' tidak ditemukan di database (tabel rooms)."
        exit 1
    fi

    # course_code asli nanti datang dari staff. Untuk testing, dibuat otomatis.
    COURSE_CODE="TEST-${KURSUS}"
    LAST_MEETING=$(psql_query "SELECT COALESCE(MAX(meeting_number), 0) FROM sessions WHERE course_code='$COURSE_CODE'")
    MEETING_NUMBER=$((LAST_MEETING + 1))

    SESSION_ID=$(psql_query "INSERT INTO sessions (course_code, module_id, room_id, meeting_number, session_date, status)
        VALUES ('$COURSE_CODE', '$MODULE_ID', '$ROOM_ID', $MEETING_NUMBER, CURRENT_DATE, 'active')
        RETURNING id")

    echo "Sesi testing dibuat: $COURSE_CODE pertemuan ke-$MEETING_NUMBER (session_id: $SESSION_ID)"
    # --- END DB INTEGRATION ---

    echo "Memulai Pembuatan Kontainer untuk Ruang $RUANG ($KURSUS)"
    for i in $(seq -f "%02g" 1 $JUMLAH_PRAKTIKAN); do
        NAME="${RUANG}-$i"
        PORT="${PORT_PREFIX}${i}"
        SLOT=$((10#$i))   # paksa basis desimal, hindari salah baca sebagai oktal

        if lxc info "$NAME" > /dev/null 2>&1; then
            echo "Lewati $NAME, sudah ada. Jalankan 'stop' dulu kalau mau ulang."
            continue
        fi

        echo "Membuat $NAME (Port: $PORT, Profile: $PROFILE)..."
        lxc copy "$MASTER_CONTAINER" "$NAME" --storage pool-lab --profile "$PROFILE"

        # --- DB INTEGRATION (SEMENTARA): generate token, simpan, inject ke container ---
        # PENTING: env var HARUS di-set SEBELUM "lxc start", karena LXD hanya
        # menerapkan config environment.* ke proses init (PID 1) pada saat
        # container start. Kalau di-set setelah container sudah nyala, PID 1
        # yang sudah jalan duluan tidak pernah membaca ulang config itu —
        # akibatnya "lxc exec ... env" tetap kelihatan (baca config LXD
        # langsung), tapi proses lain seperti sshd/TUI yang mewarisi
        # environment dari PID 1 tidak akan pernah melihatnya.
        TOKEN=$(openssl rand -hex 32)
        TOKEN_HASH=$(echo -n "$TOKEN" | sha256sum | awk '{print $1}')

        lxc config set "$NAME" environment.PRAKTIKUM_API_URL "$API_URL"
        lxc config set "$NAME" environment.PRAKTIKUM_API_TOKEN "$TOKEN"

        psql -U "$DB_USER" -d "$DB_NAME" -h "$DB_HOST" -v ON_ERROR_STOP=1 -q -c "
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
        "
        if [ $? -eq 0 ]; then
            echo "  -> Token API dibuat & terdaftar untuk $NAME"
        else
            echo "  -> GAGAL menyimpan token untuk $NAME ke database, cek error di atas!"
        fi
        # --- END DB INTEGRATION ---

        lxc start "$NAME"
        lxc config device add "$NAME" proxy22 proxy listen=tcp:0.0.0.0:$PORT connect=tcp:127.0.0.1:22

        # Tunggu sampai container benar-benar running sebelum snapshot
        for attempt in $(seq 1 10); do
            STATE=$(lxc info "$NAME" | grep "^Status:" | awk '{print $2}')
            if [ "$STATE" == "Running" ]; then
                break
            fi
            sleep 1
        done

        lxc snapshot "$NAME" clean
        echo "  -> Snapshot 'clean' dibuat untuk $NAME"

        # --- DB INTEGRATION (SEMENTARA): tandai snapshot sudah ada ---
        psql_query "UPDATE environments SET has_clean_snapshot = true WHERE container_name = '$NAME'" > /dev/null
        # --- END DB INTEGRATION ---
    done
    echo "Proses Selesai. $JUMLAH_PRAKTIKAN Kontainer dibuat untuk Ruang $RUANG"

# STOP (MENGHAPUS KONTAINER & MEMBERSIHKAN DISK)
elif [ "$ACTION" == "stop" ]; then
    echo "Membersihkan Kontainer di Ruang $RUANG"
    for i in $(seq -f "%02g" 1 $JUMLAH_PRAKTIKAN); do
        NAME="${RUANG}-$i"
        if lxc info "$NAME" > /dev/null 2>&1; then
            echo "Menghapus $NAME..."
            lxc stop "$NAME" --force
            lxc delete "$NAME"

            # --- DB INTEGRATION (SEMENTARA): hapus row environment ---
            # Perlu dihapus karena container_name UNIQUE di database,
            # supaya nama yang sama bisa dipakai lagi di provisioning berikutnya.
            psql_query "DELETE FROM environments WHERE container_name = '$NAME'" > /dev/null
            # --- END DB INTEGRATION ---
        fi
    done
    echo "Proses Telah Selesai, Ruang $RUANG telah berhasil dihapus"

# RESET 1 CONTAINER (PER PRAKTIKAN)
elif [ "$ACTION" == "reset" ]; then
    NAME="$RUANG"   # parameter kedua diisi nama container penuh, misal f491-03

    if [ -z "$NAME" ]; then
        echo "Error: Tentukan nama container yang mau direset, contoh: $0 reset f491-03"
        exit 1
    fi

    if ! lxc info "$NAME" > /dev/null 2>&1; then
        echo "Error: Container '$NAME' tidak ditemukan!"
        exit 1
    fi

    if ! lxc info "$NAME" | grep -q "clean"; then
        echo "Error: Snapshot 'clean' tidak ditemukan di '$NAME'. Container ini mungkin dibuat sebelum fitur snapshot ditambahkan."
        exit 1
    fi

    echo "Mereset $NAME ke snapshot 'clean'..."
    lxc restore "$NAME" clean
    echo "Selesai. $NAME sudah kembali ke kondisi awal."

# RESET SATU RUANGAN PENUH
elif [ "$ACTION" == "reset-room" ]; then
    echo "Mereset seluruh kontainer di Ruang $RUANG ke snapshot 'clean'"
    for i in $(seq -f "%02g" 1 $JUMLAH_PRAKTIKAN); do
        NAME="${RUANG}-$i"
        if lxc info "$NAME" > /dev/null 2>&1; then
            if lxc info "$NAME" | grep -q "clean"; then
                echo "Mereset $NAME..."
                lxc restore "$NAME" clean
            else
                echo "Lewati $NAME, tidak ada snapshot 'clean'."
            fi
        else
            echo "Lewati $NAME, container tidak ditemukan."
        fi
    done
    echo "Proses reset ruangan $RUANG selesai."

else
    usage
fi