package canonical

// AuthMode names the auth scheme on an auth block.
type AuthMode string

// Auth modes recognized by the canonical model.
const (
	AuthInherit AuthMode = "inherit"
	AuthNone    AuthMode = "none"
	AuthBearer  AuthMode = "bearer"
	AuthBasic   AuthMode = "basic"
	AuthAPIKey  AuthMode = "apikey"
	AuthAWSv4   AuthMode = "awsv4"
)

// AuthBearer holds bearer-token auth settings.
type AuthBearerCfg struct {
	Token string
}

// AuthBasicCfg holds basic auth credentials.
type AuthBasicCfg struct {
	Username string
	Password string
}

// AuthAPIKeyCfg holds API key auth settings.
type AuthAPIKeyCfg struct {
	Key       string
	Value     string
	Placement string // "header" | "query"
}

// AuthAWSv4Cfg holds AWS Signature V4 settings.
type AuthAWSv4Cfg struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Service         string
	Region          string
	ProfileName     string
}

// AuthBlock holds the resolved auth configuration for a node in the tree
// (collection, folder, or request). Mode selects which sub-config is used.
type AuthBlock struct {
	Mode   AuthMode
	Bearer *AuthBearerCfg
	Basic  *AuthBasicCfg
	APIKey *AuthAPIKeyCfg
	AWSv4  *AuthAWSv4Cfg
}
