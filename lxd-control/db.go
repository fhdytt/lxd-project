package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Room struct {
	Nama string
}

type Module struct {
	Code string
	Name string
}

type EnvironmentRow struct {
	ContainerName     string
	SSHPort           int
	Status            string
	HasCleanSnapshot  bool
	CourseCode        string
	Module            string
	PraktikanNama     string
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

func UpdateRoom(ctx context.Context, db *pgxpool.Pool, originalNama, newNama, portPrefix string, capacity int) error {
	_, err := db.Exec(ctx,
		"UPDATE rooms SET nama = $1, port_prefix = $2, capacity = $3 WHERE nama = $4",
		newNama, portPrefix, capacity, originalNama,
	)
	return err
}

func DeleteRoom(ctx context.Context, db *pgxpool.Pool, nama string) error {
	_, err := db.Exec(ctx, "DELETE FROM rooms WHERE nama = $1", nama)
	return err
}

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

func UpdateSession(ctx context.Context, db *pgxpool.Pool, id, courseCode string, meetingNumber int, sessionDate, status string) error {
	_, err := db.Exec(ctx, `
		UPDATE sessions SET course_code = $1, meeting_number = $2, session_date = $3, status = $4
		WHERE id = $5
	`, courseCode, meetingNumber, sessionDate, status, id)
	return err
}

func DeleteSession(ctx context.Context, db *pgxpool.Pool, id string) error {
	_, err := db.Exec(ctx, "DELETE FROM sessions WHERE id = $1", id)
	return err
}

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

func ListSessionsForProvision(ctx context.Context, db *pgxpool.Pool, roomNama, moduleCode string) ([]SessionDetail, error) {
	const query = `
		SELECT s.id, s.course_code, r.nama, m.code, s.meeting_number, s.session_date::text, s.status
		FROM sessions s
		JOIN rooms r   ON r.id = s.room_id
		JOIN modules m ON m.id = s.module_id
		WHERE r.nama = $1 AND m.code = $2 AND s.status IN ('scheduled', 'active')
		ORDER BY ABS(s.session_date - CURRENT_DATE) ASC, (s.status = 'active') DESC
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

func UnlinkPraktikan(ctx context.Context, db *pgxpool.Pool, containerName string) error {
	_, err := db.Exec(ctx,
		"UPDATE environments SET praktikan_id = NULL, identified_at = NULL WHERE container_name = $1",
		containerName,
	)
	return err
}

func GetRoomCurrentModule(ctx context.Context, db *pgxpool.Pool, roomNama string) (string, error) {
	const query = `
		SELECT m.code
		FROM environments e
		JOIN sessions s ON s.id = e.session_id
		JOIN rooms r    ON r.id = s.room_id
		JOIN modules m  ON m.id = s.module_id
		WHERE r.nama = $1
		LIMIT 1
	`
	var code string
	err := db.QueryRow(ctx, query, roomNama).Scan(&code)
	if err != nil {
		return "", fmt.Errorf("belum ada container di ruangan %q — provisioning dulu lewat 'Provisioning Ruangan > Start': %w", roomNama, err)
	}
	return code, nil
}

func RepointSession(ctx context.Context, db *pgxpool.Pool, containerName, newSessionID string) error {
	_, err := db.Exec(ctx, "UPDATE environments SET session_id = $1 WHERE container_name = $2", newSessionID, containerName)
	return err
}

func DeleteEnvironmentByContainerName(ctx context.Context, db *pgxpool.Pool, containerName string) error {
	_, err := db.Exec(ctx, "DELETE FROM environments WHERE container_name = $1", containerName)
	return err
}

func CreateSessionsBulk(ctx context.Context, db *pgxpool.Pool, courseCode, roomNama, moduleCode string, startMeetingNumber, meetingCount, intervalDays int, startDate string) error {
	roomID, err := getRoomIDByNama(ctx, db, roomNama)
	if err != nil {
		return err
	}
	moduleID, err := getModuleIDByCode(ctx, db, moduleCode)
	if err != nil {
		return err
	}

	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("format tanggal salah, gunakan YYYY-MM-DD: %w", err)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for i := 0; i < meetingCount; i++ {
		meetingNumber := startMeetingNumber + i
		meetingDate := start.AddDate(0, 0, i*intervalDays)

		_, err := tx.Exec(ctx, `
			INSERT INTO sessions (course_code, module_id, room_id, meeting_number, session_date, status)
			VALUES ($1, $2, $3, $4, $5, 'scheduled')
		`, courseCode, moduleID, roomID, meetingNumber, meetingDate.Format("2006-01-02"))
		if err != nil {
			return fmt.Errorf("gagal insert pertemuan ke-%d: %w", meetingNumber, err)
		}
	}

	return tx.Commit(ctx)
}
