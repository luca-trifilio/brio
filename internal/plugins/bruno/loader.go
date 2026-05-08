package bruno

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/luca-trifilio/brio/internal/canonical"
	"github.com/luca-trifilio/brio/internal/model"
	"github.com/luca-trifilio/brio/internal/plugins"
)

const formatName = "bruno"

// Loader is the Bruno CollectionLoader plugin.
type Loader struct{}

// New returns a Loader instance.
func New() *Loader { return &Loader{} }

// Name returns the canonical format name.
func (Loader) Name() string { return formatName }

// Detect reports whether root looks like a Bruno collection: presence of
// a bruno.json file or any *.bru file at any depth (cheap glob: check the
// root directory only).
func (Loader) Detect(root string) bool {
	if st, err := os.Stat(filepath.Join(root, "bruno.json")); err == nil && !st.IsDir() {
		return true
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".bru") {
			return true
		}
	}
	// Look one level deep for *.bru — Bruno collections always have at
	// least one nested folder with .bru files.
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub, err := os.ReadDir(filepath.Join(root, e.Name()))
		if err != nil {
			continue
		}
		for _, s := range sub {
			if !s.IsDir() && strings.HasSuffix(s.Name(), ".bru") {
				return true
			}
		}
	}
	return false
}

// Load parses the Bruno collection at root and returns the canonical view.
// Per-file parse failures bubble up as canonical Diagnostics rather than
// fatal errors, mirroring the existing model.LoadCollection contract.
func (Loader) Load(root string) (*canonical.Collection, []canonical.Diagnostic, error) {
	m, err := model.LoadCollection(root)
	if err != nil {
		return nil, nil, err
	}
	c := adaptCollection(m)
	// model.LoadCollection currently swallows per-file parse errors; once it
	// surfaces them, translate to canonical.Diagnostic here.
	var diags []canonical.Diagnostic
	return c, diags, nil
}

func init() {
	plugins.Register(New())
}
