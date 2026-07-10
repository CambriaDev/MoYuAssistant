//go:build module_pomodoro

package pomodoro

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
	module.Register(&PomodoroModule{})
}

// PomodoroModule is a placeholder for the pomodoro timer feature.
type PomodoroModule struct{}

func (m *PomodoroModule) Name() string        { return i18n.T("番茄钟", "Pomodoro") }
func (m *PomodoroModule) Description() string { return i18n.T("25/5 分钟工作/休息计时器", "25/5 minutes work/rest timer") }
func (m *PomodoroModule) Icon() fyne.Resource { return theme.MediaPlayIcon() }

func (m *PomodoroModule) CreateUI(w fyne.Window) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(i18n.T("🍅 番茄钟", "🍅 Pomodoro"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	desc := widget.NewLabel(i18n.T("此模块将实现番茄工作法计时器功能，敬请期待。", "This module will implement Pomodoro timer, coming soon."))
	desc.Alignment = fyne.TextAlignCenter

	return container.New(layout.NewCenterLayout(),
		container.NewVBox(title, desc),
	)
}

func (m *PomodoroModule) OnInit()    {}
func (m *PomodoroModule) OnDestroy() {}
