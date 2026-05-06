package interp

import (
	"path/filepath"
	"strings"

	"github.com/luca-trifilio/bruno-tui/internal/model"
)

// ResolveAuth walks the inheritance chain for the given request and returns
// the first concrete (non-inherit) auth block.
//
// Order: request → folder ancestors (leaf → root) → collection.
// Behaviour:
//   - if the request's own auth block is non-inherit, return it
//   - otherwise, walk parent folders (their folder.bru auth blocks)
//   - finally fall back to the collection's auth block
//   - if none is found, return AuthBlock{Mode: AuthNone}
func ResolveAuth(c *model.Collection, req *model.Request) *model.AuthBlock {
	if req.Auth != nil && req.Auth.Mode != "" && req.Auth.Mode != model.AuthInherit {
		return req.Auth
	}
	for _, f := range folderChainFor(c, req) {
		if f.FolderAuth != nil && f.FolderAuth.Mode != "" && f.FolderAuth.Mode != model.AuthInherit {
			return f.FolderAuth
		}
	}
	if c.CollectionAuth != nil && c.CollectionAuth.Mode != "" && c.CollectionAuth.Mode != model.AuthInherit {
		return c.CollectionAuth
	}
	return &model.AuthBlock{Mode: model.AuthNone}
}

// folderChainFor returns the folder chain leaf → root (excluding the
// collection root sentinel) for the given request.
func folderChainFor(c *model.Collection, req *model.Request) []*model.Folder {
	if c == nil || c.Root == nil {
		return nil
	}
	rel, err := filepath.Rel(c.Root.Path, filepath.Dir(req.SourcePath))
	if err != nil {
		return nil
	}
	if rel == "." || rel == "" {
		return nil
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	var chain []*model.Folder
	cur := c.Root
	for _, p := range parts {
		var next *model.Folder
		for _, sub := range cur.Folders {
			if filepath.Base(sub.Path) == p {
				next = sub
				break
			}
		}
		if next == nil {
			break
		}
		chain = append([]*model.Folder{next}, chain...) // prepend → leaf-first
		cur = next
	}
	return chain
}

// BuildScope assembles the variable scope for executing req in env.
// Layers (lowest→highest): collection → environment → folder vars (root→leaf)
// → request vars (vars + vars:pre-request) → runtime overrides.
func BuildScope(c *model.Collection, env *model.Environment, req *model.Request, runtime map[string]string) *VarScope {
	s := &VarScope{}
	if c != nil {
		s.Push("collection", c.CollectionVars)
	}
	if env != nil {
		s.Push("env:"+env.Name, env.Vars)
	}
	chain := folderChainFor(c, req)
	// Walk chain root → leaf (chain is leaf → root, so reverse iterate).
	for i := len(chain) - 1; i >= 0; i-- {
		s.Push("folder:"+chain[i].Name, chain[i].FolderVars)
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
