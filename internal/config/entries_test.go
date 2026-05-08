package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfigFile(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	brioDir := filepath.Join(dir, "brio")
	if err := os.MkdirAll(brioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(brioDir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestEntriesReturnsCollections(t *testing.T) {
	cfg := &Config{Collections: []CollectionEntry{
		{Path: "/a"},
		{Path: "/b", Format: "bruno"},
	}}
	got := cfg.Entries()
	if len(got) != 2 {
		t.Fatalf("want 2 entries, got %d", len(got))
	}
	if got[0].Path != "/a" || got[0].Format != "" {
		t.Errorf("entry[0] = %+v", got[0])
	}
	if got[1].Path != "/b" || got[1].Format != "bruno" {
		t.Errorf("entry[1] = %+v", got[1])
	}
}

func TestEntriesNilSafe(t *testing.T) {
	var cfg *Config
	if got := cfg.Entries(); got != nil {
		t.Errorf("nil config Entries() = %v; want nil", got)
	}
}

func TestEntriesEmpty(t *testing.T) {
	cfg := &Config{}
	if got := cfg.Entries(); len(got) != 0 {
		t.Errorf("empty Entries() = %v; want []", got)
	}
}

func TestAddCollectionDedupes(t *testing.T) {
	cfg := &Config{}
	if !cfg.AddCollection("/a", "bruno") {
		t.Fatal("first add should return true")
	}
	if cfg.AddCollection("/a", "bruno") {
		t.Fatal("dup add should return false")
	}
	if !cfg.AddCollection("/b", "") {
		t.Fatal("second distinct add should return true")
	}
	if got := len(cfg.Collections); got != 2 {
		t.Errorf("want 2 collections, got %d", got)
	}
}

func TestRemoveCollection(t *testing.T) {
	cfg := &Config{Collections: []CollectionEntry{{Path: "/a"}, {Path: "/b"}}}
	if !cfg.RemoveCollection("/a") {
		t.Fatal("remove existing should return true")
	}
	if cfg.RemoveCollection("/a") {
		t.Fatal("remove missing should return false")
	}
	if got := len(cfg.Collections); got != 1 || cfg.Collections[0].Path != "/b" {
		t.Errorf("after remove: %+v", cfg.Collections)
	}
}

func TestLoadLegacyStringFormParsedAsEntries(t *testing.T) {
	writeConfigFile(t, `collections = ["/x", "/y"]`+"\n")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(cfg.Collections) != 2 {
		t.Fatalf("want 2 entries, got %d (%+v)", len(cfg.Collections), cfg.Collections)
	}
	if cfg.Collections[0].Path != "/x" || cfg.Collections[1].Path != "/y" {
		t.Errorf("unexpected entries: %+v", cfg.Collections)
	}
}

func TestLoadTableFormParsedAsEntries(t *testing.T) {
	body := `[[collections]]` + "\n" +
		`path = "/x"` + "\n" +
		`format = "bruno"` + "\n" +
		"\n" +
		`[[collections]]` + "\n" +
		`path = "/y"` + "\n"
	writeConfigFile(t, body)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(cfg.Collections) != 2 {
		t.Fatalf("want 2, got %d", len(cfg.Collections))
	}
	if cfg.Collections[0].Path != "/x" || cfg.Collections[0].Format != "bruno" {
		t.Errorf("entry[0] = %+v", cfg.Collections[0])
	}
	if cfg.Collections[1].Path != "/y" || cfg.Collections[1].Format != "" {
		t.Errorf("entry[1] = %+v", cfg.Collections[1])
	}
}

func TestSaveRoundTripUsesTableForm(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := EnsureDir(); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{Collections: []CollectionEntry{
		{Path: "/x", Format: "bruno"},
		{Path: "/y"},
	}}
	if err := Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got.Collections) != 2 {
		t.Fatalf("want 2 entries, got %d", len(got.Collections))
	}
	if got.Collections[0].Path != "/x" || got.Collections[0].Format != "bruno" {
		t.Errorf("entry[0] = %+v", got.Collections[0])
	}
	if got.Collections[1].Path != "/y" {
		t.Errorf("entry[1] = %+v", got.Collections[1])
	}
}
