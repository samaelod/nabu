package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FileBrowser struct {
	List           list.Model
	CurrentDir     string
	Selected       string
	PreviewContent string
	Height         int
	Width          int
	Err            error
	AllowedTypes   []string
}

type fileItem struct {
	name  string
	path  string
	isDir bool
	info  os.FileInfo
}

func (i fileItem) Title() string {
	if i.isDir {
		return i.name + "/"
	}
	return i.name
}
func (i fileItem) Description() string {
	if i.isDir {
		return "Directory"
	}
	return fmt.Sprintf("File • %d bytes", i.info.Size())
}
func (i fileItem) FilterValue() string { return i.name }

// Custom Delegate for Highlighting
type browserDelegate struct {
	allowedTypes []string
}

func (d browserDelegate) Height() int                               { return 1 }
func (d browserDelegate) Spacing() int                              { return 0 }
func (d browserDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d browserDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(fileItem)
	if !ok {
		return
	}

	str := i.name
	if i.isDir {
		str += "/"
	}

	var style lipgloss.Style
	isSelected := index == m.Index()

	if isSelected {
		style = styleSelected.Copy().Foreground(colorSecondary)
		str = "> " + str
	} else {
		// Unselected Logic
		if i.isDir {
			style = lipgloss.NewStyle().Foreground(colorText).Bold(true)
		} else {
			// Check extension
			allowed := false
			nameLower := strings.ToLower(i.name)
			for _, ext := range d.allowedTypes {
				if strings.HasSuffix(nameLower, strings.ToLower(ext)) {
					allowed = true
					break
				}
			}
			if allowed {
				style = lipgloss.NewStyle().Foreground(colorPrimary)
			} else {
				style = lipgloss.NewStyle().Foreground(colorSubtext).Faint(true)
			}
		}
		str = "  " + str
	}

	fmt.Fprint(w, style.Render(str))
}

func NewFileBrowser(allowedTypes []string) FileBrowser {
	cwd, _ := os.Getwd()

	// Create list with custom delegate
	delegate := browserDelegate{allowedTypes: allowedTypes}
	l := list.New([]list.Item{}, delegate, 0, 0)

	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = styleTitle

	fb := FileBrowser{
		List:         l,
		CurrentDir:   cwd,
		AllowedTypes: allowedTypes,
	}
	fb.refreshDir()
	return fb
}

func (fb *FileBrowser) refreshDir() {
	entries, err := os.ReadDir(fb.CurrentDir)
	if err != nil {
		fb.Err = err
		return
	}

	items := []list.Item{}

	// Add ".." if not root
	if filepath.Dir(fb.CurrentDir) != fb.CurrentDir {
		items = append(items, fileItem{name: "..", path: filepath.Dir(fb.CurrentDir), isDir: true})
	}

	// Sort: Dirs first, then files
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() && !entries[j].IsDir() {
			return true
		}
		if !entries[i].IsDir() && entries[j].IsDir() {
			return false
		}
		return entries[i].Name() < entries[j].Name()
	})

	for _, e := range entries {
		// Skip hidden
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}

		info, _ := e.Info()
		items = append(items, fileItem{
			name:  e.Name(),
			path:  filepath.Join(fb.CurrentDir, e.Name()),
			isDir: e.IsDir(),
			info:  info,
		})
	}

	fb.List.SetItems(items)
	fb.updatePreview()
}

func (fb *FileBrowser) HasValidFilesInDir(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		for _, allowed := range fb.AllowedTypes {
			if ext == strings.ToLower(allowed) {
				return true
			}
		}
	}
	return false
}

func (fb *FileBrowser) SelectedHasValidExtension() bool {
	if fb.Selected == "" {
		return false
	}
	ext := strings.ToLower(filepath.Ext(fb.Selected))
	for _, allowed := range fb.AllowedTypes {
		if ext == strings.ToLower(allowed) {
			return true
		}
	}
	return false
}

func (fb *FileBrowser) updatePreview() {
	item := fb.List.SelectedItem()
	if item == nil {
		fb.PreviewContent = ""
		return
	}

	fi := item.(fileItem)
	if fi.isDir {
		fb.PreviewContent = fmt.Sprintf("Directory: %s\n\nItems: %d", fi.name, 0) // Simplified
		return
	}

	fb.Selected = fi.path

	// Case-insensitive util
	nameLower := strings.ToLower(fi.name)

	// Check if allowed
	allowed := false
	for _, ext := range fb.AllowedTypes {
		if strings.HasSuffix(nameLower, strings.ToLower(ext)) {
			allowed = true
			break
		}
	}

	if !allowed {
		fb.PreviewContent = "File type not supported."
		return
	}

	var contentStr string

	if strings.HasSuffix(nameLower, ".lua") {
		content, err := os.ReadFile(fi.path)
		if err != nil {
			contentStr = "Error reading file"
		} else {
			contentStr = string(content)
		}
	} else if strings.HasSuffix(nameLower, ".pcap") || strings.HasSuffix(nameLower, ".pcapng") || strings.HasSuffix(nameLower, ".cap") {
		contentStr = fmt.Sprintf("Capture file\nSize: %d bytes\n\nPreview not available for binary files.", fi.info.Size())
	} else {
		contentStr = "Preview unavailable for this file type."
	}

	// TRUNCATION LOGIC
	// Ensure we don't exceed visual height
	lines := strings.Split(contentStr, "\n")
	maxLines := fb.Height
	if maxLines <= 0 {
		maxLines = 10
	}

	if len(lines) > maxLines {
		contentStr = strings.Join(lines[:maxLines], "\n") + "\n... (truncated)"
	} else {
		contentStr = strings.Join(lines, "\n")
	}

	fb.PreviewContent = contentStr
}

func (fb FileBrowser) Update(msg tea.Msg) (FileBrowser, tea.Cmd) {
	var cmd tea.Cmd
	fb.List, cmd = fb.List.Update(msg)

	// Check for selection change (cursor moved)
	// Bubbles list doesn't emit a distinct msg for this easily, so we just update preview every time
	// Optimization: Check if index changed
	fb.updatePreview()

	// Handle navigation
	if msg, ok := msg.(tea.KeyMsg); ok {
		if msg.String() == "enter" {
			item := fb.List.SelectedItem()
			if item != nil {
				fi, ok := item.(fileItem)
				if !ok {
					return fb, cmd
				}
				if fi.isDir {
					fb.CurrentDir = fi.path
					fb.refreshDir()
					fb.List.ResetSelected()
				}
				// If file, parent will handle it by checking fb.Selected
			}
		}
		if msg.String() == "backspace" || msg.String() == "left" {
			parent := filepath.Dir(fb.CurrentDir)
			if parent != fb.CurrentDir {
				fb.CurrentDir = parent
				fb.refreshDir()
				fb.List.ResetSelected()
			}
		}
	}

	return fb, cmd
}

func (fb *FileBrowser) SetSize(width, height int) {
	fb.Width = width
	fb.Height = height
	fb.List.SetSize(width, height)
}

func (fb FileBrowser) View() string {
	return fb.List.View()
}
