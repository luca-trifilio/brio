package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCorpus walks real Bruno collections (when available on the dev machine)
// and asserts every `.bru` file parses without error. Build mirrors are
// skipped to avoid double-counting.
func TestCorpus(t *testing.T) {
	roots := []string{
		"/Users/luca.trifilio/Progetti/bck_transaction/src/main/resources/api",
		"/Users/luca.trifilio/Progetti/bck_notification/src/main/resources/api",
		"/Users/luca.trifilio/Progetti/bck_material/core/core1/collections/satispay-api",
	}

	skipSegments := []string{
		string(os.PathSeparator) + "bin" + string(os.PathSeparator),
		string(os.PathSeparator) + "build" + string(os.PathSeparator),
		string(os.PathSeparator) + "node_modules" + string(os.PathSeparator),
	}

	var (
		total int
		fails int
	)

	for _, root := range roots {
		if _, err := os.Stat(root); err != nil {
			t.Logf("skip missing corpus root: %s", root)
			continue
		}
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".bru") {
				return nil
			}
			for _, seg := range skipSegments {
				if strings.Contains(path, seg) {
					return nil
				}
			}
			total++
			if _, perr := ParseFile(path); perr != nil {
				fails++
				t.Errorf("parse %s: %v", path, perr)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}

	if total == 0 {
		t.Skip("no corpus available — install bck_transaction/bck_notification/bck_material to run this test")
	}
	t.Logf("parsed %d .bru files, %d failures", total, fails)
}
