package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/state"
	"github.com/loom-cli/gryd/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var statusJSONFlag bool

type ProjectStatusJSON struct {
	Name        string                       `json:"name"`
	ConfigPath  string                       `json:"config_path,omitempty"`
	LogFilePath string                       `json:"log_file_path,omitempty"`
	Running     bool                         `json:"running"`
	Services    map[string]ServiceStatusJSON `json:"services"`
}

type ServiceStatusJSON struct {
	Command string `json:"command"`
	Port    int    `json:"port,omitempty"`
	PID     int    `json:"pid,omitempty"`
	Running bool   `json:"running"`
}

var statusCmd = &cobra.Command{
	Use:   "status [project]",
	Short: "Show the current status of a project (Running / Terminated / Not Running)",
	Long: `Display detailed runtime status for configured services in a project.
If [project] is omitted, Gryd checks the local directory project or lists all active projects.
Pass --json for machine-readable JSON output suitable for scripts and CI/CD pipelines.`,
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
		if statusJSONFlag {
			data, _ := json.MarshalIndent(map[string]string{
				"error": fmt.Sprintf("project '%s' was not found", projectName),
			}, "", "  ")
			fmt.Println(string(data))
			return nil
		}
		return fmt.Errorf("project '%s' was not found (no local config or active state found)", projectName)
	}

	if statusJSONFlag {
		output := buildProjectStatusJSON(projectName, cfg, st)
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
		return nil
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
	if statusJSONFlag {
		var list []ProjectStatusJSON
		for _, st := range runningProjects {
			var cfg *config.Config
			if st.ConfigPath != "" {
				cfg, _ = config.Load(st.ConfigPath)
			}
			list = append(list, buildProjectStatusJSON(st.ProjectName, cfg, st))
		}
		data, _ := json.MarshalIndent(list, "", "  ")
		fmt.Println(string(data))
		return nil
	}

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

func buildProjectStatusJSON(projectName string, cfg *config.Config, st *state.ProjectState) ProjectStatusJSON {
	out := ProjectStatusJSON{
		Name:     projectName,
		Services: make(map[string]ServiceStatusJSON),
	}

	if st != nil {
		out.ConfigPath = st.ConfigPath
		out.LogFilePath = st.LogFilePath
	} else if cfg != nil {
		out.ConfigPath = cfg.ConfigPath
	}

	running := false
	if cfg != nil {
		for sName, s := range cfg.Services {
			sJSON := ServiceStatusJSON{
				Command: s.Command,
				Port:    s.Port,
			}
			if st != nil {
				if proc, ok := st.Processes[sName]; ok && proc.PID > 0 && state.IsProcessAlive(proc.PID) {
					sJSON.PID = proc.PID
					sJSON.Running = true
					running = true
				}
			}
			out.Services[sName] = sJSON
		}
	} else if st != nil {
		for sName, p := range st.Processes {
			alive := p.PID > 0 && state.IsProcessAlive(p.PID)
			if alive {
				running = true
			}
			out.Services[sName] = ServiceStatusJSON{
				Command: p.Command,
				PID:     p.PID,
				Running: alive,
			}
		}
	}

	out.Running = running
	return out
}

func init() {
	statusCmd.Flags().BoolVarP(&statusJSONFlag, "json", "", false, "Output project status in machine-readable JSON format")
}
