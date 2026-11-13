package state

import (
	"os"
	"testing"
)

func TestGetProcessTelemetry(t *testing.T) {
	currentPID := os.Getpid()
	telem, err := GetProcessTelemetry(currentPID)
	if err != nil {
		t.Fatalf("failed to get telemetry for current process: %v", err)
	}

	if telem.PID != int32(currentPID) {
		t.Errorf("expected PID %d, got %d", currentPID, telem.PID)
	}

	if telem.MemoryRSS == 0 {
		t.Errorf("expected positive RSS memory usage, got %d", telem.MemoryRSS)
	}

	formatted := FormatBytes(telem.MemoryRSS)
	if formatted == "" {
		t.Errorf("FormatBytes returned empty string")
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		bytes uint64
		want  string
	}{
		{500, "500 B"},
		{1024, "1.0KB"},
		{1024 * 1024 * 50, "50.0MB"},
		{1024 * 1024 * 1024 * 2, "2.0GB"},
	}

	for _, tc := range cases {
		got := FormatBytes(tc.bytes)
		if got != tc.want {
			t.Errorf("FormatBytes(%d) = %s, want %s", tc.bytes, got, tc.want)
		}
	}
}
