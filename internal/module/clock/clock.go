//go:build module_clock

package clock

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"moyu-assistant/internal/i18n"
	"moyu-assistant/internal/module"
)

func init() {
	module.Register(&ClockModule{})
}

// ClockModule is a placeholder for the world clock feature.
type ClockModule struct{}

func (m *ClockModule) Name() string        { return i18n.T("世界时钟", "World Clock") }
func (m *ClockModule) Description() string { return i18n.T("多时区时钟显示，实时更新", "Multi-timezone clock display, real-time update") }
func (m *ClockModule) Icon() fyne.Resource { return theme.HistoryIcon() }

func (m *ClockModule) CreateUI(w fyne.Window) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(i18n.T("🕐 世界时钟", "🕐 World Clock"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	desc := widget.NewLabel(i18n.T("此模块将实现多时区时钟显示功能，敬请期待。", "This module will implement multi-timezone clock display, coming soon."))
	desc.Alignment = fyne.TextAlignCenter

	return container.New(layout.NewCenterLayout(),
		container.NewVBox(title, desc),
	)
}

func (m *ClockModule) OnInit()    {}
func (m *ClockModule) OnDestroy() {}
