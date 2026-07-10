//go:build module_notes

package notes

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"moyu-assistant/internal/module"
)

func init() {
	module.Register(&NotesModule{})
}

// NotesModule is a placeholder for the quick notes feature.
type NotesModule struct{}

func (m *NotesModule) Name() string        { return "快捷笔记" }
func (m *NotesModule) Description() string { return "快速记录文本笔记" }
func (m *NotesModule) Icon() fyne.Resource { return theme.FileTextIcon() }

func (m *NotesModule) CreateUI(w fyne.Window) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("📝 快捷笔记", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	desc := widget.NewLabel("此模块将实现快速笔记记录功能，敬请期待。")
	desc.Alignment = fyne.TextAlignCenter

	return container.New(layout.NewCenterLayout(),
		container.NewVBox(title, desc),
	)
}

func (m *NotesModule) OnInit()    {}
func (m *NotesModule) OnDestroy() {}
