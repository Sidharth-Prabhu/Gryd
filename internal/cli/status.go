package cli

import (
	"fmt"
	"os"

	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/state"
	"github.com/loom-cli/gryd/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var statusCmd = &cobra.Command{
	Use:   "status [project]",
	Short: "Show the current status of a project (Running / Terminated / Not Running)",
	Long: `Display detailed runtime status for configured services in a project.
If [project] is omitted, Gryd checks the local directory project or lists all active projects.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var projectName string
		if len(args) > 0 {
			projectName = args[0]
		}

		// Case 1: Specific project provided
		if projectName != "" {
			return showProjectStatus(projectName)
		}

		// Case 2: No argument provided -> check current directory for local config
		localConfigPath, err := config.FindConfigFile("", "")
		if err == nil {
			localCfg, err := config.Load(localConfigPath)
			if err == nil {
				return showProjectStatus(localCfg.Name)
			}
		}

		// Case 3: Show all active and known projects
		return showAllProjectsStatus()
	},
}

func showProjectStatus(projectName string) error {
	st, _ := state.LoadState(projectName)

	var cfg *config.Config
	configPath, err := config.FindConfigFile(configFile, projectName)
	if err == nil {
		cfg, _ = config.Load(configPath)
	} else if st != nil && st.ConfigPath != "" {
		cfg, _ = config.Load(st.ConfigPath)
	}

	if st == nil && cfg == nil {
		return fmt.Errorf("project '%s' was not found (no local config or active state found)", projectName)
	}

	termW := 80
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 40 {
		termW = w
	}

	fmt.Println()
	fmt.Println(tui.RenderStatusCard(projectName, cfg, st, termW))
	fmt.Println()

	aliveCount := 0
	if st != nil {
		for _, p := range st.Processes {
			if p.PID > 0 && state.IsProcessAlive(p.PID) {
				aliveCount++
			}
		}
	}

	if aliveCount > 0 {
		fmt.Printf("View live logs with: \033[1mgryd look %s\033[0m\n\n", projectName)
	}

	return nil
}

func showAllProjectsStatus() error {
	runningProjects, _ := state.ListRunningProjects()
	if len(runningProjects) == 0 {
		fmt.Println("\nNo active Gryd projects currently running.")
		fmt.Println("Run 'gryd run' in a project directory to start services.")
		fmt.Println()
		return nil
	}

	termW := 80
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 40 {
		termW = w
	}

	for _, st := range runningProjects {
		var cfg *config.Config
		if st.ConfigPath != "" {
			cfg, _ = config.Load(st.ConfigPath)
		}
		fmt.Println()
		fmt.Println(tui.RenderStatusCard(st.ProjectName, cfg, st, termW))
	}
	fmt.Println()

	return nil
}
