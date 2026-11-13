package tunnel

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sync"
	"syscall"
	"time"
)

var (
	reCloudflareURL = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)
	reNgrokURL      = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.ngrok-free\.app|https://[a-zA-Z0-9-]+\.ngrok\.io|https://[a-zA-Z0-9-]+\.ngrok\.app`)
	reLocaltunnel   = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.loca\.lt`)
)

// ActiveTunnel represents a running tunnel instance.
type ActiveTunnel struct {
	ServiceName string
	Port        int
	Provider    string
	PublicURL   string
	Cmd         *exec.Cmd
	cancel      context.CancelFunc
	mu          sync.Mutex
}

// Manager manages running tunnels for a Gryd session.
type Manager struct {
	tunnels map[string]*ActiveTunnel
	mu      sync.Mutex
}

var globalManager = &Manager{
	tunnels: make(map[string]*ActiveTunnel),
}

// GetGlobalManager returns the singleton tunnel manager.
func GetGlobalManager() *Manager {
	return globalManager
}

// DetectProvider checks which tunneling CLI is installed on the host.
func DetectProvider(preferred string) (string, error) {
	if preferred != "" {
		if _, err := exec.LookPath(preferred); err == nil {
			return preferred, nil
		}
		if preferred == "localtunnel" {
			if _, err := exec.LookPath("lt"); err == nil {
				return "lt", nil
			}
		}
		return "", fmt.Errorf("requested tunnel provider '%s' is not installed or not in PATH", preferred)
	}

	// Priority order: cloudflared -> ngrok -> localtunnel (lt)
	if _, err := exec.LookPath("cloudflared"); err == nil {
		return "cloudflared", nil
	}
	if _, err := exec.LookPath("ngrok"); err == nil {
		return "ngrok", nil
	}
	if _, err := exec.LookPath("lt"); err == nil {
		return "lt", nil
	}

	return "", fmt.Errorf("no tunneling provider found. Please install cloudflared (`brew install cloudflared`) or ngrok (`brew install ngrok`)")
}

// StartTunnel launches a public tunnel for the specified service and port.
func (m *Manager) StartTunnel(ctx context.Context, serviceName string, port int, preferredProvider string) (*ActiveTunnel, error) {
	m.mu.Lock()
	if existing, ok := m.tunnels[serviceName]; ok && existing != nil {
		m.mu.Unlock()
		return existing, nil
	}
	m.mu.Unlock()

	provider, err := DetectProvider(preferredProvider)
	if err != nil {
		return nil, err
	}

	tCtx, cancel := context.WithCancel(ctx)
	var cmd *exec.Cmd
	var urlRegex *regexp.Regexp

	switch provider {
	case "cloudflared":
		cmd = exec.CommandContext(tCtx, "cloudflared", "tunnel", "--url", fmt.Sprintf("http://localhost:%d", port))
		urlRegex = reCloudflareURL

	case "ngrok":
		cmd = exec.CommandContext(tCtx, "ngrok", "http", fmt.Sprintf("%d", port), "--log=stdout")
		urlRegex = reNgrokURL

	case "lt", "localtunnel":
		cmd = exec.CommandContext(tCtx, "lt", "--port", fmt.Sprintf("%d", port))
		urlRegex = reLocaltunnel

	default:
		cancel()
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	// Process group isolation
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start %s tunnel: %w", provider, err)
	}

	urlChan := make(chan string, 1)
	scanOutput := func(scanner *bufio.Scanner) {
		for scanner.Scan() {
			line := scanner.Text()
			if match := urlRegex.FindString(line); match != "" {
				select {
				case urlChan <- match:
				default:
				}
			}
		}
	}

	go scanOutput(bufio.NewScanner(stdout))
	go scanOutput(bufio.NewScanner(stderr))

	var publicURL string
	select {
	case u := <-urlChan:
		publicURL = u
	case <-time.After(12 * time.Second):
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		cancel()
		return nil, fmt.Errorf("timeout waiting for %s to generate public tunnel URL", provider)
	}

	tun := &ActiveTunnel{
		ServiceName: serviceName,
		Port:        port,
		Provider:    provider,
		PublicURL:   publicURL,
		Cmd:         cmd,
		cancel:      cancel,
	}

	m.mu.Lock()
	m.tunnels[serviceName] = tun
	m.mu.Unlock()

	return tun, nil
}

// StopTunnel stops and cleans up an active tunnel for a service.
func (m *Manager) StopTunnel(serviceName string) error {
	m.mu.Lock()
	tun, ok := m.tunnels[serviceName]
	if !ok || tun == nil {
		m.mu.Unlock()
		return nil
	}
	delete(m.tunnels, serviceName)
	m.mu.Unlock()

	tun.cancel()
	if tun.Cmd != nil && tun.Cmd.Process != nil {
		_ = syscall.Kill(-tun.Cmd.Process.Pid, syscall.SIGTERM)
		time.Sleep(100 * time.Millisecond)
		_ = syscall.Kill(-tun.Cmd.Process.Pid, syscall.SIGKILL)
	}

	return nil
}

// GetTunnel returns an active tunnel for a service if present.
func (m *Manager) GetTunnel(serviceName string) *ActiveTunnel {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tunnels[serviceName]
}

// ListTunnels returns all active tunnels.
func (m *Manager) ListTunnels() map[string]*ActiveTunnel {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make(map[string]*ActiveTunnel)
	for k, v := range m.tunnels {
		res[k] = v
	}
	return res
}
