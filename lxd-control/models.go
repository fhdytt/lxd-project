package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
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

	// --- Kelola Ruangan (CRUD) ---
	screenRoomsMenu
	screenRoomsList
	screenRoomFormNama
	screenRoomFormPortPrefix
	screenRoomFormCapacity
	screenRoomPickForEdit
	screenRoomPickForDelete
	screenRoomDeleteConfirm

	// --- Kelola Sesi (CRUD) ---
	screenSessionsMenu
	screenSessionsList
	screenSessionPickRoom
	screenSessionPickModule
	screenSessionFormCourseCode
	screenSessionFormMeetingNumber
	screenSessionFormDate
	screenSessionPickStatus
	screenSessionPickForEdit
	screenSessionPickForDelete
	screenSessionDeleteConfirm
)

// Daftar menu tetap, dipakai ulang di beberapa tempat supaya tidak perlu
// menuliskan literalnya berkali-kali (dan menghindari salah ketik/tidak
// sinkron antar tempat).
var mainMenuItems = []string{
	"Lihat Daftar Environment",
	"Provisioning Ruangan",
	"Reset Environment",
	"Kelola Ruangan",
	"Kelola Sesi",
	"Keluar",
}

var roomsMenuItems = []string{
	"Lihat Daftar Ruangan",
	"Tambah Ruangan",
	"Edit Ruangan",
	"Hapus Ruangan",
	"Kembali ke Menu Utama",
}

var sessionsMenuItems = []string{
	"Lihat Daftar Sesi",
	"Tambah Sesi",
	"Edit Sesi",
	"Hapus Sesi",
	"Kembali ke Menu Utama",
}

var sessionStatusOptions = []string{"scheduled", "active", "completed", "cancelled"}

// model adalah state Bubble Tea tunggal untuk seluruh aplikasi lxd-control.
type model struct {
	cfg *Config
	db  *pgxpool.Pool

	state  screenState
	errMsg string
	spin   spinner.Model

	// returnTo menentukan layar mana yang dituju saat pengguna menekan
	// Enter/Esc di layar hasil/daftar (screenCommandResult, screenRoomsList,
	// dst). Method expression (model.backToMainMenu, model.gotoRoomsMenu,
	// dst) dipakai supaya satu field ini bisa mewakili "kembali ke mana
	// saja" tanpa perlu banyak flag boolean terpisah. WAJIB di-set ulang di
	// setiap titik masuk alur baru — jangan mengandalkan nilai lama.
	returnTo func(model) (tea.Model, tea.Cmd)

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

	// --- Kelola Ruangan ---
	roomsDetailed           []RoomDetail
	roomFormMode            string // "create" atau "update"
	roomInputNama           textinput.Model
	roomInputPortPrefix     textinput.Model
	roomInputCapacity       textinput.Model
	editingRoomOriginalNama string

	// --- Kelola Sesi ---
	sessionsDetailed          []SessionDetail
	sessionPickIDs            []string // paralel dengan menuItems saat pick-for-edit/delete
	sessionFormMode           string   // "create" atau "update"
	sessionInputCourseCode    textinput.Model
	sessionInputMeetingNumber textinput.Model
	sessionInputDate          textinput.Model
	sessionRoom               string // dipilih sebelum form, hanya saat create
	sessionModule             string // dipilih sebelum form, hanya saat create
	sessionStatus             string
	editingSessionID          string
}

func initialModel(cfg *Config, db *pgxpool.Pool) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(accent)

	newInput := func(placeholder string) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = 100
		ti.Width = 40
		ti.PromptStyle = lipgloss.NewStyle().Foreground(accent)
		return ti
	}

	return model{
		cfg:       cfg,
		db:        db,
		state:     screenMainMenu,
		menuItems: mainMenuItems,
		spin:      sp,

		roomInputNama:       newInput("Nama ruangan, misal f491"),
		roomInputPortPrefix: newInput("Port prefix 2 digit, misal 21"),
		roomInputCapacity:   newInput("Kapasitas, misal 5"),

		sessionInputCourseCode:    newInput("Contoh : 1CNAR261442K"),
		sessionInputMeetingNumber: newInput("Pertemuan ke-"),
		sessionInputDate:          newInput("YYYY-MM-DD"),
	}
}

// ==================== STYLES ====================

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
type roomsDetailedLoadedMsg struct {
	rooms []RoomDetail
	err   error
}
type sessionsLoadedMsg struct {
	sessions []SessionDetail
	err      error
}
type roomMutatedMsg struct {
	err error
}
type sessionMutatedMsg struct {
	err error
}