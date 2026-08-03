//go:build module_clock

package clock

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

func (m *ClockModule) Name() string { return i18n.T("世界时钟", "World Clock") }
func (m *ClockModule) Description() string {
	return i18n.T("多时区时钟显示，实时更新", "Multi-timezone clock display, real-time update")
}
func (m *ClockModule) Icon() fyne.Resource { return theme.HistoryIcon() }

func (m *ClockModule) CreateUI(w fyne.Window) fyne.CanvasObject {
	header := module.CreateHeader(i18n.T("🕐 世界时钟", "🕐 World Clock"), i18n.T("多时区时钟显示，实时更新", "Multi-timezone clock display"))
	return container.NewBorder(header, nil, nil, nil, container.NewCenter(widget.NewLabel(i18n.T("开发中...", "Under construction..."))))
}

func (m *ClockModule) OnInit()          {}
func (m *ClockModule) OnDestroy()       {}
func (m *ClockModule) Category() string { return "" }
