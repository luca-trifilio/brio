// Package plugins defines the CollectionLoader plugin interface and a
// registry that maps format names to concrete loader implementations.
//
// Loader plugins translate a source format (Bruno, Postman, Insomnia, etc.)
// into the format-agnostic types in internal/canonical. The TUI consumes
// only canonical types and does not know which loader produced them.
package plugins

import "github.com/luca-trifilio/brio/internal/canonical"

// CollectionLoader is the contract every collection-format plugin satisfies.
type CollectionLoader interface {
	// Name returns the canonical format name (e.g. "bruno", "postman").
	Name() string
	// Detect reports whether this loader can handle the directory at root.
	// Implementations should be cheap (filesystem stat / glob, not full parse).
	Detect(root string) bool
	// Load parses the collection at root, returning the canonical view, any
	// non-fatal diagnostics, and a fatal error when the collection cannot be
	// loaded at all.
	Load(root string) (*canonical.Collection, []canonical.Diagnostic, error)
}

// AutodetectLoader is an optional interface a CollectionLoader may implement
// to suggest candidate collection roots without an explicit user-supplied
// path. The import flow probes loaders for this capability via type assertion.
type AutodetectLoader interface {
	// Autodetect returns deduplicated absolute paths to candidate collections.
	Autodetect() []string
}
