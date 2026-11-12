package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Service defines configuration for an individual service/process.
type Service struct {
	Directory string            `yaml:"directory"`
	Command   string            `yaml:"command"`
	Port      int               `yaml:"port,omitempty"`
	DependsOn []string          `yaml:"depends_on,omitempty"`
	Env       map[string]string `yaml:"env,omitempty"`
}

// Settings contains global execution options for the project.
type Settings struct {
	HealthCheck      bool `yaml:"health_check"`
	RestartOnFailure bool `yaml:"restart_on_failure"`
}

// Config represents the complete gryd.yaml (or legacy loom.yaml) configuration file.
type Config struct {
	Name             string             `yaml:"name"`
	WorkingDirectory string             `yaml:"working_directory"`
	Services         map[string]Service `yaml:"services"`
	Settings         Settings           `yaml:"settings"`

	// ConfigPath is the absolute path to the file this config was loaded from.
	ConfigPath string `yaml:"-"`
}

// Load reads and parses a gryd.yaml configuration file from the specified path.
func Load(path string) (*Config, error) {
	resolvedPath, err := resolveFilePath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config path: %w", err)
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", resolvedPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML configuration: %w", err)
	}

	cfg.ConfigPath = resolvedPath

	if err := cfg.ValidateAndResolve(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// ValidateAndResolve validates the config values and resolves relative and home paths.
func (c *Config) ValidateAndResolve() error {
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("project 'name' is required")
	}

	if len(c.Services) == 0 {
		return errors.New("at least one service must be defined under 'services'")
	}

	// Resolve working directory
	if c.WorkingDirectory == "" {
		// Default to the directory where the config file is located
		c.WorkingDirectory = filepath.Dir(c.ConfigPath)
	} else {
		resolvedWorkingDir, err := resolveFilePath(c.WorkingDirectory)
		if err != nil {
			return fmt.Errorf("invalid working_directory: %w", err)
		}
		c.WorkingDirectory = resolvedWorkingDir
	}

	// Validate each service
	for name, svc := range c.Services {
		if strings.TrimSpace(svc.Command) == "" {
			return fmt.Errorf("service '%s' must have a non-empty 'command'", name)
		}

		// Ensure dependencies exist
		for _, dep := range svc.DependsOn {
			if dep == name {
				return fmt.Errorf("service '%s' cannot depend on itself", name)
			}
			if _, exists := c.Services[dep]; !exists {
				return fmt.Errorf("service '%s' depends on unknown service '%s'", name, dep)
			}
		}
	}

	return nil
}

// FindConfigFile attempts to locate the appropriate config file based on project name or path.
func FindConfigFile(specifiedPath string, projectName string) (string, error) {
	if specifiedPath != "" {
		resolved, err := resolveFilePath(specifiedPath)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(resolved); err != nil {
			return "", fmt.Errorf("specified config file '%s' does not exist", specifiedPath)
		}
		return resolved, nil
	}

	// If projectName is provided and looks like a path or file
	if projectName != "" {
		// 1. Check if projectName is a direct path to a file
		if strings.HasSuffix(projectName, ".yaml") || strings.HasSuffix(projectName, ".yml") {
			if _, err := os.Stat(projectName); err == nil {
				return resolveFilePath(projectName)
			}
		}

		// 2. Check if projectName is a directory containing gryd.yaml or loom.yaml
		for _, fname := range []string{"gryd.yaml", "gryd.yml", "loom.yaml", "loom.yml"} {
			dirCfg := filepath.Join(projectName, fname)
			if _, err := os.Stat(dirCfg); err == nil {
				return resolveFilePath(dirCfg)
			}
		}

		// 3. Check user config directory ~/.config/gryd/<projectName>.yaml or ~/.config/loom/
		if home, err := os.UserHomeDir(); err == nil {
			for _, dirName := range []string{"gryd", "loom"} {
				userConfig := filepath.Join(home, ".config", dirName, projectName+".yaml")
				if _, err := os.Stat(userConfig); err == nil {
					return userConfig, nil
				}
			}
		}
	}

	// Default to local ./gryd.yaml, ./gryd.yml, or fallback to ./loom.yaml
	for _, candidate := range []string{"gryd.yaml", "gryd.yml", "loom.yaml", "loom.yml"} {
		if _, err := os.Stat(candidate); err == nil {
			return resolveFilePath(candidate)
		}
	}

	return "", errors.New("no gryd.yaml (or loom.yaml) found in current directory or search paths. Use --config or run 'gryd init'")
}

// resolveFilePath expands ~ home directory and cleans the path.
func resolveFilePath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, path[2:])
		}
	}

	return filepath.Abs(path)
}
