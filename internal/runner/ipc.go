package runner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/state"
	"go.uber.org/zap"
)

// IPCRequest represents a command message sent over the Unix socket.
type IPCRequest struct {
	Action  string   `json:"action"` // "restart", "stop", "exec"
	Service string   `json:"service"`
	Command []string `json:"command,omitempty"`
}

// IPCResponse represents the response returned over the Unix socket.
type IPCResponse struct {
	Success bool   `json:"success"`
	Output  string `json:"output,omitempty"`
	Error   string `json:"error,omitempty"`
}

// GetSocketPath returns the standard Unix domain socket path for a project.
func GetSocketPath(projectName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	sockDir := filepath.Join(home, ".gryd", "run")
	_ = os.MkdirAll(sockDir, 0755)
	return filepath.Join(sockDir, fmt.Sprintf("%s.sock", projectName)), nil
}

// IPCServer listens on a Unix domain socket and handles incoming remote control requests.
type IPCServer struct {
	projectName string
	sockPath    string
	listener    net.Listener
	runner      *Runner
	log         *zap.Logger
	mu          sync.Mutex
	closed      bool
}

// NewIPCServer initializes a new IPC listener for the runner.
func NewIPCServer(projectName string, r *Runner, log *zap.Logger) (*IPCServer, error) {
	sockPath, err := GetSocketPath(projectName)
	if err != nil {
		return nil, err
	}

	// Remove stale socket if present
	_ = os.Remove(sockPath)

	l, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on socket %s: %w", sockPath, err)
	}

	srv := &IPCServer{
		projectName: projectName,
		sockPath:    sockPath,
		listener:    l,
		runner:      r,
		log:         log,
	}

	return srv, nil
}

// Start begins accepting IPC requests in a background goroutine.
func (s *IPCServer) Start() {
	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				s.mu.Lock()
				closed := s.closed
				s.mu.Unlock()
				if closed {
					return
				}
				continue
			}
			go s.handleConnection(conn)
		}
	}()
}

// Close stops the IPC server and removes the socket file.
func (s *IPCServer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	var err error
	if s.listener != nil {
		err = s.listener.Close()
	}
	_ = os.Remove(s.sockPath)
	return err
}

func (s *IPCServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return
	}

	var req IPCRequest
	if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
		sendIPCResponse(conn, IPCResponse{Success: false, Error: "invalid JSON request"})
		return
	}

	resp := s.executeRequest(req)
	sendIPCResponse(conn, resp)
}

func (s *IPCServer) executeRequest(req IPCRequest) IPCResponse {
	switch req.Action {
	case "restart":
		if req.Service == "" {
			return IPCResponse{Success: false, Error: "service name required"}
		}
		if s.runner != nil {
			err := s.runner.RestartService(req.Service)
			if err != nil {
				return IPCResponse{Success: false, Error: err.Error()}
			}
			return IPCResponse{Success: true, Output: fmt.Sprintf("Service '%s' restarted", req.Service)}
		}
		// Fallback to standalone single service restart
		err := RestartSingleService(s.projectName, req.Service)
		if err != nil {
			return IPCResponse{Success: false, Error: err.Error()}
		}
		return IPCResponse{Success: true, Output: fmt.Sprintf("Service '%s' restarted", req.Service)}

	case "stop":
		if req.Service == "" {
			return IPCResponse{Success: false, Error: "service name required"}
		}
		if s.runner != nil {
			s.runner.mu.Lock()
			proc, exists := s.runner.processes[req.Service]
			if exists && proc.Cmd != nil && proc.Cmd.Process != nil {
				_ = killProcessGroup(proc.Cmd, syscall.SIGKILL)
				delete(s.runner.processes, req.Service)
			}
			s.runner.mu.Unlock()
			return IPCResponse{Success: true, Output: fmt.Sprintf("Service '%s' stopped", req.Service)}
		}
		// Standalone stop
		st, err := state.LoadState(s.projectName)
		if err != nil {
			return IPCResponse{Success: false, Error: err.Error()}
		}
		if proc, exists := st.Processes[req.Service]; exists && proc.PID > 0 {
			_ = StopSinglePID(proc.PID)
			delete(st.Processes, req.Service)
			_ = state.SaveState(st)
			return IPCResponse{Success: true, Output: fmt.Sprintf("Service '%s' stopped", req.Service)}
		}
		return IPCResponse{Success: false, Error: fmt.Sprintf("service '%s' is not running", req.Service)}

	case "exec":
		if req.Service == "" {
			return IPCResponse{Success: false, Error: "service name required"}
		}
		if len(req.Command) == 0 {
			return IPCResponse{Success: false, Error: "command required"}
		}

		st, _ := state.LoadState(s.projectName)
		var workDir string
		if st != nil && st.ConfigPath != "" {
			if cfg, err := config.Load(st.ConfigPath); err == nil {
				if svc, ok := cfg.Services[req.Service]; ok {
					if svc.Directory != "" {
						if filepath.IsAbs(svc.Directory) {
							workDir = svc.Directory
						} else {
							workDir = filepath.Join(cfg.WorkingDirectory, svc.Directory)
						}
					} else {
						workDir = cfg.WorkingDirectory
					}
				}
			}
		}

		cmd := exec.Command(req.Command[0], req.Command[1:]...)
		if workDir != "" {
			cmd.Dir = workDir
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return IPCResponse{Success: false, Output: string(out), Error: err.Error()}
		}
		return IPCResponse{Success: true, Output: string(out)}

	default:
		return IPCResponse{Success: false, Error: fmt.Sprintf("unknown action: %s", req.Action)}
	}
}

func sendIPCResponse(conn net.Conn, resp IPCResponse) {
	data, _ := json.Marshal(resp)
	data = append(data, '\n')
	_, _ = conn.Write(data)
}

// SendIPCCommand sends an IPC request to a running Gryd project daemon.
func SendIPCCommand(projectName string, req IPCRequest) (*IPCResponse, error) {
	sockPath, err := GetSocketPath(projectName)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTimeout("unix", sockPath, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("could not connect to project '%s' socket (is it running?): %w", projectName, err)
	}
	defer conn.Close()

	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	reqData = append(reqData, '\n')

	if _, err := conn.Write(reqData); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty response from daemon")
	}

	var resp IPCResponse
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse daemon response: %w", err)
	}

	return &resp, nil
}

// StopSinglePID kills a single process and its group.
func StopSinglePID(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	_ = p.Signal(os.Interrupt)
	time.Sleep(100 * time.Millisecond)
	_ = p.Kill()
	return nil
}
