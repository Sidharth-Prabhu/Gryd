package vc

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type remoteOption struct {
	Name string
	URL  string
}

type remoteSelectorModel struct {
	options  []remoteOption
	cursor   int
	chosen   string
	quitting bool
}

func (m remoteSelectorModel) Init() tea.Cmd {
	return nil
}

func (m remoteSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.chosen = m.options[m.cursor].Name
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m remoteSelectorModel) View() string {
	if m.quitting {
		return "\nSelection cancelled.\n"
	}
	if m.chosen != "" {
		return fmt.Sprintf("\nSelected remote: \033[1;32m%s\033[0m\n", m.chosen)
	}

	var s strings.Builder
	s.WriteString("\n\033[1mSelect a remote repository to push to:\033[0m\n")
	s.WriteString("\033[2mUse ↑/↓ arrow keys to navigate, Enter to select, q/Ctrl+C to cancel\033[0m\n\n")

	for i, opt := range m.options {
		cursor := "  "
		itemStyle := opt.Name
		if m.cursor == i {
			cursor = "\033[1;36m❯ \033[0m"
			itemStyle = fmt.Sprintf("\033[1;36m%s\033[0m", opt.Name)
		}

		s.WriteString(fmt.Sprintf("%s%-14s \033[2m(%s)\033[0m\n", cursor, itemStyle, opt.URL))
	}

	s.WriteString("\n")
	return s.String()
}

// SelectRemote displays an interactive Bubble Tea menu to select a remote repository.
func SelectRemote(remotes []RemoteInfo) (string, error) {
	// Deduplicate remotes by name
	seen := make(map[string]bool)
	var options []remoteOption

	for _, r := range remotes {
		if !seen[r.Name] {
			seen[r.Name] = true
			options = append(options, remoteOption{
				Name: r.Name,
				URL:  r.URL,
			})
		}
	}

	if len(options) == 0 {
		return "", fmt.Errorf("no remotes available to select")
	}

	// If there's only 1 remote, we can either select it directly or present it
	p := tea.NewProgram(remoteSelectorModel{
		options: options,
	})

	m, err := p.Run()
	if err != nil {
		return "", err
	}

	finalModel := m.(remoteSelectorModel)
	if finalModel.quitting || finalModel.chosen == "" {
		return "", fmt.Errorf("remote selection was cancelled")
	}

	return finalModel.chosen, nil
}
