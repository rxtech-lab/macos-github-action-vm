package tui

import (
	"errors"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rxtech-lab/rvmm/internal/updater"
)

type updaterStatusMsg struct {
	status    updater.Status
	installed bool
}

func tickUpdaterStatus() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		status, err := updater.ReadStatus(updater.StatusPath)
		if err != nil {
			return updaterStatusMsg{installed: !errors.Is(err, os.ErrNotExist)}
		}
		return updaterStatusMsg{status: status, installed: true}
	})
}

func formatUpdaterStatus(status updater.Status, installed bool) string {
	if !installed {
		return "not installed"
	}
	if status.State == "" {
		return "waiting for first check"
	}
	result := status.State
	if status.LatestVersion != "" {
		result += " (" + status.LatestVersion + ")"
	}
	if status.RestartNeeded {
		result += " - restart pending"
	}
	return result
}
