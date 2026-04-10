package cmd

import (
	"fmt"
	"time"

	"github.com/MrHalder/moor/internal/config"
	"github.com/MrHalder/moor/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var watchRefresh int

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Interactive TUI port dashboard",
	Long:  "Launch the full-screen interactive port monitor with real-time updates.",
	RunE:  runWatch,
}

func init() {
	watchCmd.Flags().IntVar(&watchRefresh, "refresh", 2, "refresh interval in seconds")
	rootCmd.AddCommand(watchCmd)
}

func runWatch(cmd *cobra.Command, args []string) error {
	interval := time.Duration(watchRefresh) * time.Second
	if interval < 1*time.Second {
		interval = 1 * time.Second
	}

	var gracePeriod time.Duration
	if cfg, err := config.Load(); err == nil {
		gracePeriod = time.Duration(cfg.Settings.GracePeriodSecs) * time.Second
	}

	model := tui.New(interval, gracePeriod)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
