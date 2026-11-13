package cli

import (
	"testing"
)

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		latest   string
		current  string
		expected bool
	}{
		{"v0.2.0", "0.1.0-dev", true},
		{"v0.2.0", "v0.1.0", true},
		{"v0.2.0", "0.2.0", false},
		{"v0.2.0", "v0.2.0", false},
		{"v0.1.0", "v0.2.0", false},
		{"v0.3.0", "v0.2.0", true},
	}

	for _, tc := range tests {
		got := isNewerVersion(tc.latest, tc.current)
		if got != tc.expected {
			t.Errorf("isNewerVersion(%q, %q) = %v; expected %v", tc.latest, tc.current, got, tc.expected)
		}
	}
}
