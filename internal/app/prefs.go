package app

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Prefs holds small UI preferences that persist between launches — window
// size for now. Kept in its own file rather than folded into events.json so
// a corrupt/unreadable prefs file can never take the actual timer data down
// with it.
type Prefs struct {
	WindowWidth  int `json:"windowWidth"`
	WindowHeight int `json:"windowHeight"`
}

const (
	defaultWindowWidth  = 640
	defaultWindowHeight = 420
)

func prefsFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(dir, "wails-timer")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(appDir, "prefs.json"), nil
}

// LoadPrefs returns saved preferences, or sane defaults if none are saved
// yet (first launch) or the file can't be read/parsed.
func LoadPrefs() Prefs {
	defaults := Prefs{WindowWidth: defaultWindowWidth, WindowHeight: defaultWindowHeight}

	path, err := prefsFilePath()
	if err != nil {
		return defaults
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return defaults
	}
	var p Prefs
	if err := json.Unmarshal(data, &p); err != nil {
		return defaults
	}
	if p.WindowWidth <= 0 || p.WindowHeight <= 0 {
		return defaults
	}
	return p
}

// SavePrefs persists preferences, overwriting whatever was saved before.
func SavePrefs(p Prefs) error {
	path, err := prefsFilePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, data, 0o644)
}
