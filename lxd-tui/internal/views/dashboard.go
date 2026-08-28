package views

import (
    "fmt"
    "strings"

    "lxd-tui/internal/state"
    "lxd-tui/internal/styles"
)

func RenderDashboard(state *state.AppState) string {
    info := state.EnvInfo
    if info == nil {
        return "Tidak ada data environment"
    }

    identStatus := styles.ValueStyle.Render("Belum diisi")
    if info.AlreadyIdentified {
        identStatus = styles.ValueStyle.
            Bold(true).
            Foreground(styles.Accent).
            Render("Sudah terdaftar (perlu verifikasi)")
    }

    divider := styles.DividerStyle.Render(strings.Repeat("─", 50))

    return fmt.Sprintf(
        "%s\n%s\n%s\n\n%s %s\n%s %s\n%s %s\n%s %d\n%s %s\n%s %s\n%s %s\n\n%s\n%s",
        styles.TitleStyle.Render(" DASHBOARD SESI PRAKTIKUM "),
        styles.SubtitleStyle.Render("Sistem Manajemen Environment Praktikum"),
        divider,
        styles.LabelStyle.Render("Kode Kursus  :"), styles.ValueStyle.Render(info.CourseCode),
        styles.LabelStyle.Render("Modul        :"), styles.ValueStyle.Render(info.Module),
        styles.LabelStyle.Render("Ruangan      :"), styles.ValueStyle.Render(info.Room),
        styles.LabelStyle.Render("Pertemuan ke :"), info.MeetingNumber,
        styles.LabelStyle.Render("Tanggal      :"), styles.ValueStyle.Render(info.SessionDate),
        styles.LabelStyle.Render("Status Env   :"), styles.ValueStyle.Render(info.Status),
        styles.LabelStyle.Render("Identifikasi :"), identStatus,
        divider,
        styles.HintStyle.Render("[Enter] Lanjutkan  |  [Ctrl+C] Keluar"),
    )
}