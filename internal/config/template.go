package config

// DefaultTemplate returns a commented starter config written to disk the first
// time the user opens the settings modal and no config file exists yet.
func DefaultTemplate() string {
	return `# brio configuration
# ~/.config/brio/config.toml
#
# Credential refresh hooks fire automatically when an HTTP response matches
# the trigger conditions. The script runs, credentials are captured, and the
# original request is retried with the new values injected.

# ── Example: non-interactive stdout hook ─────────────────────────────────────
#
# [[hooks]]
# name = "aws-token-refresh"
#
# [hooks.trigger]
# status = [401, 403]                          # HTTP status codes that fire the hook
# body   = "ExpiredToken|InvalidClientTokenId" # optional regex on response body
# tier   = "danger"                            # optional: "safe" | "caution" | "danger"
#
# [hooks.script]
# path = "~/bin/my-refresh.sh"                 # ~ and $ENV vars are expanded
# [hooks.script.env]
# AWS_DEFAULT_REGION = "eu-west-1"             # extra env vars passed to the script
#
# [hooks.output]
# type = "stdout"                              # script writes KEY=VALUE lines to stdout
#
# [hooks.vars]
# ACCESS_KEY    = "aws_access_key_id"          # output key → brio runtime variable
# SECRET_KEY    = "aws_secret_access_key"
# SESSION_TOKEN = "aws_session_token"

# ── Example: interactive file hook ───────────────────────────────────────────
#
# [[hooks]]
# name = "vault-login"
#
# [hooks.trigger]
# status = [401]
#
# [hooks.script]
# path = "~/bin/vault-login.sh"               # TUI suspends while this runs
#
# [hooks.output]
# type   = "file"                             # script writes credentials to a file
# path   = "~/.vault-creds.json"
# format = "json"                             # "dotenv" | "json" | "yaml" | "bruno-env"
#
# [hooks.vars]
# token = "vault_token"
`
}
