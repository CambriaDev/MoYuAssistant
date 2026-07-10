package module

import "fyne.io/fyne/v2"

// Module defines the interface that all functional modules must implement.
// Each module is an independent feature unit that can be included or excluded
// at compile time via Go build tags.
type Module interface {
	// Name returns the display name of the module (shown in the tab bar).
	Name() string
	// Description returns a short description of what the module does.
	Description() string
	// Icon returns the icon resource for the module tab.
	Icon() fyne.Resource
	// CreateUI builds and returns the module's UI widget tree.
	CreateUI(w fyne.Window) fyne.CanvasObject
	// OnInit is called once when the application starts (after registration).
	OnInit()
	// OnDestroy is called once when the application is shutting down.
	OnDestroy()
}

// registry holds all registered modules.
var registry []Module

// Register adds a module to the global registry.
// This should be called from init() functions in module packages.
func Register(m Module) {
	registry = append(registry, m)
}

// All returns all registered modules in registration order.
func All() []Module {
	return registry
}

// Count returns the number of registered modules.
func Count() int {
	return len(registry)
}
