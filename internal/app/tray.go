package app

import (
	"bytes"
	"image"
	"image/color"
	"image/png"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"moyu-assistant/internal/i18n"
)

// setupTray configures the system tray icon and menu, and intercepts the
// window close event to hide the window (minimize to tray) instead of quitting.
func setupTray(a fyne.App, w fyne.Window, appTitle string) {
	// Only desktop platforms support system tray
	desk, ok := a.(desktop.App)
	if !ok {
		return
	}

	// Build the tray right-click menu
	menu := fyne.NewMenu(appTitle,
		fyne.NewMenuItem(i18n.T("显示主窗口", "Show Main Window"), func() {
			w.Show()
			w.RequestFocus()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem(i18n.T("退出", "Quit"), func() {
			a.Quit()
		}),
	)
	desk.SetSystemTrayMenu(menu)

	// Intercept window close: hide to tray instead of quitting
	w.SetCloseIntercept(func() {
		w.Hide()
	})
}

func newTrayIcon() fyne.Resource {
	const iconSize = 32

	img := image.NewNRGBA(image.Rect(0, 0, iconSize, iconSize))
	background := color.NRGBA{R: 0x1F, G: 0x29, B: 0x37, A: 0xFF}
	highlight := color.NRGBA{R: 0x22, G: 0xC5, B: 0x5E, A: 0xFF}
	foreground := color.NRGBA{R: 0xF9, G: 0xFA, B: 0xFB, A: 0xFF}

	for y := 0; y < iconSize; y++ {
		for x := 0; x < iconSize; x++ {
			img.SetNRGBA(x, y, background)
		}
	}

	for y := 5; y < 27; y++ {
		for x := 5; x < 27; x++ {
			img.SetNRGBA(x, y, highlight)
		}
	}

	for y := 9; y < 23; y++ {
		for x := 9; x < 13; x++ {
			img.SetNRGBA(x, y, foreground)
		}
	}

	for y := 16; y < 20; y++ {
		for x := 13; x < 23; x++ {
			img.SetNRGBA(x, y, foreground)
		}
	}

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		return nil
	}

	return fyne.NewStaticResource("tray-icon.png", buffer.Bytes())
}
