package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
)

// main sengaja setipis mungkin — cuma "merakit" komponen: baca env var,
// buka koneksi API, jalankan program TUI, lalu exec ke shell/login kalau
// sukses. Semua logic sesungguhnya ada di file lain: model.go (state &
// tipe data), commands.go (operasi async), update.go (transisi state),
// view.go (rendering tampilan), api.go (HTTP client ke backend),
// local_auth.go (baca /etc/passwd & verifikasi /etc/shadow).
func main() {
	apiURL := getContainerEnv("PRAKTIKUM_API_URL")
	token := getContainerEnv("PRAKTIKUM_API_TOKEN")

	if apiURL == "" || token == "" {
		fmt.Fprintln(os.Stderr, "Error: PRAKTIKUM_API_URL atau PRAKTIKUM_API_TOKEN belum di-set pada environment ini.")
		fmt.Fprintln(os.Stderr, "Hubungi asisten/admin lab, environment ini belum terkonfigurasi dengan benar.")
		os.Exit(1)
	}

	client := NewAPIClient(apiURL, token)

	p := tea.NewProgram(initialModel(client), tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "TUI error:", err)
		os.Exit(1)
	}

	m := finalModel.(model)

	if !m.shouldContinueToShell {
		fmt.Println("Sesi ditutup.")
		os.Exit(0)
	}

	execAsUser(m.selectedUsername)
}

// getContainerEnv membaca env var yang di-inject LXD lewat
// "lxc config set <container> environment.KEY=value". Fallback ke
// /proc/1/environ karena env var itu tidak diwariskan ke sesi SSH secara
// otomatis — lihat dokumentasi TUI § 4.3.
func getContainerEnv(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	data, err := os.ReadFile("/proc/1/environ")
	if err != nil {
		return ""
	}
	for _, entry := range strings.Split(string(data), "\x00") {
		if strings.HasPrefix(entry, key+"=") {
			return strings.TrimPrefix(entry, key+"=")
		}
	}
	return ""
}

// execAsUser menggantikan proses TUI dengan sesi login sebagai user Linux
// yang dipilih. Memakai "login -f" (force, tanpa password lagi — karena
// password SUDAH diverifikasi manual oleh TUI lewat /etc/shadow) daripada
// "su", supaya sesi tercatat rapi di utmp/wtmp seperti login normal.
// TUI selalu jalan sebagai root (lihat keputusan desain project), jadi
// "login -f" ini valid dijalankan untuk user manapun, termasuk root sendiri.
func execAsUser(username string) {
	env := os.Environ()
	if err := syscall.Exec("/bin/login", []string{"login", "-f", username}, env); err != nil {
		fmt.Fprintln(os.Stderr, "Gagal masuk sebagai", username, ":", err)
		os.Exit(1)
	}
}