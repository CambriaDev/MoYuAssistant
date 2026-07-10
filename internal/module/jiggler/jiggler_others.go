//go:build module_jiggler && !windows

package jiggler

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
	module.Register(&JigglerModule{})
}

type JigglerModule struct{}

func (m *JigglerModule) Name() string        { return i18n.T("假装在线", "Fake Online") }
func (m *JigglerModule) Description() string { return i18n.T("180s无操作自动晃动鼠标", "Auto jiggle mouse after 180s of inactivity") }
func (m *JigglerModule) Icon() fyne.Resource { return theme.ComputerIcon() }

func (m *JigglerModule) CreateUI(w fyne.Window) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(i18n.T("🖱️ 假装在线", "🖱️ Fake Online"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	desc := widget.NewLabel(i18n.T("当前操作系统不支持此模块。\n该功能依赖 Windows API。", "This module is not supported on the current OS.\nIt relies on Windows APIs."))
	desc.Alignment = fyne.TextAlignCenter

	return container.New(layout.NewCenterLayout(),
		container.NewVBox(title, desc),
	)
}

func (m *JigglerModule) OnInit()    {}
func (m *JigglerModule) OnDestroy() {}
