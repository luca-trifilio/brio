package cli

import (
	"fmt"
	"runtime/debug"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/luca-trifilio/brio/internal/config"
	"github.com/luca-trifilio/brio/internal/history"
	_ "github.com/luca-trifilio/brio/internal/plugins/bruno" // self-registers Bruno loader
	"github.com/luca-trifilio/brio/internal/tui"
)

// Build metadata injected at link time by GoReleaser:
//
//	-X github.com/luca-trifilio/brio/internal/cli.version=x.y.z
//	-X github.com/luca-trifilio/brio/internal/cli.commit=abc1234
//	-X github.com/luca-trifilio/brio/internal/cli.date=2026-05-07T12:00:00Z
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	// When installed via `go install` without ldflags (e.g. from a tagged
	// module), read the version embedded by the Go toolchain in the binary.
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
	}
}

// buildVersion returns the full version string shown by --version.
func buildVersion() string {
	if commit == "none" && date == "unknown" {
		return version
	}
	return fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)
}

// NewRootCmd builds the top-level cobra command.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "brio [COLLECTION_PATH...]",
		Short: "A vim-style TUI for Bruno API collections",
		Long: `brio opens one or more Bruno v1 collection directories,
renders them in a vim-style modal TUI, and executes HTTP requests with
full variable interpolation, AWS SigV4 signing, environment switching,
and history.

Read-only by design — .bru files are never written.`,
		Args:    cobra.ArbitraryArgs,
		Version: buildVersion(),
		RunE: func(cmd *cobra.Command, args []string) error {
			paths := args
			if len(paths) == 0 {
				cfg, err := config.Load()
				if err != nil {
					return fmt.Errorf("loading config: %w", err)
				}

				// 1. config.toml → collections
				if len(cfg.Collections) > 0 {
					warn := func(p string) {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: collection path not found, skipping: %s\n", p)
					}
					paths = config.ResolvedCollections(cfg, warn)
					if len(paths) > 0 {
						_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "loading collections from config:\n  %s\n\n",
							strings.Join(paths, "\n  "))
					}
				}

				// 2. Nothing found — fall through and start the TUI with an
				// empty collection list. The TUI will pop the import modal
				// on first paint so the user can add collections without
				// quitting.
			}
			entries := make([]config.CollectionEntry, 0, len(paths))
			for _, p := range paths {
				entries = append(entries, config.CollectionEntry{Path: p})
			}
			collections, diags := tui.LoadCollections(entries)
			store, err := history.NewDefault()
			if err != nil {
				return fmt.Errorf("history store: %w", err)
			}
			m := tui.NewModelWithDiagnostics(collections, diags, store)
			// Empty-state: no collections loaded → open manager modal.
			if len(collections) == 0 {
				m.OpenCollMgrIfEmpty()
			}
			prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
			if _, err := prog.Run(); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}
