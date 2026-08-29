package main

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Init (bagian dari interface tea.Model) — men-trigger command async pertama
// kali dijalankan: cuma spinner tick, karena layar awal (Menu Utama) tidak
// butuh data apapun dari database.
func (m model) Init() tea.Cmd {
	return m.spin.Tick
}

// ==================== COMMANDS (operasi async) ====================
// Semua fungsi di sini mengembalikan tea.Cmd: closure yang dijalankan
// Bubble Tea di goroutine terpisah, hasilnya dikirim balik sebagai tea.Msg
// ke Update() (lihat update.go).

func loadRoomsCmd(db *pgxpool.Pool) tea.Cmd {
	return func() tea.Msg {
		rooms, err := ListRooms(context.Background(), db)
		return roomsLoadedMsg{rooms: rooms, err: err}
	}
}

func loadModulesCmd(db *pgxpool.Pool) tea.Cmd {
	return func() tea.Msg {
		modules, err := ListModules(context.Background(), db)
		return modulesLoadedMsg{modules: modules, err: err}
	}
}

func loadEnvironmentsCmd(db *pgxpool.Pool, room string) tea.Cmd {
	return func() tea.Msg {
		rows, err := ListEnvironmentsForRoom(context.Background(), db, room)
		return environmentsLoadedMsg{rows: rows, err: err}
	}
}

func loadContainersCmd(db *pgxpool.Pool, room string) tea.Cmd {
	return func() tea.Msg {
		names, err := ListContainerNamesForRoom(context.Background(), db, room)
		return containersLoadedMsg{names: names, err: err}
	}
}

// runCommandCmd menjalankan kelola-lxd.sh (lewat actions.go) sebagai
// subprocess. Dijalankan blocking di dalam goroutine command Bubble Tea,
// jadi UI tetap responsif (spinner tetap berputar) selama proses berjalan.
func runCommandCmd(cfg *Config, args ...string) tea.Cmd {
	return func() tea.Msg {
		out, err := RunKelolaScript(cfg, args...)
		return commandDoneMsg{output: out, err: err}
	}
}

// --- Kelola Ruangan ---

func loadRoomsDetailedCmd(db *pgxpool.Pool) tea.Cmd {
	return func() tea.Msg {
		rooms, err := ListRoomsDetailed(context.Background(), db)
		return roomsDetailedLoadedMsg{rooms: rooms, err: err}
	}
}

func createRoomCmd(db *pgxpool.Pool, nama, portPrefix string, capacity int) tea.Cmd {
	return func() tea.Msg {
		err := CreateRoom(context.Background(), db, nama, portPrefix, capacity)
		return roomMutatedMsg{err: err}
	}
}

func updateRoomCmd(db *pgxpool.Pool, originalNama, nama, portPrefix string, capacity int) tea.Cmd {
	return func() tea.Msg {
		err := UpdateRoom(context.Background(), db, originalNama, nama, portPrefix, capacity)
		return roomMutatedMsg{err: err}
	}
}

func deleteRoomCmd(db *pgxpool.Pool, nama string) tea.Cmd {
	return func() tea.Msg {
		err := DeleteRoom(context.Background(), db, nama)
		return roomMutatedMsg{err: err}
	}
}

// --- Kelola Sesi ---

func loadSessionsCmd(db *pgxpool.Pool) tea.Cmd {
	return func() tea.Msg {
		sessions, err := ListSessions(context.Background(), db)
		return sessionsLoadedMsg{sessions: sessions, err: err}
	}
}

func createSessionCmd(db *pgxpool.Pool, courseCode, room, module string, meetingNumber int, date, status string) tea.Cmd {
	return func() tea.Msg {
		err := CreateSession(context.Background(), db, courseCode, room, module, meetingNumber, date, status)
		return sessionMutatedMsg{err: err}
	}
}

func updateSessionCmd(db *pgxpool.Pool, id, courseCode string, meetingNumber int, date, status string) tea.Cmd {
	return func() tea.Msg {
		err := UpdateSession(context.Background(), db, id, courseCode, meetingNumber, date, status)
		return sessionMutatedMsg{err: err}
	}
}

func deleteSessionCmd(db *pgxpool.Pool, id string) tea.Cmd {
	return func() tea.Msg {
		err := DeleteSession(context.Background(), db, id)
		return sessionMutatedMsg{err: err}
	}
}