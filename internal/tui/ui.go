package tui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/runner"
	"github.com/loom-cli/gryd/internal/state"
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
)

type logMsg string
type tickMsg time.Time

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
	}

	m.readExistingLogs()
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
	if len(m.allLogs) > 3000 {
		m.allLogs = m.allLogs[len(m.allLogs)-3000:]
	}

	for _, sName := range m.serviceNames[1:] {
		if strings.HasPrefix(line, "["+sName+"]") || strings.Contains(line, "["+sName+"]") {
			m.logs[sName] = append(m.logs[sName], line)
			if len(m.logs[sName]) > 1500 {
				m.logs[sName] = m.logs[sName][len(m.logs[sName])-1500:]
			}
			break
		}
	}
}

func (m *UIModel) updateViewportContent() {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lines []string
	if m.selectedIdx == 0 {
		lines = m.allLogs
	} else {
		targetSvc := m.serviceNames[m.selectedIdx]
		lines = m.logs[targetSvc]
	}

	if len(lines) == 0 {
		m.viewport.SetContent("\n  \033[2mNo logs recorded yet for this service.\033[0m\n")
	} else {
		m.viewport.SetContent(strings.Join(lines, "\n"))
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

		// Determine if compact mode (terminal width under 85 columns)
		m.compactMode = (msg.Width < 85)

		// Responsive layout math
		var sidebarWidth int
		var mainWidth int
		var contentHeight int

		// Vertical heights: header(1) + footer(1) + margin gaps(3)
		contentHeight = msg.Height - 5
		if contentHeight < 6 {
			contentHeight = 6
		}

		if m.compactMode {
			// In compact mode, sidebar becomes a top tab bar or takes minimal 22 cols
			sidebarWidth = 22
			mainWidth = msg.Width - sidebarWidth - 5
		} else {
			// Proportional sidebar: 25% of width bounded between 24 and 34
			sidebarWidth = int(float64(msg.Width) * 0.24)
			if sidebarWidth < 24 {
				sidebarWidth = 24
			} else if sidebarWidth > 34 {
				sidebarWidth = 34
			}
			mainWidth = msg.Width - sidebarWidth - 5
		}

		if mainWidth < 20 {
			mainWidth = 20
		}

		m.viewport.Width = mainWidth - 2 // account for borders & padding
		m.viewport.Height = contentHeight - 2
		m.updateViewportContent()

	case logMsg:
		chunk := string(msg)
		if chunk != "" {
			lines := strings.Split(chunk, "\n")
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
		}
		cmds = append(cmds, m.ticker())

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "c":
			m.detach = true
			return m, tea.Quit

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

		case "r":
			m.statusMessage = "Reloading services..."
			go func() {
				_, _ = runner.ReloadBackgroundProject(m.cfg.Name)
			}()

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
		for _, proc := range m.projectState.Processes {
			if state.IsProcessAlive(proc.PID) {
				activeCount++
			}
		}
	}
	countLabel := fmt.Sprintf(" %d/%d Active ", activeCount, len(m.cfg.Services))

	headerBar := lipgloss.JoinHorizontal(lipgloss.Center,
		pillProject.Render(titleLabel),
		" ",
		pillUptime.Render(uptimeLabel),
		" ",
		pillCount.Render(countLabel),
	)

	// Compute available widths
	sidebarWidth := int(float64(m.width) * 0.24)
	if sidebarWidth < 22 {
		sidebarWidth = 22
	} else if sidebarWidth > 32 {
		sidebarWidth = 32
	}
	mainWidth := m.width - sidebarWidth - 4
	if mainWidth < 20 {
		mainWidth = 20
	}

	contentHeight := m.height - 5
	if contentHeight < 6 {
		contentHeight = 6
	}

	// 2. Sidebar Rendering
	var sidebarItems []string
	sidebarHeader := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(" SERVICES")
	sidebarItems = append(sidebarItems, sidebarHeader, "")

	for i, name := range m.serviceNames {
		statusBadge := ""
		if i == 0 {
			statusBadge = lipgloss.NewStyle().Foreground(colorAccent).Render("✦")
		} else {
			if m.projectState != nil {
				if proc, ok := m.projectState.Processes[name]; ok && state.IsProcessAlive(proc.PID) {
					portBadge := ""
					if svc, ok := m.cfg.Services[name]; ok && svc.Port > 0 {
						portBadge = fmt.Sprintf(":%d", svc.Port)
					}
					statusBadge = lipgloss.NewStyle().Foreground(colorSuccess).Render("●" + portBadge)
				} else {
					statusBadge = lipgloss.NewStyle().Foreground(colorDanger).Render("✖")
				}
			} else {
				statusBadge = lipgloss.NewStyle().Foreground(colorMuted).Render("○")
			}
		}

		maxNameLen := sidebarWidth - 10
		if maxNameLen < 6 {
			maxNameLen = 6
		}
		displayName := name
		if len(displayName) > maxNameLen {
			displayName = displayName[:maxNameLen-1] + "…"
		}

		// Calculate padding to right-align status badge
		padLen := sidebarWidth - len(displayName) - lipgloss.Width(statusBadge) - 6
		if padLen < 1 {
			padLen = 1
		}
		spacer := strings.Repeat(" ", padLen)

		rowText := fmt.Sprintf("%s%s%s", displayName, spacer, statusBadge)

		if i == m.selectedIdx {
			sidebarItems = append(sidebarItems, itemActive.Width(sidebarWidth-4).Render("❯ "+rowText))
		} else {
			sidebarItems = append(sidebarItems, itemInactive.Width(sidebarWidth-4).Render("  "+rowText))
		}
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
	titleRight := lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("%d lines ", logLineCount))

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
	if m.width > 90 {
		footerText = fmt.Sprintf("%s %s   %s %s   %s %s   %s %s   %s %s",
			shortcutKey.Render("↑/↓/Tab"), shortcutDesc.Render("select"),
			shortcutKey.Render("u/d"), shortcutDesc.Render("scroll"),
			shortcutKey.Render("r"), shortcutDesc.Render("reload"),
			shortcutKey.Render("c"), shortcutDesc.Render("detach"),
			shortcutKey.Render("q"), shortcutDesc.Render("quit"),
		)
	} else {
		footerText = fmt.Sprintf("%s %s  %s %s  %s %s",
			shortcutKey.Render("Tab"), shortcutDesc.Render("service"),
			shortcutKey.Render("r"), shortcutDesc.Render("reload"),
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
