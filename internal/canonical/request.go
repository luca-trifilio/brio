package canonical

// HTTPMethod is one of GET/POST/PUT/PATCH/DELETE/HEAD/OPTIONS/TRACE/CONNECT.
type HTTPMethod string

// Header is a request header line.
type Header struct {
	Name     string
	Value    string
	Disabled bool
}

// Param is a query/path parameter.
type Param struct {
	Name     string
	Value    string
	Disabled bool
}

// Body holds the request body. Only one form is populated at a time.
type Body struct {
	// Type is the body kind: "none", "json", "text", "xml", "graphql",
	// "formUrlEncoded", "multipartForm".
	Type string
	// Raw is the verbatim body content for json/text/xml/graphql.
	Raw string
}

// ScriptBlock holds extracted script source text. Pre runs before the
// request, Post after. Both may be empty.
type ScriptBlock struct {
	Pre  string
	Post string
}

// Request is the canonical view of a single HTTP request.
type Request struct {
	// SourcePath is the absolute filesystem path of the source file (if any).
	SourcePath string
	// Name is the display name of the request.
	Name string
	// Seq is the sort key within its folder (0 when absent).
	Seq int
	// Method is GET/POST/etc.
	Method HTTPMethod
	// URL is the raw URL template (may contain {{vars}}).
	URL string
	// Headers, query/path params, vars are preserved in source order.
	Headers     []Header
	QueryParams []Param
	PathParams  []Param
	// Vars are request-level variables (`vars { ... }`).
	Vars []Var
	// PreVars are pre-request variables (`vars:pre-request { ... }`).
	PreVars []Var
	// Auth is the auth block declared on this request (one link in the chain).
	Auth *AuthBlock
	// Body is the request body for json/text/xml.
	Body Body
	// Settings are request-level settings.
	Settings Settings
	// Scripts holds pre/post-response script source as plain text.
	Scripts ScriptBlock
	// Extra is a bag for opaque vendor metadata. Hot paths must not read this.
	Extra map[string]any
}
