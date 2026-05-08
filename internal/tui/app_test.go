package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/luca-trifilio/brio/internal/canonical"
)

func testModel() *Model {
	coll := &canonical.Collection{
		Root: "/fake/root",
		Name: "test",
		RootFolder: &canonical.Folder{
			Path: "/fake/root",
			Requests: []*canonical.Request{
				{Name: "req-a", URL: "http://a"},
				{Name: "req-b", URL: "http://b"},
				{Name: "req-c", URL: "http://c"},
			},
		},
	}
	m := NewModel([]*canonical.Collection{coll}, nil)
	m.tree.Expanded[coll.Root] = true
	m.tree.Rebuild()
	return m
}

func pressKey(m *Model, key string) *Model {
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	nm := next.(*Model)
	return nm
}

//nolint:unused // kept for completeness; some tests use special keys.
func pressSpecial(m *Model, t tea.KeyType) *Model {
	next, _ := m.Update(tea.KeyMsg{Type: t})
	nm := next.(*Model)
	return nm
}

func TestYDoesNotMoveCursor(t *testing.T) {
	m := testModel()
	m.focused = PaneTree

	initialCursor := m.tree.Cursor

	m = pressKey(m, "y")

	if m.tree.Cursor != initialCursor {
		t.Errorf("pressing 'y' moved tree cursor from %d to %d", initialCursor, m.tree.Cursor)
	}
	if !m.pendingYank {
		t.Error("pressing 'y' should set pendingYank=true")
	}
}

func TestYCDoesNotMoveCursor(t *testing.T) {
	m := testModel()
	m.focused = PaneTree

	initialCursor := m.tree.Cursor

	m = pressKey(m, "y")
	m = pressKey(m, "c")

	if m.tree.Cursor != initialCursor {
		t.Errorf("pressing 'yc' moved tree cursor from %d to %d", initialCursor, m.tree.Cursor)
	}
	if m.pendingYank {
		t.Error("after 'yc' pendingYank should be false")
	}
}

func TestJMovesDown(t *testing.T) {
	m := testModel()
	m.focused = PaneTree

	initialCursor := m.tree.Cursor
	m = pressKey(m, "j")

	if m.tree.Cursor != initialCursor+1 {
		t.Errorf("pressing 'j' should move cursor down: got %d want %d", m.tree.Cursor, initialCursor+1)
	}
}

func TestStripJSONComments(t *testing.T) {
	cases := []struct{ in, want string }{
		{
			in:   `{"a": 1} // line comment`,
			want: `{"a": 1} `,
		},
		{
			in:   `{"url": "http://example.com"} // keep url intact`,
			want: `{"url": "http://example.com"} `,
		},
		{
			in:   "{\n  \"parent_payment_uid\": \"abc-123\", // toBusiness implicit id\n  \"amount_unit\": 10\n  // \"disabled_field\": 99\n}",
			want: "{\n  \"parent_payment_uid\": \"abc-123\", \n  \"amount_unit\": 10\n  \n}",
		},
		{
			in:   `{"a": /* block */ 1}`,
			want: `{"a":  1}`,
		},
		{
			in:   `{"a": "has // inside string"}`,
			want: `{"a": "has // inside string"}`,
		},
		{
			in:   `{"a": "has /* inside */ string"}`,
			want: `{"a": "has /* inside */ string"}`,
		},
	}
	for _, c := range cases {
		got := stripJSONComments(c.in)
		if got != c.want {
			t.Errorf("stripJSONComments(%q)\n got  %q\n want %q", c.in, got, c.want)
		}
	}
}
