package main

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (m model) Init() tea.Cmd {
	return m.spin.Tick
}

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

func runCommandCmd(cfg *Config, args ...string) tea.Cmd {
	return func() tea.Msg {
		out, err := RunKelolaScript(cfg, args...)
		return commandDoneMsg{output: out, err: err}
	}
}