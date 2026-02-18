package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/samaelod/nabu/types"
)

type endpointItem types.Endpoint

func (e endpointItem) Title() string {
	return fmt.Sprintf("[%d] %s:%d", e.ID, e.Address, e.Port)
}
func (e endpointItem) Description() string { return "" }
func (e endpointItem) FilterValue() string { return e.Title() }

type endpointsDelegate struct{}

func renderScrollbar(vp viewport.Model, height int) string {
	total := vp.TotalLineCount()
	visible := vp.VisibleLineCount()

	if total <= visible {
		return ""
	}

	trackHeight := height
	if trackHeight < 1 {
		trackHeight = visible
	}

	scrollPercent := vp.ScrollPercent()

	thumbPos := int(float64(trackHeight-1) * scrollPercent)
	if thumbPos < 0 {
		thumbPos = 0
	}
	if thumbPos > trackHeight-1 {
		thumbPos = trackHeight - 1
	}

	var sb strings.Builder
	for i := 0; i < trackHeight; i++ {
		if i == thumbPos {
			sb.WriteString(scrollbarThumb.Render("█"))
		} else {
			sb.WriteString(scrollbarTrack.Render("│"))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (d endpointsDelegate) Height() int                               { return 1 }
func (d endpointsDelegate) Spacing() int                              { return 0 }
func (d endpointsDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d endpointsDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(endpointItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("[%d] %s:%d", i.ID, i.Address, i.Port)
	isSelected := index == m.Index()

	if isSelected {
		style := styleSelected.Copy().Foreground(colorSecondary)
		str = "> " + str
		fmt.Fprint(w, style.Render(str))
	} else {
		style := lipgloss.NewStyle().Foreground(colorText)
		str = "  " + str
		fmt.Fprint(w, style.Render(str))
	}
}

func (m Model) View() string {
	var content string

	// Calculate inner dimensions
	// Window border (2) + padding (2) + margin (2) = ~6 vertical space used by chrome
	windowWidth := m.width - 4
	windowHeight := m.height - 4

	if windowWidth < minWindowWidth || windowHeight < minWindowHeight {
		return styleScreenTooSmall.
			Width(m.width).
			Height(m.height).
			Render("Terminal window is too small.\nPlease resize.")
	}

	if windowWidth < 0 {
		windowWidth = 0
	}
	if windowHeight < 0 {
		windowHeight = 0
	}

	switch m.screen {

	case screenSourceSelect:
		// App title
		appTitle := styleAppTitle.Width(windowWidth).Render("NABU " + m.version)

		// Custom Card View for Menu
		menuTitle := styleTitle.Render("Select Source")

		var cardPCAP, cardLua string

		if m.menuCursor == 0 {
			cardPCAP = styleMenuItemSelected.Render("PCAP File")
			cardLua = styleMenuItem.Render("Lua Script")
		} else {
			cardPCAP = styleMenuItem.Render("PCAP File")
			cardLua = styleMenuItemSelected.Render("Lua Script")
		}

		menuContent := lipgloss.JoinVertical(lipgloss.Center,
			menuTitle,
			"\n",
			lipgloss.JoinHorizontal(lipgloss.Center, cardPCAP, cardLua),
		)

		// Title at top, menu centered in remaining space
		content = lipgloss.JoinVertical(lipgloss.Top,
			appTitle,
			lipgloss.Place(
				windowWidth, windowHeight-1,
				lipgloss.Center, lipgloss.Center,
				styleMenuContainer.Render(menuContent),
			),
		)

	case screenFilePicker:
		// App title
		appTitle := styleAppTitle.Width(windowWidth).Render("NABU " + m.version)

		// Split View: Browser (1/3) | Preview (2/3)
		listWidth := windowWidth / 3
		previewWidth := windowWidth - listWidth
		panelHeight := windowHeight - 1 // -1 for title

		// Determine border colors
		browserColor := colorSecondary
		if m.fileBrowser.HasValidFilesInDir(m.fileBrowser.CurrentDir) {
			browserColor = colorSuccess
		}

		previewColor := colorSecondary
		if m.fileBrowser.Selected != "" {
			// Check if selected item is a file (not directory)
			item := m.fileBrowser.List.SelectedItem()
			if item != nil {
				fi, ok := item.(fileItem)
				if ok && !fi.isDir {
					if m.fileBrowser.SelectedHasValidExtension() {
						previewColor = colorSuccess
					} else {
						previewColor = colorError
					}
				}
			}
		}

		// Left panel (File Browser) with title inside
		browserTitle := styleTitle.MarginBottom(1).Render("Select File")
		browserContent := browserTitle + "\n" + m.fileBrowser.View()

		browserView := stylePanelTitled.
			BorderForeground(browserColor).
			Width(listWidth - 4).
			Height(panelHeight).
			Render(browserContent)

		// Preview Panel with title inside
		previewTitle := styleTitle.MarginBottom(1).Render("File Preview")

		// Truncate content to fit panel
		contentHeight := panelHeight - 5 // -2 border, -1 title, -1 margin, -1 dots
		previewLines := strings.Split(m.fileBrowser.PreviewContent, "\n")
		if len(previewLines) > contentHeight {
			previewLines = previewLines[:contentHeight-1]
			previewLines = append(previewLines, "...")
		}
		truncatedPreview := strings.Join(previewLines, "\n")
		previewWithTitle := previewTitle + "\n" + truncatedPreview

		previewView := stylePanelTitled.
			BorderForeground(previewColor).
			Width(previewWidth).
			Height(panelHeight).
			Render(previewWithTitle)

		content = lipgloss.Place(
			windowWidth, windowHeight,
			lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Top,
				appTitle,
				lipgloss.JoinHorizontal(lipgloss.Top, browserView, previewView),
			),
		)

	case screenLoading:
		appTitle := styleAppTitle.Width(windowWidth).Render("NABU " + m.version)

		var status string
		if m.err != nil {
			status = styleSubtext.Render("Error: " + m.err.Error())
		} else {
			status = "Loading..."
		}

		content = lipgloss.Place(
			windowWidth, windowHeight,
			lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Center,
				appTitle,
				"\n",
				status,
			),
		)

	case screenViewConfig:
		// App title
		appTitle := styleAppTitle.Width(windowWidth).Render("NABU " + m.version)

		// Layout: Left (List) | Right (Details / Logs)
		availWidth := windowWidth
		availHeight := windowHeight - 1 // -1 for title

		// Reserve space for footer (3 lines: 1 content + 2 border)
		availHeight -= footerHeight

		// 1. Calculate Dimensions
		// Left List Width
		listWidth := defaultListWidth
		if listWidth > availWidth/3 {
			listWidth = availWidth / 3
		}
		if listWidth < minListWidth {
			listWidth = 20
		}

		rightWidth := availWidth - listWidth
		if rightWidth < 0 {
			rightWidth = 0
		}

		// Vertical Split for Right Side
		// Strategy:
		// If logs not focused: Details takes 50%, Logs takes 50%.
		// If logs focused: Logs takes more space? Or simply fixed 50/50?

		var logsHeight int
		if m.activeView == 1 {
			// Logs Focused: 70%
			logsHeight = (availHeight * 70) / 100
		} else {
			// Logs NOT Focused: 40%
			logsHeight = (availHeight * 40) / 100
		}

		detailsHeight := availHeight - logsHeight
		if detailsHeight < 10 { // Enforce minimum for details
			detailsHeight = 10
			logsHeight = availHeight - detailsHeight
		}

		// 2. Left Column (Endpoints List) - Two panels: Servers and Clients
		listHeight := (availHeight - 6) / 2 // -6 for borders and spacing

		// Set sizes for both endpoint lists
		m.serverEndpoints.SetSize(listWidth-4, listHeight)
		m.clientEndpoints.SetSize(listWidth-4, listHeight)

		// Servers Panel
		serverTitle := styleTitle.MarginBottom(1).Render("Servers")
		serverContent := serverTitle + "\n" + m.serverEndpoints.View()
		serverBorderColor := colorSubtext
		if m.activeView == 0 && m.activeEndpointPanel == 0 {
			serverBorderColor = colorSecondary
		}
		serverPanel := stylePanelTitled.
			BorderForeground(serverBorderColor).
			Width(listWidth - 4).
			Height(listHeight + 2). // +2 for border
			Render(serverContent)

		// Clients Panel
		clientTitle := styleTitle.MarginBottom(1).Render("Clients")
		clientContent := clientTitle + "\n" + m.clientEndpoints.View()
		clientBorderColor := colorSubtext
		if m.activeView == 0 && m.activeEndpointPanel == 1 {
			clientBorderColor = colorSecondary
		}
		clientPanel := stylePanelTitled.
			BorderForeground(clientBorderColor).
			Width(listWidth - 4).
			Height(listHeight + 2). // +2 for border
			Render(clientContent)

		// Stack the two panels
		leftColumn := lipgloss.JoinVertical(lipgloss.Top, serverPanel, clientPanel)

		// 3. Right Top (Details)
		// Content width: rightWidth - 4 (Panel overhead)
		// Content height: detailsHeight - 3 (2 border, 1 title)
		detailsContentHeight := detailsHeight - 3
		if detailsContentHeight < 4 {
			detailsContentHeight = 4
		}
		detailsContent := renderEndpointDetails(m, rightWidth-4, detailsContentHeight)
		// Add margin below title to match Endpoints panel spacing
		detailsTitle := styleTitle.MarginBottom(1).Render("Endpoint Details")
		detailsWithTitle := detailsTitle + "\n" + detailsContent

		detailsBorderColor := colorSubtext
		if m.config != nil && len(m.config.Endpoints) > 0 {
			var item list.Item
			if m.activeEndpointPanel == 0 {
				item = m.serverEndpoints.SelectedItem()
			} else {
				item = m.clientEndpoints.SelectedItem()
			}
			if ep, ok := item.(endpointItem); ok {
				if m.engine != nil {
					st := m.engine.GetStatus(ep.ID)
					switch st {
					case types.StatusRunning:
						detailsBorderColor = colorSecondary // Orange
					case types.StatusCompleted:
						detailsBorderColor = colorSuccess // Green
					case types.StatusError:
						detailsBorderColor = colorError // Red
					}
				}
			}
		}

		// Use stylePanelTitled (no top padding) so title sits at the top like list title
		rightTop := stylePanelTitled.
			BorderForeground(detailsBorderColor).
			Width(rightWidth).     // Content width
			Height(detailsHeight). // -2 for thick border
			Render(detailsWithTitle)

		// 4. Right Bottom (Logs)
		// Panel height = logsHeight - 3 (2 border, 1 bottom padding)
		// Content space = logsHeight - 6 (3 panel overhead + 1 title + 1 margin + 1 viewport gap)
		logsContentHeight := logsHeight - 6
		if logsContentHeight < 2 {
			logsContentHeight = 2
		}

		m.logViewport.Width = rightWidth - 7 // Width minus padding, border, and scrollbar
		m.logViewport.Height = logsContentHeight

		logsColor := colorSubtext
		if m.activeView == 1 {
			logsColor = colorSecondary
		}

		// Add margin below title to match Endpoints panel spacing
		logsTitle := styleTitle.MarginBottom(1).Render("Logs")

		// Render viewport and scrollbar side by side
		viewportContent := m.logViewport.View()
		scrollbar := renderScrollbar(m.logViewport, logsContentHeight)

		// Use fixed-width scrollbar (1 char wide)
		scrollbarCol := scrollbarTrack.Width(1).Render(scrollbar)
		logsContent := lipgloss.JoinHorizontal(lipgloss.Top, viewportContent, scrollbarCol)
		logsContent = logsTitle + "\n" + logsContent

		logsStyle := stylePanelTitled.
			BorderForeground(logsColor)

		logsView := logsStyle.
			Width(rightWidth).
			Height(logsHeight - 2). // -2 for thick border
			Render(logsContent)

		rightBottom := logsView

		// 5. Combine Right Column
		rightColumn := lipgloss.JoinVertical(lipgloss.Top, rightTop, rightBottom)

		// 6. Combine Top Area
		topArea := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightColumn)

		// 7. Footer with border
		keyStyle := lipgloss.NewStyle().Foreground(colorSecondary).Bold(true)
		descStyle := lipgloss.NewStyle().Foreground(colorSubtext)
		sep := descStyle.Render(" • ")

		// Add Tab hint - match format: key in orange, description in gray
		tabHint := keyStyle.Render("<tab>") + descStyle.Render(" switch focus")

		var footer string
		if m.activeView == 0 {
			// Endpoints focused: arrows switch servers/clients, e, u, r, s
			arrowHint := keyStyle.Render("←/→") + descStyle.Render(" switch")
			footer = lipgloss.JoinHorizontal(lipgloss.Center,
				tabHint,
				sep,
				arrowHint,
				sep,
				keyStyle.Render("e"), descStyle.Render(" edit"),
				sep,
				keyStyle.Render("u"), descStyle.Render(" update"),
				sep,
				keyStyle.Render("r"), descStyle.Render(" run"),
				sep,
				keyStyle.Render("s"), descStyle.Render(" stop"),
				sep,
				keyStyle.Render("q"), descStyle.Render(" quit"),
			)
		} else {
			// Logs focused: e (editor), g (top), G (bottom)
			footer = lipgloss.JoinHorizontal(lipgloss.Center,
				tabHint,
				sep,
				keyStyle.Render("e"), descStyle.Render(" editor"),
				sep,
				keyStyle.Render("g"), descStyle.Render(" top"),
				sep,
				keyStyle.Render("G"), descStyle.Render(" bottom"),
				sep,
				keyStyle.Render("q"), descStyle.Render(" quit"),
			)
		}

		// Wrap footer in a thin border panel
		footerStyle := lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(colorSubtext).
			Padding(0, 1)

		footerView := footerStyle.
			Width(windowWidth - 2).
			Render(footer)

		content = lipgloss.JoinVertical(lipgloss.Top,
			appTitle,
			lipgloss.JoinVertical(lipgloss.Top, topArea, footerView),
		)
	}

	// Apply global window style
	return styleWindow.
		Width(m.width - 2). // Margin accounts for some
		Height(m.height - 2).
		Render(content)
}

func renderEndpointDetails(m Model, width, height int) string {
	if m.config == nil || len(m.config.Endpoints) == 0 {
		return "No endpoint selected"
	}

	var item list.Item
	if m.activeEndpointPanel == 0 {
		item = m.serverEndpoints.SelectedItem()
	} else {
		item = m.clientEndpoints.SelectedItem()
	}
	ep, ok := item.(endpointItem)
	if !ok {
		return "No endpoint selected"
	}

	// Get messages for this endpoint (both sent and received)
	// Use MessagesByFrom for sent messages, scan for received
	var endpointMessages []types.Message
	if m.config.MessagesByFrom != nil {
		endpointMessages = append(endpointMessages, m.config.MessagesByFrom[ep.ID]...)
	}
	// Also include messages sent TO this endpoint
	for _, msg := range m.config.Messages {
		if msg.To == ep.ID {
			endpointMessages = append(endpointMessages, msg)
		}
	}

	// Calculate available width for values
	// Width includes padding from styleDetails (0,1) -> -2 width available
	// Now we wrapped it in stylePanel (Border+Padding) -> -4 total overhead
	// view pass "width" which is rightWidth-4 (content width of panel).
	// But detailsContent uses styleDetails (Padding 0,1).
	// So inside the panel, we lose another 2 chars.
	contentWidth := width - 2
	if contentWidth < 0 {
		contentWidth = 0
	}

	// Label width 10. Gap 1.
	labelWidth := 10
	valueMaxWidth := contentWidth - labelWidth - 1
	if valueMaxWidth < 5 {
		valueMaxWidth = 5
	}

	// Helper to render label+value rows with truncation
	row := func(label, value string) string {
		if len(value) > valueMaxWidth {
			value = value[:valueMaxWidth-1] + "…"
		}
		return lipgloss.JoinHorizontal(lipgloss.Left,
			styleLabel.Render(label),
			styleValue.Render(value),
		)
	}

	header := lipgloss.JoinVertical(lipgloss.Left,
		row("ID:", fmt.Sprintf("%d", ep.ID)),
		row("Kind:", ep.Kind),
		row("Address:", ep.Address),
		row("Port:", fmt.Sprintf("%d", ep.Port)),
	)

	headerHeight := 4 // 4 rows

	messagesHeader := lipgloss.NewStyle().
		MarginTop(1).
		Foreground(colorSecondary).
		Bold(true).
		Render("Messages")

	// Calculate remaining height for messages
	// Height - Header(4) - MsgHeader(1 with marginTop(1))
	availMsgHeight := height - headerHeight - 3
	if availMsgHeight < 0 {
		availMsgHeight = 0
	}

	var msgContent string
	count := 0
	truncated := false

	for _, msg := range endpointMessages {
		if count >= availMsgHeight {
			msgContent += styleSubtext.Render("... and more ...")
			truncated = true
			break
		}

		var kindStr string
		if msg.Kind != "" {
			kindStr = fmt.Sprintf("[%s] ", strings.ToUpper(msg.Kind))
		}

		var line string
		if msg.From == ep.ID {
			line = fmt.Sprintf("→ %sto %d (+%d ms)", kindStr, msg.To, msg.TDelta)
		} else if msg.To == ep.ID {
			line = fmt.Sprintf("← %sfrom %d (+%d ms)", kindStr, msg.From, msg.TDelta)
		}

		if line != "" {
			// Truncate line if too long
			if len(line) > contentWidth {
				line = line[:contentWidth-1] + "…"
			}
			msgContent += line + "\n"
			count++
		}
	}

	if msgContent == "" {
		msgContent = styleSubtext.Render("No messages recorded for this endpoint.")
	}

	// Join all content and ensure it fills exactly the height
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		messagesHeader,
		msgContent,
	)

	// Handle truncation and padding
	lines := strings.Split(content, "\n")
	if truncated {
		// Already has "... and more ..." - just ensure we don't exceed height
		if len(lines) > height {
			lines = lines[:height]
		}
	} else {
		// Pad with empty lines to fill remaining height
		for len(lines) < height {
			lines = append(lines, "")
		}
		if len(lines) > height {
			lines = lines[:height]
		}
	}

	return strings.Join(lines, "\n")
}

// Additional style needed for subtext which I missed in styles.go
var styleSubtext = lipgloss.NewStyle().Foreground(colorSubtext)
