package tui

import (
	"net/http"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/luca-trifilio/bruno-tui/internal/history"
	"github.com/luca-trifilio/bruno-tui/internal/httpx"
	"github.com/luca-trifilio/bruno-tui/internal/interp"
	"github.com/luca-trifilio/bruno-tui/internal/model"
)

// resolveRequest builds an httpx.ResolvedRequest by interpolating req against
// the given collection/env/runtime overrides and resolving the auth chain.
func resolveRequest(c *model.Collection, env *model.Environment, req *model.Request, runtime map[string]string) (httpx.ResolvedRequest, *interp.VarScope) {
	scope := interp.BuildScope(c, env, req, runtime)
	out := httpx.ResolvedRequest{
		Method:    string(req.Method),
		URL:       scope.Interpolate(req.URL),
		BodyType:  req.Body.Type,
		TimeoutMS: req.Settings.TimeoutMS,
		EncodeURL: req.Settings.EncodeURL,
	}
	hdr := http.Header{}
	for _, h := range req.Headers {
		if h.Disabled {
			continue
		}
		hdr.Add(scope.Interpolate(h.Name), scope.Interpolate(h.Value))
	}
	out.Headers = hdr
	for _, p := range req.QueryParams {
		if p.Disabled {
			continue
		}
		out.QueryParams = append(out.QueryParams, [2]string{
			scope.Interpolate(p.Name), scope.Interpolate(p.Value),
		})
	}
	if req.Body.Raw != "" {
		out.Body = []byte(scope.Interpolate(req.Body.Raw))
	}
	auth := interp.ResolveAuth(c, req)
	out.AuthMode = string(auth.Mode)
	if auth.Mode == model.AuthModeAWSv4 && auth.AWSv4 != nil {
		out.AWSv4 = &httpx.AWSCreds{
			AccessKeyID:     scope.Interpolate(auth.AWSv4.AccessKeyID),
			SecretAccessKey: scope.Interpolate(auth.AWSv4.SecretAccessKey),
			SessionToken:    scope.Interpolate(auth.AWSv4.SessionToken),
			Service:         scope.Interpolate(auth.AWSv4.Service),
			Region:          scope.Interpolate(auth.AWSv4.Region),
		}
	}
	return out, scope
}

// executeMsg is sent when an HTTP request completes.
type executeMsg struct {
	Resp        httpx.Response
	Resolved    httpx.ResolvedRequest
	Collection  *model.Collection
	Request     *model.Request
	Environment string
}

// errMsg is for non-HTTP errors surfaced to the status line.
type errMsg struct{ Err error }

// runRequestCmd returns a tea.Cmd that runs the request asynchronously.
func runRequestCmd(c *model.Collection, env *model.Environment, req *model.Request, runtime map[string]string) tea.Cmd {
	return func() tea.Msg {
		resolved, _ := resolveRequest(c, env, req, runtime)
		ex := httpx.NewExecutor()
		resp := ex.Execute(resolved)
		envName := ""
		if env != nil {
			envName = env.Name
		}
		return executeMsg{
			Resp:        resp,
			Resolved:    resolved,
			Collection:  c,
			Request:     req,
			Environment: envName,
		}
	}
}

// historyEntryFromExecute converts an executeMsg into a history.Entry.
func historyEntryFromExecute(m executeMsg) history.Entry {
	headers := map[string]string{}
	if m.Resolved.Headers != nil {
		for k, v := range m.Resolved.Headers {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}
	}
	preview := ""
	if len(m.Resp.Body) > 0 {
		if len(m.Resp.Body) > history.MaxPreviewBytes {
			preview = string(m.Resp.Body[:history.MaxPreviewBytes])
		} else {
			preview = string(m.Resp.Body)
		}
	}
	errStr := ""
	if m.Resp.Err != nil {
		errStr = m.Resp.Err.Error()
	}
	return history.Entry{
		Timestamp:       time.Now().UTC(),
		Collection:      collectionDir(m.Collection),
		RequestPath:     m.Request.SourcePath,
		RequestName:     m.Request.Name,
		Environment:     m.Environment,
		Method:          m.Resolved.Method,
		URL:             m.Resp.URL,
		Status:          m.Resp.Status,
		StatusCode:      m.Resp.StatusCode,
		ElapsedMS:       m.Resp.Elapsed.Milliseconds(),
		ResponsePreview: preview,
		Error:           errStr,
		Headers:         headers,
	}
}

func collectionDir(c *model.Collection) string {
	if c == nil {
		return ""
	}
	return filepath.Base(c.Path)
}
