package httpx

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Executor wraps an *http.Client.
type Executor struct {
	Client *http.Client
}

// NewExecutor builds an executor with sane defaults: TLS enabled, no global
// timeout (per-request timeout is honoured via context).
func NewExecutor() *Executor {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}
	return &Executor{
		Client: &http.Client{Transport: tr},
	}
}

// NewExecutorInsecure builds an executor that skips TLS certificate
// verification. Required for endpoints served by a private CA
// (e.g. internal-admin-api.satispay.aws).
func NewExecutorInsecure() *Executor {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true, //nolint:gosec
		},
	}
	return &Executor{
		Client: &http.Client{Transport: tr},
	}
}

// Execute runs r and returns a Response. Errors are captured in Response.Err
// rather than returned, so callers can render them in the TUI uniformly.
func (e *Executor) Execute(r ResolvedRequest) Response {
	start := time.Now()
	resp := Response{Method: r.Method}

	// Merge query params into URL.
	finalURL, err := mergeQuery(r.URL, r.QueryParams)
	if err != nil {
		resp.Err = err
		resp.URL = r.URL
		resp.Elapsed = time.Since(start)
		return resp
	}
	resp.URL = finalURL

	method := r.Method
	if method == "" {
		method = http.MethodGet
	}

	var bodyReader io.Reader
	if len(r.Body) > 0 {
		bodyReader = bytes.NewReader(r.Body)
	}

	ctx := context.Background()
	if r.TimeoutMS > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(r.TimeoutMS)*time.Millisecond)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, method, finalURL, bodyReader)
	if err != nil {
		resp.Err = err
		resp.Elapsed = time.Since(start)
		return resp
	}
	if r.Headers != nil {
		req.Header = r.Headers.Clone()
	}
	// Default Content-Type by body type when unset.
	if len(r.Body) > 0 && req.Header.Get("Content-Type") == "" {
		switch r.BodyType {
		case "json":
			req.Header.Set("Content-Type", "application/json")
		case "xml":
			req.Header.Set("Content-Type", "application/xml")
		case "text":
			req.Header.Set("Content-Type", "text/plain")
		}
	}

	if r.AuthMode == "awsv4" {
		if err := signAWSv4(req, r.AWSv4); err != nil {
			resp.Err = err
			resp.Elapsed = time.Since(start)
			return resp
		}
	}

	httpResp, err := e.Client.Do(req)
	if err != nil {
		resp.Err = err
		resp.Elapsed = time.Since(start)
		return resp
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Err = err
	}
	resp.Status = httpResp.Status
	resp.StatusCode = httpResp.StatusCode
	resp.Headers = httpResp.Header
	resp.Body = body
	resp.Elapsed = time.Since(start)
	return resp
}

// mergeQuery appends params (in order) to the URL's query string. Disabled
// params are filtered upstream.
func mergeQuery(rawURL string, params [][2]string) (string, error) {
	if len(params) == 0 {
		return rawURL, nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for _, p := range params {
		q.Add(p[0], p[1])
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
