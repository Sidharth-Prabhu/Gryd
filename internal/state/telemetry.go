package state

import (
	"fmt"
	"syscall"

	"github.com/shirou/gopsutil/v3/process"
)

// ProcessTelemetry holds real-time CPU, memory, and process tree info.
type ProcessTelemetry struct {
	PID        int32
	CPUPercent float64
	MemoryRSS  uint64 // in bytes
	NumFDs     int32
	NumThreads int32
	Children   []ChildProcessInfo
}

// ChildProcessInfo represents details about a child process.
type ChildProcessInfo struct {
	PID        int32
	Name       string
	CPUPercent float64
	MemoryRSS  uint64
	Command    string
}

// GetProcessTelemetry fetches resource metrics for a process and all its children.
func GetProcessTelemetry(pid int) (*ProcessTelemetry, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("invalid pid: %d", pid)
	}

	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, err
	}

	// Verify process is actually alive via signal 0
	if err := syscall.Kill(pid, 0); err != nil {
		return nil, fmt.Errorf("process %d is not running: %w", pid, err)
	}

	cpuPercent, _ := p.CPUPercent()
	memInfo, _ := p.MemoryInfo()
	var rss uint64
	if memInfo != nil {
		rss = memInfo.RSS
	}

	numFDs, _ := p.NumFDs()
	numThreads, _ := p.NumThreads()

	telemetry := &ProcessTelemetry{
		PID:        int32(pid),
		CPUPercent: cpuPercent,
		MemoryRSS:  rss,
		NumFDs:     numFDs,
		NumThreads: numThreads,
		Children:   nil,
	}

	// Recursively collect children
	if children, err := p.Children(); err == nil {
		for _, child := range children {
			childPID := child.Pid
			childCPU, _ := child.CPUPercent()
			childMem, _ := child.MemoryInfo()
			var childRSS uint64
			if childMem != nil {
				childRSS = childMem.RSS
			}
			childName, _ := child.Name()
			childCmd, _ := child.Cmdline()

			// Add child usage to aggregate
			telemetry.CPUPercent += childCPU
			telemetry.MemoryRSS += childRSS

			telemetry.Children = append(telemetry.Children, ChildProcessInfo{
				PID:        childPID,
				Name:       childName,
				CPUPercent: childCPU,
				MemoryRSS:  childRSS,
				Command:    childCmd,
			})
		}
	}

	return telemetry, nil
}

// FormatBytes formats bytes into human-readable strings (KB, MB, GB).
func FormatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "KMGTPE"[exp])
}
