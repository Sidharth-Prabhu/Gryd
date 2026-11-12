package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type PortRow struct {
	Port        int
	ProjectName string
	ServiceName string
	PIDs        string
	Status      string
	ProcessName string
}

// RenderPortsTable formats the ports audit view into a styled modern TUI table.
func RenderPortsTable(rows []PortRow, termWidth int) string {
	if termWidth <= 0 {
		termWidth = 80
	}

	headerLeft := BadgeProject.Render(" GRYD PORTS ")
	subtext := lipgloss.NewStyle().Foreground(ColorTextMuted).Render(" Active Port Listeners & Allocations")
	topHeader := lipgloss.JoinHorizontal(lipgloss.Center, headerLeft, subtext)

	colPortW := 10
	colProjW := 14
	colSvcW := 14
	colStatW := 14
	colPidW := 10

	tableHeader := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(
		fmt.Sprintf("  %-*s %-*s %-*s %-*s %-*s %s",
			colPortW, "PORT",
			colProjW, "PROJECT",
			colSvcW, "SERVICE",
			colStatW, "STATUS",
			colPidW, "PID",
			"PROCESS"),
	)

	var tableLines []string
	tableLines = append(tableLines, tableHeader)
	tableLines = append(tableLines, "  "+strings.Repeat("─", termWidth-10))

	for _, r := range rows {
		portStr := fmt.Sprintf(":%d", r.Port)

		var statBadge string
		if strings.Contains(strings.ToLower(r.Status), "active") {
			statBadge = lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render("● Listening")
		} else {
			statBadge = lipgloss.NewStyle().Foreground(ColorMuted).Render("○ Inactive")
		}

		line := fmt.Sprintf("  %-*s %-*s %-*s %-*s %-*s %s",
			colPortW, portStr,
			colProjW, r.ProjectName,
			colSvcW, r.ServiceName,
			colStatW+lipgloss.Width(statBadge)-len(lipgloss.NewStyle().Render(statBadge)), statBadge,
			colPidW, r.PIDs,
			r.ProcessName,
		)
		tableLines = append(tableLines, line)
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		topHeader,
		"",
		strings.Join(tableLines, "\n"),
	)

	return BoxCard.Width(termWidth - 2).Render(body)
}
