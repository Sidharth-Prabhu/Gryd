package vc

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorGitBranch  = lipgloss.Color("#A6E3A1") // Emerald Green
	colorGitCommit  = lipgloss.Color("#F9E2AF") // Warm Gold
	colorGitRemote  = lipgloss.Color("#89B4FA") // Sapphire Blue
	colorGitMuted   = lipgloss.Color("#6C7086") // Slate Gray
	colorGitHead    = lipgloss.Color("#CBA6F7") // Mauve
	colorGitCardBdr = lipgloss.Color("#313244")

	pillGit = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#11111B")).
		Background(colorGitRemote).
		Padding(0, 1)

	boxGitCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGitCardBdr).
			Padding(0, 1)
)

// PrintGitStatusCard formats the git status overview inside a styled TUI card.
func PrintGitStatusCard(info GitStatusInfo, termWidth int) {
	if termWidth <= 0 {
		termWidth = 80
	}

	headerLeft := pillGit.Render(" GRYD VERSION CONTROL ")
	topHeader := lipgloss.JoinHorizontal(lipgloss.Center, headerLeft)

	if !info.Installed {
		errBox := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F38BA8")).Render("✖ Git is not installed on this system.")
		fmt.Println("\n" + boxGitCard.Width(termWidth-2).Render(lipgloss.JoinVertical(lipgloss.Left, topHeader, "", errBox)) + "\n")
		return
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("  \033[1;32m✓\033[0m \033[1mGit Engine:\033[0m          %s", info.Version))

	if info.IsConfigured {
		lines = append(lines, fmt.Sprintf("  \033[1;32m✓\033[0m \033[1mUser Identity:\033[0m       %s <%s>", info.UserName, info.UserEmail))
	} else {
		lines = append(lines, "  \033[1;31m✖\033[0m \033[1mUser Identity:\033[0m       Not configured (run: git config --global user.name ...)")
	}

	if info.GHInstalled {
		if info.GHAuthenticated {
			ghUser := info.GHUser
			if ghUser == "" {
				ghUser = "Authenticated"
			}
			lines = append(lines, fmt.Sprintf("  \033[1;32m✓\033[0m \033[1mGitHub CLI (gh):\033[0m     Logged in as %s", ghUser))
		} else {
			lines = append(lines, "  \033[1;33m!\033[0m \033[1mGitHub CLI (gh):\033[0m     Not authenticated (run 'gh auth login')")
		}
	}

	if info.InGitRepo {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  \033[1mRepository Branch:\033[0m   \033[1;32m%s\033[0m", info.RepoBranch))
		if info.RepoOrigin != "" {
			lines = append(lines, fmt.Sprintf("  \033[1mRemote Origin:\033[0m       \033[1;34m%s\033[0m", info.RepoOrigin))
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		topHeader,
		"",
		strings.Join(lines, "\n"),
	)

	fmt.Println()
	fmt.Println(boxGitCard.Width(termWidth - 2).Render(content))
	fmt.Println()
}
