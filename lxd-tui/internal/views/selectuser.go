package views

import (
    "fmt"
    "strings"

    "lxd-tui/internal/state"
    "lxd-tui/internal/styles"
)

func RenderSelectUser(state *state.AppState) string {
    if len(state.LocalUsers) == 0 {
        return fmt.Sprintf("%s Memuat daftar user...", state.Spinner.View())
    }

    var b strings.Builder
    b.WriteString(styles.TitleStyle.Render(" MASUK SEBAGAI "))
    b.WriteString("\n\n")

    for i, u := range state.LocalUsers {
        if i == state.UserCursor {
            itemText := fmt.Sprintf("[-] %02d. %s", i+1, u.Username)
            b.WriteString(styles.SelectedItemStyle.Render(itemText))
        } else {
            itemText := fmt.Sprintf("    %02d. %s", i+1, u.Username)
            b.WriteString(styles.InactiveItemStyle.Render(itemText))
        }
        b.WriteString("\n")
    }

    b.WriteString("\n")
    b.WriteString(styles.DividerStyle.Render(strings.Repeat("─", 50)))
    b.WriteString("\n")
    b.WriteString(styles.HintStyle.Render("[UP/DOWN] Pilih  |  [ENTER] Konfirmasi  |  [CTRL+C] Batal"))
    return b.String()
}