//go:build !windows

package runner

import (
	"syscall"
)

func terminatePID(pid int) error {
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	return syscall.Kill(pid, syscall.SIGTERM)
}

func killPID(pid int) error {
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	return syscall.Kill(pid, syscall.SIGKILL)
}
