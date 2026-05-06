// Package parser implements a hand-rolled parser for Bruno's `.bru`
// block-based plain-text format.
//
// Grammar (informal):
//
//	doc      := block*
//	block    := name (":" subtype)? ws "{" body "}"
//	body     := raw_body | kv_body
//	kv_body  := kv_line*
//	kv_line  := ("~")? key ":" value
//	raw_body := arbitrary text (with balanced braces) for body:* / script:* / tests
//
// The parser is line-oriented and keeps brace depth so that JSON bodies (which
// contain `{` and `}`) parse cleanly inside `body:json`.
package parser

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// rawBlockNames holds blocks whose body is treated as opaque text rather than
// key/value pairs. Both whole-name and "name:" prefix matches are checked.
var rawBlockPrefixes = []string{
	"body:",   // body, body:json, body:text, body:xml, body:graphql, ...
	"script:", // script:pre-request, script:post-response
}

var rawBlockExact = map[string]bool{
	"tests":   true,
	"docs":    true,
	"body":    true, // edge: bare `body` block (rare)
	"script":  true,
	"query":   true, // body:graphql:vars uses query — defensive
	"graphql": true,
}

func isRawBlock(name, subtype string) bool {
	full := name
	if subtype != "" {
		full = name + ":" + subtype
	}
	if rawBlockExact[full] {
		return true
	}
	for _, p := range rawBlockPrefixes {
		if strings.HasPrefix(full+":", p) || strings.HasPrefix(full, p) {
			return true
		}
	}
	// Special: "tests" and "docs" are raw even without subtype (already in map).
	return false
}

// ParseFile reads and parses a `.bru` file.
func ParseFile(path string) (*BruDoc, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	doc, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return doc, nil
}

// ParseString is a convenience wrapper around Parse.
func ParseString(s string) (*BruDoc, error) {
	return Parse(strings.NewReader(s))
}

// Parse reads a `.bru` document from r.
func Parse(r io.Reader) (*BruDoc, error) {
	br := bufio.NewReader(r)
	// Read the whole doc into memory — `.bru` files are small.
	data, err := io.ReadAll(br)
	if err != nil {
		return nil, err
	}
	p := &parser{src: string(data)}
	return p.parseDoc()
}

type parser struct {
	src string
	pos int // byte offset
	ln  int // 1-based line for error messages
}

func (p *parser) parseDoc() (*BruDoc, error) {
	doc := &BruDoc{}
	for {
		p.skipWhitespaceAndComments()
		if p.pos >= len(p.src) {
			break
		}
		blk, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		doc.Blocks = append(doc.Blocks, blk)
	}
	return doc, nil
}

// parseBlock parses `name[:subtype] { ... }`.
func (p *parser) parseBlock() (Block, error) {
	header, err := p.readHeader()
	if err != nil {
		return Block{}, err
	}
	name, subtype := splitHeader(header)
	if name == "" {
		return Block{}, p.errf("empty block name")
	}
	raw := isRawBlock(name, subtype)
	body, err := p.readBracedBody(raw)
	if err != nil {
		return Block{}, fmt.Errorf("block %q: %w", header, err)
	}
	blk := Block{Name: name, Subtype: subtype}
	if raw {
		blk.IsRaw = true
		blk.Raw = strings.Trim(body, "\n")
	} else {
		blk.Lines = parseKVLines(body)
	}
	return blk, nil
}

// readHeader reads up to (but not including) the opening `{`. It strips a
// trailing colon-less / colon-bearing identifier sequence and returns it.
func (p *parser) readHeader() (string, error) {
	start := p.pos
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if c == '{' {
			break
		}
		if c == '\n' {
			p.ln++
		}
		p.pos++
	}
	if p.pos >= len(p.src) {
		return "", p.errf("expected '{' after block header")
	}
	header := strings.TrimSpace(p.src[start:p.pos])
	return header, nil
}

// readBracedBody assumes p.src[p.pos] == '{', consumes through the matching
// '}', and returns the inner contents.
//
// When raw is true (body:* / script:* / tests / docs), the body may contain
// arbitrary text including code with apostrophes, so we only track JSON-style
// double-quoted strings (which can't legally contain unescaped braces) and
// `//` line comments. Single quotes are NOT treated as string delimiters in
// raw mode — that would break prose containing words like "doesn't".
//
// When raw is false, we still skip braces inside double-quoted strings so
// that values like `header: "{x}"` parse cleanly.
func (p *parser) readBracedBody(raw bool) (string, error) {
	if p.pos >= len(p.src) || p.src[p.pos] != '{' {
		return "", p.errf("expected '{'")
	}
	p.pos++ // consume opening
	depth := 1
	bodyStart := p.pos
	inStr := false
	escape := false
	inLineComment := false

	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if c == '\n' {
			p.ln++
			inLineComment = false
			p.pos++
			continue
		}
		if inLineComment {
			p.pos++
			continue
		}
		if inStr {
			if escape {
				escape = false
			} else if c == '\\' {
				escape = true
			} else if c == '"' {
				inStr = false
			}
			p.pos++
			continue
		}
		// Detect "//" line comments only in raw blocks (JS scripts).
		if raw && c == '/' && p.pos+1 < len(p.src) && p.src[p.pos+1] == '/' {
			inLineComment = true
			p.pos += 2
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				body := p.src[bodyStart:p.pos]
				p.pos++ // consume closing
				return body, nil
			}
		}
		p.pos++
	}
	return "", p.errf("unterminated block (missing '}')")
}

func (p *parser) skipWhitespaceAndComments() {
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		switch {
		case c == ' ' || c == '\t' || c == '\r':
			p.pos++
		case c == '\n':
			p.ln++
			p.pos++
		default:
			return
		}
	}
}

func (p *parser) errf(format string, args ...any) error {
	return fmt.Errorf("line %d: "+format, append([]any{p.ln + 1}, args...)...)
}

func splitHeader(header string) (name, subtype string) {
	// Header is a sequence like "auth:awsv4" or "params:query" or "meta".
	// Whitespace inside header is unusual; strip just in case.
	header = strings.TrimSpace(header)
	if i := strings.Index(header, ":"); i >= 0 {
		return strings.TrimSpace(header[:i]), strings.TrimSpace(header[i+1:])
	}
	return header, ""
}

// parseKVLines parses the body of a kv block.
func parseKVLines(body string) []Line {
	var out []Line
	scanner := bufio.NewScanner(strings.NewReader(body))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		disabled := false
		if strings.HasPrefix(trimmed, "~") {
			disabled = true
			trimmed = strings.TrimSpace(trimmed[1:])
		}
		idx := strings.Index(trimmed, ":")
		if idx < 0 {
			// Allow lines without a colon — treat as key with empty value.
			out = append(out, Line{Disabled: disabled, Key: trimmed})
			continue
		}
		key := strings.TrimSpace(trimmed[:idx])
		val := strings.TrimSpace(trimmed[idx+1:])
		out = append(out, Line{Disabled: disabled, Key: key, Value: val})
	}
	return out
}
