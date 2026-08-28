package views

import (
    "fmt"
    "strings"

    "lxd-tui/internal/state"
    "lxd-tui/internal/styles"
)

func RenderInputNama(state *state.AppState) string {
    hint := "Masukkan nama lengkap:"
    if state.EnvInfo != nil && state.EnvInfo.AlreadyIdentified {
        hint = "Environment ini sudah terdaftar. Masukkan nama untuk verifikasi:"
    }

    return fmt.Sprintf(
        "%s\n\n%s\n%s\n\n%s\n%s",
        styles.TitleStyle.Render(" IDENTIFIKASI PRAKTIKAN "),
        styles.LabelStyle.Render(hint),
        state.InputNama.View(),
        styles.DividerStyle.Render(strings.Repeat("─", 50)),
        styles.HintStyle.Render("[Enter] Lanjut  |  [Ctrl+C] Batal"),
    )
}

func RenderInputNPM(state *state.AppState) string {
    return fmt.Sprintf(
        "%s\n\n%s %s\n\n%s\n%s\n\n%s\n%s",
        styles.TitleStyle.Render(" IDENTIFIKASI PRAKTIKAN "),
        styles.LabelStyle.Render("Nama:"), styles.ValueStyle.Render(state.InputNama.Value()),
        styles.LabelStyle.Render("Masukkan NPM:"),
        state.InputNPM.View(),
        styles.DividerStyle.Render(strings.Repeat("─", 50)),
        styles.HintStyle.Render("[Enter] Kirim  |  [Esc] Kembali  |  [Ctrl+C] Batal"),
    )
}

func RenderSubmitting(state *state.AppState) string {
    return fmt.Sprintf("%s Memverifikasi identitas...", state.Spinner.View())
}