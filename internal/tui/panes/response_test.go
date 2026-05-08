package panes

import (
	"strings"
	"testing"
)

func TestApplyJQ(t *testing.T) {
	body := []byte(`{"id":"abc","amount":100,"items":[{"k":"a"},{"k":"b"}]}`)

	tests := []struct {
		name       string
		query      string
		wantSubstr string
		wantErr    bool
	}{
		{
			name:       "empty query passes through",
			query:      "",
			wantSubstr: `"id"`,
		},
		{
			name:       "select field",
			query:      ".id",
			wantSubstr: `"abc"`,
		},
		{
			name:       "array index",
			query:      ".items[0].k",
			wantSubstr: `"a"`,
		},
		{
			name:    "invalid syntax",
			query:   ".[",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := applyJQ(body, tt.query)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("want error, got nil; out=%q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(got, tt.wantSubstr) {
				t.Errorf("output %q does not contain %q", got, tt.wantSubstr)
			}
		})
	}
}

func TestApplyJQNonJSON(t *testing.T) {
	_, err := applyJQ([]byte("not json"), ".foo")
	if err == nil {
		t.Error("expected error for non-JSON body")
	}
}
