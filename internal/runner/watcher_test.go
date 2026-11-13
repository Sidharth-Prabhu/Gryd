package runner

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/loom-cli/gryd/internal/config"
)

func TestFileWatcherDebounce(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gryd-watch-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	subDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	cfg := &config.Config{
		Name:             "test-watch",
		WorkingDirectory: tempDir,
		Services: map[string]config.Service{
			"api": {
				Directory: "src",
				Command:   "echo api",
				Watch:     []string{"src/**"},
				Ignore:    []string{"*.log"},
			},
		},
	}

	var restarts int32
	fw, err := NewFileWatcher(cfg, nil, func(serviceName string) error {
		if serviceName == "api" {
			atomic.AddInt32(&restarts, 1)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to create file watcher: %v", err)
	}
	defer fw.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := fw.Start(ctx); err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}

	// Wait for watcher to register paths
	time.Sleep(100 * time.Millisecond)

	// Write a file in src
	testFile := filepath.Join(subDir, "main.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Wait for debounce (300ms + buffer)
	time.Sleep(600 * time.Millisecond)

	count := atomic.LoadInt32(&restarts)
	if count < 1 {
		t.Errorf("expected at least 1 restart triggered by file modification, got %d", count)
	}

	// Write an ignored file (.log) and verify no restart is triggered
	ignoredFile := filepath.Join(subDir, "app.log")
	if err := os.WriteFile(ignoredFile, []byte("some log"), 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	time.Sleep(500 * time.Millisecond)
	if atomic.LoadInt32(&restarts) > count {
		t.Errorf("ignored file modification should not trigger restart")
	}
}
