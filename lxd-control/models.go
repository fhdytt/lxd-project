package main

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ==================== STATE ====================

type screenState int

const (
	screenMainMenu screenState = iota
	screenPickRoomForList
	screenListEnvironments
	screenPickRoomForProvision
	screenPickProvisionAction
	screenPickModuleForStart
	screenConfirmProvision
	screenRunningCommand
	screenCommandResult
	screenPickRoomForReset
	screenPickResetMode
	screenPickContainerForReset
	screenConfirmReset
	screenError
)

// model adalah state Bubble Tea tunggal untuk seluruh aplikasi lxd-control.
type model struct {
	cfg *Config
	db  *pgxpool.Pool

	state  screenState
	errMsg string
	spin   spinner.Model

	// Menu generik: dipakai ulang di semua layar berbasis daftar pilihan.
	menuItems  []string
	menuCursor int

	rooms   []Room
	modules []Module
	envRows []EnvironmentRow

	selectedRoom      string
	selectedModule    string
	selectedAction    string // "start" atau "stop"
	selectedResetMode string // "room" atau "container"
	selectedContainer string

	commandOutput string
	commandFailed bool
}

func initialModel(cfg *Config, db *pgxpool.Pool) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(accent)

	return model{
		cfg:       cfg,
		db:        db,
		state:     screenMainMenu,
		menuItems: []string{"Lihat Daftar Environment", "Provisioning Ruangan", "Reset Environment", "Keluar"},
		spin:      sp,
	}
}

// Init (bagian dari interface tea.Model) ada di commands.go, karena dia
// men-trigger command async pertama kali (spinner tick).

var (
	accent      = lipgloss.Color("39")
	danger      = lipgloss.Color("203")
	success     = lipgloss.Color("42")
	muted       = lipgloss.Color("241")
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(accent).Padding(0, 1)
	boxStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(1, 3)
	hintStyle   = lipgloss.NewStyle().Foreground(muted).Italic(true)
	errStyle    = lipgloss.NewStyle().Foreground(danger).Bold(true)
	okStyle     = lipgloss.NewStyle().Foreground(success).Bold(true)
	cursorStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	labelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

// ==================== MESSAGES ====================
// Tipe pesan async yang dikirim balik ke Update() setelah operasi
// database/subprocess selesai — didefinisikan di sini karena erat kaitannya
// dengan bentuk data model, implementasinya (fungsi generator) ada di commands.go.

type roomsLoadedMsg struct {
	rooms []Room
	err   error
}
type modulesLoadedMsg struct {
	modules []Module
	err     error
}
type environmentsLoadedMsg struct {
	rows []EnvironmentRow
	err  error
}
type containersLoadedMsg struct {
	names []string
	err   error
}
type commandDoneMsg struct {
	output string
	err    error
}