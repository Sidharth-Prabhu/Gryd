package runner

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/loom-cli/gryd/internal/config"
	"go.uber.org/zap"
)

// ContainerRuntime detects whether docker or podman is available and running.
type ContainerRuntime struct {
	Binary string // "docker" or "podman"
}

// DetectContainerRuntime checks for docker or podman binaries and verifies daemon connectivity.
func DetectContainerRuntime() (*ContainerRuntime, error) {
	for _, bin := range []string{"docker", "podman"} {
		path, err := exec.LookPath(bin)
		if err == nil && path != "" {
			// Verify daemon is responsive with a quick ping
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			cmd := exec.CommandContext(ctx, bin, "info")
			err := cmd.Run()
			cancel()
			if err == nil {
				return &ContainerRuntime{Binary: bin}, nil
			}
		}
	}
	return nil, fmt.Errorf("no container runtime (docker or podman) is installed or responsive")
}

// SidecarManager supervises containerized dependencies defined in gryd.yaml.
type SidecarManager struct {
	projectName string
	runtime     *ContainerRuntime
	deps        map[string]config.Dependency
	log         *zap.Logger
}

// NewSidecarManager creates a new SidecarManager.
func NewSidecarManager(projectName string, deps map[string]config.Dependency, log *zap.Logger) *SidecarManager {
	return &SidecarManager{
		projectName: projectName,
		deps:        deps,
		log:         log,
	}
}

// ContainerName returns the canonical container name for a dependency sidecar.
func (sm *SidecarManager) ContainerName(depName string) string {
	cleanProj := strings.ToLower(strings.ReplaceAll(sm.projectName, " ", "-"))
	return fmt.Sprintf("gryd-%s-%s", cleanProj, depName)
}

// StartAll spins up all declared sidecars and waits for their ports to be ready.
func (sm *SidecarManager) StartAll(ctx context.Context) error {
	if len(sm.deps) == 0 {
		return nil
	}

	runtime, err := DetectContainerRuntime()
	if err != nil {
		return fmt.Errorf("cannot start dependency sidecars: %w", err)
	}
	sm.runtime = runtime

	for depName, dep := range sm.deps {
		containerName := sm.ContainerName(depName)

		// Check if container already exists
		inspectCmd := exec.Command(sm.runtime.Binary, "inspect", "-f", "{{.State.Running}}", containerName)
		out, err := inspectCmd.Output()
		isRunning := strings.TrimSpace(string(out)) == "true"

		if err == nil && isRunning {
			if sm.log != nil {
				sm.log.Info("Sidecar container already running", zap.String("dependency", depName), zap.String("container", containerName))
			}
			continue
		}

		if err == nil && !isRunning {
			// Existing container stopped, start it
			startCmd := exec.Command(sm.runtime.Binary, "start", containerName)
			if startOut, err := startCmd.CombinedOutput(); err != nil {
				// If start fails, remove and recreate
				_ = exec.Command(sm.runtime.Binary, "rm", "-f", containerName).Run()
			} else {
				_ = startOut
				continue
			}
		}

		// Run new container
		args := []string{"run", "-d", "--name", containerName}

		// Add ports
		for _, p := range dep.Ports {
			args = append(args, "-p", p)
		}

		// Add volumes
		for _, v := range dep.Volumes {
			args = append(args, "-v", v)
		}

		// Add environment variables
		for k, v := range dep.Env {
			args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
		}

		// Image
		args = append(args, dep.Image)

		// Command if any
		if dep.Command != "" {
			args = append(args, strings.Fields(dep.Command)...)
		}

		runCmd := exec.CommandContext(ctx, sm.runtime.Binary, args...)
		if runOut, err := runCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to start sidecar '%s' (%s): %s (%w)",
				depName, dep.Image, strings.TrimSpace(string(runOut)), err)
		}

		if sm.log != nil {
			sm.log.Info("Started sidecar container", zap.String("dependency", depName), zap.String("image", dep.Image))
		}
	}

	// Wait for published host ports to become active
	for depName, dep := range sm.deps {
		for _, portMap := range dep.Ports {
			hostPort := extractHostPort(portMap)
			if hostPort > 0 {
				_ = waitForPort(hostPort, 5*time.Second)
			}
		}
		_ = depName
	}

	return nil
}

// StopAll terminates and removes the sidecar containers.
func (sm *SidecarManager) StopAll() {
	if sm.runtime == nil || len(sm.deps) == 0 {
		return
	}

	for depName := range sm.deps {
		containerName := sm.ContainerName(depName)
		cmd := exec.Command(sm.runtime.Binary, "rm", "-f", containerName)
		_ = cmd.Run()
		if sm.log != nil {
			sm.log.Info("Stopped and removed sidecar container", zap.String("dependency", depName), zap.String("container", containerName))
		}
	}
}

// extractHostPort extracts the host port integer from a "host:container", "ip:host:container", or "port" mapping.
func extractHostPort(portMapping string) int {
	parts := strings.Split(portMapping, ":")
	if len(parts) >= 3 {
		// ip:hostPort:containerPort
		if p, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
			return p
		}
	} else if len(parts) == 2 {
		// hostPort:containerPort
		if p, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
			return p
		}
	} else if len(parts) == 1 {
		if p, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
			return p
		}
	}
	return 0
}

func waitForPort(port int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if IsPortInUse(port) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
