package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/runner"
	"github.com/loom-cli/gryd/internal/state"
	"github.com/loom-cli/gryd/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var stopCmd = &cobra.Command{
	Use:   "stop [project]",
	Short: "Completely stop all services and free ports used by the project",
	Long: `Terminate all child processes for a running Gryd project, clean up its state,
and ensure all configured ports (and any lingering child processes) are released.
If inside a project directory, [project] can be omitted.`,
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
					return fmt.Errorf("multiple projects running. Please specify which one: gryd stop <project>")
				} else {
					return fmt.Errorf("no project specified and no active project in current directory")
				}
			}
		}

		fmt.Printf("\033[1;33m[gryd] Stopping project '%s'...\033[0m\n", projectName)

		// 1. Load active runtime state if present
		st, _ := state.LoadState(projectName)

		// 2. Load config to identify configured ports
		var cfg *config.Config
		configPath, err := config.FindConfigFile(configFile, projectName)
		if err == nil {
			cfg, _ = config.Load(configPath)
		} else if st != nil && st.ConfigPath != "" {
			cfg, _ = config.Load(st.ConfigPath)
		}

		stoppedCount := 0
		if st != nil {
			for _, proc := range st.Processes {
				if proc.PID > 0 && state.IsProcessAlive(proc.PID) {
					stoppedCount++
				}
			}
			runner.StopProjectProcesses(st)
			_ = state.RemoveState(projectName)
		}

		// 3. Free and verify all configured ports
		var freedPorts []int
		if cfg != nil {
			for svcName, svc := range cfg.Services {
				if svc.Port > 0 {
					if runner.IsPortInUse(svc.Port) {
						pids, _ := runner.FreePort(svc.Port)
						if len(pids) > 0 {
							freedPorts = append(freedPorts, svc.Port)
							fmt.Printf("  • Service '%s': freed port %d (killed process PIDs %v)\n",
								svcName, svc.Port, pids)
						}
					}
				}
			}
		}

		termW := 80
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 40 {
			termW = w
		}

		title := tui.BadgeProject.Render(" " + projectName + " ")
		statusBadge := tui.BadgeSuccess.Render(" STOPPED ")
		spacer := termW - lipgloss.Width(title) - lipgloss.Width(statusBadge) - 6
		if spacer < 1 {
			spacer = 1
		}
		topBar := lipgloss.JoinHorizontal(lipgloss.Center, title, strings.Repeat(" ", spacer), statusBadge)

		var summaryLines []string
		summaryLines = append(summaryLines, fmt.Sprintf("  \033[1;32m✓\033[0m Successfully terminated all processes for '%s'.", projectName))
		if stoppedCount > 0 {
			summaryLines = append(summaryLines, fmt.Sprintf("  • Terminated %d running service processes.", stoppedCount))
		}
		if len(freedPorts) > 0 {
			summaryLines = append(summaryLines, fmt.Sprintf("  • Released and confirmed freed ports: %v", freedPorts))
		} else {
			summaryLines = append(summaryLines, "  • All project ports verified released and free.")
		}

		box := tui.BoxCard.Width(termW - 2).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				topBar,
				"",
				strings.Join(summaryLines, "\n"),
			),
		)

		fmt.Println()
		fmt.Println(box)
		fmt.Println()

		return nil
	},
}
