package tui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/logger"
	"github.com/loom-cli/gryd/internal/runner"
	"github.com/loom-cli/gryd/internal/state"
	"github.com/loom-cli/gryd/internal/tunnel"
)

var (
	// Theme Palette
	colorBgDark   = lipgloss.Color("#181825")
	colorPrimary = lipgloss.Color("#89B4FA") // Sapphire Blue
	colorAccent   = lipgloss.Color("#CBA6F7") // Mauve / Purple
	colorSuccess  = lipgloss.Color("#A6E3A1") // Emerald Green
	colorWarning  = lipgloss.Color("#F9E2AF") // Warm Gold
	colorDanger   = lipgloss.Color("#F38BA8") // Crimson
	colorMuted    = lipgloss.Color("#6C7086") // Slate Gray
	colorBorder   = lipgloss.Color("#313244") // Subtle Border
	colorActive   = lipgloss.Color("#45475A") // Active Item Highlight
	colorText     = lipgloss.Color("#CDD6F4") // Primary Text

	// Component Styles
	pillProject = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#11111B")).
			Background(colorPrimary).
			Padding(0, 1)

	pillUptime = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorBorder).
			Padding(0, 1)

	pillCount = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#11111B")).
			Background(colorSuccess).
			Padding(0, 1)

	boxSidebar = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	boxMain = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 1)

	itemActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(colorActive).
			Padding(0, 1)

	itemInactive = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 1)

	shortcutKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	shortcutDesc = lipgloss.NewStyle().
			Foreground(colorMuted)

	highlightStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#11111B")).
			Background(lipgloss.Color("#F9E2AF"))
)

type logMsg string
type tickMsg time.Time
type clearStatusMsg struct{}

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

// UIModel is the responsive Bubble Tea model for the interactive dashboard.
type UIModel struct {
	cfg           *config.Config
	projectState  *state.ProjectState
	serviceNames  []string
	selectedIdx   int
	width         int
	height        int
	viewport      viewport.Model
	logs          map[string][]string
	allLogs       []string
	logFile       *os.File
	fileOffset    int64
	quitting      bool
	detach        bool
	mu            sync.Mutex
	statusMessage string
	compactMode   bool

	// Search state
	searching   bool
	searchQuery string
	matchCount  int

	// Log level filter ("ALL", "ERROR", "WARN", "INFO")
	levelFilter string

	// Background file watcher for TUI mode
	watcher *runner.FileWatcher

	// Resource Telemetry (CPU% & RAM per service)
	telemetry   map[string]*state.ProcessTelemetry
	inspectMode bool // toggle with 'i' or 'p'

	// Active public tunnels (service -> URL)
	tunnels map[string]string

	// JSON format mode: "compact" (single-line parsed), "pretty" (expanded multiline), "raw"
	jsonViewMode string
}

// NewUIModel creates a new responsive UIModel.
func NewUIModel(cfg *config.Config, st *state.ProjectState) (*UIModel, error) {
	logPath := st.LogFilePath
	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open project logs: %w", err)
	}

	var services []string
	services = append(services, "All (Combined)")
	for name := range cfg.Services {
		services = append(services, name)
	}

	vp := viewport.New(80, 20)
	vp.YPosition = 0

	m := &UIModel{
		cfg:          cfg,
		projectState: st,
		serviceNames: services,
		selectedIdx:  0,
		viewport:     vp,
		logs:         make(map[string][]string),
		allLogs:      nil,
		logFile:      file,
		levelFilter:  "ALL",
		telemetry:    make(map[string]*state.ProcessTelemetry),
		tunnels:      make(map[string]string),
		jsonViewMode: "compact",
	}

	m.readExistingLogs()

	// Initialize file watcher in TUI mode
	fw, err := runner.NewFileWatcher(cfg, nil, func(serviceName string) error {
		return runner.RestartSingleService(cfg.Name, serviceName)
	})
	if err == nil {
		m.watcher = fw
		_ = fw.Start(context.Background())
	}

	return m, nil
}

func (m *UIModel) IsDetached() bool {
	return m.detach
}

