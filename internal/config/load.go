package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Path returns the resolved path to the brio config file.
// Respects $XDG_CONFIG_HOME, falls back to ~/.config/brio/config.toml.
func Path() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "~/.config/brio/config.toml"
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "brio", "config.toml")
}

// EnsureDir creates the config directory (and any parents) if it does not exist.
func EnsureDir() error {
	return os.MkdirAll(filepath.Dir(Path()), 0o755)
}

// Load reads and parses the config file.
// Returns an empty Config (no hooks) if the file does not exist — that is not
// an error; it simply means no hooks are configured.
func Load() (*Config, error) {
	p := Path()

	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
