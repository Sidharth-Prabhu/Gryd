package logger

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

var (
	jsonKeyColor    = "\033[1;36m" // Cyan
	jsonStringColor = "\033[32m"   // Green
	jsonNumberColor = "\033[33m"   // Yellow
	jsonBoolColor   = "\033[35m"   // Magenta
	jsonNullColor   = "\033[2;37m" // Dim Gray
	jsonBraceColor  = "\033[1;37m" // Bold White
)

// IsJSONLog detects if a string contains valid JSON payload (e.g. `{"level":"info","msg":"..."}`).
func IsJSONLog(line string) (map[string]interface{}, string, bool) {
	trimmed := strings.TrimSpace(line)
	// Check if there is a prefix like [backend]
	prefix := ""
	if strings.HasPrefix(trimmed, "[") {
		closeIdx := strings.Index(trimmed, "]")
		if closeIdx > 0 && closeIdx+1 < len(trimmed) {
			prefix = trimmed[:closeIdx+1] + " "
			trimmed = strings.TrimSpace(trimmed[closeIdx+1:])
		}
	}

	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return nil, "", false
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return nil, "", false
	}

	return parsed, prefix, true
}

// FormatJSONCompact renders a structured JSON log into a clean single-line view:
// [backend] [INFO] Request completed (method="GET", path="/api", status=200)
func FormatJSONCompact(parsed map[string]interface{}, prefix string) string {
	var level, msg, timeStr string

	for _, k := range []string{"level", "lvl", "severity", "status"} {
		if v, ok := parsed[k]; ok {
			level = fmt.Sprintf("%v", v)
			delete(parsed, k)
			break
		}
	}

	for _, k := range []string{"msg", "message", "error"} {
		if v, ok := parsed[k]; ok {
			msg = fmt.Sprintf("%v", v)
			delete(parsed, k)
			break
		}
	}

	for _, k := range []string{"time", "timestamp", "ts"} {
		if v, ok := parsed[k]; ok {
			timeStr = fmt.Sprintf("%v", v)
			delete(parsed, k)
			break
		}
	}

	// Format remaining fields as key=value
	var keys []string
	for k := range parsed {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var fields []string
	for _, k := range keys {
		val := parsed[k]
		fields = append(fields, fmt.Sprintf("%s%s%s=%s", jsonKeyColor, k, ansiReset, formatJSONValue(val)))
	}

	var sb strings.Builder
	if prefix != "" {
		sb.WriteString(prefix)
	}

	if timeStr != "" {
		sb.WriteString(fmt.Sprintf("\033[2m%s\033[0m ", timeStr))
	}

	if level != "" {
		upperLvl := strings.ToUpper(level)
		coloredLvl := ColorizeLogLevels(upperLvl)
		sb.WriteString(fmt.Sprintf("[%s] ", coloredLvl))
	}

	if msg != "" {
		sb.WriteString(fmt.Sprintf("\033[1m%s\033[0m ", msg))
	}

	if len(fields) > 0 {
		sb.WriteString(fmt.Sprintf("(%s)", strings.Join(fields, ", ")))
	}

	return strings.TrimSpace(sb.String())
}

// FormatJSONPretty formats the JSON object with colorized syntax across indented lines.
func FormatJSONPretty(parsed map[string]interface{}, prefix string) string {
	data, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	var formatted []string
	for _, l := range lines {
		formatted = append(formatted, ColorizeJSONLine(l))
	}

	if prefix != "" {
		return prefix + "\n" + strings.Join(formatted, "\n")
	}
	return strings.Join(formatted, "\n")
}

// ColorizeJSONLine applies syntax highlighting to a single JSON line.
func ColorizeJSONLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "{" || trimmed == "}" || trimmed == "[" || trimmed == "]" || trimmed == "}," || trimmed == "]," {
		return jsonBraceColor + line + ansiReset
	}

	parts := strings.SplitN(line, ":", 2)
	if len(parts) == 2 {
		keyPart := parts[0]
		valPart := parts[1]

		coloredKey := jsonKeyColor + keyPart + ansiReset
		coloredVal := colorizeValueToken(valPart)

		return coloredKey + ":" + coloredVal
	}

	return line
}

func colorizeValueToken(v string) string {
	trimmed := strings.TrimSpace(v)
	trailingComma := ""
	if strings.HasSuffix(trimmed, ",") {
		trailingComma = ","
		trimmed = strings.TrimSuffix(trimmed, ",")
	}

	indent := v[:len(v)-len(strings.TrimLeft(v, " "))]

	var colored string
	if strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") {
		colored = jsonStringColor + trimmed + ansiReset
	} else if trimmed == "true" || trimmed == "false" {
		colored = jsonBoolColor + trimmed + ansiReset
	} else if trimmed == "null" {
		colored = jsonNullColor + trimmed + ansiReset
	} else {
		colored = jsonNumberColor + trimmed + ansiReset
	}

	return indent + colored + trailingComma
}

func formatJSONValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%s%q%s", jsonStringColor, val, ansiReset)
	case bool:
		return fmt.Sprintf("%s%v%s", jsonBoolColor, val, ansiReset)
	case nil:
		return fmt.Sprintf("%snull%s", jsonNullColor, ansiReset)
	default:
		return fmt.Sprintf("%s%v%s", jsonNumberColor, val, ansiReset)
	}
}
