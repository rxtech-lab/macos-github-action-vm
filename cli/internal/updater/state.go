package updater

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	StateDirectory   = "/Library/Application Support/RVMM/Updater"
	RequestDirectory = StateDirectory + "/requests"
	StatusPath       = StateDirectory + "/status.json"
	LockPath         = StateDirectory + "/update.lock"
)

type Status struct {
	State          string    `json:"state"`
	CurrentVersion string    `json:"current_version,omitempty"`
	LatestVersion  string    `json:"latest_version,omitempty"`
	Message        string    `json:"message,omitempty"`
	RestartNeeded  bool      `json:"restart_needed,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func ReadStatus(path string) (Status, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Status{}, err
	}
	var status Status
	if err := json.Unmarshal(data, &status); err != nil {
		return Status{}, fmt.Errorf("decode updater status: %w", err)
	}
	return status, nil
}

func writeStatus(path string, status Status) error {
	status.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("encode updater status: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create updater state directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".status-*")
	if err != nil {
		return fmt.Errorf("create updater status: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0644); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func RequestUpdate(directory string) error {
	info, err := os.Stat(directory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.New("background updater is not installed; install the latest RVMM package once with administrator approval")
		}
		return fmt.Errorf("inspect updater request directory: %w", err)
	}
	if !info.IsDir() {
		return errors.New("updater request path is not a directory")
	}
	name := fmt.Sprintf("request-%d-%d", time.Now().UnixNano(), os.Getpid())
	file, err := os.OpenFile(filepath.Join(directory, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("queue update request: %w", err)
	}
	return file.Close()
}
