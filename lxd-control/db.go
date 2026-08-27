package main

import (
	"context"

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