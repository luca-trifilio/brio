package bruno

import (
	"os"
	"path/filepath"

	"github.com/luca-trifilio/brio/internal/brunoprefs"
)

// Autodetect returns absolute paths to candidate Bruno collections discovered
// from Bruno's preferences.json (lastOpenedCollections) and the current
// working directory. Each candidate is verified via Detect before inclusion.
// The result is deduplicated by absolute path.
func (l Loader) Autodetect() []string {
	seen := map[string]bool{}
	var out []string

	add := func(p string) {
		abs, err := filepath.Abs(p)
		if err != nil {
			return
		}
		if seen[abs] {
			return
		}
		if !l.Detect(abs) {
			return
		}
		seen[abs] = true
		out = append(out, abs)
	}

	if paths, err := brunoprefs.CollectionPaths(); err == nil {
		for _, p := range paths {
			add(p)
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		add(cwd)
		// Also peek one level into CWD: Bruno collections often sit alongside
		// other directories in a workspace folder.
		if entries, err := os.ReadDir(cwd); err == nil {
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				add(filepath.Join(cwd, e.Name()))
			}
		}
	}

	return out
}
