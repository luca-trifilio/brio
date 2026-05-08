package plugins

import (
	"errors"
	"testing"

	"github.com/luca-trifilio/brio/internal/canonical"
)

type fakeLoader struct {
	name   string
	detect bool
}

func (f *fakeLoader) Name() string         { return f.name }
func (f *fakeLoader) Detect(_ string) bool { return f.detect }
func (f *fakeLoader) Load(_ string) (*canonical.Collection, []canonical.Diagnostic, error) {
	return nil, nil, errors.New("not implemented")
}

func TestRegistryResolveByName(t *testing.T) {
	r := NewRegistry()
	bruno := &fakeLoader{name: "bruno", detect: true}
	r.Register(bruno)

	got, err := r.Resolve("bruno", "/tmp/x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != bruno {
		t.Fatalf("expected bruno loader, got %v", got)
	}
}

func TestRegistryResolveByDetect(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeLoader{name: "no", detect: false})
	yes := &fakeLoader{name: "yes", detect: true}
	r.Register(yes)

	got, err := r.Resolve("", "/tmp/x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != yes {
		t.Fatalf("expected yes loader, got %v", got)
	}
}

func TestRegistryResolveUnknownFormat(t *testing.T) {
	r := NewRegistry()
	if _, err := r.Resolve("missing", "/tmp/x"); err == nil {
		t.Fatal("expected error for unknown format")
	}
}

func TestRegistryResolveNoDetect(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeLoader{name: "no", detect: false})
	if _, err := r.Resolve("", "/tmp/x"); err == nil {
		t.Fatal("expected error when no loader detects")
	}
}

func TestDetectCollectionsAtRoot(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeLoader{name: "fake", detect: true})
	dir := t.TempDir()
	got, fmtName, err := r.DetectCollections(dir)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if fmtName != "fake" {
		t.Errorf("want fmt fake, got %q", fmtName)
	}
	if len(got) == 0 {
		t.Errorf("expected root in results, got %v", got)
	}
}

func TestDetectCollectionsNoMatch(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeLoader{name: "x", detect: false})
	dir := t.TempDir()
	if _, _, err := r.DetectCollections(dir); err == nil {
		t.Fatal("expected error when nothing detected")
	}
}

func TestRegistryDetectAll(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeLoader{name: "a", detect: true})
	r.Register(&fakeLoader{name: "b", detect: false})
	r.Register(&fakeLoader{name: "c", detect: true})

	names := r.DetectAll("/tmp/x")
	if len(names) != 2 {
		t.Fatalf("expected 2 detected loaders, got %d (%v)", len(names), names)
	}
}
