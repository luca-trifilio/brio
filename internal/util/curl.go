// Package util has small helpers shared across packages.
package util

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/luca-trifilio/bruno-tui/internal/httpx"
)

// ToCurl renders r as a curl command. AWS SigV4 auth is replaced by a
// placeholder comment — callers should set credentials via env files.
func ToCurl(r httpx.ResolvedRequest) string {
	var b strings.Builder
	b.WriteString("curl")
	method := r.Method
	if method == "" {
		method = "GET"
	}
	if method != "GET" {
		b.WriteString(" -X ")
		b.WriteString(method)
	}

	finalURL := r.URL
	if len(r.QueryParams) > 0 {
		if u, err := url.Parse(finalURL); err == nil {
			q := u.Query()
			for _, p := range r.QueryParams {
				q.Add(p[0], p[1])
			}
			u.RawQuery = q.Encode()
			finalURL = u.String()
		}
	}

	for name, vals := range r.Headers {
		for _, v := range vals {
			b.WriteString(fmt.Sprintf(" -H %s", shellQuote(name+": "+v)))
		}
	}

	if r.AuthMode == "awsv4" {
		b.WriteString(" \\\n  # AWS SigV4 auth — set credentials in env file")
	}

	if len(r.Body) > 0 {
		b.WriteString(" --data ")
		b.WriteString(shellQuote(string(r.Body)))
	}
	b.WriteString(" ")
	b.WriteString(shellQuote(finalURL))
	return b.String()
}

// shellQuote single-quote-escapes s for POSIX shells.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n'\"\\$`*?#&|;<>()[]{}!~") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
