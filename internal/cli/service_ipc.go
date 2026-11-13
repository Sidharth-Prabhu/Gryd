package cli

import (
	"fmt"

	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/runner"
	"github.com/spf13/cobra"
)

var serviceRestartCmd = &cobra.Command{
	Use:   "restart <service>",
	Short: "Restart a single service via daemon IPC",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]
		projectName, err := resolveTargetProject(serviceProjectFlag)
		if err != nil {
			return err
		}

		resp, err := runner.SendIPCCommand(projectName, runner.IPCRequest{
			Action:  "restart",
			Service: serviceName,
		})
		if err != nil {
			// Fallback directly to standalone single service restart
			err = runner.RestartSingleService(projectName, serviceName)
			if err != nil {
				return err
			}
			fmt.Printf("\033[1;32m✓ Service '%s' restarted successfully in '%s'.\033[0m\n", serviceName, projectName)
			return nil
		}

		if !resp.Success {
			return fmt.Errorf("daemon error: %s", resp.Error)
		}

		fmt.Printf("\033[1;32m✓ %s\033[0m\n", resp.Output)
		return nil
	},
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop <service>",
	Short: "Stop a single service via daemon IPC",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]
		projectName, err := resolveTargetProject(serviceProjectFlag)
		if err != nil {
			return err
		}

		resp, err := runner.SendIPCCommand(projectName, runner.IPCRequest{
			Action:  "stop",
			Service: serviceName,
		})
		if err != nil {
			return fmt.Errorf("failed to communicate with daemon: %w", err)
		}

		if !resp.Success {
			return fmt.Errorf("daemon error: %s", resp.Error)
		}

		fmt.Printf("\033[1;32m✓ %s\033[0m\n", resp.Output)
		return nil
	},
}

var serviceExecCmd = &cobra.Command{
	Use:   "exec <service> -- <cmd...>",
	Short: "Execute a command inside a service directory via daemon IPC",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]
		execArgs := args[1:]
		if len(execArgs) == 0 {
			return fmt.Errorf("no command provided to execute. Example: gryd service exec %s -- ls -la", serviceName)
		}

		projectName, err := resolveTargetProject(serviceProjectFlag)
		if err != nil {
			return err
		}

		resp, err := runner.SendIPCCommand(projectName, runner.IPCRequest{
			Action:  "exec",
			Service: serviceName,
			Command: execArgs,
		})
		if err != nil {
			return fmt.Errorf("failed to communicate with daemon: %w", err)
		}

		if resp.Output != "" {
			fmt.Print(resp.Output)
		}

		if !resp.Success {
			return fmt.Errorf("execution error: %s", resp.Error)
		}

		return nil
	},
}

func resolveTargetProject(flagProject string) (string, error) {
	if flagProject != "" {
		return flagProject, nil
	}
	detected, err := config.DetectCurrentProject()
	if err == nil {
		return detected, nil
	}
	return "", fmt.Errorf("could not detect active project. Please specify with --project <name>")
}

func init() {
	serviceCmd.AddCommand(serviceRestartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceExecCmd)
}
