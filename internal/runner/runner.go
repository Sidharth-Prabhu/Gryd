package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/logger"
	"go.uber.org/zap"
)

// ProcessState tracks an individual active service.
type ProcessState struct {
	Name    string
	Service config.Service
	Cmd     *exec.Cmd
	Dir     string
	Started time.Time
}

// Runner supervises and manages concurrent service execution.
type Runner struct {
	cfg         *config.Config
	log         *zap.Logger
	multiplexer *logger.LogMultiplexer
	processes   map[string]*ProcessState
	mu          sync.Mutex
	stopOnce    sync.Once
}

// New creates a new Runner instance.
func New(cfg *config.Config, log *zap.Logger) *Runner {
	return &Runner{
		cfg:         cfg,
		log:         log,
		multiplexer: logger.NewLogMultiplexer(os.Stdout),
		processes:   make(map[string]*ProcessState),
	}
}

// Run launches all services concurrently and blocks until completion or interrupt signal.
func (r *Runner) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Capture interrupt signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	r.log.Info("Starting Gryd project",
		zap.String("project", r.cfg.Name),
		zap.String("working_dir", r.cfg.WorkingDirectory),
		zap.Int("services_count", len(r.cfg.Services)),
	)

	// Channel to report process exit results
	type exitResult struct {
		name string
		err  error
	}
	exitChan := make(chan exitResult, len(r.cfg.Services))

	var wg sync.WaitGroup

	// Launch each service concurrently
	for name, svc := range r.cfg.Services {
		// Resolve service directory
		svcDir := r.cfg.WorkingDirectory
		if svc.Directory != "" {
			if filepath.IsAbs(svc.Directory) {
				svcDir = svc.Directory
			} else {
				svcDir = filepath.Join(r.cfg.WorkingDirectory, svc.Directory)
			}
		}

		// Ensure directory exists
		if _, err := os.Stat(svcDir); os.IsNotExist(err) {
			r.log.Warn("Service directory does not exist, creating it",
				zap.String("service", name),
				zap.String("path", svcDir),
			)
			if mkErr := os.MkdirAll(svcDir, 0755); mkErr != nil {
				r.shutdownAll()
				return fmt.Errorf("failed to create directory for service '%s': %w", name, mkErr)
			}
		}

		cmd := shellCommand(svc.Command)
		cmd.Dir = svcDir

		// Inherit environment and apply service-specific env vars
		cmd.Env = os.Environ()
		for k, v := range svc.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}

		// Route stdout and stderr through the multiplexer
		outWriter := r.multiplexer.NewServiceWriter(name)
		cmd.Stdout = outWriter
		cmd.Stderr = outWriter

		// Set process group for group termination
		setProcessGroup(cmd)

		if err := cmd.Start(); err != nil {
			r.shutdownAll()
			return fmt.Errorf("failed to start service '%s': %w", name, err)
		}

		procState := &ProcessState{
			Name:    name,
			Service: svc,
			Cmd:     cmd,
			Dir:     svcDir,
			Started: time.Now(),
		}

		r.mu.Lock()
		r.processes[name] = procState
		r.mu.Unlock()

		r.log.Info("Service started",
			zap.String("service", name),
			zap.Int("pid", cmd.Process.Pid),
			zap.String("command", svc.Command),
		)

		wg.Add(1)
		go func(serviceName string, command *exec.Cmd) {
			defer wg.Done()
			err := command.Wait()
			exitChan <- exitResult{name: serviceName, err: err}
		}(name, cmd)
	}

	// Wait for shutdown trigger or process exit
	select {
	case sig := <-sigChan:
		fmt.Printf("\nReceived signal %s. Initiating graceful shutdown...\n", sig)
		r.shutdownAll()
	case res := <-exitChan:
		if res.err != nil {
			r.log.Error("Service exited with error",
				zap.String("service", res.name),
				zap.Error(res.err),
			)
		} else {
			r.log.Info("Service exited normally", zap.String("service", res.name))
		}

		// If restart_on_failure is disabled, stop the entire environment
		if !r.cfg.Settings.RestartOnFailure {
			r.log.Info("Stopping all remaining services...")
			r.shutdownAll()
		}
	case <-ctx.Done():
		r.shutdownAll()
	}

	// Wait for all processes to finish terminating
	wg.Wait()
	r.log.Info("All services cleanly terminated")
	return nil
}

// shutdownAll cleanly terminates all active service processes.
func (r *Runner) shutdownAll() {
	r.stopOnce.Do(func() {
		r.mu.Lock()
		defer r.mu.Unlock()

		if len(r.processes) == 0 {
			return
		}

		fmt.Println("Shutting down services...")

		// Step 1: Send SIGTERM to all process groups
		for name, proc := range r.processes {
			if proc.Cmd != nil && proc.Cmd.Process != nil {
				r.log.Debug("Sending SIGTERM to service process group",
					zap.String("service", name),
					zap.Int("pid", proc.Cmd.Process.Pid),
				)
				_ = killProcessGroup(proc.Cmd, syscall.SIGTERM)
			}
		}

		// Step 2: Give processes up to 5 seconds to exit gracefully
		done := make(chan struct{})
		go func() {
			for _, proc := range r.processes {
				if proc.Cmd != nil && proc.Cmd.Process != nil {
					_, _ = proc.Cmd.Process.Wait()
				}
			}
			close(done)
		}()

		select {
		case <-done:
			// Clean exit
		case <-time.After(5 * time.Second):
			// Step 3: Force kill any remaining stubborn processes with SIGKILL
			r.log.Warn("Shutdown timed out; force killing remaining processes...")
			for _, proc := range r.processes {
				if proc.Cmd != nil && proc.Cmd.Process != nil {
					_ = killProcessGroup(proc.Cmd, syscall.SIGKILL)
				}
			}
		}
	})
}
