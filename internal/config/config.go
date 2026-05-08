package config

// Config is the top-level brio configuration loaded from config.toml.
type Config struct {
	// Collections is the list of configured collections. Each entry has a
	// path and an optional format hint (empty = autodetect via plugin
	// registry). Loaded from `[[collections]]` TOML tables; the legacy
	// `collections = ["..."]` flat-string form is also accepted at load time.
	Collections []CollectionEntry `toml:"collections"`
	// ActiveCollection is the name (or path) of the collection to focus on
	// startup. Empty means "first collection".
	ActiveCollection string `toml:"active_collection,omitempty"`
	// Theme names a theme plugin. Reserved for future use; currently unused.
	Theme string `toml:"theme,omitempty"`
	Hooks []Hook `toml:"hooks"`
}

// CollectionEntry is the in-memory representation of a configured collection.
// Format may be empty when auto-detection is desired.
type CollectionEntry struct {
	Path   string `toml:"path"`
	Format string `toml:"format,omitempty"`
}

// Entries returns the configured collections.
func (c *Config) Entries() []CollectionEntry {
	if c == nil {
		return nil
	}
	out := make([]CollectionEntry, len(c.Collections))
	copy(out, c.Collections)
	return out
}

// AddCollection appends a collection entry, deduping by expanded absolute path.
// Returns true if added, false if an entry with the same path already exists.
func (c *Config) AddCollection(path, format string) bool {
	if c == nil {
		return false
	}
	target := ExpandPath(path)
	for _, e := range c.Collections {
		if ExpandPath(e.Path) == target {
			return false
		}
	}
	c.Collections = append(c.Collections, CollectionEntry{Path: path, Format: format})
	return true
}

// RemoveCollection deletes the entry whose expanded path matches absPath.
// Returns true if an entry was removed.
func (c *Config) RemoveCollection(absPath string) bool {
	if c == nil {
		return false
	}
	target := ExpandPath(absPath)
	for i, e := range c.Collections {
		if ExpandPath(e.Path) == target {
			c.Collections = append(c.Collections[:i], c.Collections[i+1:]...)
			return true
		}
	}
	return false
}

// Hook describes a single credential-refresh hook.
type Hook struct {
	Name    string            `toml:"name"`
	Trigger Trigger           `toml:"trigger"`
	Script  Script            `toml:"script"`
	Output  Output            `toml:"output"`
	Vars    map[string]string `toml:"vars"` // output key → runtime var name
}

// Trigger defines when a hook fires.
type Trigger struct {
	Status []int  `toml:"status"`         // HTTP status codes (required)
	Body   string `toml:"body,omitempty"` // optional regex matched on response body
	Tier   string `toml:"tier,omitempty"` // "safe" | "caution" | "danger" | "" (any)
}

// Script describes the executable that runs when the hook fires.
type Script struct {
	Path string            `toml:"path"`
	Env  map[string]string `toml:"env,omitempty"` // extra env vars passed to the script
}

// Output describes how the script returns credentials back to brio.
type Output struct {
	Type   string `toml:"type"`             // "stdout" | "file"
	Path   string `toml:"path,omitempty"`   // only for type=file
	Format string `toml:"format,omitempty"` // only for type=file: "dotenv"|"json"|"yaml"|"bruno-env"
}
