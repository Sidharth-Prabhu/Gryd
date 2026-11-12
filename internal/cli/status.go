package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/state"
	"github.com/spf13/cobra"
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
	// Try loading active runtime state
	st, _ := state.LoadState(projectName)

	// Try loading static config file to get service metadata (ports, dirs, commands)
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

	actualName := projectName
	if cfg != nil {
		actualName = cfg.Name
	} else if st != nil {
		actualName = st.ProjectName
	}

	fmt.Printf("\n\033[1mProject:\033[0m          %s\n", actualName)
	if cfg != nil {
		fmt.Printf("\033[1mWorking Dir:\033[0m      %s\n", cfg.WorkingDirectory)
		fmt.Printf("\033[1mConfig File:\033[0m      %s\n", cfg.ConfigPath)
	}
	if st != nil && st.LogFilePath != "" {
		fmt.Printf("\033[1mLog File:\033[0m         %s\n", st.LogFilePath)
	}

	// Determine overall project status
	overallStatus, statusColor := determineProjectStatus(st, cfg)
	fmt.Printf("\033[1mOverall Status:\033[0m   %s%s%s\n", statusColor, overallStatus, "\033[0m")
	if st != nil && (overallStatus == "Running" || overallStatus == "Partially Running") {
		uptime := time.Since(st.StartedAt).Round(time.Second)
		fmt.Printf("\033[1mUptime:\033[0m           %s (started %s)\n", uptime, st.StartedAt.Format("15:04:05"))
	}
	fmt.Println()

	// Print services table
	fmt.Printf("  \033[1m%-16s %-12s %-8s %-8s %s\033[0m\n", "SERVICE", "STATUS", "PID", "PORT", "COMMAND")
	fmt.Printf("  %s\n", strings.Repeat("─", 68))

	// Collect service names
	serviceNames := make(map[string]bool)
	if cfg != nil {
		for name := range cfg.Services {
			serviceNames[name] = true
		}
	}
	if st != nil {
		for name := range st.Processes {
			serviceNames[name] = true
		}
	}

	for name := range serviceNames {
		cmdStr := "-"
		portStr := "-"
		if cfg != nil {
			if svc, ok := cfg.Services[name]; ok {
				cmdStr = svc.Command
				if svc.Port > 0 {
					portStr = fmt.Sprintf("%d", svc.Port)
				}
			}
		}

		pidStr := "-"
		statusBadge := "\033[31mNot Running\033[0m"

		if st != nil {
			if proc, ok := st.Processes[name]; ok {
				if proc.Command != "" && cmdStr == "-" {
					cmdStr = proc.Command
				}
				if proc.PID > 0 {
					if state.IsProcessAlive(proc.PID) {
						statusBadge = "\033[32mRunning\033[0m"
						pidStr = fmt.Sprintf("%d", proc.PID)
					} else {
						statusBadge = "\033[31mTerminated\033[0m"
						pidStr = fmt.Sprintf("%d", proc.PID)
					}
				}
			}
		}

		if len(cmdStr) > 30 {
			cmdStr = cmdStr[:27] + "..."
		}

		fmt.Printf("  %-16s %-21s %-8s %-8s %s\n", name, statusBadge, pidStr, portStr, cmdStr)
	}
	fmt.Println()

	if overallStatus == "Running" || overallStatus == "Partially Running" {
		fmt.Printf("View live logs with: \033[1mgryd look %s\033[0m\n\n", actualName)
	}

	return nil
}

func determineProjectStatus(st *state.ProjectState, cfg *config.Config) (string, string) {
	if st == nil {
		return "Not Running", "\033[33m" // Yellow
	}

	aliveCount := 0
	totalProcesses := len(st.Processes)

	for _, p := range st.Processes {
		if p.PID > 0 && state.IsProcessAlive(p.PID) {
			aliveCount++
		}
	}

	if aliveCount == 0 {
		return "Terminated", "\033[31m" // Red
	}

	targetCount := totalProcesses
	if cfg != nil && len(cfg.Services) > targetCount {
		targetCount = len(cfg.Services)
	}

	if aliveCount == targetCount {
		return "Running", "\033[32m" // Green
	}

	return "Partially Running", "\033[33m" // Yellow
}

func showAllProjectsStatus() error {
	running, _ := state.ListRunningProjects()

	fmt.Println("\n\033[1mGryd Project Status Overview\033[0m")
	fmt.Printf("%s\n", strings.Repeat("─", 60))

	foundAny := false

	if len(running) > 0 {
		foundAny = true
		fmt.Printf("\n\033[1;32mActive Projects:\033[0m\n")
		for _, st := range running {
			status, color := determineProjectStatus(st, nil)
			uptime := time.Since(st.StartedAt).Round(time.Second)
			fmt.Printf("  • \033[1m%-16s\033[0m %s%-16s\033[0m (uptime: %s, %d services)\n",
				st.ProjectName, color, status, uptime, len(st.Processes))
			fmt.Printf("      Logs: %s (inspect: gryd status %s)\n", st.LogFilePath, st.ProjectName)
		}
	}

	// Also check known config files
	home, err := os.UserHomeDir()
	if err == nil {
		configDir := filepath.Join(home, ".config", "gryd")
		entries, err := os.ReadDir(configDir)
		if err == nil {
			var idleConfigs []string
			for _, entry := range entries {
				if !entry.IsDir() && (filepath.Ext(entry.Name()) == ".yaml" || filepath.Ext(entry.Name()) == ".yml") {
					name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
					// check if not already in running
					isRunning := false
					for _, r := range running {
						if r.ProjectName == name {
							isRunning = true
							break
						}
					}
					if !isRunning {
						idleConfigs = append(idleConfigs, name)
					}
				}
			}
			if len(idleConfigs) > 0 {
				foundAny = true
				fmt.Printf("\n\033[1mConfigured Projects (Not Running):\033[0m\n")
				for _, name := range idleConfigs {
					fmt.Printf("  • %-16s \033[33mNot Running\033[0m (run with: gryd run %s)\n", name, name)
				}
			}
		}
	}

	if !foundAny {
		fmt.Println("No projects currently running or configured.")
		fmt.Println("Run 'gryd init' to create a new project or 'gryd status <project>' to inspect one.")
	} else {
		fmt.Println("\nRun \033[1mgryd status <project>\033[0m for detailed service breakdown.")
	}
	fmt.Println()

	return nil
}
