package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// ProcessInfo holds information about a running service process.
type ProcessInfo struct {
	Name    string `json:"name"`
	PID     int    `json:"pid"`
	Command string `json:"command"`
}

// ProjectState stores the active running state of a project.
type ProjectState struct {
	ProjectName string                 `json:"project_name"`
	ConfigPath  string                 `json:"config_path"`
	LogFilePath string                 `json:"log_file_path"`
	StartedAt   time.Time              `json:"started_at"`
	Processes   map[string]ProcessInfo `json:"processes"`
}

// GetGrydDir returns ~/.gryd directory, ensuring it exists.
func GetGrydDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	grydDir := filepath.Join(home, ".gryd")
	if err := os.MkdirAll(filepath.Join(grydDir, "run"), 0755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(grydDir, "logs"), 0755); err != nil {
		return "", err
	}
	return grydDir, nil
}

// GetLogFilePath returns the log file path for a project.
func GetLogFilePath(projectName string) (string, error) {
	grydDir, err := GetGrydDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(grydDir, "logs", projectName+".log"), nil
}

// SaveState records the current project state to ~/.gryd/run/<project>.json
func SaveState(st *ProjectState) error {
	grydDir, err := GetGrydDir()
	if err != nil {
		return err
	}
	statePath := filepath.Join(grydDir, "run", st.ProjectName+".json")
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath, data, 0644)
}

// LoadState reads project state from ~/.gryd/run/<project>.json
func LoadState(projectName string) (*ProjectState, error) {
	grydDir, err := GetGrydDir()
	if err != nil {
		return nil, err
	}
	statePath := filepath.Join(grydDir, "run", projectName+".json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("project '%s' is not currently running (no state found at %s)", projectName, statePath)
	}
	var st ProjectState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

// RemoveState deletes the project state file.
func RemoveState(projectName string) error {
	grydDir, err := GetGrydDir()
	if err != nil {
		return err
	}
	statePath := filepath.Join(grydDir, "run", projectName+".json")
	_ = os.Remove(statePath)
	return nil
}

// IsProcessAlive checks if a process with the given PID is still running.
func IsProcessAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, Signal(0) tests whether process exists without killing it
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// ListRunningProjects returns all active projects saved in ~/.gryd/run/
func ListRunningProjects() ([]*ProjectState, error) {
	grydDir, err := GetGrydDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Join(grydDir, "run"))
	if err != nil {
		return nil, err
	}

	var projects []*ProjectState
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			name := entry.Name()[:len(entry.Name())-5]
			st, err := LoadState(name)
			if err == nil {
				// Clean up if all processes are dead
				anyAlive := false
				for _, proc := range st.Processes {
					if IsProcessAlive(proc.PID) {
						anyAlive = true
						break
					}
				}
				if anyAlive {
					projects = append(projects, st)
				} else {
					_ = RemoveState(name)
				}
			}
		}
	}
	return projects, nil
}