func (m *UIModel) readExistingLogs() {
	if m.logFile == nil {
		return
	}

	stat, err := m.logFile.Stat()
	if err == nil {
		if stat.Size() > 64*1024 {
			_, _ = m.logFile.Seek(-64*1024, io.SeekEnd)
		} else {
			_, _ = m.logFile.Seek(0, io.SeekStart)
		}
	}

	scanner := bufio.NewScanner(m.logFile)
	for scanner.Scan() {
		line := scanner.Text()
		m.appendLogLine(line)
	}
	m.fileOffset, _ = m.logFile.Seek(0, io.SeekCurrent)
	m.updateViewportContent()
}

func (m *UIModel) appendLogLine(line string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.allLogs = append(m.allLogs, line)
	if len(m.allLogs) > 2000 {
		m.allLogs = m.allLogs[len(m.allLogs)-2000:]
	}

	// Try extracting service tag e.g. [backend]
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "[") {
		endIdx := strings.Index(trimmed, "]")
		if endIdx > 1 {
			svcTag := trimmed[1:endIdx]
			if _, exists := m.cfg.Services[svcTag]; exists {
				m.logs[svcTag] = append(m.logs[svcTag], line)
				if len(m.logs[svcTag]) > 2000 {
					m.logs[svcTag] = m.logs[svcTag][len(m.logs[svcTag])-2000:]
				}
			}
		}
	}
}

func matchesLogLevel(line, filter string) bool {
	if filter == "" || filter == "ALL" {
		return true
	}
	upper := strings.ToUpper(line)
	switch filter {
	case "ERROR":
		return strings.Contains(upper, "ERROR") || strings.Contains(upper, "ERR") || strings.Contains(upper, "FATAL") || strings.Contains(upper, "PANIC")
	case "WARN":
		return strings.Contains(upper, "WARN") || strings.Contains(upper, "WARNING")
	case "INFO":
		return strings.Contains(upper, "INFO")
	}
	return true
}

func highlightMatches(line, query string) string {
	if query == "" {
		return line
	}
	// Case-insensitive regex search with fallback to literal quote
	re, err := regexp.Compile("(?i)" + query)
	if err != nil {
		re, err = regexp.Compile("(?i)" + regexp.QuoteMeta(query))
		if err != nil {
			return line
		}
	}
	return re.ReplaceAllStringFunc(line, func(match string) string {
		return highlightStyle.Render(match)
	})
}

func lineMatchesQuery(line, query string) bool {
	if query == "" {
		return true
	}
	// First try regex match
	if re, err := regexp.Compile("(?i)" + query); err == nil {
		if re.MatchString(line) {
			return true
		}
	}
	// Fallback to substring match
	return strings.Contains(strings.ToLower(line), strings.ToLower(query))
}

