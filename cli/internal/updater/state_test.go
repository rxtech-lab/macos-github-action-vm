package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRequestUpdateCreatesQueueEntry(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	if err := RequestUpdate(directory); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d request entries, want 1", len(entries))
	}
}

func TestStatusRoundTrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "status.json")
	want := Status{State: "installed", CurrentVersion: "v1.0.0", LatestVersion: "v1.1.0", RestartNeeded: true}
	if err := writeStatus(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadStatus(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != want.State || got.LatestVersion != want.LatestVersion || !got.RestartNeeded {
		t.Fatalf("ReadStatus() = %+v, want %+v", got, want)
	}
}
