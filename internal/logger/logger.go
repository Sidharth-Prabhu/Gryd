package logger

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	ansiColors = []string{
		"\033[36m", // Cyan
		"\033[32m", // Green
		"\033[33m", // Yellow
		"\033[35m", // Magenta
		"\033[34m", // Blue
		"\033[96m", // Light Cyan
		"\033[92m", // Light Green
		"\033[93m", // Light Yellow
	}
	ansiReset = "\033[0m"
	ansiBold  = "\033[1m"

	ansiStripRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
)

// LogMultiplexer assigns consistent prefixes and colors to different service streams
// and broadcasts lines to both terminal output (with colors) and file log (clean plain text).
type LogMultiplexer struct {
	mu           sync.Mutex
	colorMap     map[string]string
	colorIndex   int
	maxPrefixLen int
	termWriter   io.Writer
	fileWriter   io.Writer
}

// NewLogMultiplexer creates a new multiplexer for streaming child process outputs.
func NewLogMultiplexer(termOut io.Writer, fileOut io.Writer) *LogMultiplexer {
	return &LogMultiplexer{
		colorMap:   make(map[string]string),
		termWriter: termOut,
		fileWriter: fileOut,
	}
}

// SetTermWriter allows updating or detaching the terminal writer (e.g. on background detach).
func (lm *LogMultiplexer) SetTermWriter(w io.Writer) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.termWriter = w
}

// GetColor returns a color string for the given service name.
func (lm *LogMultiplexer) GetColor(serviceName string) string {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	color, exists := lm.colorMap[serviceName]
	if !exists {
		color = ansiColors[lm.colorIndex%len(ansiColors)]
		lm.colorMap[serviceName] = color
		lm.colorIndex++
	}

	if len(serviceName) > lm.maxPrefixLen {
		lm.maxPrefixLen = len(serviceName)
	}

	return color
}

// NewServiceWriter returns an io.Writer that prefixes each line with [serviceName].
func (lm *LogMultiplexer) NewServiceWriter(serviceName string) io.Writer {
	color := lm.GetColor(serviceName)
	return &serviceLogWriter{
		multiplexer: lm,
		serviceName: serviceName,
		color:       color,
	}
}

type serviceLogWriter struct {
	multiplexer *LogMultiplexer
	serviceName string
	color       string
}

func (w *serviceLogWriter) Write(p []byte) (n int, err error) {
	w.multiplexer.mu.Lock()
	defer w.multiplexer.mu.Unlock()

	pad := ""
	if diff := w.multiplexer.maxPrefixLen - len(w.serviceName); diff > 0 {
		pad = fmt.Sprintf("%*s", diff, "")
	}

	coloredPrefix := fmt.Sprintf("%s%s[%s]%s%s ", ansiBold, w.color, w.serviceName, ansiReset, pad)
	plainPrefix := fmt.Sprintf("[%s]%s ", w.serviceName, pad)

	scanner := bufio.NewScanner(bufio.NewReader(io.MultiReader(
		&sliceReader{p: p},
	)))

	for scanner.Scan() {
		line := scanner.Text()
		if w.multiplexer.termWriter != nil {
			coloredLine := ColorizeLogLevels(line)
			fmt.Fprintf(w.multiplexer.termWriter, "%s%s\n", coloredPrefix, coloredLine)
		}
		if w.multiplexer.fileWriter != nil {
			cleanLine := ansiStripRegex.ReplaceAllString(line, "")
			fmt.Fprintf(w.multiplexer.fileWriter, "%s%s\n", plainPrefix, cleanLine)
		}
	}

	return len(p), scanner.Err()
}

type sliceReader struct {
	p   []byte
	idx int
}

func (r *sliceReader) Read(buf []byte) (int, error) {
	if r.idx >= len(r.p) {
		return 0, io.EOF
	}
	n := copy(buf, r.p[r.idx:])
	r.idx += n
	return n, nil
}

// NewZapLogger creates an initialized Zap logger based on verbose/debug flag.
func NewZapLogger(debug bool) (*zap.Logger, error) {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	if !debug {
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	} else {
		cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}
	return cfg.Build()
}
