package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ProjectRegistryEntry stores known project locations.
type ProjectRegistryEntry struct {
	Name       string    `json:"name"`
	ConfigPath string    `json:"config_path"`
	WorkingDir string    `json:"working_dir"`
	LastSeen   time.Time `json:"last_seen"`
}

func getRegistryPath() (string, error) {
	grydDir, err := GetGrydDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(grydDir, "projects.json"), nil
}

// RegisterProject records or updates a project's location in the global registry.
func RegisterProject(name, configPath, workingDir string) error {
	regPath, err := getRegistryPath()
	if err != nil {
		return err
	}

	registry := LoadProjectRegistry()
	registry[strings.ToLower(name)] = ProjectRegistryEntry{
		Name:       name,
		ConfigPath: configPath,
		WorkingDir: workingDir,
		LastSeen:   time.Now(),
	}

	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(regPath, data, 0644)
}

// LoadProjectRegistry reads all known projects.
func LoadProjectRegistry() map[string]ProjectRegistryEntry {
	registry := make(map[string]ProjectRegistryEntry)
	regPath, err := getRegistryPath()
	if err != nil {
		return registry
	}

	data, err := os.ReadFile(regPath)
	if err != nil {
		return registry
	}

	_ = json.Unmarshal(data, &registry)
	return registry
}

// FindProjectInRegistry looks up a project path by name in the registry.
func FindProjectInRegistry(name string) (ProjectRegistryEntry, bool) {
	registry := LoadProjectRegistry()
	entry, ok := registry[strings.ToLower(name)]
	return entry, ok
}
