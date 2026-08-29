package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Room struct {
	Nama string
}

type Module struct {
	Code string
	Name string
}

// EnvironmentRow merepresentasikan satu baris untuk ditampilkan di layar
// daftar environment — sudah di-JOIN dengan sessions, modules, praktikan.
type EnvironmentRow struct {
	ContainerName     string
	SSHPort           int
	Status            string
	HasCleanSnapshot  bool
	CourseCode        string
	Module            string
	PraktikanNama     string // kosong kalau belum diisi
	PraktikanNPM      string
}

func ListRooms(ctx context.Context, db *pgxpool.Pool) ([]Room, error) {
	rows, err := db.Query(ctx, "SELECT nama FROM rooms ORDER BY nama")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Room
	for rows.Next() {
		var r Room
		if err := rows.Scan(&r.Nama); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func ListModules(ctx context.Context, db *pgxpool.Pool) ([]Module, error) {
	rows, err := db.Query(ctx, "SELECT code, name FROM modules ORDER BY code")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Module
	for rows.Next() {
		var m Module
		if err := rows.Scan(&m.Code, &m.Name); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// ListEnvironmentsForRoom mengambil semua environment yang SAAT INI ada
// (masih tercatat di database — row-nya dihapus otomatis saat "stop") untuk
// satu ruangan, diurutkan berdasarkan slot.
func ListEnvironmentsForRoom(ctx context.Context, db *pgxpool.Pool, roomNama string) ([]EnvironmentRow, error) {
	const query = `
		SELECT
			e.container_name, e.ssh_port, e.status, e.has_clean_snapshot,
			s.course_code, m.code AS module,
			COALESCE(p.nama, ''), COALESCE(p.npm, '')
		FROM environments e
		JOIN sessions s ON s.id = e.session_id
		JOIN rooms r    ON r.id = s.room_id
		JOIN modules m  ON m.id = s.module_id
		LEFT JOIN praktikan p ON p.id = e.praktikan_id
		WHERE r.nama = $1
		ORDER BY e.slot_number
	`
	rows, err := db.Query(ctx, query, roomNama)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []EnvironmentRow
	for rows.Next() {
		var e EnvironmentRow
		if err := rows.Scan(
			&e.ContainerName, &e.SSHPort, &e.Status, &e.HasCleanSnapshot,
			&e.CourseCode, &e.Module, &e.PraktikanNama, &e.PraktikanNPM,
		); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// ListContainerNamesForRoom dipakai layar "reset 1 container" — cuma butuh
// nama container-nya saja untuk daftar pilihan.
func ListContainerNamesForRoom(ctx context.Context, db *pgxpool.Pool, roomNama string) ([]string, error) {
	const query = `
		SELECT e.container_name
		FROM environments e
		JOIN sessions s ON s.id = e.session_id
		JOIN rooms r    ON r.id = s.room_id
		WHERE r.nama = $1
		ORDER BY e.slot_number
	`
	rows, err := db.Query(ctx, query, roomNama)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		result = append(result, name)
	}
	return result, rows.Err()
}

// ==================== CRUD: ROOMS ====================

type RoomDetail struct {
	Nama       string
	PortPrefix string
	Capacity   int
}

func ListRoomsDetailed(ctx context.Context, db *pgxpool.Pool) ([]RoomDetail, error) {
	rows, err := db.Query(ctx, "SELECT nama, port_prefix, capacity FROM rooms ORDER BY nama")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []RoomDetail
	for rows.Next() {
		var r RoomDetail
		if err := rows.Scan(&r.Nama, &r.PortPrefix, &r.Capacity); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func CreateRoom(ctx context.Context, db *pgxpool.Pool, nama, portPrefix string, capacity int) error {
	_, err := db.Exec(ctx,
		"INSERT INTO rooms (nama, port_prefix, capacity) VALUES ($1, $2, $3)",
		nama, portPrefix, capacity,
	)
	return err
}

// UpdateRoom mencari row berdasarkan nama LAMA (originalNama), lalu
// menimpanya dengan nilai baru — termasuk kemungkinan nama itu sendiri berubah.
func UpdateRoom(ctx context.Context, db *pgxpool.Pool, originalNama, newNama, portPrefix string, capacity int) error {
	_, err := db.Exec(ctx,
		"UPDATE rooms SET nama = $1, port_prefix = $2, capacity = $3 WHERE nama = $4",
		newNama, portPrefix, capacity, originalNama,
	)
	return err
}

// DeleteRoom akan GAGAL (bukan silent success) kalau masih ada sessions yang
// mereferensikan ruangan ini — constraint FK sessions.room_id memakai
// ON DELETE RESTRICT secara sengaja, supaya ruangan yang masih punya
// riwayat sesi tidak bisa terhapus tanpa sadar.
func DeleteRoom(ctx context.Context, db *pgxpool.Pool, nama string) error {
	_, err := db.Exec(ctx, "DELETE FROM rooms WHERE nama = $1", nama)
	return err
}

// ==================== CRUD: SESSIONS ====================

type SessionDetail struct {
	ID            string
	CourseCode    string
	RoomNama      string
	ModuleCode    string
	MeetingNumber int
	SessionDate   string
	Status        string
}

func ListSessions(ctx context.Context, db *pgxpool.Pool) ([]SessionDetail, error) {
	const query = `
		SELECT s.id, s.course_code, r.nama, m.code, s.meeting_number, s.session_date::text, s.status
		FROM sessions s
		JOIN rooms r   ON r.id = s.room_id
		JOIN modules m ON m.id = s.module_id
		ORDER BY s.session_date DESC, s.created_at DESC
	`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SessionDetail
	for rows.Next() {
		var s SessionDetail
		if err := rows.Scan(&s.ID, &s.CourseCode, &s.RoomNama, &s.ModuleCode, &s.MeetingNumber, &s.SessionDate, &s.Status); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func getRoomIDByNama(ctx context.Context, db *pgxpool.Pool, nama string) (string, error) {
	var id string
	err := db.QueryRow(ctx, "SELECT id FROM rooms WHERE nama = $1", nama).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("ruangan %q tidak ditemukan: %w", nama, err)
	}
	return id, nil
}

func getModuleIDByCode(ctx context.Context, db *pgxpool.Pool, code string) (string, error) {
	var id string
	err := db.QueryRow(ctx, "SELECT id FROM modules WHERE code = $1", code).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("modul %q tidak ditemukan: %w", code, err)
	}
	return id, nil
}

// CreateSession sengaja menerima nama ruangan & kode modul (bukan UUID
// langsung) supaya pemanggilnya (update.go) tidak perlu tahu detail UUID
// internal — sesuai bagaimana admin berinteraksi lewat menu (pilih nama,
// bukan ID).
func CreateSession(ctx context.Context, db *pgxpool.Pool, courseCode, roomNama, moduleCode string, meetingNumber int, sessionDate, status string) error {
	roomID, err := getRoomIDByNama(ctx, db, roomNama)
	if err != nil {
		return err
	}
	moduleID, err := getModuleIDByCode(ctx, db, moduleCode)
	if err != nil {
		return err
	}

	_, err = db.Exec(ctx, `
		INSERT INTO sessions (course_code, module_id, room_id, meeting_number, session_date, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, courseCode, moduleID, roomID, meetingNumber, sessionDate, status)
	return err
}

// UpdateSession SENGAJA tidak mengizinkan mengubah ruangan/modul sesi yang
// sudah ada — environment yang sudah diprovisioning terikat ke kombinasi
// ruangan+modul saat sesi itu dibuat, mengubahnya belakangan berisiko bikin
// data tidak konsisten. Yang bisa diubah: course_code, nomor pertemuan,
// tanggal, dan status.
func UpdateSession(ctx context.Context, db *pgxpool.Pool, id, courseCode string, meetingNumber int, sessionDate, status string) error {
	_, err := db.Exec(ctx, `
		UPDATE sessions SET course_code = $1, meeting_number = $2, session_date = $3, status = $4
		WHERE id = $5
	`, courseCode, meetingNumber, sessionDate, status, id)
	return err
}

// DeleteSession akan ikut MENGHAPUS semua baris di tabel environments yang
// terhubung ke sesi ini (ON DELETE CASCADE di skema). Container LXD-nya
// sendiri TIDAK ikut terhapus otomatis — jadi row yang hilang ini bisa
// membuat container "yatim" (ada di LXD tapi tidak lagi tercatat di
// database). Peringatan ini WAJIB ditampilkan ke admin sebelum konfirmasi,
// lihat breadcrumb() di view.go.
func DeleteSession(ctx context.Context, db *pgxpool.Pool, id string) error {
	_, err := db.Exec(ctx, "DELETE FROM sessions WHERE id = $1", id)
	return err
}

// ==================== PROVISIONING (dipakai oleh commands.go) ====================

// ModuleDetail mirip Module, tapi ikut membawa master_container & lxd_profile
// — informasi yang dibutuhkan untuk benar-benar menjalankan provisioning
// (bukan cuma ditampilkan di menu).
type ModuleDetail struct {
	Code            string
	Name            string
	MasterContainer string
	LxdProfile      string
}

func GetModuleByCode(ctx context.Context, db *pgxpool.Pool, code string) (*ModuleDetail, error) {
	var m ModuleDetail
	err := db.QueryRow(ctx,
		"SELECT code, name, master_container, lxd_profile FROM modules WHERE code = $1", code,
	).Scan(&m.Code, &m.Name, &m.MasterContainer, &m.LxdProfile)
	if err != nil {
		return nil, fmt.Errorf("modul %q tidak ditemukan: %w", code, err)
	}
	return &m, nil
}

func GetRoomByNama(ctx context.Context, db *pgxpool.Pool, nama string) (*RoomDetail, error) {
	var r RoomDetail
	err := db.QueryRow(ctx,
		"SELECT nama, port_prefix, capacity FROM rooms WHERE nama = $1", nama,
	).Scan(&r.Nama, &r.PortPrefix, &r.Capacity)
	if err != nil {
		return nil, fmt.Errorf("ruangan %q tidak ditemukan: %w", nama, err)
	}
	return &r, nil
}

// ListSessionsForProvision mencari sesi yang LAYAK dipakai untuk
// provisioning (status scheduled/active) untuk kombinasi ruangan+modul
// tertentu — dipakai admin untuk "menempel" provisioning ke sesi yang sudah
// dibuat lewat menu Kelola Sesi, bukan bikin sesi baru asal-asalan.
func ListSessionsForProvision(ctx context.Context, db *pgxpool.Pool, roomNama, moduleCode string) ([]SessionDetail, error) {
	const query = `
		SELECT s.id, s.course_code, r.nama, m.code, s.meeting_number, s.session_date::text, s.status
		FROM sessions s
		JOIN rooms r   ON r.id = s.room_id
		JOIN modules m ON m.id = s.module_id
		WHERE r.nama = $1 AND m.code = $2 AND s.status IN ('scheduled', 'active')
		ORDER BY (s.status = 'active') DESC, s.session_date DESC
	`
	rows, err := db.Query(ctx, query, roomNama, moduleCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SessionDetail
	for rows.Next() {
		var s SessionDetail
		if err := rows.Scan(&s.ID, &s.CourseCode, &s.RoomNama, &s.ModuleCode, &s.MeetingNumber, &s.SessionDate, &s.Status); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// InsertEnvironment upsert (aman dipanggil ulang tanpa "stop" dulu, sama
// seperti perilaku kelola-lxd.sh versi lama).
func InsertEnvironment(ctx context.Context, db *pgxpool.Pool, sessionID, containerName string, slot, port int, tokenHash string) error {
	_, err := db.Exec(ctx, `
		INSERT INTO environments (session_id, container_name, slot_number, ssh_port, status, api_token_hash)
		VALUES ($1, $2, $3, $4, 'running', $5)
		ON CONFLICT (container_name) DO UPDATE SET
			session_id = EXCLUDED.session_id,
			slot_number = EXCLUDED.slot_number,
			ssh_port = EXCLUDED.ssh_port,
			status = EXCLUDED.status,
			api_token_hash = EXCLUDED.api_token_hash,
			praktikan_id = NULL,
			identified_at = NULL,
			has_clean_snapshot = false
	`, sessionID, containerName, slot, port, tokenHash)
	return err
}

func MarkSnapshotReady(ctx context.Context, db *pgxpool.Pool, containerName string) error {
	_, err := db.Exec(ctx, "UPDATE environments SET has_clean_snapshot = true WHERE container_name = $1", containerName)
	return err
}

// UnlinkPraktikan melepas kaitan environment ini dari praktikan yang
// sebelumnya mengidentifikasi diri di sini. WAJIB dipanggil setiap kali
// environment di-reset (lihat resetContainerCmd/resetRoomCmd di
// commands.go) — kalau tidak, environment ini akan TERKUNCI SELAMANYA ke
// identitas praktikan lama walau isi container-nya sudah bersih, karena
// verifikasi identitas (lihat API § 5.4a) membandingkan ke row praktikan
// yang tercatat di sini.
func UnlinkPraktikan(ctx context.Context, db *pgxpool.Pool, containerName string) error {
	_, err := db.Exec(ctx,
		"UPDATE environments SET praktikan_id = NULL, identified_at = NULL WHERE container_name = $1",
		containerName,
	)
	return err
}

func DeleteEnvironmentByContainerName(ctx context.Context, db *pgxpool.Pool, containerName string) error {
	_, err := db.Exec(ctx, "DELETE FROM environments WHERE container_name = $1", containerName)
	return err
}