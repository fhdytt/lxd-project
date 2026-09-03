package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/jackc/pgx/v5/pgxpool"
)

type screenState int

const (
	screenMainMenu screenState = iota
	screenPickRoomForList
	screenListEnvironments
	screenPickRoomForProvision
	screenPickProvisionAction
	screenPickModuleForStart
	screenPickSessionForProvision
	screenConfirmProvision
	screenRunningCommand
	screenCommandResult
	screenPickRoomForReset
	screenPickResetMode
	screenPickContainerForReset
	screenConfirmReset
	screenError

	// Sesi Ruangan
	screenPickRoomForNextMeeting
	screenPickSessionForNextMeeting
	screenConfirmNextMeeting

	// Kelola Ruangan
	screenRoomsMenu
	screenRoomsList
	screenRoomFormNama
	screenRoomFormPortPrefix
	screenRoomFormCapacity
	screenRoomPickForEdit
	screenRoomPickForDelete
	screenRoomDeleteConfirm

	// Kelola Sesi
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

	//  Tambah Sesi
	screenBulkPickRoom
	screenBulkPickModule
	screenBulkFormCourseCode
	screenBulkFormStartMeeting
	screenBulkFormCount
	screenBulkFormStartDate
	screenBulkFormInterval
	screenBulkConfirm
)

// Daftar menu
var mainMenuItems = []string{
	"Lihat Daftar Environment",
	"Persiapan Ruangan",
	"Reset Sesi ",
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
	"Tambah Banyak Sesi",
	"Edit Sesi",
	"Hapus Sesi",
	"Kembali ke Menu Utama",
}

var sessionStatusOptions = []string{"scheduled", "active", "completed", "cancelled"}

type model struct {
	cfg *Config
	db  *pgxpool.Pool

	state  screenState
	errMsg string
	spin   spinner.Model

	returnTo func(model) (tea.Model, tea.Cmd)

	menuItems  []string
	menuCursor int

	rooms   []Room
	modules []Module
	envRows []EnvironmentRow

	selectedRoom      string
	selectedModule    string
	selectedAction    string
	selectedResetMode string
	selectedContainer string

	provisionSessionIDs []string
	selectedSessionID   string

	commandOutput string
	commandFailed bool

	//  Kelola Ruangan
	roomsDetailed           []RoomDetail
	roomFormMode            string
	roomInputNama           textinput.Model
	roomInputPortPrefix     textinput.Model
	roomInputCapacity       textinput.Model
	editingRoomOriginalNama string

	//  Kelola Sesi
	sessionsDetailed          []SessionDetail
	sessionPickIDs            []string
	sessionFormMode           string
	sessionInputCourseCode    textinput.Model
	sessionInputMeetingNumber textinput.Model
	sessionInputDate          textinput.Model
	sessionRoom               string
	sessionModule             string
	sessionStatus             string
	editingSessionID          string

	//  Tambah Banyak Sesi
	bulkRoom              string
	bulkModule            string
	bulkInputCourseCode   textinput.Model
	bulkInputStartMeeting textinput.Model
	bulkInputCount        textinput.Model
	bulkInputStartDate    textinput.Model
	bulkInputInterval     textinput.Model
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

		sessionInputCourseCode:    newInput("Contoh: 1NFBR261131K"),
		sessionInputMeetingNumber: newInput("Contoh: 2"),
		sessionInputDate:          newInput("YYYY-MM-DD"),

		bulkInputCourseCode:   newInput("Contoh: 1NFBR261131K"),
		bulkInputStartMeeting: newInput("Contoh: 2"),
		bulkInputCount:        newInput("Contoh: 6"),
		bulkInputStartDate:    newInput("Tanggal pertemuan pertaman(YYYY-MM-DD)"),
		bulkInputInterval:     newInput("Contoh: 7 (dalam hari)"),
	}
}

// Styling
var (
	dark      = lipgloss.Color("#1A1A2E")
	border    = lipgloss.Color("#E94560")
	secondary = lipgloss.Color("#FFD23F")
	accent    = lipgloss.Color("#F39C12")
	base      = lipgloss.Color("#F5F5F5")
	success   = lipgloss.Color("#2ECC71")
	danger    = lipgloss.Color("#FF3B30")

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(accent).Padding(0, 1)
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(border).Padding(1, 3)
	hintStyle  = lipgloss.NewStyle().Foreground(secondary).Italic(true)
	errStyle   = lipgloss.NewStyle().Foreground(danger).Bold(true)
	okStyle    = lipgloss.NewStyle().Foreground(success).Bold(true)
	cursorStyle = lipgloss.NewStyle().Foreground(dark).Background(accent).Bold(true).Padding(0, 1)
	dimStyle    = lipgloss.NewStyle().Foreground(secondary)
	labelStyle  = lipgloss.NewStyle().Foreground(base)
	logoColor    = lipgloss.Color("#F8F8F2")
	logoStyle    = lipgloss.NewStyle().Foreground(logoColor).Italic(true)
	logoSubStyle = lipgloss.NewStyle().Foreground(secondary).Bold(true)
)

const Logo = `     __       ______  ____    _  __  ____    __  ___
    /  /     / ____/ / __ \  / //_/ / __ \  /  |/  /
   /  /     / /___  / /_/ / / ,<   / / / / / /|_/ /
  /  /___  / /___  / ____/ / /| | / /_/ / / /  / /
 /______/ /_____/ /_/     /_/ |_| \____/ /_/  /_/    `

func renderLogo() string {
	width := lipgloss.Width(Logo)
	logo := logoStyle.Render(Logo)
	sub := logoSubStyle.Render(lipgloss.PlaceHorizontal(width, lipgloss.Center, "G U N A D A R M A"))
	return logo + "\n" + sub
}

// MSG
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
type sessionsForProvisionLoadedMsg struct {
	sessions []SessionDetail
	err      error
}
type roomModuleLoadedMsg struct {
	module string
	err    error
}
type roomMutatedMsg struct {
	err error
}
type sessionMutatedMsg struct {
	err error
}
