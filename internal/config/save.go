package config

import (
	"bytes"
	"os"

	"github.com/BurntSushi/toml"
)

// Save encodes cfg as TOML and writes it to the config file.
// All structured data (collections, hooks) is preserved. TOML comments in the
// original file are not preserved — this is a known limitation of the
// BurntSushi encoder.
func Save(cfg *Config) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return err
	}
	return os.WriteFile(Path(), buf.Bytes(), 0o644)
}
