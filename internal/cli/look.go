package cli

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/runner"
	"github.com/loom-cli/gryd/internal/state"
	"github.com/loom-cli/gryd/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var lookCmd = &cobra.Command{
	Use:   "look [project]",
	Short: "Stream live logs for a running background project",
	Long: `Attach to and stream real-time logs for a background project started by Gryd.
If inside a project directory, [project] can be omitted.
Interactive controls: 'r' to reload, 'c' to run in background (detach), or 'q' / Ctrl+C to stop.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var projectName string
		if len(args) > 0 {
			projectName = args[0]
		} else {
			detected, err := config.DetectCurrentProject()
			if err == nil {
				projectName = detected
			} else {
				running, _ := state.ListRunningProjects()
				if len(running) == 1 {
					projectName = running[0].ProjectName
				} else if len(running) > 1 {
					return fmt.Errorf("multiple projects running. Please specify which one: gryd look <project>")
				} else {
					return fmt.Errorf("no project specified and no active project in current directory")
				}
			}
		}

		projectState, err := state.LoadState(projectName)
		if err != nil {
			return err
		}

		aliveCount := 0
		for _, p := range projectState.Processes {
			if state.IsProcessAlive(p.PID) {
				aliveCount++
			}
		}

		if aliveCount == 0 {
			_ = state.RemoveState(projectName)
			return fmt.Errorf("project '%s' is no longer running (all processes terminated)", projectName)
		}

		configPath, err := config.FindConfigFile(configFile, projectName)
		if err != nil {
			return err
		}

		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		if !term.IsTerminal(int(os.Stdout.Fd())) {
			return fmt.Errorf("gryd look requires an interactive terminal")
		}

		// Connect full interactive dashboard to running project
		model, err := tui.NewUIModel(cfg, projectState)
		if err != nil {
			return err
		}

		p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("error running UI: %w", err)
		}

		if m, ok := finalModel.(*tui.UIModel); ok && !m.IsDetached() {
			fmt.Printf("\n\033[1;33m[gryd] Stopping project '%s'...\033[0m\n", cfg.Name)
			runner.StopProjectProcesses(projectState)
			_ = state.RemoveState(cfg.Name)
			fmt.Printf("\033[1;32m✓ All services stopped cleanly.\033[0m\n\n")
		}

		return nil
	},
}
