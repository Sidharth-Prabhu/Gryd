package vc

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type branchSelectorModel struct {
	allBranches   []string
	filtered      []string
	currentBranch string
	cursor        int
	scrollOffset  int
	windowHeight  int
	filterText    string
	chosen        string
	isNew         bool
	quitting      bool
}

func newBranchSelectorModel(branches []string, current string) branchSelectorModel {
	m := branchSelectorModel{
		allBranches:   branches,
		currentBranch: current,
		windowHeight:  15, // Show up to 15 branches in viewport
	}
	m.applyFilter()

	// Locate current branch
	for i, b := range m.filtered {
		if b == current {
			m.cursor = i
			break
		}
	}
	m.adjustScroll()
	return m
}

func (m *branchSelectorModel) applyFilter() {
	if strings.TrimSpace(m.filterText) == "" {
		m.filtered = make([]string, len(m.allBranches))
		copy(m.filtered, m.allBranches)
	} else {
		query := strings.ToLower(strings.TrimSpace(m.filterText))
		m.filtered = nil
		for _, b := range m.allBranches {
			if strings.Contains(strings.ToLower(b), query) {
				m.filtered = append(m.filtered, b)
			}
		}
	}
	if m.cursor >= len(m.filtered)+1 { // +1 for "+ Create new branch"
		m.cursor = 0
	}
	m.adjustScroll()
}

func (m *branchSelectorModel) adjustScroll() {
	totalItems := len(m.filtered) + 1 // branches + "+ Create"
	if totalItems <= m.windowHeight {
		m.scrollOffset = 0
		return
	}

	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	} else if m.cursor >= m.scrollOffset+m.windowHeight {
		m.scrollOffset = m.cursor - m.windowHeight + 1
	}

	maxOffset := totalItems - m.windowHeight
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func (m branchSelectorModel) Init() tea.Cmd {
	return nil
}

func (m branchSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		totalItems := len(m.filtered) + 1 // filtered branches + Create button

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit

		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = totalItems - 1
			}
			m.adjustScroll()
			return m, nil

		case tea.KeyDown:
			if m.cursor < totalItems-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
			m.adjustScroll()
			return m, nil

		case tea.KeyPgUp:
			m.cursor -= m.windowHeight
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.adjustScroll()
			return m, nil

		case tea.KeyPgDown:
			m.cursor += m.windowHeight
			if m.cursor >= totalItems {
				m.cursor = totalItems - 1
			}
			m.adjustScroll()
			return m, nil

		case tea.KeyBackspace, tea.KeyDelete:
			if len(m.filterText) > 0 {
				m.filterText = m.filterText[:len(m.filterText)-1]
				m.applyFilter()
			}
			return m, nil

		case tea.KeyEnter:
			if m.cursor < len(m.filtered) {
				m.chosen = m.filtered[m.cursor]
				m.isNew = false
				return m, tea.Quit
			}

			// If cursor is on "+ Create new branch"
			trimmed := strings.TrimSpace(m.filterText)
			if trimmed != "" {
				m.chosen = trimmed
				m.isNew = true
				return m, tea.Quit
			}
			return m, nil

		case tea.KeyRunes, tea.KeySpace:
			// Append character to filter search
			m.filterText += msg.String()
			m.applyFilter()
			return m, nil
		}
	}
	return m, nil
}

func (m branchSelectorModel) View() string {
	if m.quitting {
		return "\nBranch selection cancelled.\n"
	}

	var s strings.Builder
	s.WriteString("\n\033[1mSelect a target Git branch:\033[0m\n")
	s.WriteString("\033[2mUse ↑/↓ arrows to navigate, type to filter/create, Enter to confirm, Esc/Ctrl+C to quit\033[0m\n\n")

	// Filter / Search bar
	if m.filterText != "" {
		s.WriteString(fmt.Sprintf("  \033[1mFilter / New Branch:\033[0m \033[1;33m%s\033[0m\033[7m \033[0m\n\n", m.filterText))
	} else {
		s.WriteString("  \033[2mType to filter or create a new branch...\033[0m\n\n")
	}

	totalItems := len(m.filtered) + 1
	endIndex := m.scrollOffset + m.windowHeight
	if endIndex > totalItems {
		endIndex = totalItems
	}

	// Render visible items
	for i := m.scrollOffset; i < endIndex; i++ {
		cursor := "  "
		isSelected := (m.cursor == i)

		if i < len(m.filtered) {
			b := m.filtered[i]
			style := b
			if b == m.currentBranch {
				style = fmt.Sprintf("%s \033[2m(current)\033[0m", b)
			}

			if isSelected {
				cursor = "\033[1;36m❯ \033[0m"
				style = fmt.Sprintf("\033[1;36m%s\033[0m", b)
				if b == m.currentBranch {
					style = fmt.Sprintf("\033[1;36m%s\033[0m \033[2m(current)\033[0m", b)
				}
			}
			s.WriteString(fmt.Sprintf("%s%s\n", cursor, style))
		} else {
			// "+ Create new branch"
			createLabel := "+ Create new branch..."
			if strings.TrimSpace(m.filterText) != "" {
				createLabel = fmt.Sprintf("+ Create and switch to new branch '%s'", strings.TrimSpace(m.filterText))
			}

			if isSelected {
				cursor = "\033[1;36m❯ \033[0m"
				s.WriteString(fmt.Sprintf("%s\033[1;33m%s\033[0m\n", cursor, createLabel))
			} else {
				s.WriteString(fmt.Sprintf("%s\033[33m%s\033[0m\n", cursor, createLabel))
			}
		}
	}

	// Scroll position indicator if items exceed viewport
	if totalItems > m.windowHeight {
		s.WriteString(fmt.Sprintf("\n  \033[2mShowing %d-%d of %d branches (scroll with ↑/↓)\033[0m\n",
			m.scrollOffset+1, endIndex, totalItems))
	} else {
		s.WriteString("\n")
	}

	return s.String()
}

// SelectBranchPrompt lets the user select an existing branch or type a new one.
func SelectBranchPrompt() (selectedBranch string, isNew bool, err error) {
	branches, current, err := GetLocalBranches()
	if err != nil {
		return "", false, err
	}

	model := newBranchSelectorModel(branches, current)
	p := tea.NewProgram(model)

	m, err := p.Run()
	if err != nil {
		return "", false, err
	}

	res := m.(branchSelectorModel)
	if res.quitting || res.chosen == "" {
		return "", false, fmt.Errorf("branch selection cancelled")
	}

	return res.chosen, res.isNew, nil
}
