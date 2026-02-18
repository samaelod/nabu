package lua

import (
	"fmt"

	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"

	"github.com/samaelod/nabu/types"
)

func ReadLuaConfig(path string) (*types.Config, error) {
	L := lua.NewState()
	defer L.Close()

	// Execute Lua file
	if err := L.DoFile(path); err != nil {
		return nil, err
	}

	// Lua file returns config table
	lv := L.Get(-1)
	table, ok := lv.(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("lua file did not return a table")
	}

	var cfg types.Config

	// Map Lua table → Go struct
	if err := gluamapper.Map(table, &cfg); err != nil {
		return nil, err
	}

	// Validate the config
	if err := ValidateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Index messages for O(1) lookup
	cfg.IndexMessages()

	return &cfg, nil
}

func ValidateConfig(cfg *types.Config) error {
	endpoints := make(map[int]bool)

	for _, ep := range cfg.Endpoints {
		endpoints[ep.ID] = true
	}

	for i, msg := range cfg.Messages {
		if !endpoints[msg.From] {
			return fmt.Errorf("message %d: invalid from id %d", i, msg.From)
		}
		if !endpoints[msg.To] {
			return fmt.Errorf("message %d: invalid to id %d", i, msg.To)
		}
	}

	return nil
}
