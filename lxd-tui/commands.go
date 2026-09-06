package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchEnvInfoCmd(m.client), m.spin.Tick)
}

func fetchEnvInfoCmd(client *APIClient) tea.Cmd {
	return func() tea.Msg {
		info, err := client.FetchEnvInfo()
		return envInfoFetchedMsg{info: info, err: err}
	}
}

func submitIdentityCmd(client *APIClient, nama, npm string) tea.Cmd {
	return func() tea.Msg {
		err := client.SubmitIdentity(nama, npm)
		return identitySubmittedMsg{err: err}
	}
}

func loadLocalUsersCmd() tea.Cmd {
	return func() tea.Msg {
		users, err := listLocalUsers()
		return localUsersLoadedMsg{users: users, err: err}
	}
}

func verifyPasswordCmd(username, password string) tea.Cmd {
	return func() tea.Msg {
		ok, err := verifyLocalPassword(username, password)
		return passwordVerifiedMsg{ok: ok, err: err}
	}
}