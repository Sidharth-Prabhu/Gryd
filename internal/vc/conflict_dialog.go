package vc

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type ConflictResolutionOption struct {
	ID          string
	Title       string
	Description string
}

type conflictModel struct {
	options      []ConflictResolutionOption
	cursor       int
	chosen       string
	targetBranch string
	currentBranch string
	quitting     bool
}

func (m conflictModel) Init() tea.Cmd {
	return nil
}

func (m conflictModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.options) - 1
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
		case "enter", " ":
			m.chosen = m.options[m.cursor].ID
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m conflictModel) View() string {
	if m.quitting {
		return "\nOperation cancelled.\n"
	}

	var s strings.Builder
	s.WriteString("\n\033[1;33m⚠ Uncommitted local changes would be overwritten by switching to '" + m.targetBranch + "'.\033[0m\n")
	s.WriteString("\033[1mHow would you like to resolve this?\033[0m\n")
	s.WriteString("\033[2mUse ↑/↓ arrow keys to choose an action, Enter to confirm, Esc to abort\033[0m\n\n")

	for i, opt := range m.options {
		cursor := "  "
		titleStyle := opt.Title
		if m.cursor == i {
			cursor = "\033[1;36m❯ \033[0m"
			titleStyle = fmt.Sprintf("\033[1;36m%s\033[0m", opt.Title)
		}

		s.WriteString(fmt.Sprintf("%s%s\n", cursor, titleStyle))
		s.WriteString(fmt.Sprintf("    \033[2m%s\033[0m\n\n", opt.Description))
	}

	return s.String()
}

// PromptBranchConflictResolution presents an interactive menu to resolve checkout conflict.
func PromptBranchConflictResolution(currentBranch, targetBranch string) (string, error) {
	options := []ConflictResolutionOption{
		{
			ID:          "stash-pop",
			Title:       "1. Stash changes, switch branch, and re-apply (Recommended)",
			Description: "Shelves your uncommitted work into git stash, switches to '" + targetBranch + "', and automatically pops/restores your changes there.",
		},
		{
			ID:          "commit",
			Title:       "2. Commit changes on current branch first",
			Description: "Prompts for a commit message and commits your uncommitted work to '" + currentBranch + "' before switching to '" + targetBranch + "'.",
		},
		{
			ID:          "stash-keep",
			Title:       "3. Stash changes and switch (Keep stash saved)",
			Description: "Safely saves your uncommitted changes into git stash and switches to '" + targetBranch + "', leaving the working tree completely clean.",
		},
		{
			ID:          "force",
			Title:       "4. Force switch and discard uncommitted changes",
			Description: "Permanently discards conflicting uncommitted local changes and forces checkout of '" + targetBranch + "'. (Warning: Changes will be lost).",
		},
		{
			ID:          "cancel",
			Title:       "5. Cancel / Abort",
			Description: "Cancels the operation and stays on your current branch ('" + currentBranch + "') without modifying anything.",
		},
	}

	p := tea.NewProgram(conflictModel{
		options:       options,
		currentBranch: currentBranch,
		targetBranch:  targetBranch,
	})

	m, err := p.Run()
	if err != nil {
		return "", err
	}

	res := m.(conflictModel)
	if res.quitting || res.chosen == "" || res.chosen == "cancel" {
		return "cancel", nil
	}

	return res.chosen, nil
}
