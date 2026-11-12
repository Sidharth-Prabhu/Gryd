//go:build windows

package runner

import (
	"os/exec"
	"strconv"
	"syscall"
)

func setProcessGroup(cmd *exec.Cmd) {
	// Handled natively or via job objects if necessary in future
}

func killProcessGroup(cmd *exec.Cmd, sig syscall.Signal) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	// On Windows, use taskkill /F /T /PID to terminate the process and all child processes
	killCmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid))
	return killCmd.Run()
}

func shellCommand(command string) *exec.Cmd {
	return exec.Command("cmd.exe", "/C", command)
}
