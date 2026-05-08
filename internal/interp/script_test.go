package interp

import (
	"strings"
	"testing"
)

func TestRunPostResponseScript(t *testing.T) {
	body := []byte(`{"id":"pay_abc123","amount":100,"nested":{"token":"tok_xyz"}}`)

	tests := []struct {
		name   string
		script string
		want   map[string]string
	}{
		{
			name:   "dot notation top-level field",
			script: `bru.setVar("parent_payment_uid", res.body.id)`,
			want:   map[string]string{"parent_payment_uid": "pay_abc123"},
		},
		{
			name:   "bracket notation",
			script: `bru.setVar("parent_payment_uid", res.body["id"])`,
			want:   map[string]string{"parent_payment_uid": "pay_abc123"},
		},
		{
			name:   "nested path",
			script: `bru.setVar("tok", res.body.nested.token)`,
			want:   map[string]string{"tok": "tok_xyz"},
		},
		{
			name: "multiple setVar calls",
			script: `
				bru.setVar("parent_payment_uid", res.body.id);
				bru.setVar("amount", res.body.amount);
			`,
			want: map[string]string{
				"parent_payment_uid": "pay_abc123",
				"amount":             "100",
			},
		},
		{
			name:   "unknown path silently skipped",
			script: `bru.setVar("x", res.body.does_not_exist)`,
			want:   map[string]string{},
		},
		{
			name:   "empty script",
			script: "",
			want:   map[string]string{},
		},
		{
			name:   "invalid json body",
			script: `bru.setVar("x", res.body.id)`,
			want:   map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyArg := body
			if tt.name == "invalid json body" {
				bodyArg = []byte("not json")
			}
			got := RunPostResponseScript(tt.script, bodyArg)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("key %q: got %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestRunPreRequestScript_UUID(t *testing.T) {
	// Mirrors the collection.bru script:pre-request in bck_transaction.
	script := `
		const uuidv4 = require('uuid').v4;
		let id = uuidv4();
		bru.setEnvVar("uuid", id);
	`
	got := RunPreRequestScript(script)
	uuidVal, ok := got["uuid"]
	if !ok {
		t.Fatal("expected 'uuid' key in result")
	}
	// Basic format check: 8-4-4-4-12 hex chars
	parts := strings.Split(uuidVal, "-")
	if len(parts) != 5 {
		t.Fatalf("uuid %q: want 5 dash-separated parts, got %d", uuidVal, len(parts))
	}
	// Two calls should produce different UUIDs.
	got2 := RunPreRequestScript(script)
	if got["uuid"] == got2["uuid"] {
		t.Error("two calls returned the same UUID — should be random")
	}
}

func TestRunPreRequestScript_NoUUID(t *testing.T) {
	// Script without uuid usage → empty result.
	script := `console.log("hello");`
	got := RunPreRequestScript(script)
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestRunPostResponseScript_RealRefundPattern(t *testing.T) {
	// Mirrors what create.bru does: extract id from payment response → parent_payment_uid
	script := `bru.setVar("parent_payment_uid", res.body.id)`
	body := []byte(`{
		"id": "pay_0197f2e4-0b6d-7f24-b0d4-34a90b9a1c8e",
		"type": "SHOP_TO_CONSUMER",
		"amount_unit": 100,
		"currency": "EUR",
		"status": "PENDING"
	}`)

	got := RunPostResponseScript(script, body)
	const want = "pay_0197f2e4-0b6d-7f24-b0d4-34a90b9a1c8e"
	if got["parent_payment_uid"] != want {
		t.Errorf("got %q, want %q", got["parent_payment_uid"], want)
	}
}
