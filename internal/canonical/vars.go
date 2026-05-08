package canonical

// Var is a key/value variable entry. Local marks variables that should not
// be exposed outside their declaring scope (mirrors Bruno's `@var(secret)`).
type Var struct {
	Name     string
	Value    string
	Disabled bool
	Local    bool
}
