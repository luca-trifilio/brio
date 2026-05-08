package panes

import (
	"testing"

	"github.com/luca-trifilio/brio/internal/canonical"
)

func TestMatchScore(t *testing.T) {
	cs := []*canonical.Collection{
		{Name: "Payments API", Root: "/srv/payments"},
		{Name: "User Service", Root: "/srv/users"},
		{Name: "", Root: "/var/legacy/billing"},
	}

	cases := []struct {
		query string
		want  []int // indexes that should match (in any order, all >=0)
	}{
		{query: "", want: []int{0, 1, 2}},
		{query: "payments", want: []int{0}},
		{query: "user", want: []int{1}},    // matches both Name "User Service" and path "users"
		{query: "billing", want: []int{2}}, // base path fallback
		{query: "zzznope", want: nil},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			got := []int{}
			for i, c := range cs {
				if matchScore(c, tc.query) >= 0 {
					got = append(got, i)
				}
			}
			if len(got) != len(tc.want) {
				t.Fatalf("query %q: got %v, want %v", tc.query, got, tc.want)
			}
			seen := map[int]bool{}
			for _, i := range got {
				seen[i] = true
			}
			for _, want := range tc.want {
				if !seen[want] {
					t.Errorf("query %q: missing %d in result %v", tc.query, want, got)
				}
			}
		})
	}
}
