package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/loom-cli/gryd/internal/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured projects and their status",
	Long:  `Scan known Gryd project directories and display configured projects.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Configured Gryd Projects:")
		found := false

		// 1. Check local directory
		for _, name := range []string{"gryd.yaml", "gryd.yml", "loom.yaml", "loom.yml"} {
			if _, err := os.Stat(name); err == nil {
				cfg, err := config.Load(name)
				if err == nil {
					fmt.Printf("  • %s (local: ./%s, %d services)\n", cfg.Name, name, len(cfg.Services))
					for svcName, svc := range cfg.Services {
						fmt.Printf("      - %s (cmd: '%s', port: %d)\n", svcName, svc.Command, svc.Port)
					}
					found = true
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
							if err == nil {
								fmt.Printf("  • %s (global: %s, %d services)\n", cfg.Name, cfgPath, len(cfg.Services))
								found = true
							}
						}
					}
				}
			}
		}

		if !found {
			fmt.Println("  No Gryd configurations found in current directory or ~/.config/gryd/")
		}

		return nil
	},
}
