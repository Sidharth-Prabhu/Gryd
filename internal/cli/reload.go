package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/runner"
	"github.com/loom-cli/gryd/internal/state"
	"github.com/loom-cli/gryd/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var reloadCmd = &cobra.Command{
	Use:   "reload [project]",
	Short: "Perform a zero-downtime hot reload of a project's services",
	Long: `Performs a hot reload on running services without stopping sidecars or destroying the session.
If you need a hard process kill and full port teardown, use 'gryd restart'.`,
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
					return fmt.Errorf("multiple projects running. Please specify which one: gryd reload <project>")
				} else {
					return fmt.Errorf("no project specified and no active project in current directory")
				}
			}
		}

		fmt.Printf("\033[1;33m[gryd] Hot reloading project '%s'...\033[0m\n", projectName)

		newState, err := runner.HotReloadBackgroundProject(projectName)
		if err != nil {
			return fmt.Errorf("failed to hot reload project '%s': %w", projectName, err)
		}

		termW := 80
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 40 {
			termW = w
		}

		var cfg *config.Config
		if newState.ConfigPath != "" {
			cfg, _ = config.Load(newState.ConfigPath)
		}

		fmt.Println()
		header := tui.BadgeSuccess.Render(" ✓ PROJECT HOT RELOADED ")
		title := tui.BadgeProject.Render(" " + projectName + " ")
		topBar := lipgloss.JoinHorizontal(lipgloss.Center, header, " ", title)

		card := tui.RenderStatusCard(projectName, cfg, newState, termW)
		fmt.Println(lipgloss.JoinVertical(lipgloss.Left, topBar, "", card))
		fmt.Printf("\nServices are running in background.\nView live logs at any time with: \033[1mgryd look %s\033[0m\n\n", projectName)

		return nil
	},
}
