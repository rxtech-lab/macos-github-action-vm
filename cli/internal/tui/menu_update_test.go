package tui

import (
	"testing"

	"github.com/rxtech-lab/rvmm/internal/updater"
)

func TestRootMenuContainsUpdate(t *testing.T) {
	t.Parallel()
	for _, entry := range rootMenuEntries() {
		if entry.Title() == "Update rvmm" {
			return
		}
	}
	t.Fatal("root menu does not contain Update rvmm")
}

func TestFormatUpdaterStatus(t *testing.T) {
	t.Parallel()
	got := formatUpdaterStatus(updater.Status{State: "installed", LatestVersion: "v1.1.0", RestartNeeded: true}, true)
	if got != "installed (v1.1.0) - restart pending" {
		t.Fatalf("formatUpdaterStatus() = %q", got)
	}
}
