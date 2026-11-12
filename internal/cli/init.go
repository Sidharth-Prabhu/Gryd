package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var initWizardFlag bool

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
Use --wizard to launch an interactive step-by-step setup to configure project name,
working directories, individual services, commands, ports, and environment variables.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := "MyProject"
		if len(args) > 0 {
			name = args[0]
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

func init() {
	initCmd.Flags().BoolVarP(&initWizardFlag, "wizard", "w", false, "Launch interactive configuration wizard")
}
