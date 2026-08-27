package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	screenSelectUser    // NEW: pilih mau login sebagai user Linux apa
	screenLocalPassword // NEW: masukkan password akun Linux yang dipilih
	screenError
)

type model struct {
	client *APIClient

	state   screenState
	envInfo *EnvInfo
	errMsg  string

	// shouldContinueToShell HANYA true kalau seluruh alur (identifikasi API
	// + verifikasi password akun Linux) benar-benar selesai sukses. Ctrl+C
	// atau error apapun TIDAK mengubah flag ini.
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

// ==================== STYLES (DISESUAIKAN) ====================

var (
	// Palet Hex Earthy
	bgDark    = lipgloss.Color("#344e41")
	bgMid     = lipgloss.Color("#3a5a40")
	accent    = lipgloss.Color("#588157")
	textSoft  = lipgloss.Color("#a3b18a")
	textLight = lipgloss.Color("#dad7cd")
	danger    = lipgloss.Color("#e63946")

	// Logo Style & ASCII Art Tegak
	logoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent)

	asciiLogo = `
  _        ______  ____   _  __   ____    __  ___ 
 | |      / ____/ / __ \ | |/ /  / __ \  /  |/  / 
 | |     / /___  / /_/ / | ' /  / / / / / /|_/ /  
 | |___ / /___  / ____/  | . \ / /_/ / / /  / /   
 |_____/_____/ /_/       |_|\_\\____/ /_/  /_/    
                                                  
              G U N A D A R M A                   
`

	// Component Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textLight).
			Background(bgDark).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(textSoft).
			Italic(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(1, 2)

	labelStyle = lipgloss.NewStyle().
			Foreground(textSoft)

	valueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textLight)

	hintStyle = lipgloss.NewStyle().
			Foreground(textSoft).
			Italic(true)

	errStyle = lipgloss.NewStyle().
			Foreground(danger).
			Bold(true)

	cursorStyle = lipgloss.NewStyle().
			Foreground(textLight).
			Bold(true)

	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(textLight).
				Background(accent).
				Padding(0, 1)

	inactiveItemStyle = lipgloss.NewStyle().
				Foreground(textSoft).
				Padding(0, 1)

	dividerStyle = lipgloss.NewStyle().
			Foreground(accent)
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

// ==================== INIT ====================

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

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchEnvInfoCmd(m.client), m.spin.Tick)
}

func fetchEnvInfoCmd(client *APIClient) tea.Cmd {
	return func() tea.Msg {
		info, err := client.FetchEnvInfo()
		return envInfoFetchedMsg{info: info, err: err}
	}
}

func submitIdentityCmd(client *APIClient, nama, npm string) tea.Cmd {
	return func() tea.Msg {
		err := client.SubmitIdentity(nama, npm)
		return identitySubmittedMsg{err: err}
	}
}

func loadLocalUsersCmd() tea.Cmd {
	return func() tea.Msg {
		users, err := listLocalUsers()
		return localUsersLoadedMsg{users: users, err: err}
	}
}

func verifyPasswordCmd(username, password string) tea.Cmd {
	return func() tea.Msg {
		ok, err := verifyLocalPassword(username, password)
		return passwordVerifiedMsg{ok: ok, err: err}
	}
}

