package app

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func dataFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(dir, "wails-timer")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(appDir, "events.json"), nil
}

func loadTimers() ([]TimerEvent, error) {
	path, err := dataFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var events []TimerEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func saveTimers(events []TimerEvent) error {
	path, err := dataFilePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, data, 0o644)
}

// writeFileAtomic writes data so the file at path always ends up either
// fully the old content or fully the new content, never a partial write:
// the content lands completely in a temp file first, then one os.Rename
// swaps it into place. Writing directly with os.WriteFile would leave the
// file truncated (and unparseable, silently losing every saved timer on
// next launch) if the app is killed, the disk fills up, or the write
// otherwise fails partway through.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
