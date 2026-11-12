package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	configFile string
	verbose    bool

	rootCmd = &cobra.Command{
		Use:   "gryd [command]",
		Short: "Gryd orchestrates and runs multi-service development environments",
		Long: `Gryd is a cross-platform CLI tool that allows developers to configure,
manage, and launch entire multi-service development environments using a
single terminal command.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
)

// Execute runs the root CLI command.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
	}
	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to configuration file (default is ./gryd.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose/debug logging output")

	// Register subcommands
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(initCmd)
}
