package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

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

// ExpandPath expands a leading "~/" to the user's home directory, then
// resolves all $VAR / ${VAR} environment-variable references, and finally
// cleans the resulting path.
func ExpandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			p = filepath.Join(home, p[2:])
		}
	}
	p = os.ExpandEnv(p)
	return filepath.Clean(p)
}

// ResolvedCollections returns the expanded, on-disk-verified subset of
// cfg.Collections. Each path is expanded via ExpandPath; paths that do not
// exist on disk are skipped and reported via warn (never nil-safe to call —
// pass a no-op if you don't need warnings).
func ResolvedCollections(cfg *Config, warn func(string)) []string {
	var out []string
	for _, e := range cfg.Collections {
		p := ExpandPath(e.Path)
		if _, err := os.Stat(p); err != nil {
			warn(p)
			continue
		}
		out = append(out, p)
	}
	return out
}

// rawConfig mirrors Config but accepts the legacy `collections = ["..."]`
// flat-string form alongside the new `[[collections]]` table form. Used as
// an intermediate decode target for backward-compat reading.
type rawConfig struct {
	Collections      []toml.Primitive `toml:"collections"`
	ActiveCollection string           `toml:"active_collection,omitempty"`
	Theme            string           `toml:"theme,omitempty"`
	Hooks            []Hook           `toml:"hooks"`
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

	var raw rawConfig
	meta, err := toml.Decode(string(data), &raw)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		ActiveCollection: raw.ActiveCollection,
		Theme:            raw.Theme,
		Hooks:            raw.Hooks,
	}

	for _, p := range raw.Collections {
		// Try string first (legacy flat form), then table form.
		var s string
		if err := meta.PrimitiveDecode(p, &s); err == nil {
			cfg.Collections = append(cfg.Collections, CollectionEntry{Path: s})
			continue
		}
		var entry CollectionEntry
		if err := meta.PrimitiveDecode(p, &entry); err != nil {
			return nil, err
		}
		cfg.Collections = append(cfg.Collections, entry)
	}

	return cfg, nil
}
