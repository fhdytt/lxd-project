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
	nama.PromptStyle = lipgloss.NewStyle().Foreground(mintPop)

	npm := textinput.New()
	npm.Placeholder = "NPM"
	npm.CharLimit = 30
	npm.Width = 40
	npm.PromptStyle = lipgloss.NewStyle().Foreground(mintPop)

	pw := textinput.New()
	pw.Placeholder = "Password"
	pw.CharLimit = 100
	pw.Width = 40
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '•'
	pw.PromptStyle = lipgloss.NewStyle().Foreground(mintPop)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(mintPop)

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

var (
	harborNavy = lipgloss.Color("#14213D")
	slateMist  = lipgloss.Color("#8D99AE")
	mintPop    = lipgloss.Color("#57CC99")
	cloudWhite = lipgloss.Color("#EDF2F4")
	leafGreen  = lipgloss.Color("#38B000")
	crimsonRed = lipgloss.Color("#D90429")
	black	  = lipgloss.Color("#000000")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(mintPop).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(slateMist).
			Italic(true)

	// boxStyle TANPA Background() supaya menyatu dengan terminal (dipertahankan
	// sesuai preferensi kamu sebelumnya).
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(harborNavy).
			Padding(1, 3)

	labelStyle = lipgloss.NewStyle().Foreground(slateMist)

	valueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(cloudWhite)

	hintStyle = lipgloss.NewStyle().Foreground(slateMist).Italic(true)
	errStyle  = lipgloss.NewStyle().Foreground(crimsonRed).Bold(true)

	// cursorStyle: pill solid (teks navy gelap di atas background mint)
	// supaya pilihan yang aktif terasa mencolok & interaktif.
	cursorStyle = lipgloss.NewStyle().Foreground(black).Background(cloudWhite).Bold(true).Padding(0, 1)
	menuItemDim = lipgloss.NewStyle().Foreground(slateMist)

	// Logo LEPKOM — tetap standar lintas-aplikasi: hijau tua (#2C4533), italic.
	logoColor    = lipgloss.Color("#F8F8F2")
	logoStyle    = lipgloss.NewStyle().Foreground(logoColor).Italic(true)
	logoSubStyle = lipgloss.NewStyle().Foreground(slateMist).Bold(true)
)

// lepkomLogo — sama persis dengan versi lxd-control.
const lepkomLogo = `     __       ______  ____    _  __  ____    __  ___ 
    /  /     / ____/ / __ \  / //_/ / __ \  /  |/  / 
   /  /     / /___  / /_/ / / ,<   / / / / / /|_/ /  
  /  /___  / /___  / ____/ / /| | / /_/ / / /  / /   
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