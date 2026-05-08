package canonical

// Severity is the level of a Diagnostic.
type Severity string

// Severity levels.
const (
	SeverityError Severity = "error"
	SeverityWarn  Severity = "warn"
	SeverityInfo  Severity = "info"
)

// Diagnostic describes a problem encountered while loading a collection.
// Diagnostics are non-fatal: a loader returns as much of the collection as
// it could parse, alongside a slice of Diagnostics for the issues found.
type Diagnostic struct {
	Severity Severity
	// Path is the source file (or directory) the diagnostic is anchored to.
	Path string
	// Line is the 1-based line number when applicable, 0 otherwise.
	Line int
	// Msg is a human-readable description of the issue.
	Msg string
}
