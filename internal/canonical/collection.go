// Package canonical defines the format-agnostic collection model used by brio.
//
// All loader plugins (Bruno, Postman, Insomnia, etc.) translate their native
// format into these types. The TUI, interpolation, auth resolution, and HTTP
// execution layers consume only canonical types — they have no knowledge of
// the source format.
package canonical

// Collection is a top-level API collection rooted at Root.
type Collection struct {
	// Name is a display name for the collection.
	Name string
	// Description is an optional human-readable description.
	Description string
	// Root is the absolute filesystem path the collection was loaded from.
	Root string
	// Format identifies the loader that produced this collection (e.g. "bruno").
	Format string
	// RootFolder is the synthetic root folder containing all top-level
	// requests and sub-folders.
	RootFolder *Folder
	// Environments lists named environments available for this collection.
	Environments []*Environment
	// Vars are collection-level variables (root of the variable scope chain).
	Vars []Var
	// Auth is the collection-level auth block (root of the auth inheritance chain).
	Auth *AuthBlock
	// Settings are collection-wide settings.
	Settings Settings
	// Scripts holds collection-level pre/post-request script source as plain
	// text. Either field may be empty.
	Scripts ScriptBlock
	// Extra is a bag for opaque vendor metadata. Hot paths must not read this.
	Extra map[string]any
}

// Folder is one node in the collection tree. A folder may contain requests
// and nested folders.
type Folder struct {
	// Path is the absolute filesystem path of the folder (when applicable).
	Path string
	// Name is the display name of the folder.
	Name string
	// Seq is the sort key within its parent (0 when absent).
	Seq int
	// Auth is the folder-level auth block (one link in the inheritance chain).
	Auth *AuthBlock
	// Vars are folder-level variables.
	Vars []Var
	// Folders are nested sub-folders.
	Folders []*Folder
	// Requests are requests directly under this folder.
	Requests []*Request
	// Scripts holds folder-level pre/post-request script source as plain text.
	Scripts ScriptBlock
	// Extra is a bag for opaque vendor metadata.
	Extra map[string]any
}

// EnvByName returns the environment with the given name (or nil).
func (c *Collection) EnvByName(name string) *Environment {
	if c == nil || name == "" {
		return nil
	}
	for _, e := range c.Environments {
		if e.Name == name {
			return e
		}
	}
	return nil
}

// EnvNames returns the environment names in declaration order.
func (c *Collection) EnvNames() []string {
	if c == nil {
		return nil
	}
	out := make([]string, 0, len(c.Environments))
	for _, e := range c.Environments {
		out = append(out, e.Name)
	}
	return out
}

// DisplayName returns Name (falling back to Root's basename).
func (c *Collection) DisplayName() string {
	if c == nil {
		return ""
	}
	if c.Name != "" {
		return c.Name
	}
	return c.Root
}

// AllRequests returns every request in DFS order.
func (c *Collection) AllRequests() []*Request {
	if c.RootFolder == nil {
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
	walk(c.RootFolder)
	return out
}
