package main

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Update (bagian dari interface tea.Model) — satu-satunya tempat state
// model boleh berubah, sesuai pola Bubble Tea/Elm Architecture.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m.handleKey(msg)

	case roomsLoadedMsg:
		if msg.err != nil {
			return m.toError(msg.err)
		}
		m.rooms = msg.rooms
		m.menuItems = roomNames(msg.rooms)
		m.menuCursor = 0
		return m, nil

	case modulesLoadedMsg:
		if msg.err != nil {
			return m.toError(msg.err)
		}
		m.modules = msg.modules
		m.menuItems = moduleLabels(msg.modules)
		m.menuCursor = 0
		return m, nil

	case environmentsLoadedMsg:
		if msg.err != nil {
			return m.toError(msg.err)
		}
		m.envRows = msg.rows
		return m, nil

	case containersLoadedMsg:
		if msg.err != nil {
			return m.toError(msg.err)
		}
		if len(msg.names) == 0 {
			m.errMsg = "Tidak ada environment di ruangan ini."
			m.state = screenError
			return m, nil
		}
		m.menuItems = msg.names
		m.menuCursor = 0
		m.state = screenPickContainerForReset
		return m, nil

	case commandDoneMsg:
		m.commandOutput = msg.output
		m.commandFailed = msg.err != nil
		m.state = screenCommandResult
		return m, nil

	case roomsDetailedLoadedMsg:
		if msg.err != nil {
			return m.toError(msg.err)
		}
		m.roomsDetailed = msg.rooms
		// Kalau sedang di layar pilih-untuk-edit/hapus, ubah jadi daftar
		// nama untuk ditampilkan sebagai menu. Kalau sedang di layar lihat
		// daftar biasa, dibiarkan (viewRoomsTable() baca m.roomsDetailed langsung).
		if m.state == screenRoomPickForEdit || m.state == screenRoomPickForDelete {
			m.menuItems = roomDetailNames(msg.rooms)
			m.menuCursor = 0
		}
		return m, nil

	case sessionsLoadedMsg:
		if msg.err != nil {
			return m.toError(msg.err)
		}
		m.sessionsDetailed = msg.sessions
		if m.state == screenSessionPickForEdit || m.state == screenSessionPickForDelete {
			m.menuItems, m.sessionPickIDs = sessionMenuLabelsAndIDs(msg.sessions)
			m.menuCursor = 0
		}
		return m, nil

	case sessionsForProvisionLoadedMsg:
		if msg.err != nil {
			return m.toError(msg.err)
		}
		if len(msg.sessions) == 0 {
			m.errMsg = fmt.Sprintf(
				"Tidak ada sesi berstatus scheduled/active untuk ruangan %s modul %s.\nBuat dulu sesinya lewat menu 'Kelola Sesi' > 'Tambah Sesi'.",
				m.selectedRoom, m.selectedModule,
			)
			m.state = screenError
			m.returnTo = model.backToMainMenu
			return m, nil
		}
		labels := make([]string, len(msg.sessions))
		ids := make([]string, len(msg.sessions))
		for i, s := range msg.sessions {
			labels[i] = fmt.Sprintf("%s | pertemuan %d | %s | %s", s.CourseCode, s.MeetingNumber, s.SessionDate, s.Status)
			ids[i] = s.ID
		}
		m.menuItems = labels
		m.provisionSessionIDs = ids
		m.menuCursor = 0
		return m, nil

	case roomMutatedMsg:
		if msg.err != nil {
			m.commandOutput = "Gagal: " + msg.err.Error()
			m.commandFailed = true
		} else {
			m.commandOutput = "Perubahan ruangan berhasil disimpan."
			m.commandFailed = false
		}
		m.state = screenCommandResult
		return m, nil

	case sessionMutatedMsg:
		if msg.err != nil {
			m.commandOutput = "Gagal: " + msg.err.Error()
			m.commandFailed = true
		} else {
			m.commandOutput = "Perubahan sesi berhasil disimpan."
			m.commandFailed = false
		}
		m.state = screenCommandResult
		return m, nil
	}

	return m, nil
}

