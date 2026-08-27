package main

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/jackc/pgx/v5/pgxpool"
)

type screenState int

const (
	screenMainMenu screenState = iota
	screenPickRoomForList
	screenListEnvironments
	screenPickRoomForProvision
	screenPickProvisionAction
	screenPickModuleForStart
	screenConfirmProvision
	screenRunningCommand
	screenCommandResult
	screenPickRoomForReset
	screenPickResetMode
	screenPickContainerForReset
	screenConfirmReset
	screenError
)

type model struct {
	cfg *Config
	db  *pgxpool.Pool

	state  screenState
	errMsg string
	spin   spinner.Model

	menuItems  []string
	menuCursor int

	rooms   []Room
	modules []Module
	envRows []EnvironmentRow

	selectedRoom      string
	selectedModule    string
	selectedAction    string
	selectedResetMode string
	selectedContainer string

	commandOutput string
	commandFailed bool
}

func initialModel(cfg *Config, db *pgxpool.Pool) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(accent)

	return model{
		cfg:       cfg,
		db:        db,
		state:     screenMainMenu,
		menuItems: []string{"View Environment List", "Room Setup", "Reset Environment", "Exit"},
		spin:      sp,
	}
}

var (
	bgDark    = lipgloss.Color("#344e41")
	bgMid     = lipgloss.Color("#3a5a40")
	accent    = lipgloss.Color("#588157")
	textSoft  = lipgloss.Color("#a3b18a")
	textLight = lipgloss.Color("#dad7cd")

	// Style untuk ASCII Art Logo
	logoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent)

 asciiLogo = `
     __       ______  ____    _  __  ____    __  ___ 
    /  /     / ____/ / __ \  / //_/ / __ \  /  |/  / 
   /  /     / /___  / /_/ / / ,<   / / / / / /|_/ /  
  /  /___  / /___  / ____/ / /| | / /_/ / / /  / /   
 /______/ /_____/ /_/     /_/ |_| \____/ /_/  /_/    
												
    			 G U N A D A R M A                   
 `

	// Component Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textLight).
			Background(bgDark).
			Padding(0, 1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(1, 2)

	hintStyle = lipgloss.NewStyle().
			Foreground(textSoft).
			Italic(true)

	errStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e63946")).
			Bold(true)

	okStyle = lipgloss.NewStyle().
			Foreground(accent).
			Bold(true)

	cursorStyle = lipgloss.NewStyle().
			Foreground(textLight).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(textSoft)

	labelStyle = lipgloss.NewStyle().
			Foreground(textSoft)
)

type roomsLoadedMsg struct {
	rooms []Room
	err   error
}
type modulesLoadedMsg struct {
	modules []Module
	err     error
}
type environmentsLoadedMsg struct {
	rows []EnvironmentRow
	err  error
}
type containersLoadedMsg struct {
	names []string
	err   error
}
type commandDoneMsg struct {
	output string
	err    error
}