package lua

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/samaelod/nabu/config"
	"github.com/samaelod/nabu/types"
)

// SaveToRecent saves the current configuration to a new file in the 'recent' directory.
// It uses the original filename as a base and appends an incrementing number.
// Returns the path to the newly created file.
func SaveToRecent(cfg *types.Config, originalPath string) (string, error) {
	// Load config to get recent directory
	appConfig, err := config.LoadDefault()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	recentDir := appConfig.RecentDir
	if recentDir == "" {
		recentDir = "recent"
	}

	// Create 'recent' directory if it doesn't exist
	if err := os.MkdirAll(recentDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create recent directory: %w", err)
	}

	// 2. Determine base name
	baseName := filepath.Base(originalPath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	// 3. Find unique filename
	// pattern: name_1.lua, name_2.lua, etc.
	// If original was "foo.pcap", we want "foo_1.lua"

	counter := 1
	var newPath string
	for {
		newFilename := fmt.Sprintf("%s_%d.lua", nameWithoutExt, counter)
		newPath = filepath.Join(recentDir, newFilename)

		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			break // Found a free name
		}
		counter++
	}

	// 4. Create and Write file
	f, err := os.Create(newPath)
	if err != nil {
		return "", fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	if strings.HasSuffix(originalPath, ".lua") {
		// If original is Lua, copy it directly to preserve comments/structure
		src, err := os.Open(originalPath)
		if err != nil {
			return "", fmt.Errorf("failed to open source lua file: %w", err)
		}
		defer src.Close()

		if _, err := io.Copy(f, src); err != nil {
			return "", fmt.Errorf("failed to copy lua content: %w", err)
		}
	} else {
		// If PCAP (or other), generate Lua from Config struct
		if err := WriteConfig(f, cfg); err != nil {
			return "", fmt.Errorf("failed to write config to lua: %w", err)
		}
	}

	return newPath, nil
}
