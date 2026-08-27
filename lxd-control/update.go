package main

import (
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
	}

	return m, nil
}

// handleKey menangani semua input keyboard selain Ctrl+C (yang sudah
// ditangani langsung di Update). Navigasi menu digeneralisasi lewat
// isMenuScreen(), supaya tidak perlu duplikasi logic ↑/↓/Enter/Esc di
// setiap layar berbasis daftar pilihan.
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
			return m.backToMainMenu()
		case "enter":
			return m.onMenuSelect()
		}
		return m, nil
	}

	switch m.state {
	case screenListEnvironments, screenCommandResult, screenError:
		if key == "enter" || key == "esc" {
			return m.backToMainMenu()
		}
	}

	return m, nil
}

func isMenuScreen(s screenState) bool {
	switch s {
	case screenMainMenu, screenPickRoomForList, screenPickRoomForProvision,
		screenPickProvisionAction, screenPickModuleForStart, screenConfirmProvision,
		screenPickRoomForReset, screenPickResetMode, screenPickContainerForReset,
		screenConfirmReset:
		return true
	}
	return false
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
			return m, loadRoomsCmd(m.db)
		case 1:
			m.state = screenPickRoomForProvision
			return m, loadRoomsCmd(m.db)
		case 2:
			m.state = screenPickRoomForReset
			return m, loadRoomsCmd(m.db)
		case 3:
			return m, tea.Quit
		}

	case screenPickRoomForList:
		m.selectedRoom = selected
		m.state = screenListEnvironments
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
		m.state = screenConfirmProvision
		m.menuItems = []string{"Ya, jalankan", "Batal"}
		m.menuCursor = 0
		return m, nil

	case screenConfirmProvision:
		if m.menuCursor != 0 {
			return m.backToMainMenu()
		}
		m.state = screenRunningCommand
		if m.selectedAction == "start" {
			return m, runCommandCmd(m.cfg, "start", m.selectedRoom, m.selectedModule)
		}
		return m, runCommandCmd(m.cfg, "stop", m.selectedRoom)

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
		if m.selectedResetMode == "room" {
			return m, runCommandCmd(m.cfg, "reset-room", m.selectedRoom)
		}
		return m, runCommandCmd(m.cfg, "reset", m.selectedContainer)
	}

	return m, nil
}

func (m model) backToMainMenu() (tea.Model, tea.Cmd) {
	m.state = screenMainMenu
	m.menuItems = []string{"Lihat Daftar Environment", "Provisioning Ruangan", "Reset Environment", "Keluar"}
	m.menuCursor = 0
	return m, nil
}

func (m model) toError(err error) (tea.Model, tea.Cmd) {
	m.errMsg = err.Error()
	m.state = screenError
	return m, nil
}