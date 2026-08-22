package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"

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
	screenError
)

type model struct {
	client *APIClient

	state   screenState
	envInfo *EnvInfo
	errMsg  string

	inputNama textinput.Model
	inputNPM  textinput.Model
}

// ==================== STYLES ====================

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Padding(0, 1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(1, 3)

	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	valueStyle = lipgloss.NewStyle().Bold(true)
	hintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

// ==================== MESSAGES ====================

type envInfoFetchedMsg struct {
	info *EnvInfo
	err  error
}

type identitySubmittedMsg struct {
	err error
}

// ==================== INIT ====================

func initialModel(client *APIClient) model {
	nama := textinput.New()
	nama.Placeholder = "Nama lengkap"
	nama.Focus()
	nama.CharLimit = 150
	nama.Width = 40

	npm := textinput.New()
	npm.Placeholder = "NPM"
	npm.CharLimit = 30
	npm.Width = 40

	return model{
		client:    client,
		state:     screenLoading,
		inputNama: nama,
		inputNPM:  npm,
	}
}

func (m model) Init() tea.Cmd {
	return fetchEnvInfoCmd(m.client)
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

// ==================== UPDATE ====================

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
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
			m.errMsg = msg.err.Error()
			return m, nil
		}
		// Berhasil -> keluar dari Bubble Tea, lanjut exec ke shell di main()
		return m, tea.Quit
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {

	case screenDashboard:
		if msg.String() == "enter" {
			if m.envInfo.AlreadyIdentified {
				// Sudah pernah isi data sebelumnya, langsung lanjut ke shell.
				return m, tea.Quit
			}
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

	case screenError:
		if msg.String() == "enter" {
			// Coba lagi dari awal
			m.state = screenLoading
			return m, fetchEnvInfoCmd(m.client)
		}
		return m, nil
	}

	return m, nil
}

// ==================== VIEW ====================

func (m model) View() string {
	switch m.state {

	case screenLoading:
		return "\n  Menghubungkan ke server, mohon tunggu...\n"

	case screenDashboard:
		return m.viewDashboard()

	case screenInputNama:
		return boxStyle.Render(fmt.Sprintf(
			"%s\n\n%s\n%s\n\n%s",
			titleStyle.Render("Identifikasi Praktikan"),
			labelStyle.Render("Masukkan nama lengkap:"),
			m.inputNama.View(),
			hintStyle.Render("[Enter] lanjut  •  [Ctrl+C] batal"),
		))

	case screenInputNPM:
		return boxStyle.Render(fmt.Sprintf(
			"%s\n\n%s %s\n\n%s\n%s\n\n%s",
			titleStyle.Render("Identifikasi Praktikan"),
			labelStyle.Render("Nama:"), valueStyle.Render(m.inputNama.Value()),
			labelStyle.Render("Masukkan NPM:"),
			m.inputNPM.View(),
			hintStyle.Render("[Enter] kirim  •  [Esc] kembali  •  [Ctrl+C] batal"),
		))

	case screenSubmitting:
		return "\n  Mengirim data, mohon tunggu...\n"

	case screenError:
		return boxStyle.Render(fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			errStyle.Render("Terjadi kesalahan"),
			m.errMsg,
			hintStyle.Render("[Enter] coba lagi  •  [Ctrl+C] keluar"),
		))
	}
	return ""
}

func (m model) viewDashboard() string {
	info := m.envInfo

	identStatus := "Belum diisi"
	if info.AlreadyIdentified {
		identStatus = "Sudah diisi"
	}

	body := fmt.Sprintf(
		"%s\n\n%s %s\n%s %s\n%s %s\n%s %d\n%s %s\n%s %s\n%s %s\n\n%s",
		titleStyle.Render("Dashboard Sesi Praktikum"),
		labelStyle.Render("Kode Kursus  :"), valueStyle.Render(info.CourseCode),
		labelStyle.Render("Modul        :"), valueStyle.Render(info.Module),
		labelStyle.Render("Ruangan      :"), valueStyle.Render(info.Room),
		labelStyle.Render("Pertemuan ke :"), info.MeetingNumber,
		labelStyle.Render("Tanggal      :"), valueStyle.Render(info.SessionDate),
		labelStyle.Render("Status Env   :"), valueStyle.Render(info.Status),
		labelStyle.Render("Identifikasi :"), valueStyle.Render(identStatus),
		hintStyle.Render("[Enter] lanjutkan  •  [Ctrl+C] keluar"),
	)

	return boxStyle.Render(body)
}

// ==================== MAIN ====================

// getContainerEnv membaca env var yang di-inject LXD lewat
// "lxc config set <container> environment.KEY=value". Env var semacam ini
// hanya "menempel" pada proses init container (PID 1) — proses yang
// dijalankan lewat SSH login (lewat PAM) mendapat environment yang bersih
// dan TIDAK otomatis mewarisi env var itu. Makanya os.Getenv() saja tidak
// cukup; kita perlu baca langsung dari /proc/1/environ sebagai fallback.
func getContainerEnv(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	data, err := os.ReadFile("/proc/1/environ")
	if err != nil {
		return ""
	}

	// /proc/1/environ berisi pasangan KEY=VALUE yang dipisah byte NUL (\x00),
	// bukan newline seperti file teks biasa.
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

	// Kalau keluar dalam kondisi error yang belum terselesaikan (misal user Ctrl+C
	// saat masih di layar error/loading), jangan lanjut ke shell, tutup sesi saja.
	m := finalModel.(model)
	if m.state == screenError && m.envInfo == nil {
		fmt.Println("Sesi ditutup.")
		os.Exit(1)
	}

	execShell()
}

// execShell menggantikan proses TUI dengan shell login user, seperti proses
// TUI "tidak pernah ada" begitu praktikan lanjut ke sesi kerja normal.
func execShell() {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	env := os.Environ()
	if err := syscall.Exec(shell, []string{shell, "-l"}, env); err != nil {
		fmt.Fprintln(os.Stderr, "Gagal membuka shell:", err)
		os.Exit(1)
	}
}