// handleKey menangani semua input keyboard selain Ctrl+C (yang sudah
// ditangani langsung di Update). Navigasi menu digeneralisasi lewat
// isMenuScreen(), layar form text-input ditangani terpisah di bawahnya.
func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if isMenuScreen(m.state) {
		switch key {
		case "up", "k":
			if m.menuCursor > 0 {
				m.menuCursor--
			}
			return m, nil
		case "down", "j":
			if m.menuCursor < len(m.menuItems)-1 {
				m.menuCursor++
			}
			return m, nil
		case "esc":
			return m.backOrMainMenu()
		case "enter":
			return m.onMenuSelect()
		}
		return m, nil
	}

	switch m.state {
	case screenListEnvironments, screenCommandResult, screenRoomsList, screenSessionsList, screenError:
		if key == "enter" || key == "esc" {
			if m.returnTo != nil {
				return m.returnTo(m)
			}
			return m.backToMainMenu()
		}
		return m, nil

	case screenRoomFormNama:
		return m.handleRoomFormNama(msg)
	case screenRoomFormPortPrefix:
		return m.handleRoomFormPortPrefix(msg)
	case screenRoomFormCapacity:
		return m.handleRoomFormCapacity(msg)

	case screenSessionFormCourseCode:
		return m.handleSessionFormCourseCode(msg)
	case screenSessionFormMeetingNumber:
		return m.handleSessionFormMeetingNumber(msg)
	case screenSessionFormDate:
		return m.handleSessionFormDate(msg)
	}

	return m, nil
}

func isMenuScreen(s screenState) bool {
	switch s {
	case screenMainMenu, screenPickRoomForList, screenPickRoomForProvision,
		screenPickProvisionAction, screenPickModuleForStart, screenPickSessionForProvision, screenConfirmProvision,
		screenPickRoomForReset, screenPickResetMode, screenPickContainerForReset,
		screenConfirmReset,
		screenRoomsMenu, screenRoomPickForEdit, screenRoomPickForDelete, screenRoomDeleteConfirm,
		screenSessionsMenu, screenSessionPickRoom, screenSessionPickModule, screenSessionPickStatus,
		screenSessionPickForEdit, screenSessionPickForDelete, screenSessionDeleteConfirm:
		return true
	}
	return false
}

// backOrMainMenu dipakai saat Esc ditekan di layar menu — kembali ke
// returnTo kalau ada (misal dari dalam submenu Kelola Ruangan), atau ke
// menu utama kalau tidak ada konteks khusus.
func (m model) backOrMainMenu() (tea.Model, tea.Cmd) {
	if m.returnTo != nil {
		return m.returnTo(m)
	}
	return m.backToMainMenu()
}

