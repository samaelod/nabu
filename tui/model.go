package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"

	"github.com/samaelod/nabu/engine"
	"github.com/samaelod/nabu/types"
)

type screen int

const (
	screenSourceSelect screen = iota
	screenFilePicker
	screenLoading
	screenViewConfig
)

type sourceType int

const (
	sourcePCAP sourceType = iota
	sourceLua
)

type Model struct {
	screen screen
	source sourceType

	config *types.Config
	err    error

	// fileBrowser for selecting PCAP/Lua files
	fileBrowser FileBrowser

	// Separate endpoint lists for servers and clients
	serverEndpoints     list.Model
	clientEndpoints     list.Model
	activeEndpointPanel int // 0: servers, 1: clients

	width        int
	height       int
	selectedFile string

	menuCursor int // 0: PCAP, 1: Lua
	activeView int // 0: Endpoints List, 1: Logs Viewport

	version string

	engine      *engine.Engine
	logViewport viewport.Model
	logContent  string // cached log content for editor
}

const (
	minWindowWidth   = 80
	minWindowHeight  = 20
	defaultListWidth = 30
	minListWidth     = 20
	footerHeight     = 3
)
