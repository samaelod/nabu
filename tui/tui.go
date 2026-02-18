package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func New(version string) Model {
	fb := NewFileBrowser([]string{".pcap", ".pcapng", ".cap", ".lua"})

	return Model{
		screen:      screenSourceSelect,
		fileBrowser: fb,
		menuCursor:  0,
		version:     version,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func Run(version string) error {
	p := tea.NewProgram(New(version), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
