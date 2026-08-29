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
	nama.PromptStyle = lipgloss.NewStyle().Foreground(accent)

	npm := textinput.New()
	npm.Placeholder = "NPM"
	npm.CharLimit = 30
	npm.Width = 40
	npm.PromptStyle = lipgloss.NewStyle().Foreground(accent)

	pw := textinput.New()
	pw.Placeholder = "Password"
	pw.CharLimit = 100
	pw.Width = 40
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '•'
	pw.PromptStyle = lipgloss.NewStyle().Foreground(accent)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(accent)

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
	accent    = lipgloss.Color("39")
	accentDim = lipgloss.Color("245")
	danger    = lipgloss.Color("203")
	muted     = lipgloss.Color("241")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(accentDim).
			Italic(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(1, 3)

	labelStyle  = lipgloss.NewStyle().Foreground(accentDim)
	valueStyle  = lipgloss.NewStyle().Bold(true)
	hintStyle   = lipgloss.NewStyle().Foreground(muted).Italic(true)
	errStyle    = lipgloss.NewStyle().Foreground(danger).Bold(true)
	cursorStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	menuItemDim = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

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