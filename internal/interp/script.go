package interp

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/luca-trifilio/bruno-tui/internal/model"
)

// setVarRe matches: bru.setVar("varName", res.body.some.path)
// It captures the var name and the rest of the res.body expression.
var setVarRe = regexp.MustCompile(
	`bru\.setVar\(\s*["` + "`" + `']([^"` + "`" + `']+)["` + "`" + `']\s*,\s*res\.body([\w.\[\]"'` + "`" + `]*)\s*\)`,
)

var bracketRe = regexp.MustCompile(`\[["` + "`" + `']([^"` + "`" + `']+)["` + "`" + `']\]`)
var numericBracketRe = regexp.MustCompile(`\[(\d+)\]`)

// RunPostResponseScript parses a Bruno post-response script and executes the
// subset of the Bruno JS API we support:
//
//	bru.setVar("name", res.body)           — entire body as JSON string
//	bru.setVar("name", res.body.field)     — top-level field
//	bru.setVar("name", res.body.a.b.c)    — nested dot path
//	bru.setVar("name", res.body["field"]) — bracket notation
//
// Returns a map of variable name → extracted value. Unknown paths are silently
// skipped. If the body is not valid JSON the map is empty.
func RunPostResponseScript(script string, responseBody []byte) map[string]string {
	out := map[string]string{}
	if script == "" || len(responseBody) == 0 {
		return out
	}

	var body any
	if err := json.Unmarshal(responseBody, &body); err != nil {
		return out
	}

	for _, m := range setVarRe.FindAllStringSubmatch(script, -1) {
		varName := m[1]
		rawPath := m[2] // everything after "res.body"

		if rawPath == "" {
			// bru.setVar("x", res.body) — whole body
			if b, err := json.Marshal(body); err == nil {
				out[varName] = string(b)
			}
			continue
		}

		// Normalise bracket notation to dot notation.
		// ["field"] / ['field'] → .field
		// [0]        → .0  (numeric index; jsonPath handles []any)
		normalised := bracketRe.ReplaceAllString(rawPath, ".$1")
		normalised = numericBracketRe.ReplaceAllString(normalised, ".$1")
		normalised = strings.TrimPrefix(normalised, ".")

		if normalised == "" {
			continue
		}

		parts := strings.Split(normalised, ".")
		if val, ok := jsonPath(body, parts); ok {
			out[varName] = val
		}
	}
	return out
}

// jsonPath navigates a decoded JSON value (any) by successive keys/indices.
// Numeric parts (e.g. "0") are treated as array indices into []any values.
func jsonPath(v any, parts []string) (string, bool) {
	if len(parts) == 0 {
		return jsonValueToString(v)
	}
	key := parts[0]
	// Try numeric index first (handles res.body.data[0].uid → parts ["data","0","uid"]).
	if idx, err := strconv.Atoi(key); err == nil {
		arr, ok := v.([]any)
		if !ok || idx < 0 || idx >= len(arr) {
			return "", false
		}
		return jsonPath(arr[idx], parts[1:])
	}
	// Otherwise treat as object key.
	m, ok := v.(map[string]any)
	if !ok {
		return "", false
	}
	next, ok := m[key]
	if !ok {
		return "", false
	}
	return jsonPath(next, parts[1:])
}

// ---------------------------------------------------------------------------
// Pre-request script runner
// ---------------------------------------------------------------------------

// uuidFuncAssignRe matches: const/let/var X = Y() or Y.Z()
// (captures variable name X and the right-hand-side callee)
var uuidFuncAssignRe = regexp.MustCompile(`(?:const|let|var)\s+(\w+)\s*=\s*([\w.]+)\(\)`)

// bruSimpleSetVarRe matches bru.setVar / bru.setEnvVar with a simple identifier value.
var bruSimpleSetVarRe = regexp.MustCompile(
	`bru\.set(?:Env)?Var\(\s*["` + "`" + `'](\w[^"` + "`" + `']*)["` + "`" + `']\s*,\s*(\w+)\s*\)`,
)

