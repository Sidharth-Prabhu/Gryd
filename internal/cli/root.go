package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	configFile  string
	profileFlag string
	debug       bool
	showVersion bool

	rootCmd = &cobra.Command{
		Use:   "gryd [command]",
		Short: "Gryd orchestrates and runs multi-service development environments",
		Long: `Gryd is a cross-platform CLI tool that allows developers to configure,
manage, and launch entire multi-service development environments using a
single terminal command.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				fmt.Println(FormatVersion())
				return nil
			}
			return cmd.Help()
		},
	}
)

// Execute runs the root CLI command.
func Execute() error {
	CheckUpdatePrompt()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprint(os.Stderr, FormatGrydError(err))
		return err
	}
	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to configuration file (default is ./gryd.yaml)")
	rootCmd.PersistentFlags().StringVarP(&profileFlag, "profile", "p", "", "Target environment profile defined in gryd.yaml (e.g. minimal, full-stack)")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable verbose/debug logging output")

	// Version flags on root command: --version and -v / --v
	rootCmd.Flags().BoolVarP(&showVersion, "version", "", false, "Print version and release information")
	rootCmd.Flags().BoolVarP(&showVersion, "v", "v", false, "Print version and release information")

	// Register subcommands
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(serviceCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(reloadCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(portsCmd)
	rootCmd.AddCommand(vcCmd)
	rootCmd.AddCommand(lookCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateCmd)
}
