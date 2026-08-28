package views

import (
    "fmt"

    "lxd-tui/internal/state"
)

func RenderLoading(state *state.AppState) string {
    return fmt.Sprintf("%s Menghubungkan ke server...", state.Spinner.View())
}