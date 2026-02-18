package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorPrimary   = lipgloss.Color("#7D56F4") // Purple
	colorSecondary = lipgloss.Color("#F4A956") // Orange
	colorText      = lipgloss.Color("#FAFAFA") // White/Light Gray
	colorSubtext   = lipgloss.Color("#777777") // Gray
	colorSuccess   = lipgloss.Color("#43BF6D") // Green
	colorError     = lipgloss.Color("#FF5F5F") // Red

	// Layout Styles
	styleWindow = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(colorPrimary).
		// Padding removed to allow split views to touch borders
		Align(lipgloss.Center)

	stylePanel = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(colorSubtext).
			Padding(1, 1)

	// Panel style for panels with internal titles (no top padding)
	stylePanelTitled = lipgloss.NewStyle().
				Border(lipgloss.ThickBorder()).
				BorderForeground(colorSubtext).
				Padding(0, 1)

	styleTitle = lipgloss.NewStyle().
			Background(colorPrimary).
			Foreground(colorText).
			Padding(0, 1).
			Bold(true)

	styleAppTitle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true).
			Padding(0, 1).
			Align(lipgloss.Center)

	styleSelected = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	styleLabel = lipgloss.NewStyle().
			Foreground(colorSubtext).
			Width(10)

	styleValue = lipgloss.NewStyle().
			Foreground(colorText)

	// Menu/Card Styles
	styleMenuContainer = lipgloss.NewStyle().
				Padding(1)

	styleMenuItem = lipgloss.NewStyle().
			Foreground(colorText).
			Border(lipgloss.ThickBorder()).
			BorderForeground(colorSubtext).
			Padding(1, 4).
			Margin(0, 1).
			Align(lipgloss.Center).
			Width(24)

	styleMenuItemSelected = styleMenuItem.
				Foreground(colorText).
				BorderForeground(colorSecondary).
				Bold(true)

	styleScreenTooSmall = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true).
				Align(lipgloss.Center, lipgloss.Center)

	// Scrollbar styles
	scrollbarTrack = lipgloss.NewStyle().
			Foreground(colorSubtext)

	scrollbarThumb = lipgloss.NewStyle().
			Foreground(colorPrimary)
)
