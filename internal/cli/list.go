package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/state"
	"github.com/loom-cli/gryd/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured and active projects",
	Long:  `Scan known Gryd project directories and display configured and running projects.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		running, _ := state.ListRunningProjects()

		var configs []*config.Config
		seenConfigs := make(map[string]bool)

		// 1. Check local directory
		for _, name := range []string{"gryd.yaml", "gryd.yml", "loom.yaml", "loom.yml"} {
			if _, err := os.Stat(name); err == nil {
				cfg, err := config.Load(name)
				if err == nil && !seenConfigs[cfg.Name] {
					seenConfigs[cfg.Name] = true
					configs = append(configs, cfg)
				}
			}
		}

		// 2. Check ~/.config/gryd/
		home, err := os.UserHomeDir()
		if err == nil {
			for _, dirName := range []string{"gryd", "loom"} {
				configDir := filepath.Join(home, ".config", dirName)
				entries, err := os.ReadDir(configDir)
				if err == nil {
					for _, entry := range entries {
						if !entry.IsDir() && (filepath.Ext(entry.Name()) == ".yaml" || filepath.Ext(entry.Name()) == ".yml") {
							cfgPath := filepath.Join(configDir, entry.Name())
							cfg, err := config.Load(cfgPath)
							if err == nil && !seenConfigs[cfg.Name] {
								seenConfigs[cfg.Name] = true
								configs = append(configs, cfg)
							}
						}
					}
				}
			}
		}

		// 3. Check registered projects
		registry := state.LoadProjectRegistry()
		for _, entry := range registry {
			if !seenConfigs[entry.Name] && entry.ConfigPath != "" {
				cfg, err := config.Load(entry.ConfigPath)
				if err == nil && !seenConfigs[cfg.Name] {
					seenConfigs[cfg.Name] = true
					configs = append(configs, cfg)
				}
			}
		}

		termW := 80
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 40 {
			termW = w
		}

		fmt.Println()
		fmt.Println(tui.RenderProjectsList(running, configs, termW))
		fmt.Println()

		return nil
	},
}
