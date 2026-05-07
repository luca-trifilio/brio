package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "tilde expansion",
			input: "~/projects/api",
			want:  filepath.Join(home, "projects/api"),
		},
		{
			name:  "env var expansion",
			input: "$HOME/projects/api",
			want:  filepath.Join(home, "projects/api"),
		},
		{
			name:  "braced env var",
			input: "${HOME}/projects/api",
			want:  filepath.Join(home, "projects/api"),
		},
		{
			name:  "plain absolute path untouched",
			input: "/absolute/path",
			want:  "/absolute/path",
		},
		{
			name:  "cleans double slashes",
			input: "~/projects//api",
			want:  filepath.Join(home, "projects/api"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandPath(tt.input)
			if got != tt.want {
				t.Errorf("ExpandPath(%q) = %q; want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolvedCollections(t *testing.T) {
	// Create two real temp dirs and one non-existent path.
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	cfg := &Config{
		Collections: []string{dir1, missing, dir2},
	}

	var warned []string
	warn := func(p string) { warned = append(warned, p) }

	got := ResolvedCollections(cfg, warn)

	if len(got) != 2 {
		t.Fatalf("expected 2 resolved paths, got %d: %v", len(got), got)
	}
	if got[0] != dir1 || got[1] != dir2 {
		t.Errorf("unexpected resolved paths: %v", got)
	}
	if len(warned) != 1 || warned[0] != missing {
		t.Errorf("expected 1 warning for %q, got: %v", missing, warned)
	}
}

func TestResolvedCollections_Empty(t *testing.T) {
	cfg := &Config{}
	got := ResolvedCollections(cfg, func(string) {})
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}
