package interp

import (
	"path/filepath"
	"strings"

	"github.com/luca-trifilio/brio/internal/canonical"
)

// ResolveAuth walks the inheritance chain for the given request and returns
// the first concrete (non-inherit) auth block.
//
// Order: request → folder ancestors (leaf → root) → collection.
// Behavior:
//   - if the request's own auth block is non-inherit, return it
//   - otherwise, walk parent folders (their folder.bru auth blocks)
//   - finally fall back to the collection's auth block
//   - if none is found, return AuthBlock{Mode: AuthNone}.
func ResolveAuth(c *canonical.Collection, req *canonical.Request) *canonical.AuthBlock {
	if req != nil && req.Auth != nil && req.Auth.Mode != "" && req.Auth.Mode != canonical.AuthInherit {
		return req.Auth
	}
	for _, f := range FolderChainFor(c, req) {
		if f.Auth != nil && f.Auth.Mode != "" && f.Auth.Mode != canonical.AuthInherit {
			return f.Auth
		}
	}
	if c != nil && c.Auth != nil && c.Auth.Mode != "" && c.Auth.Mode != canonical.AuthInherit {
		return c.Auth
	}
	return &canonical.AuthBlock{Mode: canonical.AuthNone}
}

// FolderChainFor returns the folder chain leaf → root (excluding the
// collection root sentinel) for the given request.
func FolderChainFor(c *canonical.Collection, req *canonical.Request) []*canonical.Folder {
	if c == nil || c.RootFolder == nil || req == nil {
		return nil
	}
	rel, err := filepath.Rel(c.RootFolder.Path, filepath.Dir(req.SourcePath))
	if err != nil {
		return nil
	}
	if rel == "." || rel == "" {
		return nil
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	var chain []*canonical.Folder
	cur := c.RootFolder
	for _, p := range parts {
		var next *canonical.Folder
		for _, sub := range cur.Folders {
			if filepath.Base(sub.Path) == p {
				next = sub
				break
			}
		}
		if next == nil {
			break
		}
		chain = append([]*canonical.Folder{next}, chain...) // prepend → leaf-first
		cur = next
	}
	return chain
}

// BuildScope assembles the variable scope for executing req in env.
// Layers (lowest→highest): collection → environment → folder vars (root→leaf)
// → request vars (vars + vars:pre-request) → runtime overrides.
func BuildScope(c *canonical.Collection, env *canonical.Environment, req *canonical.Request, runtime map[string]string) *VarScope {
	s := &VarScope{}
	if c != nil {
		s.Push("collection", c.Vars)
	}
	if env != nil {
		s.Push("env:"+env.Name, env.Vars)
	}
	chain := FolderChainFor(c, req)
	// Walk chain root → leaf (chain is leaf → root, so reverse iterate).
	for i := len(chain) - 1; i >= 0; i-- {
		s.Push("folder:"+chain[i].Name, chain[i].Vars)
	}
	if req != nil {
		s.Push("request:vars", req.Vars)
		s.Push("request:pre-vars", req.PreVars)
	}
	if len(runtime) > 0 {
		s.PushOverrides("runtime", runtime)
	}
	return s
}
