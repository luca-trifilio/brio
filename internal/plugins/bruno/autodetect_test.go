package bruno

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAutodetectFindsBrunoDirInCWD(t *testing.T) {
	dir := t.TempDir()
	// Create a child dir that looks like a Bruno collection.
	coll := filepath.Join(dir, "my-coll")
	if err := os.MkdirAll(coll, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coll, "bruno.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(dir)

	got := New().Autodetect()
	want, _ := filepath.EvalSymlinks(coll)
	found := false
	for _, p := range got {
		resolved, _ := filepath.EvalSymlinks(p)
		if resolved == want {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected %s in autodetect results, got %v", want, got)
	}
}

func TestAutodetectImplementsInterface(t *testing.T) {
	var l interface{} = New()
	if _, ok := l.(interface{ Autodetect() []string }); !ok {
		t.Fatal("Loader should implement AutodetectLoader")
	}
}
