package config

import (
	"fmt"
	"path/filepath"
)

// ApplyProfile transforms the configuration based on the requested profile name.
// If profile is empty, cascading .env files are still merged into services.
func (c *Config) ApplyProfile(profile string) error {
	c.ActiveProfile = profile

	// 1. Load cascading environment files from working directory
	cascadingEnvs := LoadCascadingEnvFiles(c.WorkingDirectory, profile)

	// 2. Check if a profile definition exists in gryd.yaml
	var profileCfg *ProfileConfig
	if profile != "" {
		if p, ok := c.Profiles[profile]; ok {
			profileCfg = &p
		} else {
			return fmt.Errorf("profile '%s' not found in configuration", profile)
		}
	}

	// 3. Load explicit env_file from profile if defined
	if profileCfg != nil && profileCfg.EnvFile != "" {
		envFilePath := profileCfg.EnvFile
		if !filepath.IsAbs(envFilePath) {
			envFilePath = filepath.Join(c.WorkingDirectory, envFilePath)
		}
		parseEnvFile(envFilePath, cascadingEnvs)
	}

	// 4. Filter services if profile specifies a subset
	if profileCfg != nil && len(profileCfg.Services) > 0 {
		allowed := make(map[string]bool)
		for _, s := range profileCfg.Services {
			allowed[s] = true
		}

		filteredServices := make(map[string]Service)
		for sName, s := range c.Services {
			if allowed[sName] {
				filteredServices[sName] = s
			}
		}
		c.Services = filteredServices
	}

	// 5. Filter dependencies if profile specifies a subset
	if profileCfg != nil && len(profileCfg.Dependencies) > 0 {
		allowedDeps := make(map[string]bool)
		for _, d := range profileCfg.Dependencies {
			allowedDeps[d] = true
		}

		filteredDeps := make(map[string]Dependency)
		for dName, d := range c.Dependencies {
			if allowedDeps[dName] {
				filteredDeps[dName] = d
			}
		}
		c.Dependencies = filteredDeps
	}

	// 6. Merge environment variables into each active service
	for sName, svc := range c.Services {
		if svc.Env == nil {
			svc.Env = make(map[string]string)
		}
		// Cascading .env variables
		for k, v := range cascadingEnvs {
			if _, exists := svc.Env[k]; !exists {
				svc.Env[k] = v
			}
		}
		// Profile explicit env overrides (highest precedence)
		if profileCfg != nil {
			for k, v := range profileCfg.Env {
				svc.Env[k] = v
			}
		}
		c.Services[sName] = svc
	}

	// Also merge into sidecar dependencies
	for dName, dep := range c.Dependencies {
		if dep.Env == nil {
			dep.Env = make(map[string]string)
		}
		for k, v := range cascadingEnvs {
			if _, exists := dep.Env[k]; !exists {
				dep.Env[k] = v
			}
		}
		if profileCfg != nil {
			for k, v := range profileCfg.Env {
				dep.Env[k] = v
			}
		}
		c.Dependencies[dName] = dep
	}

	return nil
}
