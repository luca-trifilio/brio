// Package interp implements variable interpolation and auth inheritance for
// resolved Bruno requests.
package interp

import (
	"regexp"
	"strings"

	"github.com/luca-trifilio/brio/internal/canonical"
)

// VarScope is a layered map (lowest priority first). Lookup walks the layers
// from highest to lowest priority. Layers are keyed slices of canonical.Var so
// that order is preserved for diagnostics, but lookup uses a map cache.
type VarScope struct {
	layers []layer
}

type layer struct {
	name string
	vars map[string]string
}

// NewScope builds a scope with the given layers, in order from lowest to
// highest priority. Disabled vars are skipped.
//
// Conventional layer order (lowest → highest):
//
//	collection vars → environment → folder vars (root → leaf) → request vars → runtime overrides
func NewScope(layers ...NamedLayer) *VarScope {
	s := &VarScope{}
	for _, l := range layers {
		m := map[string]string{}
		for _, v := range l.Vars {
			if v.Disabled {
				continue
			}
			m[v.Name] = v.Value
		}
		s.layers = append(s.layers, layer{name: l.Name, vars: m})
	}
	return s
}

// NamedLayer pairs a label with a slice of variables.
type NamedLayer struct {
	Name string
	Vars []canonical.Var
}

// Push adds a layer at the highest priority.
func (s *VarScope) Push(name string, vars []canonical.Var) {
	m := map[string]string{}
	for _, v := range vars {
		if v.Disabled {
			continue
		}
		m[v.Name] = v.Value
	}
	s.layers = append(s.layers, layer{name: name, vars: m})
}

// PushOverrides adds a string-keyed map (used for runtime overrides).
func (s *VarScope) PushOverrides(name string, m map[string]string) {
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	s.layers = append(s.layers, layer{name: name, vars: cp})
}

// Lookup returns the highest-priority value for k.
func (s *VarScope) Lookup(k string) (string, bool) {
	for i := len(s.layers) - 1; i >= 0; i-- {
		if v, ok := s.layers[i].vars[k]; ok {
			return v, true
		}
	}
	return "", false
}

// Snapshot returns all keys with their resolved values.
func (s *VarScope) Snapshot() map[string]string {
	out := map[string]string{}
	for _, l := range s.layers {
		for k, v := range l.vars {
			out[k] = v
		}
	}
	return out
}

var tplRe = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_\-\.]*)\s*\}\}`)

// Interpolate replaces every `{{name}}` token in s using the scope. Resolution
// is recursive (a value may itself contain `{{...}}`) with a cycle guard.
// Unknown variables are left as-is.
func (s *VarScope) Interpolate(input string) string {
	return s.interp(input, map[string]bool{}, 0)
}

const maxInterpDepth = 16

func (s *VarScope) interp(input string, seen map[string]bool, depth int) string {
	if depth > maxInterpDepth {
		return input
	}
	return tplRe.ReplaceAllStringFunc(input, func(m string) string {
		sub := tplRe.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		key := sub[1]
		if seen[key] {
			return m // cycle — leave token literal
		}
		val, ok := s.Lookup(key)
		if !ok {
			return m
		}
		if !strings.Contains(val, "{{") {
			return val
		}
		newSeen := map[string]bool{key: true}
		for k := range seen {
			newSeen[k] = true
		}
		return s.interp(val, newSeen, depth+1)
	})
}
