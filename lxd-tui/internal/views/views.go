package views

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"lxd-tui/internal/state"
	"lxd-tui/internal/styles"
)

// RenderBox adalah fungsi utama untuk membuat box dengan border dan logo
func RenderBox(appState *state.AppState, content string) string {
	width := 56

	if appState.WindowWidth > 0 {
		width = appState.WindowWidth - 10

		if width > 60 {
			width = 60
		}

		if width < 30 {
			width = 30
		}
	}

	fullContent := fmt.Sprintf(
		"%s\n%s",
		styles.LogoStyle.Render(styles.AsciiLogo),
		content,
	)

	box := styles.BoxStyle.Width(width).Render(fullContent)

	if appState.WindowWidth > 0 && appState.WindowHeight > 0 {
		return lipgloss.Place(
			appState.WindowWidth,
			appState.WindowHeight,
			lipgloss.Center,
			lipgloss.Center,
			box,
		)
	}

	return box
}

// MainView memilih view berdasarkan state
func MainView(appState *state.AppState) string {
	switch appState.CurrentScreen {
	case state.ScreenLoading:
		return RenderBox(appState, RenderLoading(appState))

	case state.ScreenDashboard:
		return RenderBox(appState, RenderDashboard(appState))

	case state.ScreenInputNama:
		return RenderBox(appState, RenderInputNama(appState))

	case state.ScreenInputNPM:
		return RenderBox(appState, RenderInputNPM(appState))

	case state.ScreenSubmitting:
		return RenderBox(appState, RenderSubmitting(appState))

	case state.ScreenSelectUser:
		return RenderBox(appState, RenderSelectUser(appState))

	case state.ScreenLocalPassword:
		return RenderBox(appState, RenderLocalPassword(appState))

	case state.ScreenError:
		return RenderBox(appState, RenderError(appState))

	default:
		return ""
	}
}
