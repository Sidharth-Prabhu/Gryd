package cli

import (
	"fmt"

	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/runner"
	"github.com/loom-cli/gryd/internal/state"
	"github.com/spf13/cobra"
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
			// Terminate all recorded process groups
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

		fmt.Println()
		fmt.Printf("\033[1;32m✓ Successfully stopped project '%s'.\033[0m\n", projectName)
		if stoppedCount > 0 {
			fmt.Printf("  • Terminated %d running service processes.\n", stoppedCount)
		}
		if len(freedPorts) > 0 {
			fmt.Printf("  • Released and confirmed ports: %v\n", freedPorts)
		} else if cfg != nil {
			fmt.Println("  • All project ports verified free.")
		}
		fmt.Println()

		return nil
	},
}
