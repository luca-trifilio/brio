package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Registry maps format names to CollectionLoader implementations.
type Registry struct {
	mu      sync.RWMutex
	loaders map[string]CollectionLoader
}

// NewRegistry constructs an empty Registry.
func NewRegistry() *Registry {
	return &Registry{loaders: map[string]CollectionLoader{}}
}

// Register adds a loader to the registry, keyed by its Name().
// Re-registering the same name overwrites the previous entry.
func (r *Registry) Register(l CollectionLoader) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.loaders[l.Name()] = l
}

// Resolve returns the loader registered under format. When format is empty,
// the registry probes registered loaders via Detect(root) and returns the
// first match. Returns an error when no loader can be resolved.
func (r *Registry) Resolve(format, root string) (CollectionLoader, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if format != "" {
		if l, ok := r.loaders[format]; ok {
			return l, nil
		}
		return nil, fmt.Errorf("no loader registered for format %q", format)
	}
	for _, l := range r.loaders {
		if l.Detect(root) {
			return l, nil
		}
	}
	return nil, fmt.Errorf("no loader detected the collection at %q", root)
}

// Names returns the alphabetically-sorted names of every registered loader.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.loaders))
	for name := range r.loaders {
		out = append(out, name)
	}
	sortStrings(out)
	return out
}

// sortStrings is a tiny in-place sort (avoids importing sort here).
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// DetectAll returns the names of every loader whose Detect(root) reports true.
func (r *Registry) DetectAll(root string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []string
	for name, l := range r.loaders {
		if l.Detect(root) {
			out = append(out, name)
		}
	}
	return out
}

// defaultRegistry is the package-level registry that loaders self-register
// into via init() blocks.
var defaultRegistry = NewRegistry()

// Default returns the package-level registry.
func Default() *Registry { return defaultRegistry }

// Register adds a loader to the package-level registry.
func Register(l CollectionLoader) { defaultRegistry.Register(l) }

// Resolve looks up a loader on the package-level registry.
func Resolve(format, root string) (CollectionLoader, error) {
	return defaultRegistry.Resolve(format, root)
}

// DetectCollections probes registered loaders against root and one level
// of children. Returns the matching format name and the list of collection
// roots discovered (deduplicated, absolute). Errors when no loader matches
// anywhere in the scanned tree.
func DetectCollections(root string) ([]string, string, error) {
	return defaultRegistry.DetectCollections(root)
}

// DetectCollections is the registry-bound implementation of the package
// helper. Behavior: probe root itself; if no match, probe immediate
// subdirectories. The first loader to detect *any* candidate wins, and
// every directory it accepts (root and/or its children) is returned.
func (r *Registry) DetectCollections(root string) ([]string, string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, "", err
	}
	if !st.IsDir() {
		return nil, "", fmt.Errorf("%q is not a directory", abs)
	}

	r.mu.RLock()
	loaders := make([]CollectionLoader, 0, len(r.loaders))
	for _, l := range r.loaders {
		loaders = append(loaders, l)
	}
	r.mu.RUnlock()

	var children []string
	if entries, err := os.ReadDir(abs); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				children = append(children, filepath.Join(abs, e.Name()))
			}
		}
	}

	for _, l := range loaders {
		var found []string
		seen := map[string]bool{}
		if l.Detect(abs) {
			seen[abs] = true
			found = append(found, abs)
		}
		for _, c := range children {
			if seen[c] {
				continue
			}
			if l.Detect(c) {
				seen[c] = true
				found = append(found, c)
			}
		}
		if len(found) > 0 {
			return found, l.Name(), nil
		}
	}
	return nil, "", fmt.Errorf("no loader detected a collection at %q", abs)
}
