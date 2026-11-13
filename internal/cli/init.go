package cli

import (
	"fmt"
	"os"

	"github.com/loom-cli/gryd/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	initWizardFlag bool
	initAutoFlag   bool
)

const sampleConfig = `name: %s
working_directory: .
services:
  backend:
    directory: backend
    command: echo "Starting backend..." && sleep 3600
    port: 8080
  frontend:
    directory: frontend
    command: echo "Starting frontend..." && sleep 3600
    port: 5173
    depends_on:
      - backend
settings:
  health_check: true
  restart_on_failure: false
`

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new gryd.yaml configuration",
	Long: `Generate a starter gryd.yaml configuration file in the current directory.
Use --auto to automatically scan your workspace and detect project stacks, frameworks,
commands, ports, and docker-compose sidecar dependencies.
Use --wizard to launch an interactive step-by-step setup to configure project name,
working directories, individual services, commands, ports, and environment variables.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := "MyProject"
		if len(args) > 0 {
			name = args[0]
		}

		if initAutoFlag {
			return runInitAuto(name)
		}

		if initWizardFlag {
			return runInitWizard(name)
		}

		targetFile := "gryd.yaml"
		if _, err := os.Stat(targetFile); err == nil {
			return fmt.Errorf("'%s' already exists in the current directory. Use 'gryd init --wizard' to reconfigure or overwrite", targetFile)
		}

		content := fmt.Sprintf(sampleConfig, name)
		if err := os.WriteFile(targetFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", targetFile, err)
		}

		fmt.Printf("Created '%s' for project '%s'. Run 'gryd run' to start your services!\n", targetFile, name)
		return nil
	},
}

func runInitAuto(customName string) error {
	targetFile := "gryd.yaml"
	if _, err := os.Stat(targetFile); err == nil {
		return fmt.Errorf("'%s' already exists in the current directory. Delete or rename it before running --auto", targetFile)
	}

	cfg, err := config.AutoDetectStack(".")
	if err != nil {
		return fmt.Errorf("stack auto-detection failed: %w", err)
	}

	if customName != "" && customName != "MyProject" {
		cfg.Name = customName
	}

	yamlBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to encode generated YAML: %w", err)
	}

	if err := os.WriteFile(targetFile, yamlBytes, 0644); err != nil {
		return fmt.Errorf("failed to write '%s': %w", targetFile, err)
	}

	fmt.Printf("✓ Detected project stack and generated '%s' for '%s'\n", targetFile, cfg.Name)
	fmt.Printf("  • %d service(s) detected:\n", len(cfg.Services))
	for sName, s := range cfg.Services {
		portInfo := ""
		if s.Port > 0 {
			portInfo = fmt.Sprintf(" (port %d)", s.Port)
		}
		dirInfo := ""
		if s.Directory != "" {
			dirInfo = fmt.Sprintf(" [%s]", s.Directory)
		}
		fmt.Printf("    - %s%s%s: %s\n", sName, dirInfo, portInfo, s.Command)
	}

	if len(cfg.Dependencies) > 0 {
		fmt.Printf("  • %d sidecar dependency(ies) imported from Docker Compose:\n", len(cfg.Dependencies))
		for dName, d := range cfg.Dependencies {
			fmt.Printf("    - %s (image: %s)\n", dName, d.Image)
		}
	}

	fmt.Println("\nRun 'gryd run' to launch your services!")
	return nil
}

func init() {
	initCmd.Flags().BoolVarP(&initWizardFlag, "wizard", "w", false, "Launch interactive configuration wizard")
	initCmd.Flags().BoolVarP(&initAutoFlag, "auto", "a", false, "Auto-detect project stack from directory")
}
