package util

import (
	"net/http"
	"strings"
	"testing"

	"github.com/luca-trifilio/brio/internal/httpx"
)

func TestToCurl_Basic(t *testing.T) {
	r := httpx.ResolvedRequest{
		Method:  "GET",
		URL:     "https://example.com/api",
		Headers: http.Header{"X-Test": []string{"a b"}},
		QueryParams: [][2]string{
			{"q", "hello world"},
		},
	}
	got := ToCurl(r)
	if !strings.Contains(got, "curl") {
		t.Fatalf("missing curl: %s", got)
	}
	if !strings.Contains(got, "-H 'X-Test: a b'") {
		t.Fatalf("missing header quoting: %s", got)
	}
	if !strings.Contains(got, "q=hello+world") && !strings.Contains(got, "q=hello%20world") {
		t.Fatalf("missing query: %s", got)
	}
}

func TestToCurl_AWSv4Comment(t *testing.T) {
	r := httpx.ResolvedRequest{
		Method:   "POST",
		URL:      "https://example.com/api",
		AuthMode: "awsv4",
		Body:     []byte(`{"x":1}`),
	}
	got := ToCurl(r)
	if !strings.Contains(got, "AWS SigV4 auth") {
		t.Fatalf("missing awsv4 placeholder: %s", got)
	}
	if !strings.Contains(got, "-X POST") {
		t.Fatalf("missing method: %s", got)
	}
	if !strings.Contains(got, "--data") {
		t.Fatalf("missing body: %s", got)
	}
}
