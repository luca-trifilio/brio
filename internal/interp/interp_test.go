package interp

import (
	"testing"

	"github.com/luca-trifilio/brio/internal/canonical"
)

func TestInterpolateBasic(t *testing.T) {
	s := NewScope(NamedLayer{Name: "env", Vars: []canonical.Var{{Name: "host", Value: "example.com"}}})
	got := s.Interpolate("https://{{host}}/v1/x")
	if got != "https://example.com/v1/x" {
		t.Errorf("got %q", got)
	}
}

func TestInterpolateOverride(t *testing.T) {
	s := NewScope(
		NamedLayer{Name: "env", Vars: []canonical.Var{{Name: "id", Value: "from-env"}}},
		NamedLayer{Name: "req", Vars: []canonical.Var{{Name: "id", Value: "from-req"}}},
	)
	if got := s.Interpolate("{{id}}"); got != "from-req" {
		t.Errorf("got %q", got)
	}
}

func TestInterpolateRecursive(t *testing.T) {
	s := NewScope(NamedLayer{Name: "env", Vars: []canonical.Var{
		{Name: "host", Value: "example.com"},
		{Name: "url", Value: "https://{{host}}/api"},
	}})
	if got := s.Interpolate("{{url}}/x"); got != "https://example.com/api/x" {
		t.Errorf("got %q", got)
	}
}

func TestInterpolateCycle(t *testing.T) {
	s := NewScope(NamedLayer{Name: "env", Vars: []canonical.Var{
		{Name: "a", Value: "{{b}}"},
		{Name: "b", Value: "{{a}}"},
	}})
	got := s.Interpolate("{{a}}") // should not infinite-loop
	_ = got
}

func TestInterpolateUnknownLeftAsIs(t *testing.T) {
	s := NewScope()
	if got := s.Interpolate("{{missing}}"); got != "{{missing}}" {
		t.Errorf("got %q", got)
	}
}

func TestRuntimeOverrides(t *testing.T) {
	s := NewScope(NamedLayer{Name: "env", Vars: []canonical.Var{{Name: "x", Value: "env-x"}}})
	s.PushOverrides("runtime", map[string]string{"x": "rt-x"})
	if got := s.Interpolate("{{x}}"); got != "rt-x" {
		t.Errorf("got %q", got)
	}
}
