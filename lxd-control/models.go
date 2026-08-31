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

	//  Ganti Sesi Ruangan
	screenPickRoomForNextMeeting
	screenPickSessionForNextMeeting
	screenConfirmNextMeeting

	//  Kelola Ruangan (CRUD) 
	screenRoomsMenu
	screenRoomsList
	screenRoomFormNama
	screenRoomFormPortPrefix
	screenRoomFormCapacity
	screenRoomPickForEdit
	screenRoomPickForDelete
	screenRoomDeleteConfirm

	//  Kelola Sesi (CRUD) 
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

	//  Tambah Banyak Sesi
	screenBulkPickRoom
	screenBulkPickModule
	screenBulkFormCourseCode
	screenBulkFormStartMeeting
	screenBulkFormCount
	screenBulkFormStartDate
	screenBulkFormInterval
	screenBulkConfirm
)

// Daftar menu tetap, dipakai ulang di beberapa tempat supaya tidak perlu
// menuliskan literalnya berkali-kali (dan menghindari salah ketik/tidak
// sinkron antar tempat).
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

	// provisionSessionIDs paralel dengan menuItems saat screenPickSessionForProvision
	provisionSessionIDs []string
	selectedSessionID   string

	commandOutput string
	commandFailed bool

	//  Kelola Ruangan 
	roomsDetailed           []RoomDetail
	roomFormMode            string // "create" atau "update"
	roomInputNama           textinput.Model
	roomInputPortPrefix     textinput.Model
	roomInputCapacity       textinput.Model
	editingRoomOriginalNama string

	//  Kelola Sesi 
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

	//  Tambah Banyak Sesi Sekaligus 
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

// ==================== STYLES ====================
//
// Palet warna "retro-arcade" — pink/oranye/kuning, sengaja beda total dari
// palet Dracula yang dipakai di lxd-tui, dan tidak memakai cyan sama sekali:
//   dark      #1A1A2E  indigo gelap — teks di dalam pill accent (kontras
//                       gelap di atas oranye terang)
//   border    #E94560  pink/merah cerah — border box, bingkai mencolok ala
//                       mesin arcade
//   secondary #FFD23F  kuning emas — label / hint / item menu nonaktif
//   accent    #F39C12  oranye — title, background pill cursor
//   base      #F5F5F5  putih hangat — teks utama/value
//   success   #2ECC71  hijau — indikator status positif
//   danger    #FF3B30  merah terang — error (sengaja beda hue dari border
//                       pink, biar makna "error" tidak ketuker dengan bingkai)
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

	// cursorStyle: pill solid (teks gelap di atas background oranye) supaya
	// item yang sedang dipilih terasa mencolok & interaktif.
	cursorStyle = lipgloss.NewStyle().Foreground(dark).Background(accent).Bold(true).Padding(0, 1)
	dimStyle    = lipgloss.NewStyle().Foreground(secondary)
	labelStyle  = lipgloss.NewStyle().Foreground(base)

	logoColor    = lipgloss.Color("#F8F8F2")
	logoStyle    = lipgloss.NewStyle().Foreground(logoColor).Italic(true)
	logoSubStyle = lipgloss.NewStyle().Foreground(secondary).Bold(true)
)

// lepkomLogo adalah ASCII art nama LEPKOM. Disimpan sebagai raw string agar
// spasi/indentasinya persis seperti yang dirancang.
const lepkomLogo = `     __       ______  ____    _  __  ____    __  ___ 
    /  /     / ____/ / __ \  / //_/ / __ \  /  |/  / 
   /  /     / /___  / /_/ / / ,<   / / / / / /|_/ /  
  /  /___  / /___  / ____/ / /| | / /_/ / / /  / /   
 /______/ /_____/ /_/     /_/ |_| \____/ /_/  /_/    `

// renderLogo merender logo ASCII LEPKOM (italic, warna hijau tua) dengan
// subjudul "G U N A D A R M A" yang otomatis dipusatkan sesuai lebar logo.
func renderLogo() string {
	width := lipgloss.Width(lepkomLogo)
	logo := logoStyle.Render(lepkomLogo)
	sub := logoSubStyle.Render(lipgloss.PlaceHorizontal(width, lipgloss.Center, "G U N A D A R M A"))
	return logo + "\n" + sub
}

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