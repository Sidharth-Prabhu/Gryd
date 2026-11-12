package runner

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/loom-cli/gryd/internal/config"
	"go.uber.org/zap"
)

func TestRunnerGracefulShutdown(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		Name:             "TestRunner",
		WorkingDirectory: tempDir,
		Services: map[string]config.Service{
			"worker-1": {
				Command: "sleep 10",
			},
			"worker-2": {
				Command: "sleep 10",
			},
		},
		Settings: config.Settings{
			RestartOnFailure: false,
		},
	}

	logger, _ := zap.NewDevelopment()
	r, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	err = r.Run(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("runner returned unexpected error: %v", err)
	}

	if elapsed > 3*time.Second {
		t.Errorf("expected shutdown to finish within 3s, took %v", elapsed)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for name, proc := range r.processes {
		if proc.Cmd != nil && proc.Cmd.Process != nil {
			err := proc.Cmd.Process.Signal(os.Signal(nil))
			t.Logf("Process %s (pid %d) status check: %v", name, proc.Cmd.Process.Pid, err)
		}
	}
}
