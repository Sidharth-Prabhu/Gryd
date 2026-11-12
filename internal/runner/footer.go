package runner

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

// PinnedFooterManager manages a scrolling screen region with a pinned bottom status line.
type PinnedFooterManager struct {
	mu           sync.Mutex
	fd           int
	isTerm       bool
	height       int
	width        int
	footerHeight int
	footerText   string
	initialized  bool
}

// NewPinnedFooterManager initializes a footer pinned to the bottom of the screen.
func NewPinnedFooterManager(fd int, footerText string) *PinnedFooterManager {
	isTerm := term.IsTerminal(fd)
	return &PinnedFooterManager{
		fd:           fd,
		isTerm:       isTerm,
		footerHeight: 2, // 1 divider line + 1 status line
		footerText:   footerText,
	}
}

// Init sets up the terminal scrolling region (rows 1 to height-footerHeight).
func (m *PinnedFooterManager) Init() {
	if !m.isTerm {
		fmt.Println(m.footerText)
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	width, height, err := term.GetSize(m.fd)
	if err != nil || height <= m.footerHeight+2 {
		m.isTerm = false
		fmt.Println(m.footerText)
		return
	}

	m.width = width
	m.height = height
	m.initialized = true

	// Set scroll region: top row 1, bottom row (height - footerHeight)
	scrollBottom := height - m.footerHeight
	// DECSTBM: \033[top;bottomr
	fmt.Fprintf(os.Stdout, "\033[1;%dr", scrollBottom)

	m.renderFooter()

	// Place cursor at bottom of scrollable region
	fmt.Fprintf(os.Stdout, "\033[%d;1H", scrollBottom)
}

// renderFooter draws the footer at the bottom lines without moving scroll position.
func (m *PinnedFooterManager) renderFooter() {
	if !m.initialized || m.height <= m.footerHeight {
		return
	}

	dividerRow := m.height - 1
	statusRow := m.height

	divider := strings.Repeat("─", m.width)
	if len(divider) > 80 {
		divider = strings.Repeat("─", 80)
	}

	// Save cursor: \033[s (or \0337)
	// Move to dividerRow: \033[dividerRow;1H
	// Clear line: \033[2K
	// Write divider
	// Move to statusRow: \033[statusRow;1H
	// Clear line: \033[2K
	// Write status
	// Restore cursor: \033[u (or \0338)
	fmt.Fprintf(os.Stdout, "\0337\033[%d;1H\033[2K\033[2m%s\033[0m\033[%d;1H\033[2K%s\0338",
		dividerRow, divider, statusRow, m.footerText)
}

// Refresh updates terminal dimensions on resize.
func (m *PinnedFooterManager) Refresh() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isTerm {
		return
	}

	width, height, err := term.GetSize(m.fd)
	if err == nil && height > m.footerHeight+2 {
		m.width = width
		m.height = height
		scrollBottom := height - m.footerHeight
		fmt.Fprintf(os.Stdout, "\033[1;%dr", scrollBottom)
		m.renderFooter()
	}
}

// WriteToScrollRegion writes log lines into the scroll region and ensures footer stays intact.
func (m *PinnedFooterManager) WriteToScrollRegion(w io.Writer, p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	n, err := w.Write(p)
	if m.initialized {
		m.renderFooter()
	}
	return n, err
}

// Reset restores the terminal scrolling region to full screen.
func (m *PinnedFooterManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return
	}

	// Reset scroll margins: \033[r
	// Move cursor to bottom: \033[height;1H
	// Clear footer rows
	fmt.Fprintf(os.Stdout, "\033[r\033[%d;1H\033[2K\033[%d;1H\033[2K\033[%d;1H\n",
		m.height-1, m.height, m.height-1)
	m.initialized = false
}
