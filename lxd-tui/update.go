package main

import (
	"errors"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Update (bagian dari interface tea.Model) — satu-satunya tempat state
// model boleh berubah, sesuai pola Bubble Tea/Elm Architecture.
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

// handleKey menangani semua input keyboard selain Ctrl+C (yang sudah
// ditangani langsung di Update). Satu case per layar — pola ini konsisten
// dengan lxd-control, memudahkan kalau nanti kedua project di-maintain
// oleh orang yang sama.
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