package updater

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

type Worker struct {
	CurrentVersion string
	TeamID         string
	Client         ReleaseClient
	StatusPath     string
	LockPath       string
	RequestDir     string
	Verify         func(context.Context, string, string) error
	Install        func(context.Context, string) error
	IsRoot         func() bool
	IsSupported    func() bool
}

func NewWorker(currentVersion, teamID string) *Worker {
	return &Worker{
		CurrentVersion: currentVersion,
		TeamID:         teamID,
		Client:         NewClient(currentVersion),
		StatusPath:     StatusPath,
		LockPath:       LockPath,
		RequestDir:     RequestDirectory,
		Verify:         VerifyPackage,
		Install:        InstallPackage,
		IsRoot:         func() bool { return os.Geteuid() == 0 },
		IsSupported:    func() bool { return runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" },
	}
}

func (w *Worker) Run(ctx context.Context) error {
	if !w.IsRoot() {
		return errors.New("update worker must run as root")
	}
	if !w.IsSupported() {
		return fmt.Errorf("automatic updates are only supported on macOS arm64")
	}
	if err := os.MkdirAll(filepath.Dir(w.LockPath), 0755); err != nil {
		return fmt.Errorf("create updater state directory: %w", err)
	}
	lock, err := os.OpenFile(w.LockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("open updater lock: %w", err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return errors.New("another update worker is already running")
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	defer w.consumeRequests()
	pendingRestart := false
	if previous, err := ReadStatus(w.StatusPath); err == nil {
		pendingRestart = previous.RestartNeeded
	}

	setStatus := func(state, latest, message string, restart bool) {
		_ = writeStatus(w.StatusPath, Status{
			State:          state,
			CurrentVersion: w.CurrentVersion,
			LatestVersion:  latest,
			Message:        message,
			RestartNeeded:  restart,
		})
	}
	setStatus("checking", "", "Checking GitHub for updates", pendingRestart)

	update, err := w.Client.Check(ctx, w.CurrentVersion)
	if err != nil {
		setStatus("error", "", err.Error(), pendingRestart)
		return err
	}
	if !update.Available {
		message := "RVMM is up to date"
		if pendingRestart {
			message = "RVMM is up to date; service restart is still pending"
		}
		setStatus("up_to_date", update.LatestVersion, message, pendingRestart)
		return nil
	}

	directory, err := os.MkdirTemp("/private/tmp", "rvmm-update-*")
	if err != nil {
		setStatus("error", update.LatestVersion, err.Error(), pendingRestart)
		return fmt.Errorf("create update directory: %w", err)
	}
	defer os.RemoveAll(directory)
	packagePath := filepath.Join(directory, PackageAssetName)

	setStatus("downloading", update.LatestVersion, "Downloading signed update", pendingRestart)
	if err := w.Client.Download(ctx, update, packagePath); err != nil {
		setStatus("error", update.LatestVersion, err.Error(), pendingRestart)
		return err
	}
	setStatus("verifying", update.LatestVersion, "Verifying package signature and notarization", pendingRestart)
	if err := w.Verify(ctx, packagePath, w.TeamID); err != nil {
		setStatus("error", update.LatestVersion, err.Error(), pendingRestart)
		return err
	}
	setStatus("installing", update.LatestVersion, "Installing update", pendingRestart)
	installCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	if err := w.Install(installCtx, packagePath); err != nil {
		setStatus("error", update.LatestVersion, err.Error(), pendingRestart)
		return err
	}
	setStatus("installed", update.LatestVersion, "Update installed; active VM work was not interrupted", true)
	return nil
}

func (w *Worker) consumeRequests() {
	entries, err := os.ReadDir(w.RequestDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			_ = os.Remove(filepath.Join(w.RequestDir, entry.Name()))
		}
	}
}
