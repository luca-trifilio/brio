package tui

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/luca-trifilio/brio/internal/config"
	"github.com/luca-trifilio/brio/internal/history"
	"github.com/luca-trifilio/brio/internal/httpx"
	"github.com/luca-trifilio/brio/internal/interp"
	"github.com/luca-trifilio/brio/internal/model"
	"github.com/luca-trifilio/brio/internal/theme"
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
		body := scope.Interpolate(req.Body.Raw)
		if req.Body.Type == "json" {
			body = stripJSONComments(body)
		}
		out.Body = []byte(body)
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
	// Prod endpoints sit behind a private CA — skip TLS verification.
	if env != nil && theme.ClassifyEnv(env.Name) == theme.TierDanger {
		out.InsecureSkipTLS = true
	}
	return out, scope
}

// executeMsg is sent when an HTTP request completes.
type executeMsg struct {
	Resp          httpx.Response
	Resolved      httpx.ResolvedRequest
	Collection    *model.Collection
	Request       *model.Request
	Environment   string
	ExtractedVars map[string]string // vars captured from script:post-response
}

// errMsg is for non-HTTP errors surfaced to the status line.
type errMsg struct{ Err error }

// editorDoneMsg is sent after the external editor process exits.
type editorDoneMsg struct{ CollectionPath string }

// configEditDoneMsg is sent after the config file editor process exits.
type configEditDoneMsg struct{}

// runRequestCmd returns a tea.Cmd that runs the request asynchronously.
func runRequestCmd(c *model.Collection, env *model.Environment, req *model.Request, runtime map[string]string) tea.Cmd {
	return func() tea.Msg {
		// Run pre-request scripts (e.g. uuid generation) and merge into runtime.
		// User's explicit runtime overrides always win.
		preVars := interp.CollectPreRequestVars(c, req)
		merged := make(map[string]string, len(preVars)+len(runtime))
		for k, v := range preVars {
			merged[k] = v
		}
		for k, v := range runtime {
			merged[k] = v // runtime overrides script-generated values
		}
		resolved, _ := resolveRequest(c, env, req, merged)
		var ex *httpx.Executor
		if resolved.InsecureSkipTLS {
			ex = httpx.NewExecutorInsecure()
		} else {
			ex = httpx.NewExecutor()
		}
		resp := ex.Execute(resolved)
		envName := ""
		if env != nil {
			envName = env.Name
		}
		var extracted map[string]string
		if req.PostResponseScript != "" && resp.Err == nil {
			extracted = interp.RunPostResponseScript(req.PostResponseScript, resp.Body)
		}
		return executeMsg{
			Resp:          resp,
			Resolved:      resolved,
			Collection:    c,
			Request:       req,
			Environment:   envName,
			ExtractedVars: extracted,
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

// openConfigInEditor suspends the TUI, launches $EDITOR on the brio config
// file (creating it from the default template if it does not yet exist), then
// resumes. On return it sends configEditDoneMsg so the caller can reload.
func openConfigInEditor() tea.Cmd {
	_ = config.EnsureDir()
	p := config.Path()
	if _, err := os.Stat(p); os.IsNotExist(err) {
		_ = os.WriteFile(p, []byte(config.DefaultTemplate()), 0o644)
	}
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	c := exec.Command(editor, p) //nolint:gosec,noctx
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return configEditDoneMsg{}
	})
}

// openEnvInEditor suspends the TUI, launches $EDITOR (fallback: vi) on envPath,
// then resumes. On return it sends editorDoneMsg so the caller can reload.
func openEnvInEditor(envPath, collectionPath string) tea.Cmd {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	c := exec.Command(editor, envPath) //nolint:gosec,noctx
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorDoneMsg{CollectionPath: collectionPath}
	})
}

// stripJSONComments removes // line comments and /* block comments */ from s
// without touching content inside double-quoted strings.
// Bruno allows JS-style comments in body:json blocks; the server does not.
func stripJSONComments(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inStr := false
	esc := false
	i := 0
	for i < len(s) {
		c := s[i]
		if esc {
			esc = false
			b.WriteByte(c)
			i++
			continue
		}
		if inStr {
			switch c {
			case '\\':
				esc = true
			case '"':
				inStr = false
			}
			b.WriteByte(c)
			i++
			continue
		}
		// Outside a string.
		if c == '"' {
			inStr = true
			b.WriteByte(c)
			i++
			continue
		}
		// // line comment — skip to end of line.
		if c == '/' && i+1 < len(s) && s[i+1] == '/' {
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		// /* block comment */ — skip to closing */.
		if c == '/' && i+1 < len(s) && s[i+1] == '*' {
			i += 2
			for i < len(s) {
				if s[i] == '*' && i+1 < len(s) && s[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			continue
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}

func collectionDir(c *model.Collection) string {
	if c == nil {
		return ""
	}
	return filepath.Base(c.Path)
}
