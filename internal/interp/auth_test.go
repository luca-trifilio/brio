package interp

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/luca-trifilio/brio/internal/canonical"
	"github.com/luca-trifilio/brio/internal/plugins/bruno"
)

func testCollectionRoot() string {
	return filepath.Join("..", "testdata", "collection")
}

func loadCanonical(t *testing.T) *canonical.Collection {
	t.Helper()
	c, _, err := bruno.New().Load(testCollectionRoot())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestResolveAuthInheritance(t *testing.T) {
	c := loadCanonical(t)
	all := c.AllRequests()
	if len(all) == 0 {
		t.Fatal("no requests found in collection")
	}

	// Every request in the testdata uses auth:inherit — they should all resolve
	// to the collection-level awsv4 config.
	var inherit *canonical.Request
	for _, r := range all {
		if r.Auth == nil || r.Auth.Mode == canonical.AuthInherit || r.Auth.Mode == "" {
			inherit = r
			break
		}
	}
	if inherit == nil {
		t.Skip("no inheriting request found in collection")
	}

	resolved := ResolveAuth(c, inherit)
	if resolved.Mode != canonical.AuthAWSv4 {
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
	c := loadCanonical(t)
	env := c.EnvByName("test")
	if env == nil {
		t.Fatal("testdata collection missing 'test' environment")
	}

	// Find a request whose URL contains a template variable.
	var req *canonical.Request
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

	got := scope.Interpolate(req.URL)
	if strings.Contains(got, "{{") {
		t.Errorf("URL has unresolved template variables: %q", got)
	}
	if !strings.HasPrefix(got, "https://api.example.com") {
		t.Errorf("env var api_url not applied; got URL: %q", got)
	}

	if v := scope.Interpolate("{{_test_override}}"); v != "ok" {
		t.Errorf("runtime override not applied: got %q, want %q", v, "ok")
	}
}