// ==================== UPDATE ====================

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			// shouldContinueToShell TETAP false -> main() menutup sesi,
			// tidak pernah lanjut ke shell, dari state manapun.
			return m, tea.Quit
		}
		return m.handleKey(msg)

	case envInfoFetchedMsg:
		if msg.err != nil {
			m.state = screenError
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.envInfo = msg.info
		m.state = screenDashboard
		return m, nil

	case identitySubmittedMsg:
		if msg.err != nil {
			m.state = screenError
			if errors.Is(msg.err, ErrIdentityMismatch) {
				m.errMsg = "Nama/NPM tidak cocok dengan environment ini.\nEnvironment ini sudah terdaftar atas nama praktikan lain."
			} else {
				m.errMsg = msg.err.Error()
			}
			return m, nil
		}
		// Identifikasi berhasil -> lanjut ke pemilihan user Linux, BUKAN
		// langsung ke shell. Baca daftar user lokal dulu.
		m.state = screenSelectUser
		return m, loadLocalUsersCmd()

	case localUsersLoadedMsg:
		if msg.err != nil {
			m.state = screenError
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.localUsers = msg.users
		m.userCursor = 0
		return m, nil

	case passwordVerifiedMsg:
		if msg.err != nil {
			m.state = screenError
			m.errMsg = msg.err.Error()
			return m, nil
		}
		if !msg.ok {
			m.pwError = "Password salah, coba lagi."
			m.inputPassword.SetValue("")
			return m, nil
		}
		// Password cocok -> BARU di sini flag diizinkan lanjut ke shell.
		m.shouldContinueToShell = true
		return m, tea.Quit
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {

	case screenDashboard:
		if msg.String() == "enter" {
			// Selalu minta identifikasi/verifikasi, tidak ada jalan pintas
			// otomatis lolos walau environment ini sudah pernah diisi.
			m.state = screenInputNama
			m.inputNama.Focus()
		}
		return m, nil

	case screenInputNama:
		if msg.String() == "enter" && m.inputNama.Value() != "" {
			m.inputNama.Blur()
			m.inputNPM.Focus()
			m.state = screenInputNPM
			return m, nil
		}
		var cmd tea.Cmd
		m.inputNama, cmd = m.inputNama.Update(msg)
		return m, cmd

	case screenInputNPM:
		if msg.String() == "enter" && m.inputNPM.Value() != "" {
			m.state = screenSubmitting
			return m, submitIdentityCmd(m.client, m.inputNama.Value(), m.inputNPM.Value())
		}
		if msg.String() == "esc" {
			m.inputNPM.Blur()
			m.inputNama.Focus()
			m.state = screenInputNama
			return m, nil
		}
		var cmd tea.Cmd
		m.inputNPM, cmd = m.inputNPM.Update(msg)
		return m, cmd

	case screenSelectUser:
		if len(m.localUsers) == 0 {
			return m, nil
		}
		switch msg.String() {
		case "up", "k":
			if m.userCursor > 0 {
				m.userCursor--
			}
		case "down", "j":
			if m.userCursor < len(m.localUsers)-1 {
				m.userCursor++
			}
		case "enter":
			m.selectedUsername = m.localUsers[m.userCursor].Username
			m.pwError = ""
			m.inputPassword.SetValue("")
			m.inputPassword.Focus()
			m.state = screenLocalPassword
		}
		return m, nil

	case screenLocalPassword:
		if msg.String() == "enter" && m.inputPassword.Value() != "" {
			password := m.inputPassword.Value()
			return m, verifyPasswordCmd(m.selectedUsername, password)
		}
		if msg.String() == "esc" {
			m.state = screenSelectUser
			m.pwError = ""
			return m, nil
		}
		var cmd tea.Cmd
		m.inputPassword, cmd = m.inputPassword.Update(msg)
		return m, cmd

	case screenError:
		if msg.String() == "enter" {
			m.state = screenLoading
			m.inputNama.SetValue("")
			m.inputNPM.SetValue("")
			return m, fetchEnvInfoCmd(m.client)
		}
		return m, nil
	}

	return m, nil
}

// ==================== VIEW (TAMPILAN BARU) ====================

func (m model) renderBox(content string) string {
	width := 56
	if m.windowWidth > 0 {
		width = m.windowWidth - 10
		if width > 60 {
			width = 60
		}
		if width < 30 {
			width = 30
		}
	}

	// Render logo di bagian paling atas untuk semua layar
	fullContent := fmt.Sprintf("%s\n%s", logoStyle.Render(asciiLogo), content)
	box := boxStyle.Width(width).Render(fullContent)

	if m.windowWidth > 0 && m.windowHeight > 0 {
		return lipgloss.Place(m.windowWidth, m.windowHeight, lipgloss.Center, lipgloss.Center, box)
	}
	return box
}

func (m model) View() string {
	switch m.state {

	case screenLoading:
		return m.renderBox(fmt.Sprintf("%s Menghubungkan ke server...", m.spin.View()))

	case screenDashboard:
		return m.renderBox(m.viewDashboard())

	case screenInputNama:
		hint := "Masukkan nama lengkap:"
		if m.envInfo != nil && m.envInfo.AlreadyIdentified {
			hint = "Environment ini sudah terdaftar. Masukkan nama untuk verifikasi:"
		}
		return m.renderBox(fmt.Sprintf(
			"%s\n\n%s\n%s\n\n%s\n%s",
			titleStyle.Render(" IDENTIFIKASI PRAKTIKAN "),
			labelStyle.Render(hint),
			m.inputNama.View(),
			dividerStyle.Render(strings.Repeat("─", 50)),
			hintStyle.Render("[Enter] Lanjut  |  [Ctrl+C] Batal"),
		))

	case screenInputNPM:
		return m.renderBox(fmt.Sprintf(
			"%s\n\n%s %s\n\n%s\n%s\n\n%s\n%s",
			titleStyle.Render(" IDENTIFIKASI PRAKTIKAN "),
			labelStyle.Render("Nama:"), valueStyle.Render(m.inputNama.Value()),
			labelStyle.Render("Masukkan NPM:"),
			m.inputNPM.View(),
			dividerStyle.Render(strings.Repeat("─", 50)),
			hintStyle.Render("[Enter] Kirim  |  [Esc] Kembali  |  [Ctrl+C] Batal"),
		))

	case screenSubmitting:
		return m.renderBox(fmt.Sprintf("%s Memverifikasi identitas...", m.spin.View()))

	case screenSelectUser:
		return m.renderBox(m.viewSelectUser())

	case screenLocalPassword:
		errLine := ""
		if m.pwError != "" {
			errLine = "\n" + errStyle.Render(m.pwError) + "\n"
		}
		return m.renderBox(fmt.Sprintf(
			"%s\n\n%s %s\n\n%s\n%s%s\n%s",
			titleStyle.Render(" MASUK SEBAGAI "+strings.ToUpper(m.selectedUsername)+" "),
			labelStyle.Render("User:"), valueStyle.Render(m.selectedUsername),
			m.inputPassword.View(),
			errLine,
			dividerStyle.Render(strings.Repeat("─", 50)),
			hintStyle.Render("[Enter] Masuk  |  [Esc] Ganti User  |  [Ctrl+C] Batal"),
		))

	case screenError:
		return m.renderBox(fmt.Sprintf(
			"%s\n\n%s\n\n%s\n%s",
			errStyle.Render("[ ERROR SYSTEM ]"),
			m.errMsg,
			dividerStyle.Render(strings.Repeat("─", 50)),
			hintStyle.Render("[Enter] Coba lagi  |  [Ctrl+C] Keluar"),
		))
	}
	return ""
}

func (m model) viewDashboard() string {
	info := m.envInfo

	identStatus := valueStyle.Render("Belum diisi")
	if info.AlreadyIdentified {
		identStatus = lipgloss.NewStyle().Bold(true).Foreground(accent).Render("Sudah terdaftar (perlu verifikasi)")
	}

	divider := dividerStyle.Render(strings.Repeat("─", 50))

	body := fmt.Sprintf(
		"%s\n%s\n%s\n\n%s %s\n%s %s\n%s %s\n%s %d\n%s %s\n%s %s\n%s %s\n\n%s\n%s",
		titleStyle.Render(" DASHBOARD SESI PRAKTIKUM "),
		subtitleStyle.Render("Sistem Manajemen Environment Praktikum"),
		divider,
		labelStyle.Render("Kode Kursus  :"), valueStyle.Render(info.CourseCode),
		labelStyle.Render("Modul        :"), valueStyle.Render(info.Module),
		labelStyle.Render("Ruangan      :"), valueStyle.Render(info.Room),
		labelStyle.Render("Pertemuan ke :"), info.MeetingNumber,
		labelStyle.Render("Tanggal      :"), valueStyle.Render(info.SessionDate),
		labelStyle.Render("Status Env   :"), valueStyle.Render(info.Status),
		labelStyle.Render("Identifikasi :"), identStatus,
		divider,
		hintStyle.Render("[Enter] Lanjutkan  |  [Ctrl+C] Keluar"),
	)

	return body
}

func (m model) viewSelectUser() string {
	if len(m.localUsers) == 0 {
		return fmt.Sprintf("%s Memuat daftar user...", m.spin.View())
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(" MASUK SEBAGAI "))
	b.WriteString("\n\n")

	for i, u := range m.localUsers {
		if i == m.userCursor {
			itemText := fmt.Sprintf("[-] %02d. %s", i+1, u.Username)
			b.WriteString(selectedItemStyle.Render(itemText))
		} else {
			itemText := fmt.Sprintf("    %02d. %s", i+1, u.Username)
			b.WriteString(inactiveItemStyle.Render(itemText))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dividerStyle.Render(strings.Repeat("─", 50)))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("[UP/DOWN] Pilih  |  [ENTER] Konfirmasi  |  [CTRL+C] Batal"))
	return b.String()
}

// ==================== MAIN ====================

func getContainerEnv(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	data, err := os.ReadFile("/proc/1/environ")
	if err != nil {
		return ""
	}
	for _, entry := range strings.Split(string(data), "\x00") {
		if strings.HasPrefix(entry, key+"=") {
			return strings.TrimPrefix(entry, key+"=")
		}
	}
	return ""
}

func main() {
	apiURL := getContainerEnv("PRAKTIKUM_API_URL")
	token := getContainerEnv("PRAKTIKUM_API_TOKEN")

	if apiURL == "" || token == "" {
		fmt.Fprintln(os.Stderr, "Error: PRAKTIKUM_API_URL atau PRAKTIKUM_API_TOKEN belum di-set pada environment ini.")
		fmt.Fprintln(os.Stderr, "Hubungi asisten/admin lab, environment ini belum terkonfigurasi dengan benar.")
		os.Exit(1)
	}

	client := NewAPIClient(apiURL, token)

	p := tea.NewProgram(initialModel(client), tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "TUI error:", err)
		os.Exit(1)
	}

	m := finalModel.(model)

	if !m.shouldContinueToShell {
		fmt.Println("Sesi ditutup.")
		os.Exit(0)
	}

	execAsUser(m.selectedUsername)
}

func execAsUser(username string) {
	env := os.Environ()
	if err := syscall.Exec("/bin/login", []string{"login", "-f", username}, env); err != nil {
		fmt.Fprintln(os.Stderr, "Gagal masuk sebagai", username, ":", err)
		os.Exit(1)
	}
}