// RunPreRequestScript parses a Bruno pre-request script and executes the
// subset we support:
//
//   - UUID generation: when the script uses require('uuid') and sets a var to
//     the result of uuidv4() / v4(), we generate a real UUID v4 in Go.
//
// Returns a map of variable name → value. Unknown or unsupported patterns are
// silently skipped.
func RunPreRequestScript(script string) map[string]string {
	out := map[string]string{}
	if script == "" {
		return out
	}

	// Detect uuid package usage.
	hasUUID := strings.Contains(script, "require('uuid')") ||
		strings.Contains(script, `require("uuid")`)
	if !hasUUID {
		return out
	}

	// Collect JS identifiers that hold a uuid value:
	// e.g. "const uuidv4 = require('uuid').v4" or "let id = uuidv4()"
	uuidIdents := map[string]bool{}
	for _, m := range uuidFuncAssignRe.FindAllStringSubmatch(script, -1) {
		varName, callee := m[1], m[2]
		// Direct call to a known uuid function name, or calling a previously
		// identified uuid-producing ident.
		if strings.Contains(callee, "uuid") || strings.Contains(callee, "v4") ||
			uuidIdents[callee] {
			uuidIdents[varName] = true
		}
	}

	// Find bru.setVar / bru.setEnvVar calls whose value is a uuid ident.
	for _, m := range bruSimpleSetVarRe.FindAllStringSubmatch(script, -1) {
		envVar, valueIdent := m[1], m[2]
		if uuidIdents[valueIdent] {
			out[envVar] = generateUUIDv4()
		}
	}

	return out
}

// generateUUIDv4 returns a random UUID v4 string using crypto/rand.
func generateUUIDv4() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ---------------------------------------------------------------------------
// CollectPreRequestVars chains pre-request scripts from the full scope:
// collection → folder chain (root→leaf) → request.
// Results from later scripts override earlier ones (same priority as the
// Bruno runtime — scripts can override each other in execution order).
// ---------------------------------------------------------------------------

// CollectPreRequestVars runs every pre-request script in scope and returns the
// merged vars map. It is called before request resolution so the generated
// values (e.g. uuid) participate in template interpolation.
func CollectPreRequestVars(c *model.Collection, req *model.Request) map[string]string {
	out := map[string]string{}

	// Collection-level script.
	if c != nil && c.CollectionDoc != nil {
		if b := c.CollectionDoc.FindBlock("script", "pre-request"); b != nil {
			for k, v := range RunPreRequestScript(b.Raw) {
				out[k] = v
			}
		}
	}

	// Folder chain (root → leaf).
	if c != nil && req != nil {
		chain := folderChainFor(c, req) // leaf → root; iterate reversed
		for i := len(chain) - 1; i >= 0; i-- {
			f := chain[i]
			if f.FolderDoc == nil {
				continue
			}
			if b := f.FolderDoc.FindBlock("script", "pre-request"); b != nil {
				for k, v := range RunPreRequestScript(b.Raw) {
					out[k] = v
				}
			}
		}
	}

	// Request-level script.
	if req != nil && req.HasPreRequestScript && req.Doc != nil {
		if b := req.Doc.FindBlock("script", "pre-request"); b != nil {
			for k, v := range RunPreRequestScript(b.Raw) {
				out[k] = v
			}
		}
	}

	return out
}

// ---------------------------------------------------------------------------
// JSON path helpers (used by RunPostResponseScript)
// ---------------------------------------------------------------------------

func jsonValueToString(v any) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case float64:
		if val == math.Trunc(val) && !math.IsInf(val, 0) {
			return strconv.FormatInt(int64(val), 10), true
		}
		return strconv.FormatFloat(val, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(val), true
	case nil:
		return "null", true
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return "", false
		}
		return string(b), true
	}
}
