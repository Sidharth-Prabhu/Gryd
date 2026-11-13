package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/loom-cli/gryd/internal/config"
	"go.uber.org/zap"
)

// FileWatcher watches files and directories defined in gryd.yaml and triggers
// debounced hot reloads for specific services.
type FileWatcher struct {
	watcher     *fsnotify.Watcher
	cfg         *config.Config
	log         *zap.Logger
	restartFunc func(serviceName string) error
	mu          sync.Mutex
	timers      map[string]*time.Timer
	debounce    time.Duration
}

// NewFileWatcher creates and initializes a file watcher.
func NewFileWatcher(cfg *config.Config, log *zap.Logger, restartFunc func(serviceName string) error) (*FileWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	fw := &FileWatcher{
		watcher:     w,
		cfg:         cfg,
		log:         log,
		restartFunc: restartFunc,
		timers:      make(map[string]*time.Timer),
		debounce:    300 * time.Millisecond,
	}

	return fw, nil
}

// Start begins watching registered paths.
func (fw *FileWatcher) Start(ctx context.Context) error {
	totalWatches := 0
	for svcName, svc := range fw.cfg.Services {
		if len(svc.Watch) == 0 {
			continue
		}

		baseDir := fw.cfg.WorkingDirectory
		if svc.Directory != "" {
			if filepath.IsAbs(svc.Directory) {
				baseDir = svc.Directory
			} else {
				baseDir = filepath.Join(fw.cfg.WorkingDirectory, svc.Directory)
			}
		}

		for _, pattern := range svc.Watch {
			pathsToWatch := fw.resolveWatchPaths(baseDir, pattern)
			for _, p := range pathsToWatch {
				if err := fw.addRecursive(p, svc.Ignore); err == nil {
					totalWatches++
				}
			}
		}
		_ = svcName
	}

	if totalWatches == 0 {
		return nil
	}

	go fw.watchLoop(ctx)
	return nil
}

// Close stops the watcher.
func (fw *FileWatcher) Close() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	for _, timer := range fw.timers {
		timer.Stop()
	}
	return fw.watcher.Close()
}

func (fw *FileWatcher) addRecursive(root string, ignorePatterns []string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if shouldIgnore(path, ignorePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return fw.watcher.Add(path)
		}
		return nil
	})
}

func (fw *FileWatcher) resolveWatchPaths(baseDir, pattern string) []string {
	// If pattern contains wildcards/globbing
	if strings.Contains(pattern, "*") {
		cleanPattern := filepath.Join(baseDir, pattern)
		// Strip trailing ** if any
		cleanPattern = strings.TrimSuffix(cleanPattern, "/**")
		cleanPattern = strings.TrimSuffix(cleanPattern, "/*")
		matches, err := filepath.Glob(cleanPattern)
		if err == nil && len(matches) > 0 {
			return matches
		}
		return []string{baseDir}
	}

	target := pattern
	if !filepath.IsAbs(target) {
		target = filepath.Join(baseDir, target)
	}

	if _, err := os.Stat(target); err == nil {
		return []string{target}
	}
	return []string{baseDir}
}

func shouldIgnore(path string, ignorePatterns []string) bool {
	base := filepath.Base(path)
	// Always ignore git and standard artifacts
	if base == ".git" || base == "node_modules" || base == ".idea" || base == ".vscode" || base == "vendor" {
		return true
	}

	for _, pat := range ignorePatterns {
		pat = strings.TrimPrefix(pat, "./")
		if strings.HasSuffix(pat, "/**") {
			prefix := strings.TrimSuffix(pat, "/**")
			if strings.Contains(path, prefix) {
				return true
			}
		}
		if matched, _ := filepath.Match(pat, base); matched {
			return true
		}
		if strings.Contains(path, pat) {
			return true
		}
	}
	return false
}

func (fw *FileWatcher) watchLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// Ignore chmod/permission events
			if event.Op&fsnotify.Chmod == fsnotify.Chmod {
				continue
			}

			// If a new directory was created, watch it
			if event.Op&fsnotify.Create == fsnotify.Create {
				if fi, err := os.Stat(event.Name); err == nil && fi.IsDir() {
					_ = fw.addRecursive(event.Name, nil)
				}
			}

			// Check which service this modified file corresponds to
			for svcName, svc := range fw.cfg.Services {
				if len(svc.Watch) == 0 {
					continue
				}

				if shouldIgnore(event.Name, svc.Ignore) {
					continue
				}

				baseDir := fw.cfg.WorkingDirectory
				if svc.Directory != "" {
					if filepath.IsAbs(svc.Directory) {
						baseDir = svc.Directory
					} else {
						baseDir = filepath.Join(fw.cfg.WorkingDirectory, svc.Directory)
					}
				}

				// If file is within service watch territory
				if strings.HasPrefix(event.Name, baseDir) {
					fw.debounceRestart(svcName, event.Name)
				}
			}

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			if fw.log != nil {
				fw.log.Debug("File watcher error", zap.Error(err))
			}
		}
	}
}

func (fw *FileWatcher) debounceRestart(svcName, changedFile string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if t, exists := fw.timers[svcName]; exists {
		t.Stop()
	}

	fw.timers[svcName] = time.AfterFunc(fw.debounce, func() {
		if fw.log != nil {
			fw.log.Info("File change detected; reloading service",
				zap.String("service", svcName),
				zap.String("file", filepath.Base(changedFile)),
			)
		}
		if fw.restartFunc != nil {
			_ = fw.restartFunc(svcName)
		}
	})
}
