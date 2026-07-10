//go:build module_jiggler && !windows

package jiggler

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"moyu-assistant/internal/module"
)

func init() {
	module.Register(&JigglerModule{})
}

type JigglerModule struct{}

func (m *JigglerModule) Name() string        { return "假装在线" }
func (m *JigglerModule) Description() string { return "180s无操作自动晃动鼠标" }
func (m *JigglerModule) Icon() fyne.Resource { return theme.ComputerIcon() }

func (m *JigglerModule) CreateUI(w fyne.Window) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("🖱️ 假装在线", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	desc := widget.NewLabel("当前操作系统不支持此模块。\n该功能依赖 Windows API。")
	desc.Alignment = fyne.TextAlignCenter

	return container.New(layout.NewCenterLayout(),
		container.NewVBox(title, desc),
	)
}

func (m *JigglerModule) OnInit()    {}
func (m *JigglerModule) OnDestroy() {}
