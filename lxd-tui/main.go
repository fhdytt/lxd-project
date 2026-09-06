package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
)

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

func execAsUser(username string) {
	env := os.Environ()
	if err := syscall.Exec("/bin/login", []string{"login", "-f", username}, env); err != nil {
		fmt.Fprintln(os.Stderr, "Gagal masuk sebagai", username, ":", err)
		os.Exit(1)
	}
}