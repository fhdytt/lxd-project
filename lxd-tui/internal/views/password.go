package views

import (
    "fmt"
    "strings"

    "lxd-tui/internal/state"
    "lxd-tui/internal/styles"
)

func RenderLocalPassword(state *state.AppState) string {
    errLine := ""
    if state.PWError != "" {
        errLine = "\n" + styles.ErrStyle.Render(state.PWError) + "\n"
    }

    return fmt.Sprintf(
        "%s\n\n%s %s\n\n%s\n%s%s\n%s",
        styles.TitleStyle.Render(" MASUK SEBAGAI "+strings.ToUpper(state.SelectedUser)+" "),
        styles.LabelStyle.Render("User:"), styles.ValueStyle.Render(state.SelectedUser),
        state.InputPassword.View(),
        errLine,
        styles.DividerStyle.Render(strings.Repeat("─", 50)),
        styles.HintStyle.Render("[Enter] Masuk  |  [Esc] Ganti User  |  [Ctrl+C] Batal"),
    )
}