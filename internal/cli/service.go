package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/logger"
	"github.com/loom-cli/gryd/internal/runner"
	"github.com/loom-cli/gryd/internal/state"
	"github.com/loom-cli/gryd/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var serviceProjectFlag string

var serviceCmd = &cobra.Command{
	Use:   "service [command]",
	Short: "Manage and run individual services within a Gryd project",
	Long:  `Run or inspect specific individual services without booting the entire project environment.`,
}

var serviceRunCmd = &cobra.Command{
	Use:   "run <service1> [service2...]",
	Short: "Run only the specified service(s) from the project",
	Long: `Start only the selected services from your configuration concurrently.
Useful when you want to isolate specific components (e.g. 'gryd service run worker frontend').

Supports multiple services in a single command.
If inside a project directory, the project is auto-detected. Use --project to specify from elsewhere.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetServices := args

		configPath, err := config.FindConfigFile(configFile, serviceProjectFlag)
		if err != nil {
			return err
		}

		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		// Validate requested service names against config
		var available []string
		for name := range cfg.Services {
			available = append(available, name)
		}

		filteredServices := make(map[string]config.Service)
		var invalidNames []string

		for _, requested := range targetServices {
			if svc, ok := cfg.Services[requested]; ok {
				filteredServices[requested] = svc
			} else {
				invalidNames = append(invalidNames, requested)
			}
		}

		if len(invalidNames) > 0 {
			return fmt.Errorf("unknown service(s): %s\nAvailable services in '%s': %s",
				strings.Join(invalidNames, ", "), cfg.Name, strings.Join(available, ", "))
		}

		// Create a copy of config with only the requested services
		subsetCfg := &config.Config{
			Name:             cfg.Name,
			WorkingDirectory: cfg.WorkingDirectory,
			Services:         filteredServices,
			Settings:         cfg.Settings,
			ConfigPath:       cfg.ConfigPath,
		}

		// Fallback for non-interactive / piped environments
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			zapLog, err := logger.NewZapLogger(debug)
			if err != nil {
				return fmt.Errorf("failed to initialize logger: %w", err)
			}
			defer func() {
				_ = zapLog.Sync()
			}()

			appRunner, err := runner.New(subsetCfg, zapLog)
			if err != nil {
				return fmt.Errorf("failed to initialize runner: %w", err)
			}
			return appRunner.Run(context.Background())
		}

		// Start subset in background for UI
		projectState, err := state.LoadState(subsetCfg.Name)
		if err != nil || len(projectState.Processes) == 0 {
			newSt, err := runner.ReloadBackgroundProject(subsetCfg.Name)
			if err != nil {
				return fmt.Errorf("failed to start selected services: %w", err)
			}
			projectState = newSt
		}

		model, err := tui.NewUIModel(subsetCfg, projectState)
		if err != nil {
			return err
		}

		p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("error running UI: %w", err)
		}

		if m, ok := finalModel.(*tui.UIModel); ok && !m.IsDetached() {
			fmt.Printf("\n\033[1;33m[gryd service run] Stopping selected services for '%s'...\033[0m\n", subsetCfg.Name)
			runner.StopProjectProcesses(projectState)
			_ = state.RemoveState(subsetCfg.Name)
			fmt.Printf("\033[1;32m✓ Selected services stopped cleanly.\033[0m\n\n")
		}

		return nil
	},
}

func init() {
	serviceCmd.PersistentFlags().StringVarP(&serviceProjectFlag, "project", "p", "", "Target project name if outside project directory")
	serviceCmd.AddCommand(serviceRunCmd)
}
