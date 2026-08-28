package styles

import (
    "github.com/charmbracelet/lipgloss"
)

// Palet Hex Earthy
var (
    BgDark    = lipgloss.Color("#344e41")
    BgMid     = lipgloss.Color("#3a5a40")
    Accent    = lipgloss.Color("#588157")
    TextSoft  = lipgloss.Color("#a3b18a")
    TextLight = lipgloss.Color("#dad7cd")
    Danger    = lipgloss.Color("#e63946")
)

// Logo Style & ASCII Art Tegak
var (
    LogoStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(Accent)

    AsciiLogo = `
  _        ______  ____   _  __   ____    __  ___ 
 | |      / ____/ / __ \ | |/ /  / __ \  /  |/  / 
 | |     / /___  / /_/ / | ' /  / / / / / /|_/ /  
 | |___ / /___  / ____/  | . \ / /_/ / / /  / /   
 |_____/_____/ /_/       |_|\_\\____/ /_/  /_/    
                                                  
              G U N A D A R M A                   
`
)

// Component Styles
var (
    TitleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(TextLight).
        Background(BgDark).
        Padding(0, 1)

    SubtitleStyle = lipgloss.NewStyle().
        Foreground(TextSoft).
        Italic(true)

    BoxStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(Accent).
        Padding(1, 2)

    LabelStyle = lipgloss.NewStyle().
        Foreground(TextSoft)

    ValueStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(TextLight)

    HintStyle = lipgloss.NewStyle().
        Foreground(TextSoft).
        Italic(true)

    ErrStyle = lipgloss.NewStyle().
        Foreground(Danger).
        Bold(true)

    CursorStyle = lipgloss.NewStyle().
        Foreground(TextLight).
        Bold(true)

    SelectedItemStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(TextLight).
        Background(Accent).
        Padding(0, 1)

    InactiveItemStyle = lipgloss.NewStyle().
        Foreground(TextSoft).
        Padding(0, 1)

    DividerStyle = lipgloss.NewStyle().
        Foreground(Accent)
)