// onMenuSelect menangani Enter di layar menu manapun — transisi state
// tergantung layar SAAT INI dan item yang sedang di-highlight (m.menuCursor).
// Ini "otak" alur navigasi seluruh aplikasi.
func (m model) onMenuSelect() (tea.Model, tea.Cmd) {
	selected := m.menuItems[m.menuCursor]

	switch m.state {
	case screenMainMenu:
		switch m.menuCursor {
		case 0:
			m.state = screenPickRoomForList
			m.returnTo = model.backToMainMenu
			return m, loadRoomsCmd(m.db)
		case 1:
			m.state = screenPickRoomForProvision
			return m, loadRoomsCmd(m.db)
		case 2:
			m.state = screenPickRoomForReset
			return m, loadRoomsCmd(m.db)
		case 3:
			return m.gotoRoomsMenu()
		case 4:
			return m.gotoSessionsMenu()
		case 5:
			return m, tea.Quit
		}

	case screenPickRoomForList:
		m.selectedRoom = selected
		m.state = screenListEnvironments
		m.returnTo = model.backToMainMenu
		return m, loadEnvironmentsCmd(m.db, m.selectedRoom)

	case screenPickRoomForProvision:
		m.selectedRoom = selected
		m.menuItems = []string{"Start (provisioning baru)", "Stop (hapus semua container)"}
		m.menuCursor = 0
		m.state = screenPickProvisionAction
		return m, nil

	case screenPickProvisionAction:
		if m.menuCursor == 0 {
			m.selectedAction = "start"
			m.state = screenPickModuleForStart
			return m, loadModulesCmd(m.db)
		}
		m.selectedAction = "stop"
		m.state = screenConfirmProvision
		m.menuItems = []string{"Ya, jalankan", "Batal"}
		m.menuCursor = 0
		return m, nil

	case screenPickModuleForStart:
		m.selectedModule = m.modules[m.menuCursor].Code
		m.state = screenPickSessionForProvision
		return m, loadSessionsForProvisionCmd(m.db, m.selectedRoom, m.selectedModule)

	case screenPickSessionForProvision:
		m.selectedSessionID = m.provisionSessionIDs[m.menuCursor]
		m.state = screenConfirmProvision
		m.menuItems = []string{"Ya, jalankan", "Batal"}
		m.menuCursor = 0
		return m, nil

	case screenConfirmProvision:
		if m.menuCursor != 0 {
			return m.backToMainMenu()
		}
		m.state = screenRunningCommand
		m.returnTo = model.backToMainMenu
		if m.selectedAction == "start" {
			return m, provisionRoomCmd(m.cfg, m.db, m.selectedRoom, m.selectedModule, m.selectedSessionID)
		}
		return m, stopRoomCmd(m.cfg, m.db, m.selectedRoom)

	case screenPickRoomForReset:
		m.selectedRoom = selected
		m.menuItems = []string{"Reset seluruh ruangan", "Reset 1 container tertentu"}
		m.menuCursor = 0
		m.state = screenPickResetMode
		return m, nil

	case screenPickResetMode:
		if m.menuCursor == 0 {
			m.selectedResetMode = "room"
			m.state = screenConfirmReset
			m.menuItems = []string{"Ya, jalankan", "Batal"}
			m.menuCursor = 0
			return m, nil
		}
		m.selectedResetMode = "container"
		return m, loadContainersCmd(m.db, m.selectedRoom)

	case screenPickContainerForReset:
		m.selectedContainer = selected
		m.state = screenConfirmReset
		m.menuItems = []string{"Ya, jalankan", "Batal"}
		m.menuCursor = 0
		return m, nil

	case screenConfirmReset:
		if m.menuCursor != 0 {
			return m.backToMainMenu()
		}
		m.state = screenRunningCommand
		m.returnTo = model.backToMainMenu
		if m.selectedResetMode == "room" {
			return m, resetRoomCmd(m.cfg, m.db, m.selectedRoom)
		}
		return m, resetContainerCmd(m.cfg, m.db, m.selectedContainer)

	// ---------- Kelola Ruangan ----------

	case screenRoomsMenu:
		switch m.menuCursor {
		case 0:
			m.state = screenRoomsList
			return m, loadRoomsDetailedCmd(m.db)
		case 1:
			m.roomFormMode = "create"
			m.editingRoomOriginalNama = ""
			m.roomInputNama.SetValue("")
			m.roomInputPortPrefix.SetValue("")
			m.roomInputCapacity.SetValue("")
			m.roomInputNama.Focus()
			m.state = screenRoomFormNama
			return m, nil
		case 2:
			m.state = screenRoomPickForEdit
			return m, loadRoomsDetailedCmd(m.db)
		case 3:
			m.state = screenRoomPickForDelete
			return m, loadRoomsDetailedCmd(m.db)
		case 4:
			return m.backToMainMenu()
		}

	case screenRoomPickForEdit:
		room := m.roomsDetailed[m.menuCursor]
		m.roomFormMode = "update"
		m.editingRoomOriginalNama = room.Nama
		m.roomInputNama.SetValue(room.Nama)
		m.roomInputPortPrefix.SetValue(room.PortPrefix)
		m.roomInputCapacity.SetValue(strconv.Itoa(room.Capacity))
		m.roomInputNama.Focus()
		m.state = screenRoomFormNama
		return m, nil

	case screenRoomPickForDelete:
		m.editingRoomOriginalNama = m.roomsDetailed[m.menuCursor].Nama
		m.state = screenRoomDeleteConfirm
		m.menuItems = []string{"Ya, hapus", "Batal"}
		m.menuCursor = 0
		return m, nil

	case screenRoomDeleteConfirm:
		if m.menuCursor != 0 {
			return m.gotoRoomsMenu()
		}
		m.state = screenRunningCommand
		m.returnTo = model.gotoRoomsMenu
		return m, deleteRoomCmd(m.db, m.editingRoomOriginalNama)

	// ---------- Kelola Sesi ----------

	case screenSessionsMenu:
		switch m.menuCursor {
		case 0:
			m.state = screenSessionsList
			return m, loadSessionsCmd(m.db)
		case 1:
			m.sessionFormMode = "create"
			m.editingSessionID = ""
			m.state = screenSessionPickRoom
			return m, loadRoomsCmd(m.db)
		case 2:
			m.state = screenSessionPickForEdit
			return m, loadSessionsCmd(m.db)
		case 3:
			m.state = screenSessionPickForDelete
			return m, loadSessionsCmd(m.db)
		case 4:
			return m.backToMainMenu()
		}

	case screenSessionPickRoom:
		m.sessionRoom = selected
		m.state = screenSessionPickModule
		return m, loadModulesCmd(m.db)

	case screenSessionPickModule:
		m.sessionModule = m.modules[m.menuCursor].Code
		m.sessionInputCourseCode.SetValue("")
		m.sessionInputMeetingNumber.SetValue("")
		m.sessionInputDate.SetValue("")
		m.sessionInputCourseCode.Focus()
		m.state = screenSessionFormCourseCode
		return m, nil

	case screenSessionPickStatus:
		m.sessionStatus = sessionStatusOptions[m.menuCursor]
		meetingNumber, err := strconv.Atoi(m.sessionInputMeetingNumber.Value())
		if err != nil {
			m.errMsg = "Nomor pertemuan harus berupa angka."
			m.state = screenError
			m.returnTo = model.gotoSessionsMenu
			return m, nil
		}
		courseCode := m.sessionInputCourseCode.Value()
		date := m.sessionInputDate.Value()
		m.state = screenRunningCommand
		m.returnTo = model.gotoSessionsMenu
		if m.sessionFormMode == "create" {
			return m, createSessionCmd(m.db, courseCode, m.sessionRoom, m.sessionModule, meetingNumber, date, m.sessionStatus)
		}
		return m, updateSessionCmd(m.db, m.editingSessionID, courseCode, meetingNumber, date, m.sessionStatus)

	case screenSessionPickForEdit:
		session := m.sessionsDetailed[m.menuCursor]
		m.sessionFormMode = "update"
		m.editingSessionID = session.ID
		m.sessionInputCourseCode.SetValue(session.CourseCode)
		m.sessionInputMeetingNumber.SetValue(strconv.Itoa(session.MeetingNumber))
		m.sessionInputDate.SetValue(session.SessionDate)
		m.sessionStatus = session.Status
		m.sessionInputCourseCode.Focus()
		m.state = screenSessionFormCourseCode
		return m, nil

	case screenSessionPickForDelete:
		m.editingSessionID = m.sessionPickIDs[m.menuCursor]
		m.state = screenSessionDeleteConfirm
		m.menuItems = []string{"Ya, hapus", "Batal"}
		m.menuCursor = 0
		return m, nil

	case screenSessionDeleteConfirm:
		if m.menuCursor != 0 {
			return m.gotoSessionsMenu()
		}
		m.state = screenRunningCommand
		m.returnTo = model.gotoSessionsMenu
		return m, deleteSessionCmd(m.db, m.editingSessionID)
	}

	return m, nil
}

