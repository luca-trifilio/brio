package hooks

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/luca-trifilio/brio/internal/config"
)

// DoneMsg is sent to the Bubble Tea runtime when a hook finishes.
// Vars contains the parsed key→value pairs ready to merge into runtime vars.
// Err is non-nil if the script failed or output could not be parsed.
type DoneMsg struct {
	Vars map[string]string
	Err  error
}

// Cmd returns a tea.Cmd that executes the given hook.
//
//   - output.type == "stdout": non-interactive exec, stdout captured and parsed
//   - output.type == "file":   interactive tea.ExecProcess (TUI suspends),
//     output file read after the script exits
func Cmd(hook *config.Hook) tea.Cmd {
	switch hook.Output.Type {
	case "file":
		return fileCmd(hook)
	default: // "stdout" and anything unrecognized
		return stdoutCmd(hook)
	}
}

// stdoutCmd runs the script non-interactively, captures its stdout, and
// parses the output as dotenv KEY=VALUE pairs.
func stdoutCmd(hook *config.Hook) tea.Cmd {
	return func() tea.Msg {
		path, err := expandPath(hook.Script.Path)
		if err != nil {
			return DoneMsg{Err: fmt.Errorf("hook %q: expand script path: %w", hook.Name, err)}
		}

		c := exec.CommandContext(context.Background(), "sh", "-c", path) //nolint:gosec
		c.Env = buildEnv(hook.Script.Env)

		out, err := c.Output()
		if err != nil {
			return DoneMsg{Err: fmt.Errorf("hook %q: script error: %w", hook.Name, err)}
		}

		vars, err := Parse(hook.Output, out)
		if err != nil {
			return DoneMsg{Err: fmt.Errorf("hook %q: parse output: %w", hook.Name, err)}
		}
		return DoneMsg{Vars: vars}
	}
}

// fileCmd suspends the TUI, runs the script interactively (the user can
// interact with prompts, MFA, etc.), then reads the output file after exit.
func fileCmd(hook *config.Hook) tea.Cmd {
	path, err := expandPath(hook.Script.Path)
	if err != nil {
		return func() tea.Msg {
			return DoneMsg{Err: fmt.Errorf("hook %q: expand script path: %w", hook.Name, err)}
		}
	}

	c := exec.CommandContext(context.Background(), "sh", "-c", path) //nolint:gosec
	c.Env = buildEnv(hook.Script.Env)

	// Capture hook for the callback closure (loop-var safety).
	h := hook
	return tea.ExecProcess(c, func(execErr error) tea.Msg {
		if execErr != nil {
			return DoneMsg{Err: fmt.Errorf("hook %q: script error: %w", h.Name, execErr)}
		}
		vars, err := Parse(h.Output, nil)
		if err != nil {
			return DoneMsg{Err: fmt.Errorf("hook %q: parse output file: %w", h.Name, err)}
		}
		return DoneMsg{Vars: vars}
	})
}

// buildEnv returns an environment slice for exec.Cmd: inherits the current
// process environment and appends any extra key=value pairs from the hook.
func buildEnv(extra map[string]string) []string {
	env := os.Environ()
	for k, v := range extra {
		env = append(env, k+"="+v)
	}
	return env
}
