package lua

import (
	"fmt"
	"io"

	"github.com/samaelod/nabu/types"
)

func WriteConfig(w io.Writer, cfg *types.Config) error {
	fmt.Fprintln(w, "local config = {}")
	fmt.Fprintln(w)

	// Globals
	fmt.Fprintln(w, "-- GLOBALS ----------------------------------------")
	fmt.Fprintln(w, "config.globals = {")
	fmt.Fprintf(w, "\tprotocol = %q,\n", cfg.Globals.Protocol)
	fmt.Fprintf(w, "\tplay_mode = %q,\n", cfg.Globals.PlayMode)
	fmt.Fprintf(w, "\ttimeout = %d,\n", cfg.Globals.Timeout)
	fmt.Fprintf(w, "\tdelay = %d,\n", cfg.Globals.Delay)
	fmt.Fprintln(w, "}")
	fmt.Fprintln(w)

	// Endpoints
	fmt.Fprintln(w, "-- ENDPOINTS --------------------------------------")
	fmt.Fprintln(w, "config.endpoints = {")
	for _, ep := range cfg.Endpoints {
		fmt.Fprintln(w, "\t{")
		fmt.Fprintf(w, "\t\tid = %d,\n", ep.ID)
		fmt.Fprintf(w, "\t\tkind = %q,\n", ep.Kind)
		fmt.Fprintf(w, "\t\taddress = %q,\n", ep.Address)
		fmt.Fprintf(w, "\t\tport = %d,\n", ep.Port)
		fmt.Fprintln(w, "\t},")
	}
	fmt.Fprintln(w, "}")
	fmt.Fprintln(w)

	// Messages
	fmt.Fprintln(w, "-- MESSAGES ----------------------------------------")
	fmt.Fprintln(w, "config.messages = {")
	for _, m := range cfg.Messages {
		fmt.Fprintln(w, "\t{")
		fmt.Fprintf(w, "\t\tfrom = %d,\n", m.From)
		fmt.Fprintf(w, "\t\tto = %d,\n", m.To)
		fmt.Fprintf(w, "\t\tkind = %q,\n", m.Kind)
		fmt.Fprintf(w, "\t\tvalue = %q,\n", m.Value)
		fmt.Fprintf(w, "\t\tt_delta = %d,\n", m.TDelta)
		fmt.Fprintln(w, "\t},")
	}
	fmt.Fprintln(w, "}")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "return config")

	return nil
}
