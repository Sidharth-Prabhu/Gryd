package logger

import (
	"strings"
	"testing"
)

func TestJSONLogDetectionAndFormatting(t *testing.T) {
	rawLog := `{"level":"info","msg":"HTTP request received","method":"GET","path":"/api/users","status":200,"duration_ms":15.4}`

	parsed, prefix, isJSON := IsJSONLog(rawLog)
	if !isJSON {
		t.Fatalf("expected IsJSONLog to return true for valid JSON log")
	}

	if prefix != "" {
		t.Errorf("expected empty prefix, got %q", prefix)
	}

	compact := FormatJSONCompact(parsed, prefix)
	if !strings.Contains(compact, "HTTP request received") {
		t.Errorf("expected compact log to contain message, got: %s", compact)
	}
	if !strings.Contains(compact, "method") {
		t.Errorf("expected compact log to format attributes, got: %s", compact)
	}

	// Test prefixed JSON log e.g. [backend] {"msg":"ready"}
	prefixedLog := `[backend] {"msg":"server listening","port":8080}`
	parsed2, prefix2, isJSON2 := IsJSONLog(prefixedLog)
	if !isJSON2 {
		t.Fatalf("expected IsJSONLog to handle [backend] prefix")
	}
	if strings.TrimSpace(prefix2) != "[backend]" {
		t.Errorf("expected prefix [backend], got %q", prefix2)
	}

	compact2 := FormatJSONCompact(parsed2, prefix2)
	if !strings.HasPrefix(compact2, "[backend]") {
		t.Errorf("expected formatted log to preserve service prefix, got: %s", compact2)
	}
}
