package history

import (
	"path/filepath"
	"testing"
	"time"
)

func TestAppendAndLoad(t *testing.T) {
	dir := t.TempDir()
	s := New(filepath.Join(dir, "h.jsonl"))
	for i, m := range []string{"GET", "POST", "DELETE"} {
		err := s.Append(Entry{
			Timestamp:  time.Date(2024, 1, 1, 0, 0, i, 0, time.UTC),
			Collection: "c",
			Method:     m,
			URL:        "https://example/" + m,
			StatusCode: 200,
			ElapsedMS:  int64(10 + i),
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d", len(got))
	}
	// newest first → DELETE, POST, GET
	if got[0].Method != "DELETE" || got[2].Method != "GET" {
		t.Errorf("order: %v", []string{got[0].Method, got[1].Method, got[2].Method})
	}
}

func TestPreviewTrim(t *testing.T) {
	dir := t.TempDir()
	s := New(filepath.Join(dir, "h.jsonl"))
	big := make([]byte, MaxPreviewBytes+1000)
	for i := range big {
		big[i] = 'a'
	}
	if err := s.Append(Entry{Method: "GET", URL: "u", ResponsePreview: string(big)}); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Load()
	if len(got[0].ResponsePreview) != MaxPreviewBytes {
		t.Errorf("preview not trimmed: %d", len(got[0].ResponsePreview))
	}
}
