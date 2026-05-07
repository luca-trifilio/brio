package cli

import (
	"fmt"
	"runtime/debug"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/luca-trifilio/brio/internal/brunoprefs"
	"github.com/luca-trifilio/brio/internal/history"
	"github.com/luca-trifilio/brio/internal/model"
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
				discovered, err := brunoprefs.CollectionPaths()
				if err != nil {
					return fmt.Errorf("reading Bruno preferences: %w", err)
				}
				if len(discovered) == 0 {
					return fmt.Errorf("no collections found in Bruno preferences; pass at least one collection path as argument")
				}
				paths = discovered
				fmt.Fprintf(cmd.ErrOrStderr(), "loading collections from Bruno:\n  %s\n\n",
					strings.Join(paths, "\n  "))
			}
			var collections []*model.Collection
			for _, p := range paths {
				c, err := model.LoadCollection(p)
				if err != nil {
					return fmt.Errorf("load %s: %w", p, err)
				}
				collections = append(collections, c)
			}
			store, err := history.NewDefault()
			if err != nil {
				return fmt.Errorf("history store: %w", err)
			}
			m := tui.NewModel(collections, store)
			prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
			if _, err := prog.Run(); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}
