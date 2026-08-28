package state

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"lxd-tui/internal/models"
)

// ScreenType mendefinisikan semua layar yang tersedia
type ScreenType int

const (
	ScreenLoading ScreenType = iota
	ScreenDashboard
	ScreenInputNama
	ScreenInputNPM
	ScreenSubmitting
	ScreenSelectUser    // Pilih mau login sebagai user Linux apa
	ScreenLocalPassword // Masukkan password akun Linux yang dipilih
	ScreenError
)

// AppState menyimpan semua state aplikasi
type AppState struct {
	// Current screen
	CurrentScreen ScreenType

	// Data
	EnvInfo      *models.EnvInfo
	LocalUsers   []models.LocalUser
	SelectedUser string
	ErrorMessage string

	// Input components
	InputNama     textinput.Model
	InputNPM      textinput.Model
	InputPassword textinput.Model
	Spinner       spinner.Model

	// UI state
	UserCursor int
	PWError    string

	// Flag: hanya true jika seluruh alur sukses
	ShouldContinueToShell bool

	// Window size
	WindowWidth  int
	WindowHeight int
}

func NewAppState() *AppState {
	nama := textinput.New()
	nama.Placeholder = "Nama lengkap"
	nama.Focus()
	nama.CharLimit = 150
	nama.Width = 40
	nama.PromptStyle = lipgloss.NewStyle().SetString("")

	npm := textinput.New()
	npm.Placeholder = "NPM"
	npm.CharLimit = 30
	npm.Width = 40
	npm.PromptStyle = lipgloss.NewStyle().SetString("")

	pw := textinput.New()
	pw.Placeholder = "Password"
	pw.CharLimit = 100
	pw.Width = 40
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '•'
	pw.PromptStyle = lipgloss.NewStyle().SetString("")

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#588157"))

	return &AppState{
		CurrentScreen: ScreenLoading,
		InputNama:     nama,
		InputNPM:      npm,
		InputPassword: pw,
		Spinner:       sp,
	}
}

// Helper methods untuk mengubah state dengan aman

func (s *AppState) GoToDashboard(info *models.EnvInfo) {
	s.EnvInfo = info
	s.CurrentScreen = ScreenDashboard
}

func (s *AppState) GoToInputNama() {
	s.CurrentScreen = ScreenInputNama
	s.InputNama.Focus()
	s.InputNPM.Blur()
}

func (s *AppState) GoToInputNPM() {
	s.InputNama.Blur()
	s.InputNPM.Focus()
	s.CurrentScreen = ScreenInputNPM
}

func (s *AppState) GoToSelectUser(users []models.LocalUser) {
	s.LocalUsers = users
	s.UserCursor = 0
	s.CurrentScreen = ScreenSelectUser
}

func (s *AppState) GoToPasswordInput(username string) {
	s.SelectedUser = username
	s.PWError = ""
	s.InputPassword.SetValue("")
	s.InputPassword.Focus()
	s.CurrentScreen = ScreenLocalPassword
}

func (s *AppState) GoToError(errMsg string) {
	s.ErrorMessage = errMsg
	s.CurrentScreen = ScreenError
}

func (s *AppState) ResetForRetry() {
	s.CurrentScreen = ScreenLoading
	s.InputNama.SetValue("")
	s.InputNPM.SetValue("")
	s.ErrorMessage = ""
	s.PWError = ""
}

func (s *AppState) SetPasswordError(err string) {
	s.PWError = err
	s.InputPassword.SetValue("")
}

func (s *AppState) StartSubmitting() {
	s.CurrentScreen = ScreenSubmitting
}

func (s *AppState) MarkShellAllowed() {
	s.ShouldContinueToShell = true
}

// Keyboard navigation helpers untuk select user
func (s *AppState) UserCursorUp() {
	if s.UserCursor > 0 {
		s.UserCursor--
	}
}

func (s *AppState) UserCursorDown() {
	if s.UserCursor < len(s.LocalUsers)-1 {
		s.UserCursor++
	}
}

func (s *AppState) GetSelectedUser() *models.LocalUser {
	if len(s.LocalUsers) == 0 || s.UserCursor >= len(s.LocalUsers) {
		return nil
	}
	return &s.LocalUsers[s.UserCursor]
}