func (m *UIModel) updateViewportContent() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.inspectMode {
		targetSvc := m.serviceNames[m.selectedIdx]
		if m.selectedIdx == 0 {
			// All services overview
			var inspectLines []string
			inspectLines = append(inspectLines, "\033[1;36m=== SERVICE PROCESS & TELEMETRY OVERVIEW ===\033[0m\n")
			for _, sName := range m.serviceNames[1:] {
				telem, exists := m.telemetry[sName]
				if exists && telem != nil {
					inspectLines = append(inspectLines, fmt.Sprintf("\033[1;32m● %s\033[0m (PID: %d) — CPU: \033[1;33m%.1f%%\033[0m | RAM: \033[1;35m%s\033[0m | FDs: %d | Threads: %d",
						sName, telem.PID, telem.CPUPercent, state.FormatBytes(telem.MemoryRSS), telem.NumFDs, telem.NumThreads))
					for _, c := range telem.Children {
						inspectLines = append(inspectLines, fmt.Sprintf("    └── \033[2mPID %d (%s):\033[0m CPU: %.1f%% | RAM: %s | Cmd: %s",
							c.PID, c.Name, c.CPUPercent, state.FormatBytes(c.MemoryRSS), c.Command))
					}
				} else {
					inspectLines = append(inspectLines, fmt.Sprintf("\033[2m○ %s — Inactive / Not Running\033[0m", sName))
				}
				inspectLines = append(inspectLines, "")
			}
			m.viewport.SetContent(strings.Join(inspectLines, "\n"))
			return
		}

		// Single service detailed process tree inspection
		telem, exists := m.telemetry[targetSvc]
		if !exists || telem == nil {
			m.viewport.SetContent(fmt.Sprintf("\n  \033[33mNo active telemetry available for '%s'\033[0m (process not running)\n  \033[2mPress 'i' or 'Esc' to return to logs.\033[0m\n", targetSvc))
			return
		}

		var inspectLines []string
		inspectLines = append(inspectLines, fmt.Sprintf("\033[1;36m=== PROCESS TREE & RESOURCE TELEMETRY: %s ===\033[0m\n", strings.ToUpper(targetSvc)))
		inspectLines = append(inspectLines, fmt.Sprintf("  \033[1mRoot PID:\033[0m       %d", telem.PID))
		inspectLines = append(inspectLines, fmt.Sprintf("  \033[1mTotal CPU:\033[0m      \033[1;33m%.1f%%\033[0m", telem.CPUPercent))
		inspectLines = append(inspectLines, fmt.Sprintf("  \033[1mMemory (RSS):\033[0m   \033[1;35m%s\033[0m", state.FormatBytes(telem.MemoryRSS)))
		inspectLines = append(inspectLines, fmt.Sprintf("  \033[1mFile Descr:\033[0m     %d", telem.NumFDs))
		inspectLines = append(inspectLines, fmt.Sprintf("  \033[1mThreads:\033[0m        %d", telem.NumThreads))
		inspectLines = append(inspectLines, "")

		if len(telem.Children) > 0 {
			inspectLines = append(inspectLines, fmt.Sprintf("\033[1;34mCHILD PROCESSES (%d):\033[0m", len(telem.Children)))
			for _, c := range telem.Children {
				inspectLines = append(inspectLines, fmt.Sprintf("  ├── \033[1mPID %d\033[0m [%s] — CPU: %.1f%% | RSS: %s",
					c.PID, c.Name, c.CPUPercent, state.FormatBytes(c.MemoryRSS)))
				if c.Command != "" {
					inspectLines = append(inspectLines, fmt.Sprintf("  │   \033[2mCommand: %s\033[0m", c.Command))
				}
			}
		} else {
			inspectLines = append(inspectLines, "\033[2mNo active child sub-processes.\033[0m")
		}

		inspectLines = append(inspectLines, "\n\033[2mPress 'i' or 'Esc' to exit inspector and return to live log stream.\033[0m")
		m.viewport.SetContent(strings.Join(inspectLines, "\n"))
		return
	}

	var sourceLines []string
	if m.selectedIdx == 0 {
		sourceLines = m.allLogs
	} else {
		targetSvc := m.serviceNames[m.selectedIdx]
		sourceLines = m.logs[targetSvc]
	}

	query := strings.TrimSpace(m.searchQuery)
	var displayLines []string
	matches := 0

	for _, l := range sourceLines {
		if !matchesLogLevel(l, m.levelFilter) {
			continue
		}

		var formattedLine string
		if m.jsonViewMode == "raw" {
			formattedLine = l
		} else if parsed, prefix, isJSON := logger.IsJSONLog(l); isJSON {
			if m.jsonViewMode == "pretty" {
				formattedLine = logger.FormatJSONPretty(parsed, prefix)
			} else {
				formattedLine = logger.FormatJSONCompact(parsed, prefix)
			}
		} else {
			formattedLine = logger.ColorizeLogLevels(l)
		}

		if query != "" {
			if lineMatchesQuery(formattedLine, query) {
				matches++
				displayLines = append(displayLines, highlightMatches(formattedLine, query))
			}
		} else {
			displayLines = append(displayLines, formattedLine)
		}
	}

	if query != "" {
		m.matchCount = matches
	} else {
		m.matchCount = 0
	}

	if len(displayLines) == 0 {
		if query != "" {
			m.viewport.SetContent(fmt.Sprintf("\n  \033[33mNo logs matching search query: '%s'\033[0m  \033[2m(Esc to clear)\033[0m\n", query))
		} else if m.levelFilter != "ALL" {
			m.viewport.SetContent(fmt.Sprintf("\n  \033[33mNo logs matching level filter: %s\033[0m  \033[2m(Press 'l' to cycle)\033[0m\n", m.levelFilter))
		} else {
			m.viewport.SetContent("\n  \033[2mNo logs recorded yet for this service.\033[0m\n")
		}
	} else {
		m.viewport.SetContent(strings.Join(displayLines, "\n"))
	}
	m.viewport.GotoBottom()
}

