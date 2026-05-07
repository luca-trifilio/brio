package tui

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

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

// breakglassDoneMsg is sent after ~/bin/breakglass.sh exits.
type breakglassDoneMsg struct{ Err error }

// breakglassPending holds the context needed to retry a request after
// breakglass credentials have been refreshed.
type breakglassPending struct {
	c   *model.Collection
	env *model.Environment
	req *model.Request
}

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
	c := exec.Command(editor, envPath) //nolint:gosec
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
			if c == '\\' {
				esc = true
			} else if c == '"' {
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

// ----------------------------------------------------------------------------
// Breakglass
// ----------------------------------------------------------------------------

// breakglassCreds holds the three AWS credential values written by breakglass.sh.
type breakglassCreds struct {
	AccessKey    string
	SecretKey    string
	SessionToken string
}

// breakglassCredsPath returns the path where breakglass.sh deposits credentials.
func breakglassCredsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support",
		"bruno", "default-workspace", "environments", "AWS BREAKGLASS.yml")
}

// readBreakglassCreds parses the YAML file written by breakglass.sh:
//
//		variables:
//		  - name: AWS_ACCESS_KEY
//		    value: "ASIA..."
//		  - name: AWS_SECRET_KEY
//		    value: "..."
//		  - name: AWS_SESSION_TOKEN
//		    value: "..."
//
// No external YAML library is used — a simple line scan is sufficient for
// this well-known, machine-generated format.
func readBreakglassCreds() (breakglassCreds, error) {
	f, err := os.Open(breakglassCredsPath())
	if err != nil {
		return breakglassCreds{}, fmt.Errorf("breakglass: open creds file: %w", err)
	}
	defer f.Close()

	var creds breakglassCreds
	var lastName string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Capture the most recent "- name: FOO" line.
		if strings.HasPrefix(line, "- name:") {
			lastName = strings.TrimSpace(strings.TrimPrefix(line, "- name:"))
			continue
		}

		// On the following "value: \"...\"" line, assign to the right field.
		if strings.HasPrefix(line, "value:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "value:"))
			val = strings.Trim(val, `"`) // strip surrounding quotes
			switch lastName {
			case "AWS_ACCESS_KEY":
				creds.AccessKey = val
			case "AWS_SECRET_KEY":
				creds.SecretKey = val
			case "AWS_SESSION_TOKEN":
				creds.SessionToken = val
			}
			lastName = ""
		}
	}
	if err := scanner.Err(); err != nil {
		return breakglassCreds{}, fmt.Errorf("breakglass: read creds file: %w", err)
	}
	if creds.AccessKey == "" || creds.SecretKey == "" {
		return breakglassCreds{}, fmt.Errorf("breakglass: credentials not found in %s", breakglassCredsPath())
	}
	return creds, nil
}

// breakglassCmd suspends the TUI and runs ~/bin/breakglass.sh interactively
// (SSO login + approval wait). Sends breakglassDoneMsg on return.
//
// The environment mirrors the Apple shortcut that runs the script every morning:
//
//	export PATH="/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:$HOME/bin:$PATH"
//	export HOME="..."
//	export AWS_DEFAULT_REGION="eu-west-1"
//	export AWS_DEFAULT_OUTPUT="json"
//
// Without these, AWS CLI may use a different output format and the script's
// step-function status polling will not parse responses correctly.
func breakglassCmd() tea.Cmd {
	home, _ := os.UserHomeDir()
	script := filepath.Join(home, "bin", "breakglass.sh")
	c := exec.Command("sh", script) //nolint:gosec
	c.Env = append(os.Environ(),
		"PATH=/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:"+filepath.Join(home, "bin")+":"+os.Getenv("PATH"),
		"HOME="+home,
		"AWS_DEFAULT_REGION=eu-west-1",
		"AWS_DEFAULT_OUTPUT=json",
	)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return breakglassDoneMsg{Err: err}
	})
}

// isAWSTokenError reports whether the response is a 403 caused by an
// invalid or expired AWS security token — both require a breakglass refresh.
func isAWSTokenError(resp httpx.Response) bool {
	if resp.StatusCode != 403 {
		return false
	}
	body := string(resp.Body)
	return strings.Contains(body, "security token") &&
		(strings.Contains(body, "expired") || strings.Contains(body, "invalid"))
}
