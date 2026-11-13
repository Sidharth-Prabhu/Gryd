//go:build windows

package runner

import (
	"os/exec"
	"strconv"
)

func terminatePID(pid int) error {
	killCmd := exec.Command("taskkill", "/PID", strconv.Itoa(pid))
	return killCmd.Run()
}

func killPID(pid int) error {
	killCmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
	return killCmd.Run()
}
