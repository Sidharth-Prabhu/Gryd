//go:build !windows

package runner

import (
	"os/exec"
	"syscall"
)

func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killProcessGroup(cmd *exec.Cmd, sig syscall.Signal) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	// Kill the negative PID to terminate the entire process group
	return syscall.Kill(-cmd.Process.Pid, sig)
}

func shellCommand(command string) *exec.Cmd {
	return exec.Command("sh", "-c", command)
}
