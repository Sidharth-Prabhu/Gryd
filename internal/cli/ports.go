package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/runner"
	"github.com/loom-cli/gryd/internal/state"
	"github.com/loom-cli/gryd/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var portsCmd = &cobra.Command{
	Use:   "ports",
	Short: "Display all ports used by Gryd projects and their active processes",
	Long: `Scan all active and configured Gryd projects and display a structured overview
of assigned ports, listening status, and process IDs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Gather all running projects
		runningProjects, _ := state.ListRunningProjects()

		// 2. Map project names to their configs
		registry := state.LoadProjectRegistry()
		projectConfigs := make(map[string]*config.Config)

		for _, st := range runningProjects {
			if st.ConfigPath != "" {
				if cfg, err := config.Load(st.ConfigPath); err == nil {
					projectConfigs[cfg.Name] = cfg
				}
			}
		}

		for _, entry := range registry {
			if _, exists := projectConfigs[entry.Name]; !exists && entry.ConfigPath != "" {
				if cfg, err := config.Load(entry.ConfigPath); err == nil {
					projectConfigs[cfg.Name] = cfg
				}
			}
		}

		// Also check local directory
		for _, name := range []string{"gryd.yaml", "gryd.yml", "loom.yaml", "loom.yml"} {
			if cfg, err := config.Load(name); err == nil {
				if _, exists := projectConfigs[cfg.Name]; !exists {
					projectConfigs[cfg.Name] = cfg
				}
			}
		}

		type portEntry struct {
			Port        int
			ProjectName string
			ServiceName string
			PIDs        []int
			Status      string
			ProcessName string
		}

		var entries []portEntry

		for projName, cfg := range projectConfigs {
			for svcName, svc := range cfg.Services {
				if svc.Port <= 0 {
					continue
				}

				entry := portEntry{
					Port:        svc.Port,
					ProjectName: projName,
					ServiceName: svcName,
					Status:      "Inactive",
					ProcessName: "-",
				}

				if runner.IsPortInUse(svc.Port) {
					entry.Status = "Active (Listening)"
					details := runner.GetProcessDetailsForPort(svc.Port)
					if len(details) > 0 {
						var pids []int
						var names []string
						for _, d := range details {
							pids = append(pids, d.PID)
							if d.Command != "" {
								names = append(names, d.Command)
							}
						}
						entry.PIDs = pids
						if len(names) > 0 {
							entry.ProcessName = strings.Join(names, ", ")
						}
					}
				}

				entries = append(entries, entry)
			}
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Port < entries[j].Port
		})

		var rows []tui.PortRow
		for _, e := range entries {
			pidStr := "-"
			if len(e.PIDs) > 0 {
				var s []string
				for _, p := range e.PIDs {
					s = append(s, fmt.Sprintf("%d", p))
				}
				pidStr = strings.Join(s, ", ")
			}

			rows = append(rows, tui.PortRow{
				Port:        e.Port,
				ProjectName: e.ProjectName,
				ServiceName: e.ServiceName,
				PIDs:        pidStr,
				Status:      e.Status,
				ProcessName: e.ProcessName,
			})
		}

		termW := 80
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 40 {
			termW = w
		}

		fmt.Println()
		fmt.Println(tui.RenderPortsTable(rows, termW))
		fmt.Printf("\n  \033[2mRun gryd stop <project> to terminate a project and release its ports.\033[0m\n\n")

		return nil
	},
}
