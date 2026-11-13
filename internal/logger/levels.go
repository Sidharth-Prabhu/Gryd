package logger

import (
	"regexp"
	"strings"
)

var (
	// Regular expressions for detecting log levels
	reError = regexp.MustCompile(`(?i)\b(ERROR|FATAL|PANIC|FAIL|FAILURE|ERR)\b`)
	reWarn  = regexp.MustCompile(`(?i)\b(WARN|WARNING)\b`)
	reInfo  = regexp.MustCompile(`(?i)\b(INFO|INFORMATION)\b`)
	reDebug = regexp.MustCompile(`(?i)\b(DEBUG|TRACE)\b`)

	// ANSI Color Codes
	ansiRed    = "\033[1;31m"
	ansiYellow = "\033[1;33m"
	ansiBlue   = "\033[1;34m"
	ansiGray   = "\033[2;37m"
)

// ColorizeLogLevels detects keywords such as ERROR, WARN, INFO, DEBUG and applies ANSI colors.
func ColorizeLogLevels(line string) string {
	// If line is empty or already has level highlighting, return
	if strings.TrimSpace(line) == "" {
		return line
	}

	// Check if line is structured JSON
	if parsed, prefix, isJSON := IsJSONLog(line); isJSON {
		return FormatJSONCompact(parsed, prefix)
	}

	// Highlight ERROR in Red
	line = reError.ReplaceAllStringFunc(line, func(m string) string {
		return ansiRed + m + ansiReset
	})

	// Highlight WARN in Yellow
	line = reWarn.ReplaceAllStringFunc(line, func(m string) string {
		return ansiYellow + m + ansiReset
	})

	// Highlight INFO in Blue / Cyan
	line = reInfo.ReplaceAllStringFunc(line, func(m string) string {
		return ansiBlue + m + ansiReset
	})

	// Highlight DEBUG in Subdued Gray
	line = reDebug.ReplaceAllStringFunc(line, func(m string) string {
		return ansiGray + m + ansiReset
	})

	return line
}
