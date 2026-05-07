// Package httpx wraps net/http with helpers for executing resolved Bruno
// requests, including AWS SigV4 signing.
package httpx

import (
	"net/http"
	"time"
)

// ResolvedRequest is what the executor consumes — every field is already
// interpolated; auth fields hold concrete credentials.
type ResolvedRequest struct {
	Method          string
	URL             string
	Headers         http.Header
	QueryParams     [][2]string // ordered (name, value)
	Body            []byte
	BodyType        string // "json" / "text" / "xml" / "" / etc.
	AuthMode        string // "none" / "awsv4" / ...
	AWSv4           *AWSCreds
	TimeoutMS       int
	EncodeURL       bool
	InsecureSkipTLS bool // skip TLS verification (needed for private-CA endpoints)
}

// AWSCreds holds the inputs for SigV4 signing.
type AWSCreds struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Service         string
	Region          string
}

// Response is the result of an executed request.
type Response struct {
	Status     string
	StatusCode int
	Headers    http.Header
	Body       []byte
	Elapsed    time.Duration
	URL        string // final URL (post query-merge)
	Method     string
	Err        error
}
