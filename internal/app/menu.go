package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"moyu-assistant/internal/i18n"
)

// applyTheme reads the user's preference and applies the corresponding theme.
func applyTheme(a fyne.App) {
	pref := a.Preferences().StringWithFallback("Theme", "Auto")
	var fallback fyne.Theme
	switch pref {
	case "Dark":
		fallback = theme.DarkTheme()
	case "Light":
		fallback = theme.LightTheme()
	default:
		fallback = theme.DefaultTheme()
	}
	a.Settings().SetTheme(&cjkTheme{fallback: fallback})
}

// setupMenu creates and sets the main application menu.
func setupMenu(a fyne.App, w fyne.Window) {
	themeAutoItem := fyne.NewMenuItem(i18n.T("自动 (跟随系统)", "Auto (System)"), func() {
		a.Preferences().SetString("Theme", "Auto")
		applyTheme(a)
	})
	themeDarkItem := fyne.NewMenuItem(i18n.T("暗色", "Dark"), func() {
		a.Preferences().SetString("Theme", "Dark")
		applyTheme(a)
	})
	themeLightItem := fyne.NewMenuItem(i18n.T("亮色", "Light"), func() {
		a.Preferences().SetString("Theme", "Light")
		applyTheme(a)
	})

	themeMenu := fyne.NewMenu(i18n.T("主题", "Theme"), themeAutoItem, themeDarkItem, themeLightItem)
	settingsMenu := fyne.NewMenu(i18n.T("设置", "Settings"), fyne.NewMenuItem(i18n.T("主题", "Theme"), nil))
	// Fyne menus don't support submenus directly in all versions, but let's check.
	// Actually, Fyne supports submenus via ChildMenu.
	settingsMenu.Items[0].ChildMenu = themeMenu

	mainMenu := fyne.NewMainMenu(settingsMenu)
	w.SetMainMenu(mainMenu)
}
