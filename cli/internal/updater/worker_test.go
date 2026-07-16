package updater

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type fakeReleaseClient struct {
	update     Update
	downloaded bool
}

func (f *fakeReleaseClient) Check(context.Context, string) (Update, error) {
	return f.update, nil
}

func (f *fakeReleaseClient) Download(_ context.Context, _ Update, destination string) error {
	f.downloaded = true
	return os.WriteFile(destination, []byte("package"), 0600)
}

func TestWorkerInstallsAvailableUpdate(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	requests := filepath.Join(directory, "requests")
	if err := os.Mkdir(requests, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(requests, "request"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	fakeClient := &fakeReleaseClient{update: Update{Available: true, LatestVersion: "v1.1.0"}}
	verified := false
	installed := false
	worker := &Worker{
		CurrentVersion: "v1.0.0",
		TeamID:         "TEAMID",
		Client:         fakeClient,
		StatusPath:     filepath.Join(directory, "status.json"),
		LockPath:       filepath.Join(directory, "update.lock"),
		RequestDir:     requests,
		Verify: func(context.Context, string, string) error {
			verified = true
			return nil
		},
		Install: func(context.Context, string) error {
			installed = true
			return nil
		},
		IsRoot:      func() bool { return true },
		IsSupported: func() bool { return true },
	}
	if err := worker.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !fakeClient.downloaded || !verified || !installed {
		t.Fatalf("downloaded=%v verified=%v installed=%v", fakeClient.downloaded, verified, installed)
	}
	status, err := ReadStatus(worker.StatusPath)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "installed" || !status.RestartNeeded {
		t.Fatalf("unexpected status: %+v", status)
	}
	entries, err := os.ReadDir(requests)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("request queue was not consumed: %v", entries)
	}
}

func TestWorkerPreservesPendingRestartWhenUpToDate(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	statusPath := filepath.Join(directory, "status.json")
	if err := writeStatus(statusPath, Status{State: "installed", LatestVersion: "v1.1.0", RestartNeeded: true}); err != nil {
		t.Fatal(err)
	}
	worker := &Worker{
		CurrentVersion: "v1.1.0",
		Client:         &fakeReleaseClient{update: Update{Available: false, LatestVersion: "v1.1.0"}},
		StatusPath:     statusPath,
		LockPath:       filepath.Join(directory, "update.lock"),
		RequestDir:     filepath.Join(directory, "requests"),
		Verify:         func(context.Context, string, string) error { return nil },
		Install:        func(context.Context, string) error { return nil },
		IsRoot:         func() bool { return true },
		IsSupported:    func() bool { return true },
	}
	if err := worker.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	status, err := ReadStatus(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	if !status.RestartNeeded {
		t.Fatalf("pending restart was cleared: %+v", status)
	}
}