// ==================== Form Ruangan (text input step-by-step) ====================

func (m model) handleRoomFormNama(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "enter" && m.roomInputNama.Value() != "" {
		m.roomInputNama.Blur()
		m.roomInputPortPrefix.Focus()
		m.state = screenRoomFormPortPrefix
		return m, nil
	}
	if key == "esc" {
		return m.gotoRoomsMenu()
	}
	var cmd tea.Cmd
	m.roomInputNama, cmd = m.roomInputNama.Update(msg)
	return m, cmd
}

func (m model) handleRoomFormPortPrefix(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "enter" && m.roomInputPortPrefix.Value() != "" {
		m.roomInputPortPrefix.Blur()
		m.roomInputCapacity.Focus()
		m.state = screenRoomFormCapacity
		return m, nil
	}
	if key == "esc" {
		m.roomInputPortPrefix.Blur()
		m.roomInputNama.Focus()
		m.state = screenRoomFormNama
		return m, nil
	}
	var cmd tea.Cmd
	m.roomInputPortPrefix, cmd = m.roomInputPortPrefix.Update(msg)
	return m, cmd
}

func (m model) handleRoomFormCapacity(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "enter" && m.roomInputCapacity.Value() != "" {
		capacity, err := strconv.Atoi(m.roomInputCapacity.Value())
		if err != nil {
			m.errMsg = "Kapasitas harus berupa angka."
			m.state = screenError
			m.returnTo = model.gotoRoomsMenu
			return m, nil
		}
		nama := m.roomInputNama.Value()
		portPrefix := m.roomInputPortPrefix.Value()
		m.state = screenRunningCommand
		m.returnTo = model.gotoRoomsMenu
		if m.roomFormMode == "create" {
			return m, createRoomCmd(m.db, nama, portPrefix, capacity)
		}
		return m, updateRoomCmd(m.db, m.editingRoomOriginalNama, nama, portPrefix, capacity)
	}
	if key == "esc" {
		m.roomInputCapacity.Blur()
		m.roomInputPortPrefix.Focus()
		m.state = screenRoomFormPortPrefix
		return m, nil
	}
	var cmd tea.Cmd
	m.roomInputCapacity, cmd = m.roomInputCapacity.Update(msg)
	return m, cmd
}

