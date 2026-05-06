package model

// HTTPMethod is one of GET/POST/PUT/PATCH/DELETE/HEAD/OPTIONS/TRACE/CONNECT.
type HTTPMethod string

// AuthMode mirrors Bruno's auth { mode: ... } values.
type AuthMode string

const (
	AuthInherit AuthMode = "inherit"
	AuthNone    AuthMode = "none"
	AuthModeAWSv4 AuthMode = "awsv4"
	AuthBasic   AuthMode = "basic"
	AuthBearer  AuthMode = "bearer"
	AuthAPIKey  AuthMode = "apikey"
	AuthDigest  AuthMode = "digest"
	AuthOAuth2  AuthMode = "oauth2"
)

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

// Var is a kv variable entry.
type Var struct {
	Name     string
	Value    string
	Disabled bool
}

// AuthAWSv4 holds AWS SigV4 settings.
type AuthAWSv4 struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Service         string
	Region          string
	ProfileName     string
}

// AuthBlock holds the resolved auth configuration for a request.
// The Mode determines which sub-struct is meaningful.
type AuthBlock struct {
	Mode  AuthMode
	AWSv4 *AuthAWSv4
	// Other auth modes can be added later (basic, bearer, apikey, oauth2).
}

// Body holds the request body. Only one form is populated at a time.
type Body struct {
	// Type mirrors Bruno's body kind: "none", "json", "text", "xml",
	// "graphql", "formUrlEncoded", "multipartForm".
	Type string
	// Raw holds the verbatim body content for json/text/xml/graphql.
	Raw string
}

// Settings mirrors the `settings { ... }` block.
type Settings struct {
	EncodeURL bool
	// TimeoutMS is 0 when the .bru sets it to 0 (no client-side timeout).
	TimeoutMS int
}
