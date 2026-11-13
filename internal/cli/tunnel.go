package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/tunnel"
	"github.com/spf13/cobra"
)

var (
	tunnelProviderFlag string
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel <service>",
	Short: "Expose a local service port to a public HTTPS URL (ngrok, Cloudflare, localtunnel)",
	Long: `Creates an instant secure public HTTPS tunnel to a service running in Gryd.
Supports cloudflared, ngrok, and localtunnel (lt).
If no provider is specified, Gryd auto-detects whichever is installed on your system.

Examples:
  gryd tunnel backend
  gryd tunnel web --provider ngrok
  gryd tunnel stop backend`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]
		if serviceName == "stop" {
			return fmt.Errorf("usage: gryd tunnel stop <service>")
		}

		projectName, err := config.DetectCurrentProject()
		if err != nil {
			return err
		}

		cfgPath, err := config.FindConfigFile(configFile, projectName)
		if err != nil {
			return err
		}

		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}

		svc, exists := cfg.Services[serviceName]
		if !exists {
			return fmt.Errorf("service '%s' not found in configuration", serviceName)
		}

		if svc.Port <= 0 {
			return fmt.Errorf("service '%s' has no port configured in gryd.yaml", serviceName)
		}

		fmt.Printf("\033[1;33m[gryd] Starting public tunnel for '%s' (port :%d)...\033[0m\n", serviceName, svc.Port)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		tun, err := tunnel.GetGlobalManager().StartTunnel(ctx, serviceName, svc.Port, tunnelProviderFlag)
		if err != nil {
			return fmt.Errorf("failed to start tunnel: %w", err)
		}

		fmt.Println()
		fmt.Printf("\033[1;32m✓ Public Tunnel Active!\033[0m\n")
		fmt.Printf("  \033[1mService:\033[0m    %s\n", serviceName)
		fmt.Printf("  \033[1mLocal Port:\033[0m :%d\n", svc.Port)
		fmt.Printf("  \033[1mProvider:\033[0m   %s\n", tun.Provider)
		fmt.Printf("  \033[1;36mPublic URL: %s\033[0m\n\n", tun.PublicURL)
		fmt.Println("Press Ctrl+C to close the tunnel.")

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		fmt.Printf("\n\033[1;33mClosing tunnel for '%s'...\033[0m\n", serviceName)
		_ = tunnel.GetGlobalManager().StopTunnel(serviceName)
		fmt.Printf("\033[1;32m✓ Tunnel closed.\033[0m\n")
		return nil
	},
}

var tunnelStopCmd = &cobra.Command{
	Use:   "stop <service>",
	Short: "Stop an active tunnel for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]
		err := tunnel.GetGlobalManager().StopTunnel(serviceName)
		if err != nil {
			return err
		}
		fmt.Printf("\033[1;32m✓ Tunnel for '%s' stopped.\033[0m\n", serviceName)
		return nil
	},
}

func init() {
	tunnelCmd.Flags().StringVarP(&tunnelProviderFlag, "provider", "", "", "Tunneling provider: cloudflared, ngrok, lt")
	tunnelCmd.AddCommand(tunnelStopCmd)
	rootCmd.AddCommand(tunnelCmd)
}
