package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProfileApplication(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gryd-profile-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a cascading .env file
	envFile := filepath.Join(tempDir, ".env")
	_ = os.WriteFile(envFile, []byte("GLOBAL_VAR=default_val\nPORT=9000"), 0644)

	// Create profile-specific .env file
	prodEnvFile := filepath.Join(tempDir, ".env.production")
	_ = os.WriteFile(prodEnvFile, []byte("GLOBAL_VAR=prod_val\nPROD_SECRET=xyz"), 0644)

	cfg := &Config{
		Name:             "profile-test",
		WorkingDirectory: tempDir,
		Services: map[string]Service{
			"api": {
				Command: "run-api",
				Env:     map[string]string{"SERVICE_SPECIFIC": "1"},
			},
			"worker": {
				Command: "run-worker",
			},
			"frontend": {
				Command: "run-web",
			},
		},
		Profiles: map[string]ProfileConfig{
			"minimal": {
				Services: []string{"api"},
				Env: map[string]string{
					"MINIMAL_MODE": "true",
				},
			},
			"production": {
				Services: []string{"api", "worker", "frontend"},
			},
		},
	}

	// Test 1: Apply minimal profile
	if err := cfg.ApplyProfile("minimal"); err != nil {
		t.Fatalf("failed to apply minimal profile: %v", err)
	}

	if len(cfg.Services) != 1 {
		t.Errorf("expected 1 service in minimal profile, got %d", len(cfg.Services))
	}

	apiSvc, ok := cfg.Services["api"]
	if !ok {
		t.Fatalf("expected 'api' service to be present in minimal profile")
	}

	if apiSvc.Env["MINIMAL_MODE"] != "true" {
		t.Errorf("expected MINIMAL_MODE=true override, got %s", apiSvc.Env["MINIMAL_MODE"])
	}

	if apiSvc.Env["GLOBAL_VAR"] != "default_val" {
		t.Errorf("expected GLOBAL_VAR=default_val from cascading .env, got %s", apiSvc.Env["GLOBAL_VAR"])
	}

	// Test 2: Apply production profile with cascading .env.production
	cfgProd := &Config{
		Name:             "profile-test-prod",
		WorkingDirectory: tempDir,
		Services: map[string]Service{
			"api":    {Command: "run-api"},
			"worker": {Command: "run-worker"},
		},
		Profiles: map[string]ProfileConfig{
			"production": {
				Services: []string{"api"},
			},
		},
	}

	if err := cfgProd.ApplyProfile("production"); err != nil {
		t.Fatalf("failed to apply production profile: %v", err)
	}

	apiProd := cfgProd.Services["api"]
	if apiProd.Env["GLOBAL_VAR"] != "prod_val" {
		t.Errorf("expected GLOBAL_VAR to be overridden to prod_val by .env.production, got %s", apiProd.Env["GLOBAL_VAR"])
	}
	if apiProd.Env["PROD_SECRET"] != "xyz" {
		t.Errorf("expected PROD_SECRET to be xyz from .env.production, got %s", apiProd.Env["PROD_SECRET"])
	}
}
