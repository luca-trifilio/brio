package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExecuteSimpleGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Echo-Path", r.URL.Path)
		w.Header().Set("X-Echo-Query", r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	e := NewExecutor()
	resp := e.Execute(ResolvedRequest{
		Method: "GET",
		URL:    srv.URL + "/v1/x",
		QueryParams: [][2]string{
			{"a", "1"}, {"b", "two"},
		},
	})
	if resp.Err != nil {
		t.Fatal(resp.Err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status=%d", resp.StatusCode)
	}
	if got := resp.Headers.Get("X-Echo-Query"); !strings.Contains(got, "a=1") {
		t.Errorf("query missing: %q", got)
	}
}

func TestSigV4SignaturePresent(t *testing.T) {
	var seenAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	e := NewExecutor()
	resp := e.Execute(ResolvedRequest{
		Method:   "POST",
		URL:      srv.URL + "/v1/foo",
		Body:     []byte(`{"x":1}`),
		BodyType: "json",
		AuthMode: "awsv4",
		AWSv4: &AWSCreds{
			AccessKeyID:     "AKIDEXAMPLE",
			SecretAccessKey: "SECRETEXAMPLE",
			Service:         "execute-api",
			Region:          "eu-west-1",
		},
	})
	if resp.Err != nil {
		t.Fatal(resp.Err)
	}
	if !strings.HasPrefix(seenAuth, "AWS4-HMAC-SHA256 ") {
		t.Errorf("Authorization missing/invalid: %q", seenAuth)
	}
	if !strings.Contains(seenAuth, "Credential=AKIDEXAMPLE/") {
		t.Errorf("Credential missing: %q", seenAuth)
	}
	if !strings.Contains(seenAuth, "execute-api") {
		t.Errorf("service missing in scope: %q", seenAuth)
	}
}
