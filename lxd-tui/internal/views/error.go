package views

import (
    "fmt"

    "lxd-tui/internal/state"
    "lxd-tui/internal/styles"
)

func RenderError(state *state.AppState) string {
    return fmt.Sprintf(
        "%s\n\n%s\n\n%s\n%s",
        styles.ErrStyle.Render("[ ERROR SYSTEM ]"),
        state.ErrorMessage,
        styles.DividerStyle.Render("─"),
        styles.HintStyle.Render("[Enter] Coba lagi  |  [Ctrl+C] Keluar"),
    )
}