package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"

	"lxd-tui/internal/api"
	"lxd-tui/internal/auth"
	"lxd-tui/internal/models"
	"lxd-tui/internal/state"
	"lxd-tui/internal/views"
)

// Konfigurasi
const (
	APIBaseURL = "http://10.10.10.9:8080"
)

type model struct {
	state *state.AppState
	api   *api.APIClient
}

func initialModel() model {
	token := os.Getenv("LXD_TOKEN")
	if token == "" {
		log.Fatal("LXD_TOKEN environment variable is required")
	}

	return model{
		state: state.NewAppState(),
		api:   api.NewAPIClient(APIBaseURL, token),
	}
}

// ==================== COMMANDS ====================

func (m model) fetchEnvCmd() tea.Cmd {
	return func() tea.Msg {
		info, err := m.api.FetchEnvInfo()
		return envInfoMsg{info: info, err: err}
	}
}

func (m model) submitIdentityCmd() tea.Cmd {
	nama := m.state.InputNama.Value()
	npm := m.state.InputNPM.Value()
	return func() tea.Msg {
		err := m.api.SubmitIdentity(nama, npm)
		return identityMsg{err: err}
	}
}

func loadUsersCmd() tea.Cmd {
	return func() tea.Msg {
		users, err := auth.ListLocalUsers()
		return usersMsg{users: users, err: err}
	}
}

func verifyPasswordCmd(username, password string) tea.Cmd {
	return func() tea.Msg {
		ok, err := auth.VerifyLocalPassword(username, password)
		return passwordMsg{ok: ok, err: err}
	}
}

// ==================== MESSAGES ====================

type envInfoMsg struct {
	info *models.EnvInfo
	err  error
}

type identityMsg struct {
	err error
}

type usersMsg struct {
	users []models.LocalUser
	err   error
}

type passwordMsg struct {
	ok  bool
	err error
}

// ==================== INIT ====================

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchEnvCmd(),
		m.state.Spinner.Tick,
	)
}

// ==================== UPDATE ====================

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.state.WindowWidth = msg.Width
		m.state.WindowHeight = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			// shouldContinueToShell TETAP false -> tidak pernah lanjut ke shell
			return m, tea.Quit
		}
		return m.handleKey(msg)

	case envInfoMsg:
		if msg.err != nil {
			m.state.GoToError(msg.err.Error())
			return m, nil
		}
		m.state.GoToDashboard(msg.info)
		return m, nil

	case identityMsg:
		if msg.err != nil {
			m.state.GoToError(msg.err.Error())
			return m, nil
		}
		// Identifikasi berhasil -> lanjut ke pemilihan user Linux
		m.state.CurrentScreen = state.ScreenSelectUser
		return m, loadUsersCmd()

	case usersMsg:
		if msg.err != nil {
			m.state.GoToError(msg.err.Error())
			return m, nil
		}
		m.state.GoToSelectUser(msg.users)
		return m, nil

	case passwordMsg:
		if msg.err != nil {
			m.state.GoToError(msg.err.Error())
			return m, nil
		}
		if !msg.ok {
			m.state.SetPasswordError("Password salah, coba lagi.")
			return m, nil
		}
		// Password cocok -> BARU di sini flag diizinkan lanjut ke shell
		m.state.MarkShellAllowed()
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.state.Spinner, cmd = m.state.Spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// ==================== KEY HANDLER ====================

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state.CurrentScreen {

	case state.ScreenDashboard:
		if msg.String() == "enter" {
			m.state.GoToInputNama()
		}
		return m, nil

	case state.ScreenInputNama:
		if msg.String() == "enter" && m.state.InputNama.Value() != "" {
			m.state.GoToInputNPM()
			return m, nil
		}
		var cmd tea.Cmd
		m.state.InputNama, cmd = m.state.InputNama.Update(msg)
		return m, cmd

	case state.ScreenInputNPM:
		if msg.String() == "enter" && m.state.InputNPM.Value() != "" {
			m.state.StartSubmitting()
			return m, m.submitIdentityCmd()
		}
		if msg.String() == "esc" {
			m.state.GoToInputNama()
			return m, nil
		}
		var cmd tea.Cmd
		m.state.InputNPM, cmd = m.state.InputNPM.Update(msg)
		return m, cmd

	case state.ScreenSelectUser:
		if len(m.state.LocalUsers) == 0 {
			return m, nil
		}
		switch msg.String() {
		case "up", "k":
			m.state.UserCursorUp()
		case "down", "j":
			m.state.UserCursorDown()
		case "enter":
			if user := m.state.GetSelectedUser(); user != nil {
				m.state.GoToPasswordInput(user.Username)
			}
		}
		return m, nil

	case state.ScreenLocalPassword:
		if msg.String() == "enter" && m.state.InputPassword.Value() != "" {
			return m, verifyPasswordCmd(
				m.state.SelectedUser,
				m.state.InputPassword.Value(),
			)
		}
		if msg.String() == "esc" {
			m.state.CurrentScreen = state.ScreenSelectUser
			m.state.PWError = ""
			return m, nil
		}
		var cmd tea.Cmd
		m.state.InputPassword, cmd = m.state.InputPassword.Update(msg)
		return m, cmd

	case state.ScreenError:
		if msg.String() == "enter" {
			m.state.ResetForRetry()
			return m, m.fetchEnvCmd()
		}
		return m, nil
	}

	return m, nil
}

// ==================== VIEW ====================

func (m model) View() string {
	return views.MainView(m.state)
}

// ==================== MAIN ====================

func main() {
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	finalModel, err := p.Run()
	if err != nil {
		log.Fatal(err)
	}

	// Cek apakah boleh lanjut ke shell
	if m, ok := finalModel.(model); ok &&
		m.state.ShouldContinueToShell {
	}
}
