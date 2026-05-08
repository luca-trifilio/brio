package model

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/luca-trifilio/brio/internal/parser"
)

// alwaysSkip directories — Bruno's build mirrors and VCS noise.
var alwaysSkip = map[string]bool{
	".git":         true,
	"node_modules": true,
	"bin":          true,
	"build":        true,
}

// LoadCollection walks a Bruno collection directory and returns a fully
// populated Collection. Errors parsing individual `.bru` files are reported
// but do not abort the load (best-effort).
func LoadCollection(root string) (*Collection, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", abs)
	}

	c := &Collection{Path: abs, Environments: map[string]*Environment{}}

	// Parse bruno.json (optional but expected).
	if data, err := os.ReadFile(filepath.Join(abs, "bruno.json")); err == nil {
		_ = json.Unmarshal(data, &c.Config)
	}

	skip := skipSet(c.Config.Ignore)

	// Parse collection.bru if present.
	if doc, err := parser.ParseFile(filepath.Join(abs, "collection.bru")); err == nil {
		c.CollectionDoc = doc
		c.CollectionAuth = authFromDoc(doc)
		for _, subtype := range []string{"", "pre-request"} {
			if v := doc.FindBlock("vars", subtype); v != nil {
				for _, l := range v.Lines {
					c.CollectionVars = append(c.CollectionVars, Var{Name: l.Key, Value: l.Value, Disabled: l.Disabled})
				}
			}
		}
	}

	// Load environments.
	envDir := filepath.Join(abs, "environments")
	if entries, err := os.ReadDir(envDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".bru") {
				continue
			}
			path := filepath.Join(envDir, e.Name())
			doc, err := parser.ParseFile(path)
			if err != nil {
				continue
			}
			env := &Environment{
				Name: strings.TrimSuffix(e.Name(), ".bru"),
				Path: path,
			}
			if vb := doc.FindBlock("vars", ""); vb != nil {
				for _, l := range vb.Lines {
					env.Vars = append(env.Vars, Var{Name: l.Key, Value: l.Value, Disabled: l.Disabled})
				}
			}
			c.Environments[env.Name] = env
		}
	}

	// Walk folder tree.
	c.Root = &Folder{Path: abs, Name: c.DisplayName()}
	if err := loadFolder(c.Root, skip, true); err != nil {
		return nil, err
	}
	c.Root.SortChildren()

	return c, nil
}

func loadFolder(f *Folder, skip map[string]bool, isRoot bool) error {
	entries, err := os.ReadDir(f.Path)
	if err != nil {
		return err
	}

	// Parse folder.bru if present (skipped at the root — collection.bru fills
	// the same role).
	if !isRoot {
		if doc, err := parser.ParseFile(filepath.Join(f.Path, "folder.bru")); err == nil {
			f.FolderDoc = doc
			f.FolderAuth = authFromDoc(doc)
			if m := doc.FindBlock("meta", ""); m != nil {
				if v, ok := m.Get("name"); ok && v != "" {
					f.Name = v
				}
				if v, ok := m.Get("seq"); ok {
					if n, err := strconv.Atoi(v); err == nil {
						f.Seq = n
					}
				}
			}
			for _, subtype := range []string{"", "pre-request"} {
				if v := doc.FindBlock("vars", subtype); v != nil {
					for _, l := range v.Lines {
						f.FolderVars = append(f.FolderVars, Var{Name: l.Key, Value: l.Value, Disabled: l.Disabled})
					}
				}
			}
		}
	}

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			if alwaysSkip[name] || skip[name] || name == "environments" {
				continue
			}
			child := &Folder{Path: filepath.Join(f.Path, name), Name: name}
			if err := loadFolder(child, skip, false); err != nil {
				return err
			}
			// Skip folders with nothing useful (no requests, no subfolders, no folder.bru).
			if child.FolderDoc == nil && len(child.Folders) == 0 && len(child.Requests) == 0 {
				continue
			}
			f.Folders = append(f.Folders, child)
			continue
		}
		if !strings.HasSuffix(name, ".bru") {
			continue
		}
		// At the root level, skip collection.bru (already loaded).
		if isRoot && name == "collection.bru" {
			continue
		}
		// In nested folders, skip folder.bru (already loaded above).
		if !isRoot && name == "folder.bru" {
			continue
		}
		path := filepath.Join(f.Path, name)
		doc, err := parser.ParseFile(path)
		if err != nil {
			continue
		}
		// Only treat docs with an HTTP method block as requests.
		if !hasMethodBlock(doc) {
			continue
		}
		req := RequestFromDoc(path, doc)
		if req.Name == "" {
			req.Name = strings.TrimSuffix(name, ".bru")
		}
		f.Requests = append(f.Requests, req)
	}
	return nil
}

func hasMethodBlock(doc *parser.BruDoc) bool {
	for _, m := range methodBlocks {
		if doc.FindBlock(m, "") != nil {
			return true
		}
	}
	return false
}

func skipSet(ignore []string) map[string]bool {
	out := map[string]bool{}
	for _, s := range ignore {
		out[s] = true
	}
	return out
}
