package interp

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/luca-trifilio/brio/internal/model"
)

func testCollectionRoot() string {
	return filepath.Join("..", "testdata", "collection")
}

func TestResolveAuthInheritance(t *testing.T) {
	c, err := model.LoadCollection(testCollectionRoot())
	if err != nil {
		t.Fatal(err)
	}
	all := c.AllRequests()
	if len(all) == 0 {
		t.Fatal("no requests found in collection")
	}

	// Every request in the testdata uses auth:inherit — they should all resolve
	// to the collection-level awsv4 config.
	var inherit *model.Request
	for _, r := range all {
		if r.AuthMode == model.AuthInherit || r.AuthMode == "" {
			inherit = r
			break
		}
	}
	if inherit == nil {
		t.Skip("no inheriting request found in collection")
	}

	resolved := ResolveAuth(c, inherit)
	if resolved.Mode != model.AuthModeAWSv4 {
		t.Errorf("want awsv4 from collection-level auth, got %q for %s",
			resolved.Mode, inherit.SourcePath)
	}
	if resolved.AWSv4 == nil {
		t.Fatal("awsv4 resolved but AWSv4 creds are nil")
	}
	if resolved.AWSv4.Region != "eu-west-1" {
		t.Errorf("want region eu-west-1, got %q", resolved.AWSv4.Region)
	}
}

func TestBuildScopeInterpolatesURL(t *testing.T) {
	c, err := model.LoadCollection(testCollectionRoot())
	if err != nil {
		t.Fatal(err)
	}

	env := c.Environments["test"]
	if env == nil {
		t.Fatal("testdata collection missing 'test' environment")
	}

	// Find a request whose URL contains a template variable.
	var req *model.Request
	for _, r := range c.AllRequests() {
		if strings.Contains(r.URL, "{{") {
			req = r
			break
		}
	}
	if req == nil {
		t.Fatal("no templated URL found in testdata collection")
	}

	scope := BuildScope(c, env, req, map[string]string{
		"_test_override": "ok",
		"transaction_id": "txn-abc-123",
	})

	// api_url and transaction_id must both be resolved — no {{ should remain.
	got := scope.Interpolate(req.URL)
	if strings.Contains(got, "{{") {
		t.Errorf("URL has unresolved template variables: %q", got)
	}
	if !strings.HasPrefix(got, "https://api.example.com") {
		t.Errorf("env var api_url not applied; got URL: %q", got)
	}

	// Runtime overrides must be visible through the scope.
	if v := scope.Interpolate("{{_test_override}}"); v != "ok" {
		t.Errorf("runtime override not applied: got %q, want %q", v, "ok")
	}

	_ = filepath.Base // keep import available for future tests
}
