package main

import (
	"fmt"
	"strings"
)

// View (bagian dari interface tea.Model) — merender string yang ditampilkan
// ke terminal, murni berdasarkan isi model, tidak pernah mengubah state apapun.
func (m model) View() string {
	switch m.state {
	case screenListEnvironments:
		return boxStyle.Render(m.viewEnvironmentTable())
	case screenRunningCommand:
		return boxStyle.Render(fmt.Sprintf("%s Menjalankan kelola-lxd.sh, mohon tunggu...", m.spin.View()))
	case screenCommandResult:
		return boxStyle.Render(m.viewCommandResult())
	case screenError:
		return boxStyle.Render(fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			errStyle.Render("⚠ Terjadi Kesalahan"),
			m.errMsg,
			hintStyle.Render("[Enter] kembali ke menu utama"),
		))
	}

	if isMenuScreen(m.state) {
		return boxStyle.Render(m.viewMenu())
	}
	return ""
}

func (m model) viewMenu() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("lxd-control — Administrator"))
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render(m.breadcrumb()))
	b.WriteString("\n\n")

	if len(m.menuItems) == 0 {
		b.WriteString(fmt.Sprintf("%s Memuat...", m.spin.View()))
	}

	for i, item := range m.menuItems {
		if i == m.menuCursor {
			b.WriteString(cursorStyle.Render("› " + item))
		} else {
			b.WriteString(dimStyle.Render("  " + item))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(hintStyle.Render("[↑/↓] pilih  •  [Enter] konfirmasi  •  [Esc] kembali  •  [Ctrl+C] keluar"))
	return b.String()
}

// breadcrumb menampilkan "kamu sedang di mana" — penting karena banyak
// layar berbagi tampilan menu generik yang sama (viewMenu), breadcrumb ini
// yang membedakan konteksnya secara visual.
func (m model) breadcrumb() string {
	switch m.state {
	case screenMainMenu:
		return "Menu Utama"
	case screenPickRoomForList:
		return "Lihat Daftar Environment › Pilih Ruangan"
	case screenPickRoomForProvision:
		return "Provisioning Ruangan › Pilih Ruangan"
	case screenPickProvisionAction:
		return fmt.Sprintf("Provisioning Ruangan › %s › Start/Stop?", m.selectedRoom)
	case screenPickModuleForStart:
		return fmt.Sprintf("Provisioning Ruangan › %s › Pilih Modul", m.selectedRoom)
	case screenConfirmProvision:
		if m.selectedAction == "start" {
			return fmt.Sprintf("Konfirmasi: START ruangan %s, modul %s?", m.selectedRoom, m.selectedModule)
		}
		return fmt.Sprintf("Konfirmasi: STOP semua container di ruangan %s?", m.selectedRoom)
	case screenPickRoomForReset:
		return "Reset Environment › Pilih Ruangan"
	case screenPickResetMode:
		return fmt.Sprintf("Reset Environment › %s › Mode Reset", m.selectedRoom)
	case screenPickContainerForReset:
		return fmt.Sprintf("Reset Environment › %s › Pilih Container", m.selectedRoom)
	case screenConfirmReset:
		if m.selectedResetMode == "room" {
			return fmt.Sprintf("Konfirmasi: RESET seluruh ruangan %s?", m.selectedRoom)
		}
		return fmt.Sprintf("Konfirmasi: RESET container %s?", m.selectedContainer)
	}
	return ""
}

func (m model) viewEnvironmentTable() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Daftar Environment — " + m.selectedRoom))
	b.WriteString("\n\n")

	if len(m.envRows) == 0 {
		b.WriteString(labelStyle.Render("Tidak ada environment aktif di ruangan ini."))
	} else {
		b.WriteString(fmt.Sprintf("%-16s %-6s %-10s %-8s %-20s %s\n", "CONTAINER", "PORT", "STATUS", "SNAPSHOT", "PRAKTIKAN", "NPM"))
		b.WriteString(strings.Repeat("─", 78) + "\n")
		for _, e := range m.envRows {
			nama := e.PraktikanNama
			if nama == "" {
				nama = "(belum diisi)"
			}
			snap := "tidak"
			if e.HasCleanSnapshot {
				snap = "ada"
			}
			b.WriteString(fmt.Sprintf("%-16s %-6d %-10s %-8s %-20s %s\n", e.ContainerName, e.SSHPort, e.Status, snap, nama, e.PraktikanNPM))
		}
	}

	b.WriteString("\n")
	b.WriteString(hintStyle.Render("[Enter/Esc] kembali ke menu utama"))
	return b.String()
}

func (m model) viewCommandResult() string {
	status := okStyle.Render("✓ Berhasil")
	if m.commandFailed {
		status = errStyle.Render("✗ Gagal (cek output di bawah)")
	}
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s",
		titleStyle.Render("Hasil Eksekusi"),
		status,
		m.commandOutput,
		hintStyle.Render("[Enter/Esc] kembali ke menu utama"),
	)
}

// roomNames dan moduleLabels mengubah data dari database (db.go) menjadi
// daftar string biasa untuk ditampilkan sebagai pilihan menu.
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
		labels[i] = fmt.Sprintf("%s (%s)", mo.Code, mo.Name)
	}
	return labels
}