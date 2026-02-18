package tui

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/samaelod/nabu/config"
	"github.com/samaelod/nabu/engine"
	"github.com/samaelod/nabu/lua"
	"github.com/samaelod/nabu/pcapreader"
	"github.com/samaelod/nabu/types"
)

func setupSessionLog(loadedFilePath string) {
	baseName := filepath.Base(loadedFilePath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	logFilename := fmt.Sprintf("%s.log", nameWithoutExt)
	logPath := filepath.Join("logs", logFilename)

	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Printf("Failed to create logs directory: %v", err)
		return
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
		return
	}

	log.SetOutput(f)
	log.Printf("Session started with file: %s", loadedFilePath)
}

func openLogsInEditor(logContent string) tea.Cmd {
	// Create temp file first
	f, err := os.CreateTemp("", "nabu-logs-*.log")
	if err != nil {
		return func() tea.Msg { return logErrorMsg{err} }
	}

	_, err = f.WriteString(logContent)
	if err != nil {
		return func() tea.Msg { return logErrorMsg{err} }
	}
	f.Close()
	tempPath := f.Name()

	// Open in editor using tea.ExecProcess
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
	}
	c := exec.Command(editor, tempPath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		// Clean up temp file after editor closes
		os.Remove(tempPath)
		return nil
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// H - 2 (window) - 1 (safety) - 4 (panel border+padding) = H - 7
		// Width: 1/3 for list, 2/3 for preview
		// Sync logic with View
		availWidth := msg.Width - 4
		listWidth := 30
		if listWidth > availWidth/3 {
			listWidth = availWidth / 3
		}
		if listWidth < 20 {
			listWidth = 20
		}

		m.fileBrowser.SetSize(listWidth-4, msg.Height-7)
		if m.screen == screenViewConfig {
			// Set size for both endpoint lists
			listHeight := (msg.Height - 7) / 2
			m.serverEndpoints.SetSize(listWidth-3, listHeight)
			m.clientEndpoints.SetSize(listWidth-3, listHeight)

			// Update Log Viewport Size
			// Logic must match View():
			// availHeight = H - 4 - 1 = H - 5
			// detailsHeight = 12 (max)
			// logsHeight = availHeight - detailsHeight
			// panelHeight = logsHeight - 1
			// vpHeight = panelHeight - 4
			// Total subtraction: 5 + 12 + 1 + 4 = 22

			availHeight := msg.Height - 5
			logsHeight := availHeight / 2 // Default 50%
			if m.activeView == 1 {
				logsHeight = int(float64(availHeight) * 0.7)
			} else {
				logsHeight = int(float64(availHeight) * 0.4)
			}
			detailsHeight := availHeight - logsHeight
			if detailsHeight < 10 {
				logsHeight = availHeight - 10
			}

			vpHeight := logsHeight - 7 // -1 (Title) - 4 (Panel Border/Pad)
			if vpHeight < 0 {
				vpHeight = 0
			}

			m.logViewport.Width = availWidth - listWidth - 4 - 2 // -2 for panel border
			m.logViewport.Height = vpHeight
		}

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
	}

	// Handle global messages (like config loaded) regardless of screen
	switch msg := msg.(type) {
	case configLoadedMsg:
		m.config = msg.config
		if msg.path != "" {
			m.selectedFile = msg.path
			setupSessionLog(msg.path)
		}

		var servers, clients []types.Endpoint
		for _, ep := range m.config.Endpoints {
			if ep.Kind == "server" {
				servers = append(servers, ep)
			} else {
				clients = append(clients, ep)
			}
		}
		sort.Slice(servers, func(i, j int) bool { return servers[i].ID < servers[j].ID })
		sort.Slice(clients, func(i, j int) bool { return clients[i].ID < clients[j].ID })

		// Create server list
		serverItems := []list.Item{}
		for _, ep := range servers {
			serverItems = append(serverItems, endpointItem(ep))
		}
		m.serverEndpoints = list.New(
			serverItems,
			endpointsDelegate{},
			30,
			(m.height-7)/2,
		)
		m.serverEndpoints.SetShowHelp(false)
		m.serverEndpoints.SetShowTitle(false)

		// Create client list
		clientItems := []list.Item{}
		for _, ep := range clients {
			clientItems = append(clientItems, endpointItem(ep))
		}
		m.clientEndpoints = list.New(
			clientItems,
			endpointsDelegate{},
			30,
			(m.height-7)/2,
		)
		m.clientEndpoints.SetShowHelp(false)
		m.clientEndpoints.SetShowTitle(false)

		// Default to servers panel
		m.activeEndpointPanel = 0

		// Load app config
		appConfig, err := config.LoadDefault()
		if err != nil {
			m.err = fmt.Errorf("failed to load config: %w", err)
			return m, nil
		}

		logsDir := appConfig.LogsDir
		if logsDir == "" {
			logsDir = "logs"
		}

		logPath := ""
		if msg.path != "" {
			baseName := filepath.Base(msg.path)
			ext := filepath.Ext(baseName)
			nameWithoutExt := strings.TrimSuffix(baseName, ext)
			logPath = filepath.Join(logsDir, nameWithoutExt+".log")
		}
		m.engine = engine.NewEngine(m.config, logPath, appConfig.LogLines, m.config.Globals.Timeout, m.config.Globals.Delay)

		m.screen = screenViewConfig

		// Init Viewport
		// Note: Actual size set in view or resize
		m.logViewport = viewport.New(10, 10)
		m.logViewport.SetContent("Ready to run simulation...")
		m.logContent = "Ready to run simulation..."

		return m, nil

	case editorFinishedMsg:
		// Stop all running endpoints before reloading config
		if m.engine != nil {
			m.engine.StopAll()
		}
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		return m, loadConfigCmd(sourceLua, m.selectedFile, false)

	case logMsg:
		// Get all logs from engine logger (handles file I/O internally)
		if m.engine != nil && m.engine.Log != nil {
			m.logContent = m.engine.Log.ReadAll()
			m.logViewport.SetContent(m.logContent)
			m.logViewport.GotoBottom()
			return m, waitForLog(m.engine.Log)
		}
		return m, nil
	}

	switch m.screen {

	case screenSourceSelect:
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "up", "k", "left", "h":
				m.menuCursor--
				if m.menuCursor < 0 {
					m.menuCursor = 1
				}
			case "down", "j", "right", "l":
				m.menuCursor++
				if m.menuCursor > 1 {
					m.menuCursor = 0
				}
			case "enter":
				switch m.menuCursor {
				case 0:
					m.source = sourcePCAP
					m.fileBrowser = NewFileBrowser([]string{".pcap", ".cap", ".pcapng"})
				case 1:
					m.source = sourceLua
					m.fileBrowser = NewFileBrowser([]string{".lua"})
				}

				// Resize filepicker (Height - WindowChrome(3) - PanelChrome(4) = -7)
				listWidth := m.width / 3
				m.fileBrowser.SetSize(listWidth-4, m.height-7) // Adjusted for split view
				m.screen = screenFilePicker
				return m, nil // Browser doesn't need Init cmd
			}
		}
		return m, nil

	case screenFilePicker:
		var cmd tea.Cmd
		m.fileBrowser, cmd = m.fileBrowser.Update(msg)

		// Check if a file was confirmed (Enter key on a file item)
		if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "enter" {
			item := m.fileBrowser.List.SelectedItem()
			if item != nil {
				fi, ok := item.(fileItem)
				if !ok || fi.isDir {
					return m, nil
				}

				// Verify extension match
				allowed := false
				// Actually we should use the same logic as browser delegate or just check suffix
				// m.fileBrowser.AllowedTypes is reliable
				for _, ext := range m.fileBrowser.AllowedTypes {
					if len(fi.name) >= len(ext) && fi.name[len(fi.name)-len(ext):] == ext {
						allowed = true
						break
					}
				}

				if !allowed {
					// Ignore selection of invalid file types
					return m, nil
				}

				// It's a file, let's load it
				path := fi.path
				m.screen = screenLoading
				log.Println("\n  You selected: " + path + "\n")
				return m, loadConfigCmd(m.source, path, true)
			}
		}

		return m, cmd

	case screenLoading:
		switch msg := msg.(type) {
		case loadedMsg:
			var servers, clients []types.Endpoint
			for _, ep := range m.config.Endpoints {
				if ep.Kind == "server" {
					servers = append(servers, ep)
				} else {
					clients = append(clients, ep)
				}
			}
			sort.Slice(servers, func(i, j int) bool { return servers[i].ID < servers[j].ID })
			sort.Slice(clients, func(i, j int) bool { return clients[i].ID < clients[j].ID })

			// Create server list
			serverItems := []list.Item{}
			for _, ep := range servers {
				serverItems = append(serverItems, endpointItem(ep))
			}
			m.serverEndpoints = list.New(
				serverItems,
				endpointsDelegate{},
				30,
				(m.height-2)/2,
			)
			m.serverEndpoints.SetShowTitle(false)
			m.serverEndpoints.SetShowHelp(false)

			// Create client list
			clientItems := []list.Item{}
			for _, ep := range clients {
				clientItems = append(clientItems, endpointItem(ep))
			}
			m.clientEndpoints = list.New(
				clientItems,
				endpointsDelegate{},
				30,
				(m.height-2)/2,
			)
			m.clientEndpoints.SetShowTitle(false)
			m.clientEndpoints.SetShowHelp(false)

			// Default to servers panel
			m.activeEndpointPanel = 0
			m.screen = screenViewConfig
		case errMsg:
			m.err = msg.err
		}
	}

	if m.screen == screenViewConfig {
		var cmd tea.Cmd
		var cmds []tea.Cmd

		// Custom Key Handling for View Config
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "tab", "shift+tab":
				m.activeView++
				if m.activeView > 1 {
					m.activeView = 0
				}
				// Force resize update to animate the panel size change immediately
				// Simulate a resize with current dimensions
				return m, func() tea.Msg { return tea.WindowSizeMsg{Width: m.width, Height: m.height} }

			case "e":
				if m.activeView == 0 {
					// Edit config (Endpoints focused)
					editor := os.Getenv("EDITOR")
					if editor == "" {
						editor = "nano"
					}
					c := exec.Command(editor, m.selectedFile)
					return m, tea.ExecProcess(c, func(err error) tea.Msg {
						return editorFinishedMsg{err}
					})
				} else {
					// Open logs in editor (Logs focused)
					return m, openLogsInEditor(m.logContent)
				}
			case "u":
				if m.activeView == 0 {
					return m, loadConfigCmd(sourceLua, m.selectedFile, false)
				}
			case "left", "h":
				if m.activeView == 0 {
					// Switch to server panel
					if m.activeEndpointPanel == 1 {
						// Sync selection: save client selection, restore server selection
						m.activeEndpointPanel = 0
					}
				}
			case "right", "l":
				if m.activeView == 0 {
					// Switch to client panel
					if m.activeEndpointPanel == 0 {
						m.activeEndpointPanel = 1
					}
				}
			case "r":
				if m.activeView == 0 {
					if m.engine != nil && m.config != nil && len(m.config.Endpoints) > 0 {
						var item list.Item
						if m.activeEndpointPanel == 0 {
							item = m.serverEndpoints.SelectedItem()
						} else {
							item = m.clientEndpoints.SelectedItem()
						}
						if ep, ok := item.(endpointItem); ok {
							m.engine.StartEndpoint(ep.ID)
							return m, waitForLog(m.engine.Log)
						}
					}
				}
			case "s":
				if m.activeView == 0 {
					if m.engine != nil && m.config != nil && len(m.config.Endpoints) > 0 {
						var item list.Item
						if m.activeEndpointPanel == 0 {
							item = m.serverEndpoints.SelectedItem()
						} else {
							item = m.clientEndpoints.SelectedItem()
						}
						if ep, ok := item.(endpointItem); ok {
							m.engine.StopEndpoint(ep.ID)
						}
					}
				}
			case "g":
				if m.activeView == 1 {
					m.logViewport.GotoTop()
				}
			case "G":
				if m.activeView == 1 {
					m.logViewport.GotoBottom()
				}
			}
		}

		// Conditional Update based on Focus
		if m.activeView == 0 {
			// Update the active endpoint list
			if m.activeEndpointPanel == 0 {
				m.serverEndpoints, cmd = m.serverEndpoints.Update(msg)
			} else {
				m.clientEndpoints, cmd = m.clientEndpoints.Update(msg)
			}
			cmds = append(cmds, cmd)
		} else {
			// Update Logs Viewport only when focused
			m.logViewport, cmd = m.logViewport.Update(msg)
			cmds = append(cmds, cmd)
		}

		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func loadConfigCmd(source sourceType, path string, saveCopy bool) tea.Cmd {
	return func() tea.Msg {
		var (
			cfg *types.Config
			err error
		)

		switch source {
		case sourcePCAP:
			cfg, err = pcapreader.ReadPCAP(path)
		case sourceLua:
			cfg, err = lua.ReadLuaConfig(path)
		}

		if err != nil {
			return errMsg{err}
		}

		// Validate the config
		if err := lua.ValidateConfig(cfg); err != nil {
			return errMsg{fmt.Errorf("invalid config: %w", err)}
		}

		finalPath := path
		if saveCopy {
			newPath, err := lua.SaveToRecent(cfg, path)
			if err != nil {
				return errMsg{err}
			}
			finalPath = newPath
		}

		return configLoadedMsg{config: cfg, path: finalPath}
	}
}

type configLoadedMsg struct {
	config *types.Config
	path   string
}

type loadedMsg struct{}
type errMsg struct{ err error }
type editorFinishedMsg struct{ err error }
type logErrorMsg struct{ err error }
type logMsg string

func waitForLog(logger *engine.Logger) tea.Cmd {
	return func() tea.Msg {
		ch := logger.Chan()
		if ch == nil {
			return nil
		}
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return logMsg(msg)
	}
}
