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
		return boxStyle.Render(fmt.Sprintf("%s Memproses, mohon tunggu...", m.spin.View()))
	case screenCommandResult:
		return boxStyle.Render(m.viewCommandResult())
	case screenError:
		return boxStyle.Render(fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			errStyle.Render("Terjadi Kesalahan"),
			m.errMsg,
			hintStyle.Render("[Enter] kembali"),
		))
	case screenRoomsList:
		return boxStyle.Render(m.viewRoomsTable())
	case screenSessionsList:
		return boxStyle.Render(m.viewSessionsTable())
	case screenRoomFormNama, screenRoomFormPortPrefix, screenRoomFormCapacity:
		return boxStyle.Render(m.viewRoomForm())
	case screenSessionFormCourseCode, screenSessionFormMeetingNumber, screenSessionFormDate:
		return boxStyle.Render(m.viewSessionForm())
	}

	if isMenuScreen(m.state) {
		return boxStyle.Render(m.viewMenu())
	}
	return ""
}

func (m model) viewMenu() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("lxd-Administrator"))
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

	case screenRoomsMenu:
		return "Kelola Ruangan"
	case screenRoomPickForEdit:
		return "Kelola Ruangan › Pilih untuk Diedit"
	case screenRoomPickForDelete:
		return "Kelola Ruangan › Pilih untuk Dihapus"
	case screenRoomDeleteConfirm:
		return fmt.Sprintf(
			"Konfirmasi: HAPUS ruangan %s?\n(Akan GAGAL kalau masih ada sesi yang memakai ruangan ini — itu perilaku yang benar, bukan bug.)",
			m.editingRoomOriginalNama,
		)

	case screenSessionsMenu:
		return "Kelola Sesi"
	case screenSessionPickRoom:
		return "Kelola Sesi › Tambah Sesi › Pilih Ruangan"
	case screenSessionPickModule:
		return fmt.Sprintf("Kelola Sesi › Tambah Sesi › %s › Pilih Modul", m.sessionRoom)
	case screenSessionPickStatus:
		return "Kelola Sesi › Pilih Status"
	case screenSessionPickForEdit:
		return "Kelola Sesi › Pilih untuk Diedit"
	case screenSessionPickForDelete:
		return "Kelola Sesi › Pilih untuk Dihapus"
	case screenSessionDeleteConfirm:
		return "Konfirmasi: HAPUS sesi ini?\nSEMUA environment yang terhubung ke sesi ini akan IKUT TERHAPUS dari database (container LXD-nya sendiri TIDAK ikut terhapus otomatis — jadi bisa jadi 'yatim', tidak lagi tercatat)."
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
	status := okStyle.Render("Berhasil")
	if m.commandFailed {
		status = errStyle.Render("Gagal (cek pesan di bawah)")
	}
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s",
		titleStyle.Render("Hasil"),
		status,
		m.commandOutput,
		hintStyle.Render("[Enter/Esc] kembali"),
	)
}

func (m model) viewRoomsTable() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Daftar Ruangan"))
	b.WriteString("\n\n")

	if len(m.roomsDetailed) == 0 {
		b.WriteString(labelStyle.Render("Belum ada ruangan."))
	} else {
		b.WriteString(fmt.Sprintf("%-10s %-14s %s\n", "NAMA", "PORT PREFIX", "KAPASITAS"))
		b.WriteString(strings.Repeat("─", 40) + "\n")
		for _, r := range m.roomsDetailed {
			b.WriteString(fmt.Sprintf("%-10s %-14s %d\n", r.Nama, r.PortPrefix, r.Capacity))
		}
	}

	b.WriteString("\n")
	b.WriteString(hintStyle.Render("[Enter/Esc] kembali"))
	return b.String()
}

func (m model) viewSessionsTable() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Daftar Sesi"))
	b.WriteString("\n\n")

	if len(m.sessionsDetailed) == 0 {
		b.WriteString(labelStyle.Render("Belum ada sesi."))
	} else {
		b.WriteString(fmt.Sprintf("%-8s %-16s %-10s %-4s %-12s %s\n", "RUANGAN", "COURSE CODE", "MODUL", "KE-", "TANGGAL", "STATUS"))
		b.WriteString(strings.Repeat("─", 72) + "\n")
		for _, s := range m.sessionsDetailed {
			b.WriteString(fmt.Sprintf("%-8s %-16s %-10s %-4d %-12s %s\n", s.RoomNama, s.CourseCode, s.ModuleCode, s.MeetingNumber, s.SessionDate, s.Status))
		}
	}

	b.WriteString("\n")
	b.WriteString(hintStyle.Render("[Enter/Esc] kembali"))
	return b.String()
}

func (m model) viewRoomForm() string {
	title := "Tambah Ruangan"
	if m.roomFormMode == "update" {
		title = "Edit Ruangan: " + m.editingRoomOriginalNama
	}
	return fmt.Sprintf(
		"%s\n\n%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s",
		titleStyle.Render(title),
		labelStyle.Render("Nama ruangan:"), m.roomInputNama.View(),
		labelStyle.Render("Port prefix (2 digit):"), m.roomInputPortPrefix.View(),
		labelStyle.Render("Kapasitas:"), m.roomInputCapacity.View(),
		hintStyle.Render("[Enter] lanjut  •  [Esc] kembali  •  [Ctrl+C] keluar"),
	)
}

func (m model) viewSessionForm() string {
	title := "Tambah Sesi"
	if m.sessionFormMode == "update" {
		title = "Edit Sesi"
	}

	context := ""
	if m.sessionFormMode == "create" {
		context = fmt.Sprintf(
			"%s %s   %s %s\n\n",
			labelStyle.Render("Ruangan:"), m.sessionRoom,
			labelStyle.Render("Modul:"), m.sessionModule,
		)
	}

	return fmt.Sprintf(
		"%s\n\n%s%s\n%s\n\n%s\n%s\n\n%s\n%s\n\n%s",
		titleStyle.Render(title),
		context,
		labelStyle.Render("Course code :"), m.sessionInputCourseCode.View(),
		labelStyle.Render("Pertemuan ke-:"), m.sessionInputMeetingNumber.View(),
		labelStyle.Render("Tanggal (format YYYY-MM-DD):"), m.sessionInputDate.View(),
		hintStyle.Render("[Enter] lanjut  •  [Esc] kembali  •  [Ctrl+C] keluar"),
	)
}

// roomNames, moduleLabels, roomDetailNames, sessionMenuLabelsAndIDs mengubah
// data dari database (db.go) menjadi daftar string biasa untuk ditampilkan
// sebagai pilihan menu.

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

func roomDetailNames(rooms []RoomDetail) []string {
	names := make([]string, len(rooms))
	for i, r := range rooms {
		names[i] = fmt.Sprintf("%s (port %s, kapasitas %d)", r.Nama, r.PortPrefix, r.Capacity)
	}
	return names
}

// sessionMenuLabelsAndIDs mengembalikan 2 slice paralel: label yang
// ditampilkan di menu, dan ID sesungguhnya (UUID) yang dipakai saat submit
// aksi edit/hapus — supaya menu tidak perlu menampilkan UUID mentah ke admin.
func sessionMenuLabelsAndIDs(sessions []SessionDetail) ([]string, []string) {
	labels := make([]string, len(sessions))
	ids := make([]string, len(sessions))
	for i, s := range sessions {
		labels[i] = fmt.Sprintf("%s | %s pertemuan %d | %s | %s", s.RoomNama, s.CourseCode, s.MeetingNumber, s.SessionDate, s.Status)
		ids[i] = s.ID
	}
	return labels, ids
}