package bruno

import (
	"path/filepath"
	"testing"
)

func TestDetectOnTestdata(t *testing.T) {
	root, err := filepath.Abs("../../testdata/collection")
	if err != nil {
		t.Fatal(err)
	}
	if !New().Detect(root) {
		t.Fatalf("Bruno loader should detect %s", root)
	}
}

func TestLoadOnTestdata(t *testing.T) {
	root, err := filepath.Abs("../../testdata/collection")
	if err != nil {
		t.Fatal(err)
	}
	c, diags, err := New().Load(root)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil collection")
	}
	if c.Format != "bruno" {
		t.Errorf("want format bruno, got %q", c.Format)
	}
	if c.Name == "" {
		t.Error("missing collection name")
	}
	if c.RootFolder == nil {
		t.Fatal("missing root folder")
	}
	if len(c.AllRequests()) == 0 {
		t.Error("expected at least one request")
	}
	for _, d := range diags {
		t.Logf("diag: %+v", d)
	}
}

func TestDetectRejectsNonBrunoDir(t *testing.T) {
	tmp := t.TempDir()
	if New().Detect(tmp) {
		t.Fatalf("empty dir should not be detected as Bruno collection")
	}
}
