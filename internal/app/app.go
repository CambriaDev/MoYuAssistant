package app

import (
	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"moyu-assistant/internal/i18n"
	"moyu-assistant/internal/module"
)

const (
	appID = "com.strada.moyu-assistant"
)

// Run initializes and starts the application.
func Run() {
	appTitle := i18n.T("摸鱼助手", "MoYu Assistant")

	a := fyneapp.NewWithID(appID)
	a.Settings().SetTheme(&cjkTheme{fallback: theme.DarkTheme()})

	w := a.NewWindow(appTitle)
	w.Resize(fyne.NewSize(800, 560))
	w.CenterOnScreen()

	// Setup system tray (minimize-to-tray behavior)
	setupTray(a, w, appTitle)

	// Initialize all registered modules
	modules := module.All()
	for _, m := range modules {
		m.OnInit()
	}

	// Build and set the main UI
	w.SetContent(buildMainUI(w, modules))
	w.ShowAndRun()

	// Cleanup on exit
	for _, m := range modules {
		m.OnDestroy()
	}
}

// buildMainUI constructs the main window layout.
// If modules are loaded, it shows a tabbed interface with one tab per module.
// If no modules are loaded, it shows a helpful placeholder message.
func buildMainUI(w fyne.Window, modules []module.Module) fyne.CanvasObject {
	if len(modules) == 0 {
		return buildEmptyState()
	}

	tabs := container.NewAppTabs()
	for _, m := range modules {
		tab := container.NewTabItemWithIcon(m.Name(), m.Icon(), m.CreateUI(w))
		tabs.Append(tab)
	}
	tabs.SetTabLocation(container.TabLocationLeading)

	return tabs
}

// buildEmptyState creates the UI shown when no modules are compiled in.
func buildEmptyState() fyne.CanvasObject {
	title := widget.NewLabelWithStyle(
		i18n.T("📦 没有加载任何功能模块", "📦 No functional modules loaded"),
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	hint := widget.NewLabelWithStyle(
		i18n.T("请使用 build tags 编译所需模块，例如：\ngo build -tags \"module_clock module_todo\" -o moyu.exe .", "Please compile with build tags, e.g.:\ngo build -tags \"module_clock module_todo\" -o moyu.exe ."),
		fyne.TextAlignCenter,
		fyne.TextStyle{Monospace: true},
	)

	return container.New(layout.NewCenterLayout(),
		container.NewVBox(title, widget.NewSeparator(), hint),
	)
}
