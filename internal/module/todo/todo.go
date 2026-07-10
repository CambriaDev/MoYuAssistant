//go:build module_todo

package todo

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"moyu-assistant/internal/module"
)

func init() {
	module.Register(&TodoModule{})
}

// TodoModule is a placeholder for the todo list feature.
type TodoModule struct{}

func (m *TodoModule) Name() string        { return "待办事项" }
func (m *TodoModule) Description() string { return "添加、删除、勾选待办任务" }
func (m *TodoModule) Icon() fyne.Resource { return theme.DocumentCreateIcon() }

func (m *TodoModule) CreateUI(w fyne.Window) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("✅ 待办事项", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	desc := widget.NewLabel("此模块将实现待办事项管理功能，敬请期待。")
	desc.Alignment = fyne.TextAlignCenter

	return container.New(layout.NewCenterLayout(),
		container.NewVBox(title, desc),
	)
}

func (m *TodoModule) OnInit()    {}
func (m *TodoModule) OnDestroy() {}