func (m *UIModel) Init() tea.Cmd {
	return tea.Batch(
		m.pollNewLogs(),
		m.ticker(),
	)
}

func (m *UIModel) ticker() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *UIModel) pollNewLogs() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(100 * time.Millisecond)
		if m.logFile == nil {
			return nil
		}

		buf := make([]byte, 8192)
		n, err := m.logFile.Read(buf)
		if err == nil && n > 0 {
			return logMsg(string(buf[:n]))
		}
		return logMsg("")
	}
}

func (m *UIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.compactMode = (msg.Width < 85)

		var sidebarWidth int
		var mainWidth int
		var contentHeight int

		contentHeight = msg.Height - 5
		if contentHeight < 6 {
			contentHeight = 6
		}

		if m.compactMode {
			sidebarWidth = 22
			mainWidth = msg.Width - sidebarWidth - 3
		} else {
			sidebarWidth = 28
			if sidebarWidth > msg.Width/3 {
				sidebarWidth = msg.Width / 3
			}
			mainWidth = msg.Width - sidebarWidth - 3
		}

		if mainWidth < 20 {
			mainWidth = 20
		}

		m.viewport.Width = mainWidth - 4
		m.viewport.Height = contentHeight - 4
		m.updateViewportContent()

	case clearStatusMsg:
		m.statusMessage = ""
		return m, nil

	case logMsg:
		str := string(msg)
		if str != "" {
			lines := strings.Split(str, "\n")
			for _, l := range lines {
				if l != "" {
					m.appendLogLine(l)
				}
			}
			m.updateViewportContent()
		}
		cmds = append(cmds, m.pollNewLogs())

	case tickMsg:
		if updated, err := state.LoadState(m.cfg.Name); err == nil {
			m.projectState = updated
			// Update telemetry for each active process
			for sName, proc := range m.projectState.Processes {
				if proc.PID > 0 {
					if telem, err := state.GetProcessTelemetry(proc.PID); err == nil {
						m.telemetry[sName] = telem
					} else {
						delete(m.telemetry, sName)
					}
				}
			}
			if m.inspectMode {
				m.updateViewportContent()
			}
		}
		cmds = append(cmds, m.ticker())

	case tea.KeyMsg:
		// When search input mode is active
		if m.searching {
			switch msg.Type {
			case tea.KeyEsc:
				// Exit search and clear query
				m.searching = false
				m.searchQuery = ""
				m.updateViewportContent()
				return m, nil

			case tea.KeyEnter:
				// Keep current search active and resume navigation
				m.searching = false
				return m, nil

			case tea.KeyBackspace, tea.KeyDelete:
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					m.updateViewportContent()
				}
				return m, nil

			case tea.KeyRunes, tea.KeySpace:
				m.searchQuery += msg.String()
				m.updateViewportContent()
				return m, nil

			case tea.KeyCtrlC:
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		// Standard navigation mode
		switch msg.String() {
		case "ctrl+c", "q":
			if m.watcher != nil {
				_ = m.watcher.Close()
			}
			m.quitting = true
			return m, tea.Quit

		case "c":
			m.detach = true
			return m, tea.Quit

		case "/":
			m.searching = true
			m.statusMessage = ""
			return m, nil

		case "i", "p":
			// Toggle process inspection view
			m.inspectMode = !m.inspectMode
			if m.inspectMode {
				m.statusMessage = "Inspecting process & resource tree (press 'i' or 'Esc' to exit)"
			} else {
				m.statusMessage = "Returned to log stream"
			}
			m.updateViewportContent()
			return m, nil

		case "m":
			// Insert a visual timestamp marker
			ts := time.Now().Format("15:04:05")
			markerLine := fmt.Sprintf("\033[1;35m--- [Marker: %s] ---\033[0m", ts)
			m.appendLogLine(markerLine)
			m.statusMessage = fmt.Sprintf("Marker added at %s", ts)
			m.updateViewportContent()
			return m, nil

		case "l":
			// Cycle log level filter: ALL -> ERROR -> WARN -> INFO -> ALL
			switch m.levelFilter {
			case "ALL", "":
				m.levelFilter = "ERROR"
			case "ERROR":
				m.levelFilter = "WARN"
			case "WARN":
				m.levelFilter = "INFO"
			case "INFO":
				m.levelFilter = "ALL"
			default:
				m.levelFilter = "ALL"
			}
			m.statusMessage = fmt.Sprintf("Log level filter: %s", m.levelFilter)
			m.updateViewportContent()
			return m, nil

		case "esc":
			if m.inspectMode {
				m.inspectMode = false
				m.statusMessage = "Returned to log stream"
				m.updateViewportContent()
			} else if m.searchQuery != "" {
				m.searchQuery = ""
				m.updateViewportContent()
			} else if m.levelFilter != "ALL" {
				m.levelFilter = "ALL"
				m.statusMessage = "Log filter reset to ALL"
				m.updateViewportContent()
			}

		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
				m.updateViewportContent()
			} else {
				m.selectedIdx = len(m.serviceNames) - 1
				m.updateViewportContent()
			}

		case "down", "j":
			if m.selectedIdx < len(m.serviceNames)-1 {
				m.selectedIdx++
				m.updateViewportContent()
			} else {
				m.selectedIdx = 0
				m.updateViewportContent()
			}

		case "tab":
			m.selectedIdx = (m.selectedIdx + 1) % len(m.serviceNames)
			m.updateViewportContent()

		case "shift+tab":
			m.selectedIdx = (m.selectedIdx - 1 + len(m.serviceNames)) % len(m.serviceNames)
			m.updateViewportContent()

		case "J":
			// Cycle JSON view mode: compact -> pretty -> raw -> compact
			switch m.jsonViewMode {
			case "compact":
				m.jsonViewMode = "pretty"
			case "pretty":
				m.jsonViewMode = "raw"
			case "raw":
				m.jsonViewMode = "compact"
			default:
				m.jsonViewMode = "compact"
			}
			m.statusMessage = fmt.Sprintf("JSON view: %s", m.jsonViewMode)
			m.updateViewportContent()
			return m, clearStatusAfter(2 * time.Second)

		case "t":
			// Toggle public tunnel for selected service
			if m.selectedIdx > 0 {
				targetSvc := m.serviceNames[m.selectedIdx]
				if activeURL, exists := m.tunnels[targetSvc]; exists {
					_ = tunnel.GetGlobalManager().StopTunnel(targetSvc)
					delete(m.tunnels, targetSvc)
					m.statusMessage = fmt.Sprintf("Tunnel closed for '%s'", targetSvc)
					_ = activeURL
				} else {
					if svc, ok := m.cfg.Services[targetSvc]; ok && svc.Port > 0 {
						m.statusMessage = fmt.Sprintf("Starting public tunnel for '%s'...", targetSvc)
						go func(sName string, p int) {
							tun, err := tunnel.GetGlobalManager().StartTunnel(context.Background(), sName, p, "")
							if err == nil && tun != nil {
								m.mu.Lock()
								m.tunnels[sName] = tun.PublicURL
								m.statusMessage = fmt.Sprintf("Tunnel active: %s", tun.PublicURL)
								m.mu.Unlock()
							} else if err != nil {
								m.mu.Lock()
								m.statusMessage = fmt.Sprintf("Tunnel failed: %v", err)
								m.mu.Unlock()
							}
						}(targetSvc, svc.Port)
					} else {
						m.statusMessage = fmt.Sprintf("Service '%s' has no port configured", targetSvc)
					}
				}
				return m, clearStatusAfter(4 * time.Second)
			}

		case "r":
			m.statusMessage = "Hot reloading services..."
			go func() {
				_, _ = runner.HotReloadBackgroundProject(m.cfg.Name)
			}()
			return m, clearStatusAfter(2 * time.Second)

		case "R":
			m.statusMessage = "Hard restarting all processes & ports..."
			go func() {
				// Hard stop, free ports, and fresh boot
				if st, err := state.LoadState(m.cfg.Name); err == nil && st != nil {
					runner.StopProjectProcesses(st)
					_ = state.RemoveState(m.cfg.Name)
				}
				_, _ = runner.ReloadBackgroundProject(m.cfg.Name)
			}()
			return m, clearStatusAfter(3 * time.Second)

		case "pgup", "u":
			m.viewport.LineUp(8)

		case "pgdown", "d":
			m.viewport.LineDown(8)

		case "g":
			m.viewport.GotoTop()

		case "G":
			m.viewport.GotoBottom()
		}
	}

	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m *UIModel) View() string {
	if m.quitting {
		return "\nShutting down Gryd UI...\n"
	}
	if m.detach {
		return fmt.Sprintf("\n\033[1;32m[gryd] Detached from UI.\033[0m Services are running in background.\nRe-attach anytime with: \033[1mgryd run %s\033[0m or \033[1mgryd look %s\033[0m\n\n",
			m.cfg.Name, m.cfg.Name)
	}

	// 1. Responsive Header Bar
	uptimeStr := "-"
	if m.projectState != nil {
		uptimeStr = time.Since(m.projectState.StartedAt).Round(time.Second).String()
	}

	titleLabel := fmt.Sprintf(" GRYD: %s ", m.cfg.Name)
	uptimeLabel := fmt.Sprintf(" Uptime: %s ", uptimeStr)

	// Count active running services
	activeCount := 0
	if m.projectState != nil {
		for _, p := range m.projectState.Processes {
			if p.PID > 0 && state.IsProcessAlive(p.PID) {
				activeCount++
			}
		}
	}
	totalCount := len(m.cfg.Services)
	countLabel := fmt.Sprintf(" %d/%d Active ", activeCount, totalCount)

	headerLeft := pillProject.Render(titleLabel)
	headerMid := pillUptime.Render(uptimeLabel)
	headerRight := pillCount.Render(countLabel)

	leftGroup := lipgloss.JoinHorizontal(lipgloss.Center, headerLeft, " ", headerMid)
	spacerWidth := m.width - lipgloss.Width(leftGroup) - lipgloss.Width(headerRight)
	if spacerWidth < 1 {
		spacerWidth = 1
	}

	headerBar := lipgloss.JoinHorizontal(lipgloss.Center,
		leftGroup,
		strings.Repeat(" ", spacerWidth),
		headerRight,
	)

	// Responsive layout math
	var sidebarWidth int
	var mainWidth int
	var contentHeight int

	contentHeight = m.height - 5
	if contentHeight < 6 {
		contentHeight = 6
	}

	if m.compactMode {
		sidebarWidth = 22
		mainWidth = m.width - sidebarWidth - 3
	} else {
		sidebarWidth = 28
		if sidebarWidth > m.width/3 {
			sidebarWidth = m.width / 3
		}
		mainWidth = m.width - sidebarWidth - 3
	}

	if mainWidth < 20 {
		mainWidth = 20
	}

	// 2. Sidebar Navigation Pane
	var sidebarItems []string
	sidebarHeader := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(" SERVICES")
	sidebarItems = append(sidebarItems, sidebarHeader, "")

	for i, svc := range m.serviceNames {
		statusBadge := ""
		if i == 0 {
			statusBadge = lipgloss.NewStyle().Foreground(colorAccent).Render("✦")
		} else {
			if m.projectState != nil {
				if proc, ok := m.projectState.Processes[svc]; ok && proc.PID > 0 && state.IsProcessAlive(proc.PID) {
					portBadge := ""
					if s, exists := m.cfg.Services[svc]; exists && s.Port > 0 {
						portBadge = fmt.Sprintf(":%d", s.Port)
					}
					statusBadge = lipgloss.NewStyle().Foreground(colorSuccess).Render("●" + portBadge)
				} else {
					statusBadge = lipgloss.NewStyle().Foreground(colorDanger).Render("✖")
				}
			} else {
				statusBadge = lipgloss.NewStyle().Foreground(colorMuted).Render("○")
			}
		}

		displayName := svc
		metricBadge := ""
		if telem, ok := m.telemetry[svc]; ok && telem != nil {
			metricBadge = fmt.Sprintf(" %s", state.FormatBytes(telem.MemoryRSS))
		}

		maxNameLen := sidebarWidth - lipgloss.Width(statusBadge) - lipgloss.Width(metricBadge) - 7
		if maxNameLen > 3 && len(displayName) > maxNameLen {
			displayName = displayName[:maxNameLen-2] + "…"
		}

		isSelected := (i == m.selectedIdx)
		itemText := displayName

		padLen := sidebarWidth - len(displayName) - lipgloss.Width(metricBadge) - lipgloss.Width(statusBadge) - 6
		if padLen < 1 {
			padLen = 1
		}
		metricStyled := ""
		if metricBadge != "" {
			metricStyled = lipgloss.NewStyle().Foreground(colorMuted).Render(metricBadge)
		}
		rowContent := fmt.Sprintf("%s%s%s%s", itemText, strings.Repeat(" ", padLen), metricStyled, statusBadge)

		if isSelected {
			sidebarItems = append(sidebarItems, itemActive.Width(sidebarWidth-4).Render("❯ "+rowContent))
		} else {
			sidebarItems = append(sidebarItems, itemInactive.Width(sidebarWidth-4).Render("  "+rowContent))
		}
	}

	// Render sidecars list if any exist
	if len(m.cfg.Dependencies) > 0 {
		sidebarItems = append(sidebarItems, "", lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Render(" SIDECARS"))
		for depName, dep := range m.cfg.Dependencies {
			depBadge := lipgloss.NewStyle().Foreground(colorMuted).Render("○")
			portInfo := ""
			if len(dep.Ports) > 0 {
				hostPort := strings.Split(dep.Ports[0], ":")[0]
				portInfo = ":" + hostPort
				if p, err := strconv.Atoi(hostPort); err == nil && runner.IsPortInUse(p) {
					depBadge = lipgloss.NewStyle().Foreground(colorSuccess).Render("●" + portInfo)
				} else {
					depBadge = lipgloss.NewStyle().Foreground(colorMuted).Render("○" + portInfo)
				}
			}
			padLen := sidebarWidth - len(depName) - lipgloss.Width(depBadge) - 6
			if padLen < 1 {
				padLen = 1
			}
			sidebarItems = append(sidebarItems, itemInactive.Width(sidebarWidth-4).Render(fmt.Sprintf("  %s%s%s", depName, strings.Repeat(" ", padLen), depBadge)))
		}
	}

	for len(sidebarItems) < contentHeight-2 {
		sidebarItems = append(sidebarItems, "")
	}

	sidebarContent := strings.Join(sidebarItems, "\n")
	sidebarBox := boxSidebar.Width(sidebarWidth).Height(contentHeight).Render(sidebarContent)

	// 3. Main Viewport Pane
	selectedService := m.serviceNames[m.selectedIdx]
	var logLineCount int
	if m.selectedIdx == 0 {
		logLineCount = len(m.allLogs)
	} else {
		logLineCount = len(m.logs[selectedService])
	}

	titleLeft := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(fmt.Sprintf(" %s", selectedService))

	var searchInfo string
	if m.searching {
		searchInfo = fmt.Sprintf(" \033[1;33mSearch:\033[0m %s\033[7m \033[0m (%d matches) ", m.searchQuery, m.matchCount)
	} else if m.searchQuery != "" {
		searchInfo = fmt.Sprintf(" \033[1;33mFiltered:\033[0m '%s' (%d matches, Esc to reset) ", m.searchQuery, m.matchCount)
	}

	levelInfo := ""
	if m.levelFilter != "ALL" && m.levelFilter != "" {
		levelInfo = fmt.Sprintf(" \033[1;35m[Level: %s]\033[0m ", m.levelFilter)
	}

	// Telemetry & Tunnel header badge if inspecting or active
	var telemHeader string
	if m.selectedIdx > 0 {
		targetSvc := m.serviceNames[m.selectedIdx]
		if tunURL, ok := m.tunnels[targetSvc]; ok && tunURL != "" {
			telemHeader += fmt.Sprintf(" \033[1;32m[⚡ %s]\033[0m ", tunURL)
		}
		if telem, ok := m.telemetry[targetSvc]; ok && telem != nil {
			telemHeader += fmt.Sprintf(" \033[1;33mCPU: %.1f%%\033[0m \033[1;35mRAM: %s\033[0m ",
				telem.CPUPercent, state.FormatBytes(telem.MemoryRSS))
		}
	}

	titleRightText := fmt.Sprintf("%d lines ", logLineCount)
	if m.inspectMode {
		titleRightText = "\033[1;36m[PROCESS TREE INSPECTOR]\033[0m "
	} else if searchInfo != "" || levelInfo != "" || telemHeader != "" {
		titleRightText = telemHeader + searchInfo + levelInfo + titleRightText
	}
	titleRight := lipgloss.NewStyle().Foreground(colorMuted).Render(titleRightText)

	titleSpacerWidth := mainWidth - lipgloss.Width(titleLeft) - lipgloss.Width(titleRight) - 4
	if titleSpacerWidth < 1 {
		titleSpacerWidth = 1
	}
	paneTitleBar := lipgloss.JoinHorizontal(lipgloss.Center, titleLeft, strings.Repeat(" ", titleSpacerWidth), titleRight)

	mainContent := lipgloss.JoinVertical(lipgloss.Left, paneTitleBar, "\n", m.viewport.View())
	mainBox := boxMain.Width(mainWidth).Height(contentHeight).Render(mainContent)

	// Horizontal arrangement
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, " ", mainBox)

	// 4. Responsive Footer Bar
	var footerText string
	if m.searching {
		footerText = fmt.Sprintf("%s %s   %s %s   %s %s",
			shortcutKey.Render("Type"), shortcutDesc.Render("to search"),
			shortcutKey.Render("Enter"), shortcutDesc.Render("confirm filter"),
			shortcutKey.Render("Esc"), shortcutDesc.Render("cancel search"),
		)
	} else if m.width > 90 {
		footerText = fmt.Sprintf("%s %s   %s %s   %s %s   %s %s   %s %s   %s %s   %s %s   %s %s   %s %s   %s %s   %s %s   %s %s",
			shortcutKey.Render("/"), shortcutDesc.Render("search"),
			shortcutKey.Render("J"), shortcutDesc.Render("json"),
			shortcutKey.Render("t"), shortcutDesc.Render("tunnel"),
			shortcutKey.Render("i"), shortcutDesc.Render("inspect"),
			shortcutKey.Render("l"), shortcutDesc.Render("lvl"),
			shortcutKey.Render("m"), shortcutDesc.Render("marker"),
			shortcutKey.Render("↑/↓/Tab"), shortcutDesc.Render("select"),
			shortcutKey.Render("u/d"), shortcutDesc.Render("scroll"),
			shortcutKey.Render("r"), shortcutDesc.Render("reload"),
			shortcutKey.Render("R"), shortcutDesc.Render("restart"),
			shortcutKey.Render("c"), shortcutDesc.Render("detach"),
			shortcutKey.Render("q"), shortcutDesc.Render("quit"),
		)
	} else {
		footerText = fmt.Sprintf("%s %s  %s %s  %s %s  %s %s  %s %s  %s %s  %s %s  %s %s",
			shortcutKey.Render("/"), shortcutDesc.Render("search"),
			shortcutKey.Render("J"), shortcutDesc.Render("json"),
			shortcutKey.Render("t"), shortcutDesc.Render("tunnel"),
			shortcutKey.Render("Tab"), shortcutDesc.Render("service"),
			shortcutKey.Render("r"), shortcutDesc.Render("reload"),
			shortcutKey.Render("R"), shortcutDesc.Render("restart"),
			shortcutKey.Render("c"), shortcutDesc.Render("detach"),
			shortcutKey.Render("q"), shortcutDesc.Render("quit"),
		)
	}

	if m.statusMessage != "" {
		footerText = fmt.Sprintf("%s  •  %s", lipgloss.NewStyle().Foreground(colorWarning).Render(m.statusMessage), footerText)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		headerBar,
		"\n",
		body,
		"\n",
		footerText,
	)
}
