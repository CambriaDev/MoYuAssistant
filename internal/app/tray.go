package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
)

// setupTray configures the system tray icon and menu, and intercepts the
// window close event to hide the window (minimize to tray) instead of quitting.
func setupTray(a fyne.App, w fyne.Window) {
	// Only desktop platforms support system tray
	desk, ok := a.(desktop.App)
	if !ok {
		return
	}

	// Set tray icon (using built-in icon for now; replace with custom icon later)
	desk.SetSystemTrayIcon(theme.ComputerIcon())

	// Build the tray right-click menu
	menu := fyne.NewMenu(appTitle,
		fyne.NewMenuItem("显示主窗口", func() {
			w.Show()
			w.RequestFocus()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("退出", func() {
			a.Quit()
		}),
	)
	desk.SetSystemTrayMenu(menu)

	// Intercept window close: hide to tray instead of quitting
	w.SetCloseIntercept(func() {
		w.Hide()
	})
}
