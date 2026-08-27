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