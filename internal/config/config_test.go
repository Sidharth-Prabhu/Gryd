package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	tempDir := t.TempDir()
	yamlContent := `
name: TestProject
working_directory: .
services:
  api:
    directory: api
    command: echo "api starting"
    port: 8080
  web:
    directory: web
    command: echo "web starting"
    port: 3000
    depends_on:
      - api
settings:
  health_check: true
  restart_on_failure: false
`
	configPath := filepath.Join(tempDir, "gryd.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}

	if cfg.Name != "TestProject" {
		t.Errorf("expected name TestProject, got %s", cfg.Name)
	}
	if len(cfg.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(cfg.Services))
	}
	if cfg.Services["api"].Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Services["api"].Port)
	}
}

func TestValidationErrors(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		content string
	}{
		{
			name: "missing project name",
			content: `
services:
  web:
    command: echo web
`,
		},
		{
			name: "missing services",
			content: `
name: NoServices
services: {}
`,
		},
		{
			name: "self dependency",
			content: `
name: SelfDep
services:
  web:
    command: echo web
    depends_on:
      - web
`,
		},
		{
			name: "unknown dependency",
			content: `
name: UnknownDep
services:
  web:
    command: echo web
    depends_on:
      - db
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tempDir, tt.name+".yaml")
			_ = os.WriteFile(path, []byte(tt.content), 0644)
			_, err := Load(path)
			if err == nil {
				t.Fatalf("expected error for case '%s', got nil", tt.name)
			}
		})
	}
}
