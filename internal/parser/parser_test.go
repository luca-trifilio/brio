package parser

import (
	"strings"
	"testing"
)

func TestParseMeta(t *testing.T) {
	doc, err := ParseString(`meta {
  name: Get Transaction
  type: http
  seq: 1
}`)
	if err != nil {
		t.Fatal(err)
	}
	m := doc.FindBlock("meta", "")
	if m == nil {
		t.Fatal("missing meta block")
	}
	if v, _ := m.Get("name"); v != "Get Transaction" {
		t.Errorf("name = %q", v)
	}
	if v, _ := m.Get("seq"); v != "1" {
		t.Errorf("seq = %q", v)
	}
}

func TestParseGetWithDisabled(t *testing.T) {
	doc, err := ParseString(`get {
  url: {{base}}/v1/x
  body: none
  auth: inherit
}

params:query {
  limit: 10
  ~offset: 0
}`)
	if err != nil {
		t.Fatal(err)
	}
	q := doc.FindBlock("params", "query")
	if q == nil {
		t.Fatal("missing params:query")
	}
	if len(q.Lines) != 2 {
		t.Fatalf("want 2 lines, got %d", len(q.Lines))
	}
	if !q.Lines[1].Disabled {
		t.Error("offset should be disabled")
	}
}

func TestParseBodyJSONWithBraces(t *testing.T) {
	doc, err := ParseString(`body:json {
  {
    "a": 1,
    "b": { "c": [1,2,3] }
  }
}`)
	if err != nil {
		t.Fatal(err)
	}
	b := doc.FindBlock("body", "json")
	if b == nil || !b.IsRaw {
		t.Fatal("missing or non-raw body:json")
	}
	if !strings.Contains(b.Raw, `"c": [1,2,3]`) {
		t.Errorf("raw missing nested JSON: %q", b.Raw)
	}
}

func TestParseAuthAwsv4(t *testing.T) {
	doc, err := ParseString(`auth:awsv4 {
  accessKeyId: {{AWS_ACCESS_KEY}}
  secretAccessKey: {{AWS_SECRET_KEY}}
  service: execute-api
  region: eu-west-1
  profileName:
}`)
	if err != nil {
		t.Fatal(err)
	}
	a := doc.FindBlock("auth", "awsv4")
	if a == nil {
		t.Fatal("missing auth:awsv4")
	}
	if v, _ := a.Get("region"); v != "eu-west-1" {
		t.Errorf("region = %q", v)
	}
	if v, _ := a.Get("profileName"); v != "" {
		t.Errorf("profileName = %q", v)
	}
}

func TestParseScriptWithBraces(t *testing.T) {
	doc, err := ParseString(`script:pre-request {
  // hello
  if (x) {
    bru.setVar("k", "v");
  }
}`)
	if err != nil {
		t.Fatal(err)
	}
	s := doc.FindBlock("script", "pre-request")
	if s == nil || !s.IsRaw {
		t.Fatal("missing or non-raw script:pre-request")
	}
	if !strings.Contains(s.Raw, `bru.setVar`) {
		t.Errorf("raw missing JS: %q", s.Raw)
	}
}

func TestParseHeadersAndVars(t *testing.T) {
	doc, err := ParseString(`headers {
  Content-Type: application/json
  ~X-Disabled: foo
}

vars {
  transaction_id:
}`)
	if err != nil {
		t.Fatal(err)
	}
	h := doc.FindBlock("headers", "")
	if v, _ := h.Get("Content-Type"); v != "application/json" {
		t.Errorf("CT = %q", v)
	}
	v := doc.FindBlock("vars", "")
	if v == nil {
		t.Fatal("missing vars")
	}
}

func TestParseMultipleBlocksOrder(t *testing.T) {
	doc, err := ParseString(`meta { name: A }
get { url: u }
headers { X: y }`)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Blocks) != 3 {
		t.Fatalf("want 3 blocks, got %d", len(doc.Blocks))
	}
	if doc.Blocks[0].Name != "meta" || doc.Blocks[1].Name != "get" || doc.Blocks[2].Name != "headers" {
		t.Errorf("order: %v", doc.Blocks)
	}
}

func TestEmptyBlock(t *testing.T) {
	doc, err := ParseString(`vars {
}`)
	if err != nil {
		t.Fatal(err)
	}
	if v := doc.FindBlock("vars", ""); v == nil || len(v.Lines) != 0 {
		t.Errorf("empty vars block parse failed: %+v", v)
	}
}
