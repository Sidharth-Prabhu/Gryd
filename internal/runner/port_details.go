package runner

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ProcessPortDetail holds information about a process using a port.
type ProcessPortDetail struct {
	PID     int
	Command string
}

// GetProcessDetailsForPort returns details for processes listening on a port.
func GetProcessDetailsForPort(port int) []ProcessPortDetail {
	if port <= 0 {
		return nil
	}

	pids := FindPIDsListeningOnPort(port)
	if len(pids) == 0 {
		return nil
	}

	var results []ProcessPortDetail
	for _, pid := range pids {
		cmdName := getProcessCommand(pid)
		results = append(results, ProcessPortDetail{
			PID:     pid,
			Command: cmdName,
		})
	}
	return results
}

func getProcessCommand(pid int) string {
	// Query command name using ps
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=")
	out, err := cmd.Output()
	if err == nil {
		comm := strings.TrimSpace(string(out))
		if comm != "" {
			return comm
		}
	}
	return fmt.Sprintf("PID %d", pid)
}
