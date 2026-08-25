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
	screenError
)

type model struct {
	client *APIClient

	state   screenState
	envInfo *EnvInfo
	errMsg  string

	// shouldContinueToShell HANYA true kalau alur identifikasi benar-benar
	// selesai dengan sukses. Ctrl+C atau error apapun TIDAK mengubah flag
	// ini, sehingga main() tahu persis kapan boleh exec ke shell dan kapan
	// harus menutup sesi begitu saja.
	shouldContinueToShell bool

	inputNama textinput.Model
	inputNPM  textinput.Model
	spin      spinner.Model

	windowWidth  int
	windowHeight int
}

// ==================== STYLES ====================
// Skema warna sengaja dibuat konsisten (aksen biru/cyan) dan minim —
// styling berbasis teks seperti ini praktis tidak menambah beban CPU/memory
// dibanding tampilan polos, karena tetap murni rendering teks di terminal.

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

	labelStyle = lipgloss.NewStyle().Foreground(accentDim)
	valueStyle = lipgloss.NewStyle().Bold(true)
	hintStyle  = lipgloss.NewStyle().Foreground(muted).Italic(true)
	errStyle   = lipgloss.NewStyle().Foreground(danger).Bold(true)
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
	nama.PromptStyle = lipgloss.NewStyle().Foreground(accent)

	npm := textinput.New()
	npm.Placeholder = "NPM"
	npm.CharLimit = 30
	npm.Width = 40
	npm.PromptStyle = lipgloss.NewStyle().Foreground(accent)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(accent)

	return model{
		client:    client,
		state:     screenLoading,
		inputNama: nama,
		inputNPM:  npm,
		spin:      sp,
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
			// shouldContinueToShell TETAP false di sini -> main() akan
			// menutup sesi, BUKAN melanjutkan ke shell.
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
		// Sukses (baik pengisian baru maupun verifikasi identitas yang
		// cocok) -> baru di sini flag diizinkan lanjut ke shell.
		m.shouldContinueToShell = true
		return m, tea.Quit
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {

	case screenDashboard:
		if msg.String() == "enter" {
			// Selalu minta identifikasi/verifikasi, TIDAK ADA jalan pintas
			// otomatis lolos walau environment ini sudah pernah diisi
			// sebelumnya — praktikan lain yang menggunakan PC yang sama
			// wajib gagal verifikasi kalau NPM-nya berbeda.
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
			m.state = screenLoading
			m.inputNama.SetValue("")
			m.inputNPM.SetValue("")
			return m, fetchEnvInfoCmd(m.client)
		}
		return m, nil
	}

	return m, nil
}

// ==================== VIEW ====================

// renderBox membungkus konten dalam box bergaya, dengan lebar yang
// menyesuaikan ukuran terminal (responsive) dan diposisikan di tengah layar
// kalau ukuran terminal sudah diketahui.
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

	box := boxStyle.Width(width).Render(content)

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
			"%s\n\n%s\n%s\n\n%s",
			titleStyle.Render("Identifikasi Praktikan"),
			labelStyle.Render(hint),
			m.inputNama.View(),
			hintStyle.Render("[Enter] lanjut  •  [Ctrl+C] batal"),
		))

	case screenInputNPM:
		return m.renderBox(fmt.Sprintf(
			"%s\n\n%s %s\n\n%s\n%s\n\n%s",
			titleStyle.Render("Identifikasi Praktikan"),
			labelStyle.Render("Nama:"), valueStyle.Render(m.inputNama.Value()),
			labelStyle.Render("Masukkan NPM:"),
			m.inputNPM.View(),
			hintStyle.Render("[Enter] kirim  •  [Esc] kembali  •  [Ctrl+C] batal"),
		))

	case screenSubmitting:
		return m.renderBox(fmt.Sprintf("%s Memverifikasi identitas...", m.spin.View()))

	case screenError:
		return m.renderBox(fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			errStyle.Render("⚠ Terjadi Kesalahan"),
			m.errMsg,
			hintStyle.Render("[Enter] coba lagi  •  [Ctrl+C] keluar"),
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

	divider := lipgloss.NewStyle().Foreground(muted).Render(strings.Repeat("─", 40))

	body := fmt.Sprintf(
		"%s\n%s\n%s\n\n%s %s\n%s %s\n%s %s\n%s %d\n%s %s\n%s %s\n%s %s\n\n%s",
		titleStyle.Render("📋 Dashboard Sesi Praktikum"),
		subtitleStyle.Render("Sistem Manajemen Environment Praktikum"),
		divider,
		labelStyle.Render("Kode Kursus  :"), valueStyle.Render(info.CourseCode),
		labelStyle.Render("Modul        :"), valueStyle.Render(info.Module),
		labelStyle.Render("Ruangan      :"), valueStyle.Render(info.Room),
		labelStyle.Render("Pertemuan ke :"), info.MeetingNumber,
		labelStyle.Render("Tanggal      :"), valueStyle.Render(info.SessionDate),
		labelStyle.Render("Status Env   :"), valueStyle.Render(info.Status),
		labelStyle.Render("Identifikasi :"), identStatus,
		hintStyle.Render("[Enter] lanjutkan  •  [Ctrl+C] keluar"),
	)

	return body
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

	// Titik keputusan tunggal: HANYA lanjut ke shell kalau alur identifikasi
	// benar-benar selesai sukses. Ctrl+C kapan pun (di layar manapun) akan
	// selalu berakhir di sini dengan shouldContinueToShell == false.
	if !m.shouldContinueToShell {
		fmt.Println("Sesi ditutup.")
		os.Exit(0)
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