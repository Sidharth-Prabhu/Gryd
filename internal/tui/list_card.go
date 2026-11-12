package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/loom-cli/gryd/internal/config"
	"github.com/loom-cli/gryd/internal/state"
)

// ProjectListItem wraps info for list rendering.
type ProjectListItem struct {
	Name        string
	Type        string // "Running", "Local", "Global"
	ConfigPath  string
	LogPath     string
	ServiceCount int
	Services    map[string]config.Service
	Processes   map[string]state.ProcessInfo
}

// RenderProjectsList formats the `gryd list` output into styled cards.
func RenderProjectsList(running []*state.ProjectState, configs []*config.Config, termWidth int) string {
	if termWidth <= 0 {
		termWidth = 80
	}

	var sections []string

	// Active running section
	if len(running) > 0 {
		secTitle := lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess).Render("● ACTIVE BACKGROUND PROJECTS")
		sections = append(sections, secTitle)

		for _, st := range running {
			title := BadgeProject.Render(" " + st.ProjectName + " ")
			badge := BadgeSuccess.Render(fmt.Sprintf(" %d services active ", len(st.Processes)))

			spacer := termWidth - lipgloss.Width(title) - lipgloss.Width(badge) - 6
			if spacer < 1 {
				spacer = 1
			}
			top := lipgloss.JoinHorizontal(lipgloss.Center, title, strings.Repeat(" ", spacer), badge)

			details := fmt.Sprintf("\033[2mStarted:\033[0m %s  |  \033[2mLogs:\033[0m %s\n\033[2mView live:\033[0m \033[1mgryd look %s\033[0m",
				st.StartedAt.Format("15:04:05"), st.LogFilePath, st.ProjectName)

			card := BoxCard.Width(termWidth - 2).Render(lipgloss.JoinVertical(lipgloss.Left, top, "", details))
			sections = append(sections, card)
		}
		sections = append(sections, "")
	}

	// Configured projects section
	if len(configs) > 0 {
		secTitle := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("◆ CONFIGURED PROJECTS")
		sections = append(sections, secTitle)

		for _, cfg := range configs {
			title := lipgloss.NewStyle().Bold(true).Foreground(ColorText).Render("• " + cfg.Name)
			cfgPath := lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("(%s, %d services)", cfg.ConfigPath, len(cfg.Services)))

			var svcList []string
			for sName, s := range cfg.Services {
				portStr := ""
				if s.Port > 0 {
					portStr = fmt.Sprintf(" :%d", s.Port)
				}
				svcList = append(svcList, fmt.Sprintf("    \033[1;36m%s\033[0m%s \033[2m('%s')\033[0m", sName, portStr, s.Command))
			}

			content := lipgloss.JoinVertical(lipgloss.Left,
				fmt.Sprintf("  %s %s", title, cfgPath),
				strings.Join(svcList, "\n"),
			)
			sections = append(sections, content)
		}
	}

	if len(running) == 0 && len(configs) == 0 {
		sections = append(sections, lipgloss.NewStyle().Foreground(ColorMuted).Render("  No Gryd projects found in current directory or ~/.config/gryd/"))
	}

	return strings.Join(sections, "\n")
}
