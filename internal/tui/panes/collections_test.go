package panes

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/luca-trifilio/brio/internal/config"
	_ "github.com/luca-trifilio/brio/internal/plugins/bruno" // registers Bruno
)

// keyMsg builds a tea.KeyMsg for a single character or named key.
func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEscape}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func TestCollectionsModalCancel(t *testing.T) {
	m := NewCollections()
	m.Open(nil)
	res, _ := m.Update(keyMsg("esc"))
	if res != CollMgrCanceled {
		t.Errorf("want Canceled, got %v", res)
	}
}

func TestCollectionsModalSaveEmptyList(t *testing.T) {
	m := NewCollections()
	m.Open(nil)
	res, _ := m.Update(keyMsg("s"))
	if res != CollMgrSaved {
		t.Errorf("want Saved, got %v", res)
	}
	if len(m.Entries()) != 0 {
		t.Errorf("expected empty entries, got %v", m.Entries())
	}
}

func TestCollectionsModalRemove(t *testing.T) {
	m := NewCollections()
	m.Open([]config.CollectionEntry{{Path: "/a"}, {Path: "/b"}})
	// cursor on /a (0). press d.
	if _, _ = m.Update(keyMsg("d")); len(m.Entries()) != 1 {
		t.Fatalf("after remove expected 1, got %v", m.Entries())
	}
	if m.Entries()[0].Path != "/b" {
		t.Errorf("expected /b, got %v", m.Entries()[0])
	}
}

func TestCollectionsModalAddFlowAutodetect(t *testing.T) {
	dir := t.TempDir()
	coll := filepath.Join(dir, "c1")
	if err := os.MkdirAll(coll, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coll, "bruno.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	m := NewCollections()
	m.Open(nil)

	// Press 'a' to start add.
	if _, _ = m.Update(keyMsg("a")); m.step != stepPath {
		// With one plugin (Bruno), we skip plugin step.
		t.Fatalf("after 'a' want stepPath, got %v", m.step)
	}

	// Toggle autodetect.
	if _, _ = m.Update(keyMsg("tab")); !m.autodetect {
		t.Fatal("tab should select autodetect")
	}

	// Run autodetect (Enter).
	if _, _ = m.Update(keyMsg("enter")); m.step != stepCandList {
		t.Fatalf("after autodetect Enter want stepCandList, got %v step (status=%q)", m.step, m.statusMsg)
	}
	if len(m.candidates) == 0 {
		t.Fatal("expected candidates from autodetect")
	}

	// Confirm move to confirm step.
	if _, _ = m.Update(keyMsg("enter")); m.step != stepConfirm {
		t.Fatalf("want stepConfirm, got %v", m.step)
	}

	// Press 'y' to commit.
	if _, _ = m.Update(keyMsg("y")); m.step != stepList {
		t.Fatalf("after y want stepList, got %v", m.step)
	}
	if len(m.Entries()) == 0 {
		t.Fatal("expected at least one entry committed")
	}

	// Save.
	res, _ := m.Update(keyMsg("s"))
	if res != CollMgrSaved {
		t.Errorf("want Saved, got %v", res)
	}
}

func TestCollectionsModalCandToggle(t *testing.T) {
	m := NewCollections()
	m.Open(nil)
	m.candidates = []string{"/a", "/b"}
	m.candSelected = []bool{true, true}
	m.step = stepCandList

	// Toggle off /a.
	_, _ = m.Update(keyMsg(" "))
	if m.candSelected[0] {
		t.Error("expected /a to be deselected")
	}
	// Select all.
	_, _ = m.Update(keyMsg("a"))
	if !m.candSelected[0] || !m.candSelected[1] {
		t.Error("expected all selected after 'a'")
	}
	// None.
	_, _ = m.Update(keyMsg("n"))
	if m.candSelected[0] || m.candSelected[1] {
		t.Error("expected none selected after 'n'")
	}
}
