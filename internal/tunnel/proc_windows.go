//go:build windows

package tunnel

import (
	"os/exec"
	"strconv"
)

func setSysProcAttr(cmd *exec.Cmd) {
	// No Setpgid on Windows
}

func killCmd(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		killCmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid))
		_ = killCmd.Run()
	}
}

func terminateCmd(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		killCmd := exec.Command("taskkill", "/PID", strconv.Itoa(cmd.Process.Pid))
		_ = killCmd.Run()
	}
}
