package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Theme Palette (Catppuccin Mocha inspired)
	ColorBgDark      = lipgloss.Color("#181825")
	ColorPrimary     = lipgloss.Color("#89B4FA") // Sapphire Blue
	ColorAccent      = lipgloss.Color("#CBA6F7") // Mauve / Purple
	ColorSuccess     = lipgloss.Color("#A6E3A1") // Emerald Green
	ColorWarning     = lipgloss.Color("#F9E2AF") // Warm Gold
	ColorDanger      = lipgloss.Color("#F38BA8") // Crimson
	ColorMuted       = lipgloss.Color("#6C7086") // Slate Gray
	ColorBorder      = lipgloss.Color("#313244") // Subtle Border
	ColorActive      = lipgloss.Color("#45475A") // Active Item Highlight
	ColorText        = lipgloss.Color("#CDD6F4") // Primary Text
	ColorTextMuted   = lipgloss.Color("#A6ADC8") // Subtext

	// Reusable Components & Badges
	BadgeProject = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#11111B")).
			Background(ColorPrimary).
			Padding(0, 1)

	BadgeSuccess = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#11111B")).
			Background(ColorSuccess).
			Padding(0, 1)

	BadgeDanger = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#11111B")).
			Background(ColorDanger).
			Padding(0, 1)

	BadgeWarning = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#11111B")).
			Background(ColorWarning).
			Padding(0, 1)

	BadgeMuted = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorBorder).
			Padding(0, 1)

	BoxCard = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	TableBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "┌",
		TopRight:    "┐",
		BottomLeft:  "└",
		BottomRight: "┘",
	}

	TableHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Background(lipgloss.Color("#1E1E2E")).
			Padding(0, 1)

	TableCell = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(ColorText)
)
