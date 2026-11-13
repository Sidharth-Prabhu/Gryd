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

var restartCmd = &cobra.Command{
	Use:   "restart [project]",
	Short: "Completely terminate processes, free ports, and cleanly restart the project",
	Long: `Performs a hard restart of the project.
All active processes, process groups, child workers, and container sidecars are
completely terminated. All assigned network ports are released.
The project is then started completely fresh from configuration.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var projectName string
		if len(args) > 0 {
			// Check if user ran `gryd restart <service>` in the current project
			detected, err := config.DetectCurrentProject()
			if err == nil {
				cfgPath, err := config.FindConfigFile("", detected)
				if err == nil {
					cfg, err := config.Load(cfgPath)
					if err == nil {
						if _, isService := cfg.Services[args[0]]; isService {
							// Delegate to single service restart
							fmt.Printf("\033[1;33m[gryd] Restarting service '%s' in '%s'...\033[0m\n", args[0], detected)
							err = runner.RestartSingleService(detected, args[0])
							if err != nil {
								return err
							}
							fmt.Printf("\033[1;32m✓ Service '%s' restarted successfully.\033[0m\n", args[0])
							return nil
						}
					}
				}
			}
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
					return fmt.Errorf("multiple projects running. Please specify which one: gryd restart <project>")
				} else {
					return fmt.Errorf("no project specified and no active project in current directory")
				}
			}
		}

		fmt.Printf("\033[1;31m[gryd] Hard terminating all processes and ports for '%s'...\033[0m\n", projectName)

		// Hard stop and free ports entirely
		if st, err := state.LoadState(projectName); err == nil && st != nil {
			runner.StopProjectProcesses(st)
			_ = state.RemoveState(projectName)
		}

		fmt.Printf("\033[1;32m✓ Terminated processes & released ports.\033[0m\n")
		fmt.Printf("\033[1;33m[gryd] Booting project '%s' fresh...\033[0m\n", projectName)

		// Full background reload and fresh boot
		newState, err := runner.ReloadBackgroundProject(projectName)
		if err != nil {
			return fmt.Errorf("failed to restart project '%s': %w", projectName, err)
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
		header := tui.BadgeSuccess.Render(" ✓ PROJECT RESTARTED FRESH ")
		title := tui.BadgeProject.Render(" " + projectName + " ")
		topBar := lipgloss.JoinHorizontal(lipgloss.Center, header, " ", title)

		card := tui.RenderStatusCard(projectName, cfg, newState, termW)
		fmt.Println(lipgloss.JoinVertical(lipgloss.Left, topBar, "", card))
		fmt.Printf("\nServices are running in background.\nView live logs at any time with: \033[1mgryd look %s\033[0m\n\n", projectName)

		return nil
	},
}
