package cli

import (
	"context"
	"fmt"

	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/logger"
	"github.com/loom-cli/gryd/internal/runner"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var runCmd = &cobra.Command{
	Use:   "run [project]",
	Short: "Start all services for a project environment",
	Long: `Start all services defined in a gryd.yaml configuration concurrently.
If [project] is provided, Gryd searches for the project configuration in standard
directories or uses the local directory.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var projectName string
		if len(args) > 0 {
			projectName = args[0]
		}

		zapLog, err := logger.NewZapLogger(debug)
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}
		defer func() {
			_ = zapLog.Sync()
		}()

		configPath, err := config.FindConfigFile(configFile, projectName)
		if err != nil {
			return err
		}

		zapLog.Info("Loading Gryd configuration", zap.String("file", configPath))
		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		appRunner, err := runner.New(cfg, zapLog)
		if err != nil {
			return fmt.Errorf("failed to initialize runner: %w", err)
		}

		return appRunner.Run(context.Background())
	},
}
