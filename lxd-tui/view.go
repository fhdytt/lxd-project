package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderBox membungkus logo LEPKOM + konten dalam satu box bergaya, dengan
// lebar yang menyesuaikan ukuran terminal (responsive) dan diposisikan di
// tengah layar kalau ukuran terminal sudah diketahui.
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
	if logoWidth := lipgloss.Width(lepkomLogo); width < logoWidth {
		width = logoWidth
	}

	full := renderLogo() + "\n\n" + content
	box := boxStyle.Width(width).Render(full)

	if m.windowWidth > 0 && m.windowHeight > 0 {
		return lipgloss.Place(m.windowWidth, m.windowHeight, lipgloss.Center, lipgloss.Center, box)
	}
	return box
}

// View merender tampilan terminal berdasarkan state saat ini.
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
			titleStyle.Render("IDENTIFIKASI PRAKTIKAN"),
			labelStyle.Render(hint),
			m.inputNama.View(),
			hintStyle.Render("[Enter] lanjut  •  [Ctrl+C] batal"),
		))

	case screenInputNPM:
		return m.renderBox(fmt.Sprintf(
			"%s\n\n%s %s\n\n%s\n%s\n\n%s",
			titleStyle.Render("IDENTIFIKASI PRAKTIKAN"),
			labelStyle.Render("Nama:"), valueStyle.Render(m.inputNama.Value()),
			labelStyle.Render("Masukkan NPM:"),
			m.inputNPM.View(),
			hintStyle.Render("[Enter] kirim  •  [Esc] kembali  •  [Ctrl+C] batal"),
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
			"%s\n\n%s %s\n\n%s\n%s\n%s\n%s",
			titleStyle.Render("PRESS START: "+m.selectedUsername),
			labelStyle.Render("User:"), valueStyle.Render(m.selectedUsername),
			m.inputPassword.View(),
			errLine,
			hintStyle.Render("[Enter] masuk  •  [Esc] ganti user  •  [Ctrl+C] batal"),
			"",
		))

	case screenError:
		return m.renderBox(fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			errStyle.Render("Terjasdi kesalahan:"),
			m.errMsg,
			hintStyle.Render("[Enter] coba lagi  •  [Ctrl+C] keluar"),
		))
	}
	return ""
}

func (m model) viewDashboard() string {
	info := m.envInfo

	identStatus := hintStyle.Render("Belum diisi")
	if info.AlreadyIdentified {
		identStatus = lipgloss.NewStyle().Bold(true).Foreground(gameboyGreen).Render("Sudah terdaftar (perlu verifikasi)")
	}

	divider := lipgloss.NewStyle().Foreground(snesPurple).Render(strings.Repeat("═", 40))

	body := fmt.Sprintf(
		"%s\n%s\n%s\n\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n\n%s",
		titleStyle.Render("DASHBOARD SESI PRAKTIKUM"),
		subtitleStyle.Render("Sistem Manajemen Environment Praktikum"),
		divider,
		labelStyle.Render("Kode Kursus  :"), valueStyle.Render(info.CourseCode),
		labelStyle.Render("Modul        :"), valueStyle.Render(info.Module),
		labelStyle.Render("Ruangan      :"), valueStyle.Render(info.Room),
		labelStyle.Render("Pertemuan ke :"), valueStyle.Render(fmt.Sprintf("%d", info.MeetingNumber)),
		labelStyle.Render("Tanggal      :"), valueStyle.Render(info.SessionDate),
		labelStyle.Render("Status Env   :"), lipgloss.NewStyle().Bold(true).Foreground(gameboyGreen).Render(info.Status),
		labelStyle.Render("Identifikasi :"), identStatus,
		hintStyle.Render("PRESS [ENTER] TO CONTINUE  •  [CTRL+C] EXIT"),
	)

	return body
}

func (m model) viewSelectUser() string {
	if len(m.localUsers) == 0 {
		return fmt.Sprintf("%s Memuat daftar user...", m.spin.View())
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("SELECT USER"))
	b.WriteString("\n\n")

	for i, u := range m.localUsers {
		if i == m.userCursor {
			b.WriteString(cursorStyle.Render("> " + u.Username))
		} else {
			b.WriteString(menuItemDim.Render("  " + u.Username))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(hintStyle.Render("[↑/↓] pilih  •  [Enter] konfirmasi  •  [Ctrl+C] batal"))
	return b.String()
}