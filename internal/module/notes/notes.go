//go:build module_notes

package notes

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"moyu-assistant/internal/i18n"
	"moyu-assistant/internal/module"
)

func init() {
	module.Register(&NotesModule{})
}

// NotesModule is a placeholder for the quick notes feature.
type NotesModule struct{}

func (m *NotesModule) Name() string { return i18n.T("快捷笔记", "Quick Notes") }
func (m *NotesModule) Description() string {
	return i18n.T("快速记录文本笔记", "Quickly record text notes")
}
func (m *NotesModule) Icon() fyne.Resource { return theme.FileTextIcon() }

func (m *NotesModule) CreateUI(w fyne.Window) fyne.CanvasObject {
	header := module.CreateHeader(i18n.T("📝 备忘录", "📝 Notes"), i18n.T("随时记录灵感与待办", "Record inspirations anytime"))
	return container.NewBorder(header, nil, nil, nil, container.NewCenter(widget.NewLabel(i18n.T("开发中...", "Under construction..."))))
}

func (m *NotesModule) OnInit()          {}
func (m *NotesModule) OnDestroy()       {}
func (m *NotesModule) Category() string { return "" }
