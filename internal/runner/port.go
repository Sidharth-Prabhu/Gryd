package runner

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// IsPortInUse tests whether a TCP port is currently open and listening.
func IsPortInUse(port int) bool {
	if port <= 0 {
		return false
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 200*time.Millisecond)
	if err == nil {
		conn.Close()
		return true
	}
	return false
}

// FindPIDsListeningOnPort uses lsof to find processes listening on a port.
func FindPIDsListeningOnPort(port int) []int {
	if port <= 0 {
		return nil
	}

	cmd := exec.Command("lsof", "-t", "-i", fmt.Sprintf(":%d", port), "-sTCP:LISTEN")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var pids []int
	for _, line := range lines {
		if pid, err := strconv.Atoi(strings.TrimSpace(line)); err == nil && pid > 0 {
			pids = append(pids, pid)
		}
	}
	return pids
}

// FreePort attempts to terminate any lingering processes listening on the given port.
func FreePort(port int) ([]int, error) {
	if port <= 0 {
		return nil, nil
	}

	pids := FindPIDsListeningOnPort(port)
	if len(pids) == 0 {
		return nil, nil
	}

	// Try graceful termination first
	for _, pid := range pids {
		_ = terminatePID(pid)
	}

	time.Sleep(300 * time.Millisecond)

	// Follow up with SIGKILL/force kill if still holding the port
	remainingPids := FindPIDsListeningOnPort(port)
	for _, pid := range remainingPids {
		_ = killPID(pid)
	}

	return pids, nil
}
