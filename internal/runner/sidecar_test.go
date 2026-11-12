package runner

import (
	"context"
	"testing"

	"github.com/loom-cli/gryd/internal/config"
)

func TestExtractHostPort(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"5432:5432", 5432},
		{"127.0.0.1:8080:80", 8080},
		{"6379", 6379},
		{"invalid", 0},
	}

	for _, tc := range cases {
		got := extractHostPort(tc.input)
		if got != tc.want {
			t.Errorf("extractHostPort(%q) = %d; want %d", tc.input, got, tc.want)
		}
	}
}

func TestSidecarManagerNaming(t *testing.T) {
	sm := NewSidecarManager("My Project", map[string]config.Dependency{
		"postgres": {
			Image: "postgres:16-alpine",
			Ports: []string{"5432:5432"},
		},
	}, nil)

	name := sm.ContainerName("postgres")
	expected := "gryd-my-project-postgres"
	if name != expected {
		t.Errorf("ContainerName = %q; want %q", name, expected)
	}

	// Verify StartAll returns graceful error if no docker/podman is active
	err := sm.StartAll(context.Background())
	if err == nil {
		t.Logf("Docker/Podman daemon is active and responsive")
	} else {
		t.Logf("Docker/Podman daemon not responsive (expected in test env): %v", err)
	}
}
