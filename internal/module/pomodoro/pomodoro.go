//go:build module_pomodoro

package pomodoro

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

func (m *PomodoroModule) Name() string { return i18n.T("番茄钟", "Pomodoro") }
func (m *PomodoroModule) Description() string {
	return i18n.T("25/5 分钟工作/休息计时器", "25/5 minutes work/rest timer")
}
func (m *PomodoroModule) Icon() fyne.Resource { return theme.MediaPlayIcon() }

func (m *PomodoroModule) CreateUI(w fyne.Window) fyne.CanvasObject {
	header := module.CreateHeader(i18n.T("🍅 番茄钟", "🍅 Pomodoro"), i18n.T("保持专注，提高效率", "Stay focused, improve efficiency"))
	return container.NewBorder(header, nil, nil, nil, container.NewCenter(widget.NewLabel(i18n.T("开发中...", "Under construction..."))))
}

func (m *PomodoroModule) OnInit()          {}
func (m *PomodoroModule) OnDestroy()       {}
func (m *PomodoroModule) Category() string { return "" }
