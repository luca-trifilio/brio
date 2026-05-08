// Package brunoprefs reads Bruno desktop application preferences to discover
// collections that were last opened in the GUI.
package brunoprefs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type preferences struct {
	LastOpenedCollections []string `json:"lastOpenedCollections"`
}

// CollectionPaths returns the paths from Bruno's lastOpenedCollections list.
// Returns an error only if the preferences file exists but cannot be parsed.
// Returns (nil, nil) when the file does not exist.
func CollectionPaths() ([]string, error) {
	p, err := prefsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Bruno preferences: %w", err)
	}
	var prefs preferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return nil, fmt.Errorf("parse Bruno preferences: %w", err)
	}
	// Filter to paths that actually exist on disk.
	var valid []string
	for _, path := range prefs.LastOpenedCollections {
		if _, err := os.Stat(path); err == nil {
			valid = append(valid, path)
		}
	}
	return valid, nil
}

// prefsPath returns the OS-specific location of Bruno's preferences.json.
func prefsPath() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "bruno", "preferences.json"), nil
	case "linux":
		// Bruno on Linux follows XDG: $XDG_CONFIG_HOME/Bruno or ~/.config/Bruno
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			configDir = filepath.Join(home, ".config")
		}
		return filepath.Join(configDir, "Bruno", "preferences.json"), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA not set")
		}
		return filepath.Join(appData, "bruno", "preferences.json"), nil
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}
