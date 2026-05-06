package interp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luca-trifilio/bruno-tui/internal/model"
)

func TestResolveAuthBckTransaction(t *testing.T) {
	root := "/Users/luca.trifilio/Progetti/bck_transaction/src/main/resources/api"
	if _, err := os.Stat(root); err != nil {
		t.Skip("corpus not available")
	}
	c, err := model.LoadCollection(root)
	if err != nil {
		t.Fatal(err)
	}
	all := c.AllRequests()
	if len(all) == 0 {
		t.Fatal("no requests")
	}
	// Most requests inherit; chain should resolve to collection-level awsv4.
	var inherit *model.Request
	for _, r := range all {
		if r.AuthMode == model.AuthInherit || r.AuthMode == "" {
			inherit = r
			break
		}
	}
	if inherit == nil {
		t.Fatal("expected at least one inheriting request")
	}
	resolved := ResolveAuth(c, inherit)
	if resolved.Mode != model.AuthModeAWSv4 {
		t.Errorf("want awsv4 from collection, got %s for %s", resolved.Mode, inherit.SourcePath)
	}
}

func TestBuildScopeInterpolatesURL(t *testing.T) {
	root := "/Users/luca.trifilio/Progetti/bck_transaction/src/main/resources/api"
	if _, err := os.Stat(root); err != nil {
		t.Skip("corpus not available")
	}
	c, err := model.LoadCollection(root)
	if err != nil {
		t.Fatal(err)
	}
	env := c.Environments["Test"]
	if env == nil {
		t.Fatal("missing Test env")
	}
	// Find a request that uses {{transaction_url}}.
	var req *model.Request
	for _, r := range c.AllRequests() {
		if strings.Contains(r.URL, "{{transaction_url}}") {
			req = r
			break
		}
	}
	if req == nil {
		t.Skip("no request templated on transaction_url")
	}
	scope := BuildScope(c, env, req, map[string]string{"transaction_id": "rt-123"})
	got := scope.Interpolate(req.URL)
	if strings.Contains(got, "{{transaction_url}}") {
		t.Errorf("URL not interpolated: %q", got)
	}
	if !strings.Contains(got, "test-transaction.satispay.aws") {
		t.Errorf("env not applied: %q", got)
	}
	// runtime override visible
	got2 := scope.Interpolate("{{transaction_id}}")
	if got2 != "rt-123" {
		t.Errorf("override not applied: %q", got2)
	}
	_ = filepath.Base // keep import in case future tests need it
}
