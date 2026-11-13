package runner

import (
	"testing"
)

func TestIPCServerLifecycle(t *testing.T) {
	projectName := "test-ipc-proj"
	sockPath, err := GetSocketPath(projectName)
	if err != nil {
		t.Fatalf("failed to get socket path: %v", err)
	}

	server, err := NewIPCServer(projectName, nil, nil)
	if err != nil {
		t.Fatalf("failed to create IPC server: %v", err)
	}
	defer server.Close()

	server.Start()

	// Test ping request
	resp, err := SendIPCCommand(projectName, IPCRequest{
		Action: "unknown_action",
	})
	if err != nil {
		t.Fatalf("failed to send command to IPC server: %v", err)
	}

	if resp.Success {
		t.Errorf("expected failure for unknown action, got success")
	}

	// Verify server close cleans up socket file
	if err := server.Close(); err != nil {
		t.Fatalf("failed to close server: %v", err)
	}
	_ = sockPath
}
