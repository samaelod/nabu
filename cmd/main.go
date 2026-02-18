package main

import (
	"log"
	"os"

	"github.com/samaelod/nabu/tui"
)

var version = "dev"

func main() {
	// Only create debug log in dev builds
	if version == "dev" {
		f, err := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err == nil {
			log.SetOutput(f)
		}
	}

	if err := tui.Run(version); err != nil {
		log.Fatal(err)
	}
}
