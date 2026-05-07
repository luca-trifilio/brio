package model

import (
	"strconv"
	"strings"

	"github.com/luca-trifilio/brio/internal/parser"
)

// Request is the typed view of a single `.bru` HTTP request file.
type Request struct {
	// SourcePath is the absolute filesystem path of the .bru file.
	SourcePath string
	// Name comes from `meta.name` (falls back to filename without extension).
	Name string
	// Seq is `meta.seq` for ordering within a folder (0 when missing).
	Seq int
	// Method is GET/POST/PUT/PATCH/DELETE etc.
	Method HTTPMethod
	// URL is the raw URL template (still contains {{vars}}).
	URL string
	// BodyMode is what the method block declared (`body: none|json|text|...`).
	BodyMode string
	// AuthMode is what the method block declared (`auth: inherit|awsv4|...`).
	AuthMode AuthMode
	// Headers, params, vars are preserved in source order.
	Headers     []Header
	QueryParams []Param
	PathParams  []Param
	Vars        []Var // vars { ... }
	PreVars     []Var // vars:pre-request { ... }
	// Auth holds the auth-specific block (e.g. auth:awsv4) declared in
	// THIS file. Inheritance resolution lives elsewhere.
	Auth *AuthBlock
	// Body holds the request body for json/text/xml.
	Body Body
	// Settings is the `settings { ... }` block when present.
	Settings Settings
	// HasPreRequestScript is true when the file has a `script:pre-request`
	// block (we don't execute it in MVP; the TUI surfaces a hint).
	HasPreRequestScript bool
	// PostResponseScript holds the raw content of a `script:post-response`
	// block. We support a subset of the Bruno JS API (bru.setVar + res.body).
	PostResponseScript string
	// Doc contains the raw parsed AST (kept around for debugging / advanced
	// resolution).
	Doc *parser.BruDoc
}

// methodBlocks lists the well-known method block names in the order Bruno
// renders them.
var methodBlocks = []string{"get", "post", "put", "patch", "delete", "head", "options", "trace", "connect"}

// FromDoc converts a parsed BruDoc into a Request.
func RequestFromDoc(path string, doc *parser.BruDoc) *Request {
	r := &Request{SourcePath: path, Doc: doc}

	if m := doc.FindBlock("meta", ""); m != nil {
		if v, ok := m.Get("name"); ok {
			r.Name = v
		}
		if v, ok := m.Get("seq"); ok {
			if n, err := strconv.Atoi(v); err == nil {
				r.Seq = n
			}
		}
	}

	for _, name := range methodBlocks {
		if mb := doc.FindBlock(name, ""); mb != nil {
			r.Method = HTTPMethod(strings.ToUpper(name))
			if v, ok := mb.Get("url"); ok {
				r.URL = v
			}
			if v, ok := mb.Get("body"); ok {
				r.BodyMode = v
			}
			if v, ok := mb.Get("auth"); ok {
				r.AuthMode = AuthMode(v)
			}
			break
		}
	}

	if h := doc.FindBlock("headers", ""); h != nil {
		for _, l := range h.Lines {
			r.Headers = append(r.Headers, Header{Name: l.Key, Value: l.Value, Disabled: l.Disabled})
		}
	}
	if q := doc.FindBlock("params", "query"); q != nil {
		for _, l := range q.Lines {
			r.QueryParams = append(r.QueryParams, Param{Name: l.Key, Value: l.Value, Disabled: l.Disabled})
		}
	}
	if p := doc.FindBlock("params", "path"); p != nil {
		for _, l := range p.Lines {
			r.PathParams = append(r.PathParams, Param{Name: l.Key, Value: l.Value, Disabled: l.Disabled})
		}
	}
	if v := doc.FindBlock("vars", ""); v != nil {
		for _, l := range v.Lines {
			r.Vars = append(r.Vars, Var{Name: l.Key, Value: l.Value, Disabled: l.Disabled})
		}
	}
	if v := doc.FindBlock("vars", "pre-request"); v != nil {
		for _, l := range v.Lines {
			r.PreVars = append(r.PreVars, Var{Name: l.Key, Value: l.Value, Disabled: l.Disabled})
		}
	}

	r.Auth = authFromDoc(doc)

	// Body: pick the first body:* block (json/text/xml/graphql) that exists.
	for _, bt := range []string{"json", "text", "xml", "graphql"} {
		if bb := doc.FindBlock("body", bt); bb != nil {
			r.Body = Body{Type: bt, Raw: bb.Raw}
			break
		}
	}

	if s := doc.FindBlock("settings", ""); s != nil {
		if v, ok := s.Get("encodeUrl"); ok {
			r.Settings.EncodeURL = v == "true"
		}
		if v, ok := s.Get("timeout"); ok {
			if n, err := strconv.Atoi(v); err == nil {
				r.Settings.TimeoutMS = n
			}
		}
	}

	if doc.FindBlock("script", "pre-request") != nil {
		r.HasPreRequestScript = true
	}
	if b := doc.FindBlock("script", "post-response"); b != nil {
		r.PostResponseScript = b.Raw
	}

	return r
}

// authFromDoc extracts the auth declaration from a single .bru document.
// Returns nil when the doc has no auth block at all.
func authFromDoc(doc *parser.BruDoc) *AuthBlock {
	a := doc.FindBlock("auth", "")
	if a == nil {
		return nil
	}
	mode, _ := a.Get("mode")
	ab := &AuthBlock{Mode: AuthMode(mode)}
	switch ab.Mode {
	case AuthModeAWSv4:
		ab.AWSv4 = awsv4FromDoc(doc)
	}
	// Even when mode is something else, capture awsv4 details if the block
	// exists — useful for debugging and for resolution that might surface
	// auth:awsv4 without a top-level auth block.
	if ab.AWSv4 == nil {
		ab.AWSv4 = awsv4FromDoc(doc)
	}
	return ab
}

func awsv4FromDoc(doc *parser.BruDoc) *AuthAWSv4 {
	b := doc.FindBlock("auth", "awsv4")
	if b == nil {
		return nil
	}
	get := func(k string) string { v, _ := b.Get(k); return v }
	return &AuthAWSv4{
		AccessKeyID:     get("accessKeyId"),
		SecretAccessKey: get("secretAccessKey"),
		SessionToken:    get("sessionToken"),
		Service:         get("service"),
		Region:          get("region"),
		ProfileName:     get("profileName"),
	}
}
