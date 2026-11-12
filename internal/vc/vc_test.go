package vc

import (
	"testing"
)

func TestCheckGitStatus(t *testing.T) {
	info := CheckGitStatus()
	if !info.Installed {
		t.Skip("git not installed on test host")
	}

	if info.Version == "" {
		t.Errorf("expected non-empty git version")
	}
}
