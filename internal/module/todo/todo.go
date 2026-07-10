//go:build module_todo

package todo

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
	module.Register(&TodoModule{})
}

// TodoModule is a placeholder for the todo list feature.
type TodoModule struct{}

func (m *TodoModule) Name() string        { return i18n.T("待办事项", "Todo List") }
func (m *TodoModule) Description() string { return i18n.T("添加、删除、勾选待办任务", "Add, delete, and check todo tasks") }
func (m *TodoModule) Icon() fyne.Resource { return theme.DocumentCreateIcon() }

func (m *TodoModule) CreateUI(w fyne.Window) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(i18n.T("✅ 待办事项", "✅ Todo List"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	desc := widget.NewLabel(i18n.T("此模块将实现待办事项管理功能，敬请期待。", "This module will implement todo list management, coming soon."))
	desc.Alignment = fyne.TextAlignCenter

	return container.New(layout.NewCenterLayout(),
		container.NewVBox(title, desc),
	)
}

func (m *TodoModule) OnInit()    {}
func (m *TodoModule) OnDestroy() {}
