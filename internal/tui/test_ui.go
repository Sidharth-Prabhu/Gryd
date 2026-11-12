package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/runner"
)

type testLogMsg string
type testCheckMsg time.Time

// TestUIModel is the interactive Bubble Tea model for `gryd test`.
type TestUIModel struct {
	cfg          *config.Config
	timeout      time.Duration
	startTime    time.Time
	spinner      spinner.Model
	viewport     viewport.Model
	logLines     []string
	serviceNames []string
	results      map[string]*runner.ServiceTestResult
	readyStatus  map[string]bool
	bootTimes    map[string]time.Duration
	failedErr    error
	done         bool
	width        int
	height       int
	quitting     bool
}

// NewTestUIModel initializes the test runner UI.
func NewTestUIModel(cfg *config.Config, timeout time.Duration) *TestUIModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	var svcs []string
	results := make(map[string]*runner.ServiceTestResult)
	readyStatus := make(map[string]bool)
	for name, svc := range cfg.Services {
		svcs = append(svcs, name)
		results[name] = &runner.ServiceTestResult{
			Name:    name,
			Command: svc.Command,
			Port:    svc.Port,
		}
		readyStatus[name] = false
	}

	vp := viewport.New(80, 10)
	vp.YPosition = 0

	return &TestUIModel{
		cfg:          cfg,
		timeout:      timeout,
		startTime:    time.Now(),
		spinner:      s,
		viewport:     vp,
		serviceNames: svcs,
		results:      results,
		readyStatus:  readyStatus,
		bootTimes:    make(map[string]time.Duration),
		width:        80,
		height:       24,
	}
}

func (m *TestUIModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.pollTick(),
	)
}

func (m *TestUIModel) pollTick() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
		return testCheckMsg(t)
	})
}

func (m *TestUIModel) AddLog(line string) {
	m.logLines = append(m.logLines, line)
	if len(m.logLines) > 1000 {
		m.logLines = m.logLines[len(m.logLines)-1000:]
	}
	m.viewport.SetContent(strings.Join(m.logLines, "\n"))
	m.viewport.GotoBottom()
}

func (m *TestUIModel) SetServiceReady(name string, d time.Duration) {
	m.readyStatus[name] = true
	m.bootTimes[name] = d
	if r, ok := m.results[name]; ok {
		r.Ready = true
		r.Duration = d
	}
}

func (m *TestUIModel) SetFailed(err error) {
	m.failedErr = err
	m.done = true
}

func (m *TestUIModel) SetAllDone() {
	m.done = true
}

func (m *TestUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		vpHeight := msg.Height - 14
		if vpHeight < 6 {
			vpHeight = 6
		}
		vpWidth := msg.Width - 6
		if vpWidth < 20 {
			vpWidth = 20
		}
		m.viewport.Width = vpWidth
		m.viewport.Height = vpHeight
		m.viewport.SetContent(strings.Join(m.logLines, "\n"))
		m.viewport.GotoBottom()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case testLogMsg:
		m.AddLog(string(msg))

	case testCheckMsg:
		if !m.done {
			cmds = append(cmds, m.pollTick())
		} else {
			// Once done, pause briefly then exit UI so summary renders cleanly
			return m, tea.Quit
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *TestUIModel) View() string {
	if m.width == 0 {
		m.width = 80
	}

	elapsed := time.Since(m.startTime).Round(time.Millisecond)

	// 1. Header Bar
	headerLeft := BadgeProject.Render(" GRYD TEST ")
	projBadge := lipgloss.NewStyle().Bold(true).Foreground(ColorText).Background(ColorActive).Padding(0, 1).Render(" " + m.cfg.Name + " ")
	statusBadge := BadgeWarning.Render(fmt.Sprintf(" %s TESTING (%s) ", m.spinner.View(), elapsed.String()))
	if m.done {
		if m.failedErr != nil {
			statusBadge = BadgeDanger.Render(" ✖ FAILED ")
		} else {
			statusBadge = BadgeSuccess.Render(" ✓ ALL PASSED ")
		}
	}

	leftGroup := lipgloss.JoinHorizontal(lipgloss.Center, headerLeft, projBadge)
	availWidth := m.width - lipgloss.Width(leftGroup) - lipgloss.Width(statusBadge)
	spacer := ""
	if availWidth > 0 {
		spacer = strings.Repeat(" ", availWidth)
	}
	header := lipgloss.JoinHorizontal(lipgloss.Center, leftGroup, spacer, statusBadge)

	// 2. Services Verification Progress Table
	var svcRows []string
	colNameW := 20
	colStatusW := 14
	colPortW := 10
	colTimeW := 12

	tableHeader := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(
		fmt.Sprintf("  %-*s %-*s %-*s %-*s", colNameW, "SERVICE", colStatusW, "STATUS", colPortW, "PORT", colTimeW, "BOOT TIME"),
	)
	svcRows = append(svcRows, tableHeader)
	svcRows = append(svcRows, "  "+strings.Repeat("─", m.width-8))

	for _, name := range m.serviceNames {
		svc := m.cfg.Services[name]
		isReady := m.readyStatus[name]

		statusText := lipgloss.NewStyle().Foreground(ColorWarning).Render(m.spinner.View() + " Booting...")
		portText := "-"
		if svc.Port > 0 {
			portText = fmt.Sprintf(":%d", svc.Port)
		}

		timeText := "-"
		if isReady {
			statusText = lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render("✓ Ready")
			d := m.bootTimes[name]
			if d > 0 {
				timeText = d.Round(time.Millisecond).String()
			}
		} else if m.done && m.failedErr != nil {
			statusText = lipgloss.NewStyle().Bold(true).Foreground(ColorDanger).Render("✖ Failed")
		}

		row := fmt.Sprintf("  %-*s %-*s %-*s %s",
			colNameW, name,
			colStatusW+lipgloss.Width(statusText)-len(lipgloss.NewStyle().Render(statusText)), statusText,
			colPortW, portText,
			timeText,
		)
		svcRows = append(svcRows, row)
	}

	tableBox := BoxCard.Width(m.width - 4).Render(strings.Join(svcRows, "\n"))

	// 3. Live Log Output Box
	logHeader := lipgloss.NewStyle().Bold(true).Foreground(ColorTextMuted).Render(" REAL-TIME BUILD & BOOT LOGS")
	logBox := BoxCard.Width(m.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, logHeader, "\n", m.viewport.View()),
	)

	// 4. Footer
	footer := lipgloss.NewStyle().Foreground(ColorMuted).Render(
		fmt.Sprintf("Verifying health checks and socket listeners... Press Ctrl+C or 'q' to abort (Timeout: %s)", m.timeout),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		tableBox,
		"",
		logBox,
		"",
		footer,
	)
}
