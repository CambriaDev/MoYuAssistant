package main

import (
	"moyu-assistant/internal/app"

	// Blank import to trigger module registration via init() functions.
	// Which modules are actually compiled depends on the build tags used.
	_ "moyu-assistant/internal/imports"
)

func main() {
	app.Run()
}
