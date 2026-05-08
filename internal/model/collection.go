package model

import (
	"path/filepath"
	"sort"

	"github.com/luca-trifilio/brio/internal/parser"
)

// BrunoConfig is the on-disk shape of `bruno.json`.
type BrunoConfig struct {
	Version string   `json:"version"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Ignore  []string `json:"ignore"`
}

// Collection is a top-level Bruno collection rooted at Path.
type Collection struct {
	Path string
	// Config is the parsed bruno.json (Name, ignore rules, etc.).
	Config BrunoConfig
	// CollectionDoc is the parsed `collection.bru` (auth, vars, scripts) when
	// present.
	CollectionDoc *parser.BruDoc
	// CollectionAuth is the auth block declared in collection.bru (root of
	// the inheritance chain).
	CollectionAuth *AuthBlock
	// CollectionVars are vars from collection.bru `vars { ... }` (rare).
	CollectionVars []Var
	// Root is the top-level folder containing all requests / sub-folders.
	Root *Folder
	// Environments keyed by name (filename without extension).
	Environments map[string]*Environment
}

// Folder is one directory inside the collection. Folders may contain
// requests and nested folders. Order is by `meta.seq` then by name.
type Folder struct {
	Path string
	Name string
	Seq  int
	// FolderDoc is the parsed `folder.bru` when present (auth/vars).
	FolderDoc  *parser.BruDoc
	FolderAuth *AuthBlock
	FolderVars []Var

	Folders  []*Folder
	Requests []*Request
}

// Environment is one `environments/<name>.bru` file.
type Environment struct {
	Name string
	Path string
	Vars []Var
}

// AllRequests returns every request in DFS order.
func (c *Collection) AllRequests() []*Request {
	if c.Root == nil {
		return nil
	}
	var out []*Request
	var walk func(f *Folder)
	walk = func(f *Folder) {
		out = append(out, f.Requests...)
		for _, sub := range f.Folders {
			walk(sub)
		}
	}
	walk(c.Root)
	return out
}

// SortChildren orders the folder's requests and subfolders by Seq, then name.
func (f *Folder) SortChildren() {
	sortByseqName := func(seq func(int) int, name func(int) string, n int, swap func(i, j int)) {
		sort.SliceStable(make([]int, n), func(i, j int) bool {
			si, sj := seq(i), seq(j)
			if si != sj && si > 0 && sj > 0 {
				return si < sj
			}
			if si > 0 && sj == 0 {
				return true
			}
			if sj > 0 && si == 0 {
				return false
			}
			return name(i) < name(j)
		})
		_ = swap // unused — sort below uses real slices
	}
	_ = sortByseqName

	sort.SliceStable(f.Requests, func(i, j int) bool {
		ri, rj := f.Requests[i], f.Requests[j]
		if ri.Seq != rj.Seq && ri.Seq > 0 && rj.Seq > 0 {
			return ri.Seq < rj.Seq
		}
		if ri.Seq > 0 && rj.Seq == 0 {
			return true
		}
		if rj.Seq > 0 && ri.Seq == 0 {
			return false
		}
		return ri.Name < rj.Name
	})
	sort.SliceStable(f.Folders, func(i, j int) bool {
		ai, aj := f.Folders[i], f.Folders[j]
		if ai.Seq != aj.Seq && ai.Seq > 0 && aj.Seq > 0 {
			return ai.Seq < aj.Seq
		}
		if ai.Seq > 0 && aj.Seq == 0 {
			return true
		}
		if aj.Seq > 0 && ai.Seq == 0 {
			return false
		}
		return ai.Name < aj.Name
	})
	for _, sub := range f.Folders {
		sub.SortChildren()
	}
}

// DefaultName returns the collection name, falling back to the directory base.
func (c *Collection) DisplayName() string {
	if c.Config.Name != "" {
		return c.Config.Name
	}
	return filepath.Base(c.Path)
}
