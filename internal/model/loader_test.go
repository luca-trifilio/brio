package model

import (
	"os"
	"testing"
)

func TestLoadBckTransaction(t *testing.T) {
	root := "/Users/luca.trifilio/Progetti/bck_transaction/src/main/resources/api"
	if _, err := os.Stat(root); err != nil {
		t.Skip("corpus not available")
	}
	c, err := LoadCollection(root)
	if err != nil {
		t.Fatal(err)
	}
	if c.Config.Name == "" {
		t.Error("missing collection name")
	}
	if len(c.Environments) < 4 {
		t.Errorf("want >=4 environments, got %d (%v)", len(c.Environments), envKeys(c))
	}
	for _, n := range []string{"Local", "Test", "Staging", "Prod"} {
		if _, ok := c.Environments[n]; !ok {
			t.Errorf("missing environment %s", n)
		}
	}
	if c.CollectionAuth == nil {
		t.Error("missing collection auth")
	} else if c.CollectionAuth.Mode != AuthModeAWSv4 {
		t.Errorf("want collection auth=awsv4, got %s", c.CollectionAuth.Mode)
	}
	all := c.AllRequests()
	if len(all) < 50 {
		t.Errorf("want >=50 requests, got %d", len(all))
	}
	t.Logf("loaded %d requests across %d folders", len(all), countFolders(c.Root))
}

func envKeys(c *Collection) []string {
	out := make([]string, 0, len(c.Environments))
	for k := range c.Environments {
		out = append(out, k)
	}
	return out
}

func countFolders(f *Folder) int {
	n := 1
	for _, sub := range f.Folders {
		n += countFolders(sub)
	}
	return n
}
