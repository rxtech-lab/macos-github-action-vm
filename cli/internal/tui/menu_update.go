package tui

import tea "github.com/charmbracelet/bubbletea"

type updateMenuItem struct{}

func (updateMenuItem) Title() string {
	return "Update rvmm"
}

func (updateMenuItem) Description() string {
	return "Request an immediate signed update in the background"
}

func (updateMenuItem) OnSelect(m *model) (tea.Model, tea.Cmd) {
	m.busy = true
	m.busyLabel = "Request update"
	return *m, tea.Batch(m.runUpdateRequestCmd(), m.spinner.Tick)
}
