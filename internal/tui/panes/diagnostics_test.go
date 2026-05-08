package panes

import (
	"strings"
	"testing"

	"github.com/luca-trifilio/brio/internal/canonical"
)

func TestDiagnosticsRender(t *testing.T) {
	d := NewDiagnostics()
	d.Open([]canonical.Diagnostic{
		{Severity: canonical.SeverityError, Path: "/p/a.bru", Line: 12, Msg: "parse failed"},
		{Severity: canonical.SeverityWarn, Path: "/p/b.bru", Msg: "missing field"},
	})

	v := d.View(80, 20)
	if !strings.Contains(v, "parse failed") || !strings.Contains(v, "missing field") {
		t.Errorf("rendered view missing diagnostic text: %q", v)
	}
	if !strings.Contains(v, "/p/a.bru:12") {
		t.Errorf("rendered view missing path:line: %q", v)
	}
	if !strings.Contains(v, SeverityIcon(canonical.SeverityError)) {
		t.Error("rendered view missing severity icon")
	}
}

func TestDiagnosticsEmpty(t *testing.T) {
	d := NewDiagnostics()
	d.Open(nil)
	v := d.View(80, 20)
	if !strings.Contains(v, "No diagnostics") {
		t.Errorf("empty view missing placeholder: %q", v)
	}
}
