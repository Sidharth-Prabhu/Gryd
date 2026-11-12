package cli

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/runner"
	"github.com/loom-cli/gryd/internal/state"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var lookCmd = &cobra.Command{
	Use:   "look [project]",
	Short: "Stream live logs for a running background project",
	Long: `Attach to and stream real-time logs for a background project started by Gryd.
If inside a project directory, [project] can be omitted.
Press 'r' to reload, 'c' to run in background (detach), or Ctrl+C to quit.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var projectName string
		if len(args) > 0 {
			projectName = args[0]
		} else {
			// Auto-detect project if inside project directory
			detected, err := config.DetectCurrentProject()
			if err == nil {
				projectName = detected
			} else {
				// If only one project is currently running, default to it
				running, _ := state.ListRunningProjects()
				if len(running) == 1 {
					projectName = running[0].ProjectName
				} else if len(running) > 1 {
					return fmt.Errorf("multiple projects running. Please specify which one: gryd look <project>")
				} else {
					return fmt.Errorf("no project specified and no active project in current directory")
				}
			}
		}

		projectState, err := state.LoadState(projectName)
		if err != nil {
			return err
		}

		aliveCount := 0
		for _, p := range projectState.Processes {
			if state.IsProcessAlive(p.PID) {
				aliveCount++
			}
		}

		if aliveCount == 0 {
			_ = state.RemoveState(projectName)
			return fmt.Errorf("project '%s' is no longer running (all processes terminated)", projectName)
		}

		logFilePath := projectState.LogFilePath
		file, err := os.Open(logFilePath)
		if err != nil {
			return fmt.Errorf("failed to open log file '%s': %w", logFilePath, err)
		}
		defer file.Close()

		footerText := "\033[1;36mPress 'r' to reload\033[0m  |  \033[1;32mPress 'c' to run in background\033[0m  |  \033[2mCtrl+C to quit\033[0m"
		footerMgr := runner.NewPinnedFooterManager(int(os.Stdout.Fd()), footerText)
		footerMgr.Init()
		defer footerMgr.Reset()

		// Seek to near end of log file
		stat, err := file.Stat()
		if err == nil && stat.Size() > 4096 {
			_, _ = file.Seek(-4096, io.SeekEnd)
		}

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigChan)

		reloadChan := make(chan struct{}, 1)
		detachChan := make(chan struct{}, 1)
		quitChan := make(chan struct{}, 1)

		stopRaw := setupFullKeyboardListener(reloadChan, detachChan, quitChan)
		defer stopRaw()

		buf := make([]byte, 4096)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-sigChan:
				stopRaw()
				footerMgr.Reset()
				fmt.Printf("\n\033[1;31m[gryd] Stopping background project '%s'...\033[0m\n", projectName)
				if currentSt, err := state.LoadState(projectName); err == nil {
					runner.StopProjectProcesses(currentSt)
					_ = state.RemoveState(projectName)
				}
				fmt.Printf("\033[1;32m[gryd] All services for '%s' stopped cleanly.\033[0m\n", projectName)
				return nil

			case <-quitChan:
				stopRaw()
				footerMgr.Reset()
				fmt.Printf("\n\033[1;31m[gryd] Stopping background project '%s'...\033[0m\n", projectName)
				if currentSt, err := state.LoadState(projectName); err == nil {
					runner.StopProjectProcesses(currentSt)
					_ = state.RemoveState(projectName)
				}
				fmt.Printf("\033[1;32m[gryd] All services for '%s' stopped cleanly.\033[0m\n", projectName)
				return nil

			case <-detachChan:
				stopRaw()
				footerMgr.Reset()
				fmt.Printf("\n\033[1;32m[gryd] Detached from '%s'. Services are still running in background.\033[0m\n", projectName)
				fmt.Printf("Re-attach anytime using: \033[1mgryd look %s\033[0m\n", projectName)
				return nil

			case <-reloadChan:
				footerMgr.Reset()
				fmt.Printf("\n\033[1;33m[gryd] Reloading background project '%s'...\033[0m\n", projectName)
				newSt, err := runner.ReloadBackgroundProject(projectName)
				if err != nil {
					fmt.Printf("\033[1;31m[gryd] Reload error: %v\033[0m\n", err)
				} else {
					projectState = newSt
					fmt.Printf("\033[1;32m[gryd] Reloaded successfully (%d services active).\033[0m\n", len(newSt.Processes))
				}
				footerMgr.Init()

			case <-ticker.C:
				for {
					n, err := file.Read(buf)
					if n > 0 {
						_, _ = footerMgr.WriteToScrollRegion(os.Stdout, buf[:n])
					}
					if err != nil {
						break
					}
				}
			}
		}
	},
}

func setupFullKeyboardListener(reloadChan, detachChan, quitChan chan struct{}) func() {
	stdinFd := int(os.Stdin.Fd())
	if !term.IsTerminal(stdinFd) {
		return func() {}
	}

	oldState, err := term.MakeRaw(stdinFd)
	if err != nil {
		return func() {}
	}

	stopChan := make(chan struct{})
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			close(stopChan)
			_ = term.Restore(stdinFd, oldState)
		})
	}

	go func() {
		buf := make([]byte, 1)
		for {
			select {
			case <-stopChan:
				return
			default:
				n, err := os.Stdin.Read(buf)
				if err != nil || n == 0 {
					return
				}
				b := buf[0]
				switch b {
				case 'r', 'R':
					select {
					case reloadChan <- struct{}{}:
					default:
					}
				case 'c', 'C':
					select {
					case detachChan <- struct{}{}:
					default:
					}
				case 3: // Ctrl+C
					select {
					case quitChan <- struct{}{}:
					default:
					}
					return
				}
			}
		}
	}()

	return cleanup
}
