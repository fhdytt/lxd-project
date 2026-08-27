package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	badgeStyle = lipgloss.NewStyle().
			Foreground(textLight).
			Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(textLight).
				Background(accent).
				Padding(0, 1)

	inactiveItemStyle = lipgloss.NewStyle().
				Foreground(textSoft).
				Padding(0, 1)

	headerTableStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(textLight).
				Background(bgDark).
				Padding(0, 1)

	statusRunningStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(textLight).
				Background(accent).
				Padding(0, 1)

	statusStoppedStyle = lipgloss.NewStyle().
				Foreground(textSoft).
				Background(bgMid).
				Padding(0, 1)

	dividerStyle = lipgloss.NewStyle().
			Foreground(accent)
)

func (m model) View() string {
	var content string

	switch m.state {
	case screenListEnvironments:
		content = m.viewEnvironmentTable()
	case screenRunningCommand:
		content = fmt.Sprintf(
			"%s  Menjalankan script kelola-lxd.sh, mohon tunggu...",
			m.spin.View(),
		)
	case screenCommandResult:
		content = m.viewCommandResult()
	case screenError:
		content = fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			errStyle.Render("ERROR SYSTEM"),
			m.errMsg,
			hintStyle.Render("[Enter] Kembali ke menu utama"),
		)
	default:
		if isMenuScreen(m.state) {
			content = m.viewMenu()
		}
	}

	return boxStyle.Render(content)
}

func (m model) viewMenu() string {
	var b strings.Builder

	// 1. Tampilkan Logo ASCII LepKom Gunadarma
	b.WriteString(logoStyle.Render(asciiLogo))
	b.WriteString("\n")

	// 2. Header Banner & Breadcrumb
	b.WriteString(titleStyle.Render("CONTAINER ADMINISTRATOR"))
	b.WriteString("\n")
	b.WriteString(badgeStyle.Render("LOCATION: " + strings.ToUpper(m.breadcrumb())))
	b.WriteString("\n\n")

	// 3. Menu Counter (Interaktif: Posisi cursor)
	totalItems := len(m.menuItems)
	if totalItems == 0 {
		b.WriteString(fmt.Sprintf("%s Memuat data...", m.spin.View()))
	} else {
		b.WriteString(dimStyle.Render(fmt.Sprintf("Pilihan Menu (%d/%d):", m.menuCursor+1, totalItems)))
		b.WriteString("\n\n")
	}

	// 4. Dynamic Menu Items dengan Highlight Bar
	for i, item := range m.menuItems {
		if i == m.menuCursor {
			itemText := fmt.Sprintf("[-] %02d. %s", i+1, item)
			b.WriteString(selectedItemStyle.Render(itemText))
		} else {
			itemText := fmt.Sprintf("    %02d. %s", i+1, item)
			b.WriteString(inactiveItemStyle.Render(itemText))
		}
		b.WriteString("\n")
	}

	// 5. Footer Control Bar
	b.WriteString("\n")
	b.WriteString(dividerStyle.Render(strings.Repeat("─", 54)))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("[UP/DOWN] Pilih  |  [ENTER] Eksekusi  |  [ESC] Kembali "))

	return b.String()
}

func (m model) breadcrumb() string {
	switch m.state {
	case screenMainMenu:
		return "Menu Utama"
	case screenPickRoomForList:
		return "Pilih Ruangan"
	case screenPickRoomForProvision:
		return "Pilih Ruangan"
	case screenPickProvisionAction:
		return fmt.Sprintf("%s > Aksi", m.selectedRoom)
	case screenPickModuleForStart:
		return fmt.Sprintf("%s > Pilih Modul", m.selectedRoom)
	case screenConfirmProvision:
		if m.selectedAction == "start" {
			return fmt.Sprintf("Konfirmasi > START %s [%s]", m.selectedRoom, m.selectedModule)
		}
		return fmt.Sprintf("Konfirmasi > STOP ALL [%s]", m.selectedRoom)
	case screenPickRoomForReset:
		return "Pilih Ruangan"
	case screenPickResetMode:
		return fmt.Sprintf("%s > Mode Reset", m.selectedRoom)
	case screenPickContainerForReset:
		return fmt.Sprintf("%s > Pilih Container", m.selectedRoom)
	case screenConfirmReset:
		if m.selectedResetMode == "room" {
			return fmt.Sprintf("Konfirmasi > RESET ALL [%s]", m.selectedRoom)
		}
		return fmt.Sprintf("Konfirmasi > RESET [%s]", m.selectedContainer)
	}
	return ""
}

func (m model) viewEnvironmentTable() string {
	var b strings.Builder

	// Header & Logo
	b.WriteString(logoStyle.Render(asciiLogo))
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("DAFTAR ENVIRONMENT "))
	b.WriteString("\n\n")
	b.WriteString(badgeStyle.Render("RUANGAN: "+m.selectedRoom))
	b.WriteString("\n\n")

	if len(m.envRows) == 0 {
		b.WriteString(hintStyle.Render(" Info: Tidak ada container aktif di ruangan ini"))
		b.WriteString("\n")
	} else {
		// Informasi Total Data
		b.WriteString(dimStyle.Render(fmt.Sprintf("Total Container Aktif: %d", len(m.envRows))))
		b.WriteString("\n\n")

		// Header Tabel
		header := fmt.Sprintf("%-18s %-8s %-12s %-12s %-20s %s", "CONTAINER", "PORT", "STATUS", "SNAPSHOT", "PRAKTIKAN", "NPM")
		b.WriteString(headerTableStyle.Render(header) + "\n")

		// Body Tabel
		for _, e := range m.envRows {
			nama := e.PraktikanNama
			if nama == "" {
				nama = hintStyle.Render("(kosong)")
			}

			snap := dimStyle.Render("[-] Tidak")
			if e.HasCleanSnapshot {
				snap = okStyle.Render("[+] Ada")
			}

			var statusFormatted string
			if strings.ToLower(e.Status) == "running" {
				statusFormatted = statusRunningStyle.Render("RUNNING")
			} else {
				statusFormatted = statusStoppedStyle.Render("STOPPED")
			}

			b.WriteString(fmt.Sprintf(
				"%-18s %-8d %-12s %-12s %-22s %-s\n",
				cursorStyle.Render(e.ContainerName),
				e.SSHPort,
				statusFormatted,
				snap,
				nama,
				e.PraktikanNPM,
			))
		}
	}

	b.WriteString("\n")
	b.WriteString(dividerStyle.Render(strings.Repeat("─", 78)))
	b.WriteString("\n")
	b.WriteString(hintStyle.Render(" Tekan [ENTER / ESC] untuk kembali ke menu utama "))
	return b.String()
}

func (m model) viewCommandResult() string {
	status := okStyle.Render("[ SUKSES ]")
	if m.commandFailed {
		status = errStyle.Render("[ GAGAL ]")
	}

	return fmt.Sprintf(
		"%s\n\nSTATUS: %s\n\n%s\n\n%s\n%s",
		titleStyle.Render(" LOG HASIL EKSEKUSI "),
		status,
		m.commandOutput,
		dividerStyle.Render(strings.Repeat("─", 50)),
		hintStyle.Render(" Tekan [ENTER / ESC] untuk kembali ke menu utama "),
	)
}

func roomNames(rooms []Room) []string {
	names := make([]string, len(rooms))
	for i, r := range rooms {
		names[i] = r.Nama
	}
	return names
}

func moduleLabels(modules []Module) []string {
	labels := make([]string, len(modules))
	for i, mo := range modules {
		labels[i] = fmt.Sprintf("[%s] %s", mo.Code, mo.Name)
	}
	return labels
}