package canonical

// Environment is a named set of variables that overrides collection vars.
type Environment struct {
	// Name is the display name (e.g. "dev", "prod").
	Name string
	// Path is the source file path (when applicable).
	Path string
	// Vars are the environment variable bindings.
	Vars []Var
}
