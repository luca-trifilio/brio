package config

// Config is the top-level brio configuration loaded from config.toml.
type Config struct {
	Collections []string `toml:"collections"`
	Hooks       []Hook   `toml:"hooks"`
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
