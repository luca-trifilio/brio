package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCorpus walks the bundled testdata Bruno collection and asserts every
// `.bru` file parses without error.
func TestCorpus(t *testing.T) {
	roots := []string{filepath.Join("..", "testdata", "collection")}

	skipSegments := []string{
		string(os.PathSeparator) + "bin" + string(os.PathSeparator),
		string(os.PathSeparator) + "build" + string(os.PathSeparator),
		string(os.PathSeparator) + "node_modules" + string(os.PathSeparator),
	}

	var total, fails int

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
		t.Fatal("no .bru files found — testdata fixture may be missing")
	}
	t.Logf("parsed %d .bru files, %d failures", total, fails)
}
