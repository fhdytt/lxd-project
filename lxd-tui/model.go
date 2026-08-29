package main

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// ==================== STATE ====================

type screenState int

const (
	screenLoading screenState = iota
	screenDashboard
	screenInputNama
	screenInputNPM
	screenSubmitting
	screenSelectUser    // pilih mau login sebagai user Linux apa
	screenLocalPassword // masukkan password akun Linux yang dipilih
	screenError
)

// model adalah state Bubble Tea tunggal untuk seluruh aplikasi TUI praktikan.
type model struct {
	client *APIClient

	state   screenState
	envInfo *EnvInfo
	errMsg  string

	// shouldContinueToShell HANYA true kalau seluruh alur (identifikasi API
	// + verifikasi password akun Linux) benar-benar selesai sukses. Ctrl+C
	// atau error apapun TIDAK mengubah flag ini — lihat main.go.
	shouldContinueToShell bool
	selectedUsername      string

	inputNama     textinput.Model
	inputNPM      textinput.Model
	inputPassword textinput.Model
	spin          spinner.Model

	localUsers []LocalUser
	userCursor int
	pwError    string

	windowWidth  int
	windowHeight int
}

func initialModel(client *APIClient) model {
	nama := textinput.New()
	nama.Placeholder = "Nama lengkap"
	nama.Focus()
	nama.CharLimit = 150
	nama.Width = 40
	nama.PromptStyle = lipgloss.NewStyle().Foreground(marioRed)

	npm := textinput.New()
	npm.Placeholder = "NPM"
	npm.CharLimit = 30
	npm.Width = 40
	npm.PromptStyle = lipgloss.NewStyle().Foreground(marioRed)

	pw := textinput.New()
	pw.Placeholder = "Password"
	pw.CharLimit = 100
	pw.Width = 40
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '•'
	pw.PromptStyle = lipgloss.NewStyle().Foreground(marioRed)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(marioRed)

	return model{
		client:        client,
		state:         screenLoading,
		inputNama:     nama,
		inputNPM:      npm,
		inputPassword: pw,
		spin:          sp,
	}
}

// ==================== STYLES ====================
//
// Palet warna: Nintendo Retro / Game Boy Style (Transparan / Tanpa Background)
//   marioRed     #FF3131  Merah Nintendo / Mario (Accent, Title, Cursor)
//   snesPurple   #836FFF  Ungu SNES (Border, Labels, Subtitles)
//   gameboyGreen #9BBC0F  Hijau Khas Layar Game Boy (Values / Success)
//   coinYellow   #FEE12B  Kuning Koin Super Mario (Hints & Subtitles)
//   creamWhite   #F8F8F0  Putih Game Boy Shell (Base Text)
//   dangerRed    #FF0055  Merah Terang (Errors)
//
var (
	marioRed     = lipgloss.Color("#FF3131")
	snesPurple   = lipgloss.Color("#836FFF")
	gameboyGreen = lipgloss.Color("#9BBC0F")
	coinYellow   = lipgloss.Color("#FEE12B")
	creamWhite   = lipgloss.Color("#F8F8F0")
	dangerRed    = lipgloss.Color("#FF0055")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(marioRed).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(creamWhite).
			Italic(true)

	// boxStyle TANPA Background() agar menyatu dengan terminal
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(snesPurple).
			Padding(1, 3)

	labelStyle = lipgloss.NewStyle().Foreground(snesPurple)

	// valueStyle TANPA Background() agar tidak ada sorotan kotak hitam
	valueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(gameboyGreen)

	hintStyle = lipgloss.NewStyle().Foreground(coinYellow).Italic(true)
	errStyle  = lipgloss.NewStyle().Foreground(dangerRed).Bold(true)

	// cursorStyle untuk menu pilihan
	cursorStyle = lipgloss.NewStyle().Foreground(marioRed).Bold(true)
	menuItemDim = lipgloss.NewStyle().Foreground(snesPurple)

	// Logo LEPKOM Merah Mario & Subjudul Kuning Koin
	logoColor    = lipgloss.Color("#9BBC0F")
	logoStyle    = lipgloss.NewStyle().Foreground(logoColor).Bold(true)
	logoSubStyle = lipgloss.NewStyle().Foreground(coinYellow).Bold(true)
)

// lepkomLogo — sama persis dengan versi lxd-control.
const lepkomLogo = `    __       ______  ____    _  __  ____    __  ___ 
   / /      / ____/ / __ \  / //_/ / __ \  /  |/  / 
  / /      / /___  / /_/ / / ,<   / / / / / /|_/ /  
 / /___   / /___  / ____/ / /| | / /_/ / / /  / /   
/______/ /_____/ /_/     /_/ |_| \____/ /_/  /_/    `

// renderLogo merender logo ASCII LEPKOM dengan subjudul
// "G U N A D A R M A" yang otomatis dipusatkan sesuai lebar logo.
func renderLogo() string {
	width := lipgloss.Width(lepkomLogo)
	logo := logoStyle.Render(lepkomLogo)
	sub := logoSubStyle.Render(lipgloss.PlaceHorizontal(width, lipgloss.Center, "G U N A D A R M A"))
	return logo + "\n" + sub
}

// ==================== MESSAGES ====================

type envInfoFetchedMsg struct {
	info *EnvInfo
	err  error
}

type identitySubmittedMsg struct {
	err error
}

type localUsersLoadedMsg struct {
	users []LocalUser
	err   error
}

type passwordVerifiedMsg struct {
	ok  bool
	err error
}