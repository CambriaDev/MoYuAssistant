//go:build module_pomodoro

package pomodoro

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"moyu-assistant/internal/module"
)

func init() {
	module.Register(&PomodoroModule{})
}

// PomodoroModule is a placeholder for the pomodoro timer feature.
type PomodoroModule struct{}

func (m *PomodoroModule) Name() string        { return "番茄钟" }
func (m *PomodoroModule) Description() string { return "25/5 分钟工作/休息计时器" }
func (m *PomodoroModule) Icon() fyne.Resource { return theme.MediaPlayIcon() }

func (m *PomodoroModule) CreateUI(w fyne.Window) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("🍅 番茄钟", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	desc := widget.NewLabel("此模块将实现番茄工作法计时器功能，敬请期待。")
	desc.Alignment = fyne.TextAlignCenter

	return container.New(layout.NewCenterLayout(),
		container.NewVBox(title, desc),
	)
}

func (m *PomodoroModule) OnInit()    {}
func (m *PomodoroModule) OnDestroy() {}
