package cli

import (
	"fmt"
	"strings"

	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/runner"
	"github.com/loom-cli/gryd/internal/state"
	"github.com/spf13/cobra"
)

var reloadCmd = &cobra.Command{
	Use:   "reload [project]",
	Short: "Restart/reload a project's services in the background without opening live logs",
	Long: `Reload and restart all services for a project in the background.
If inside a project directory, [project] can be omitted.
This terminates running services, frees ports, re-reads the configuration,
starts services fresh in the background, and leaves live logs untouched.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var projectName string
		if len(args) > 0 {
			projectName = args[0]
		} else {
			// Auto-detect project if inside project directory
			detected, err := config.DetectCurrentProject()
			if err == nil {
				projectName = detected
			} else {
				// If only one project is currently running, default to it
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

		fmt.Printf("\033[1;33m[gryd] Reloading project '%s'...\033[0m\n", projectName)

		newState, err := runner.ReloadBackgroundProject(projectName)
		if err != nil {
			return fmt.Errorf("failed to reload project '%s': %w", projectName, err)
		}

		fmt.Println()
		fmt.Printf("\033[1;32m✓ Successfully reloaded project '%s'.\033[0m\n", projectName)
		fmt.Printf("  • Config File: %s\n", newState.ConfigPath)
		fmt.Printf("  • Log File:    %s\n", newState.LogFilePath)
		fmt.Println()

		fmt.Printf("  \033[1m%-16s %-10s %-8s %s\033[0m\n", "SERVICE", "STATUS", "PID", "COMMAND")
		fmt.Printf("  %s\n", strings.Repeat("─", 56))

		for name, proc := range newState.Processes {
			cmdStr := proc.Command
			if len(cmdStr) > 30 {
				cmdStr = cmdStr[:27] + "..."
			}
			fmt.Printf("  %-16s \033[32mRunning\033[0m    %-8d %s\n", name, proc.PID, cmdStr)
		}

		fmt.Printf("\nServices are running in background.\nView live logs at any time with: \033[1mgryd look %s\033[0m\n\n", projectName)

		return nil
	},
}