// ==================== Form Sesi (text input step-by-step) ====================

func (m model) handleSessionFormCourseCode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "enter" && m.sessionInputCourseCode.Value() != "" {
		m.sessionInputCourseCode.Blur()
		m.sessionInputMeetingNumber.Focus()
		m.state = screenSessionFormMeetingNumber
		return m, nil
	}
	if key == "esc" {
		return m.gotoSessionsMenu()
	}
	var cmd tea.Cmd
	m.sessionInputCourseCode, cmd = m.sessionInputCourseCode.Update(msg)
	return m, cmd
}

func (m model) handleSessionFormMeetingNumber(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "enter" && m.sessionInputMeetingNumber.Value() != "" {
		m.sessionInputMeetingNumber.Blur()
		m.sessionInputDate.Focus()
		m.state = screenSessionFormDate
		return m, nil
	}
	if key == "esc" {
		m.sessionInputMeetingNumber.Blur()
		m.sessionInputCourseCode.Focus()
		m.state = screenSessionFormCourseCode
		return m, nil
	}
	var cmd tea.Cmd
	m.sessionInputMeetingNumber, cmd = m.sessionInputMeetingNumber.Update(msg)
	return m, cmd
}

func (m model) handleSessionFormDate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "enter" && m.sessionInputDate.Value() != "" {
		m.sessionInputDate.Blur()
		m.state = screenSessionPickStatus
		m.menuItems = sessionStatusOptions
		m.menuCursor = 0
		for i, s := range sessionStatusOptions {
			if s == m.sessionStatus {
				m.menuCursor = i
			}
		}
		return m, nil
	}
	if key == "esc" {
		m.sessionInputDate.Blur()
		m.sessionInputMeetingNumber.Focus()
		m.state = screenSessionFormMeetingNumber
		return m, nil
	}
	var cmd tea.Cmd
	m.sessionInputDate, cmd = m.sessionInputDate.Update(msg)
	return m, cmd
}

// ==================== Navigasi umum ====================

func (m model) backToMainMenu() (tea.Model, tea.Cmd) {
	m.state = screenMainMenu
	m.menuItems = mainMenuItems
	m.menuCursor = 0
	m.returnTo = model.backToMainMenu
	return m, nil
}

func (m model) gotoRoomsMenu() (tea.Model, tea.Cmd) {
	m.state = screenRoomsMenu
	m.menuItems = roomsMenuItems
	m.menuCursor = 0
	m.returnTo = model.gotoRoomsMenu
	return m, nil
}

func (m model) gotoSessionsMenu() (tea.Model, tea.Cmd) {
	m.state = screenSessionsMenu
	m.menuItems = sessionsMenuItems
	m.menuCursor = 0
	m.returnTo = model.gotoSessionsMenu
	return m, nil
}

func (m model) toError(err error) (tea.Model, tea.Cmd) {
	m.errMsg = err.Error()
	m.state = screenError
	return m, nil
}