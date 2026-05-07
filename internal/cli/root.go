package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/luca-trifilio/bruno-tui/internal/brunoprefs"
	"github.com/luca-trifilio/bruno-tui/internal/history"
	"github.com/luca-trifilio/bruno-tui/internal/model"
	"github.com/luca-trifilio/bruno-tui/internal/tui"
)

// version is set at build time via -ldflags.
var version = "dev"

// NewRootCmd builds the top-level cobra command.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bruno-tui [COLLECTION_PATH...]",
		Short: "A vim-style TUI for Bruno API collections",
		Long: `bruno-tui opens one or more Bruno v1 collection directories,
renders them in a vim-style modal TUI, and executes HTTP requests with
full variable interpolation, AWS SigV4 signing, environment switching,
and history.

MVP is execute-only — .bru files are read but never written.`,
		Args:    cobra.ArbitraryArgs,
		Version: version,
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
