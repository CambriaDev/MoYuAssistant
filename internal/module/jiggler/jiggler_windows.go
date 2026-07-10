//go:build module_jiggler && windows

package jiggler

import (
	"context"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"moyu-assistant/internal/i18n"
	"moyu-assistant/internal/module"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procGetLastInputInfo = user32.NewProc("GetLastInputInfo")
	procGetTickCount     = kernel32.NewProc("GetTickCount")
	procMouseEvent       = user32.NewProc("mouse_event")
)

const (
	MOUSEEVENTF_MOVE = 0x0001
)

type LASTINPUTINFO struct {
	CbSize uint32
	DwTime uint32
}

func init() {
	module.Register(&JigglerModule{})
}

type JigglerModule struct {
	active bool
	mu     sync.Mutex
	cancel context.CancelFunc

	statusLabel *widget.Label
	toggleBtn   *widget.Button
}

func (m *JigglerModule) Name() string        { return i18n.T("假装在线", "Fake Online") }
func (m *JigglerModule) Description() string { return i18n.T("180s无操作自动晃动鼠标", "Auto jiggle mouse after 180s of inactivity") }
func (m *JigglerModule) Icon() fyne.Resource { return theme.ComputerIcon() }

func (m *JigglerModule) CreateUI(w fyne.Window) fyne.CanvasObject {
	m.statusLabel = widget.NewLabelWithStyle(i18n.T("状态：💤 未开启", "Status: 💤 Disabled"), fyne.TextAlignCenter, fyne.TextStyle{})

	m.toggleBtn = widget.NewButton(i18n.T("开启防离开模式", "Enable Anti-Away Mode"), nil)
	m.toggleBtn.OnTapped = func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.active {
			m.stop()
		} else {
			m.start()
		}
	}

	title := widget.NewLabelWithStyle(i18n.T("🖱️ 假装在线", "🖱️ Fake Online"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	desc := widget.NewLabel(i18n.T("开启后，后台会监听系统输入事件。\n如果超过 180 秒没有任何键盘鼠标活动，会自动微动鼠标。\n(任何人工或程序的键鼠活动都会重新计时)", "When enabled, listens to system input events in the background.\nIf no keyboard/mouse activity for 180s, automatically jiggles the mouse.\n(Any manual or programmatic input will reset the timer)"))
	desc.Alignment = fyne.TextAlignCenter

	return container.New(layout.NewCenterLayout(),
		container.NewVBox(
			title,
			desc,
			widget.NewSeparator(),
			m.statusLabel,
			m.toggleBtn,
		),
	)
}

func (m *JigglerModule) OnInit() {}

func (m *JigglerModule) OnDestroy() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active {
		m.stop()
	}
}

func (m *JigglerModule) start() {
	m.active = true
	m.statusLabel.SetText(i18n.T("状态：✅ 已开启 (180s 无操作自动防离开)", "Status: ✅ Enabled (Auto anti-away after 180s)"))
	m.toggleBtn.SetText(i18n.T("停止", "Stop"))
	m.toggleBtn.Importance = widget.HighImportance

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	go m.runLoop(ctx)
}

func (m *JigglerModule) stop() {
	m.active = false
	m.statusLabel.SetText(i18n.T("状态：💤 未开启", "Status: 💤 Disabled"))
	m.toggleBtn.SetText(i18n.T("开启防离开模式", "Enable Anti-Away Mode"))
	m.toggleBtn.Importance = widget.MediumImportance

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

func (m *JigglerModule) runLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second) // 每 5 秒检查一次闲置时间
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			idleTimeMs := getIdleTime()
			if idleTimeMs >= 180*1000 { // 180 seconds
				jiggleMouse()
			}
		}
	}
}

// getIdleTime 返回距离上次用户输入（键盘/鼠标）经过的毫秒数
func getIdleTime() uint32 {
	var lii LASTINPUTINFO
	lii.CbSize = uint32(unsafe.Sizeof(lii))

	ret, _, _ := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&lii)))
	if ret == 0 {
		return 0
	}

	tickCount, _, _ := procGetTickCount.Call()

	// uint32 相减处理溢出没问题（只要系统运行时间差在 49 天以内）
	return uint32(tickCount) - lii.DwTime
}

// jiggleMouse 模拟极微小的鼠标移动
func jiggleMouse() {
	// 向右下方移动 1 像素
	procMouseEvent.Call(MOUSEEVENTF_MOVE, uintptr(1), uintptr(1), 0, 0)
	time.Sleep(50 * time.Millisecond)
	// 向左上方移回 1 像素（0xFFFFFFFF 是 -1 的 uint32 表示）
	procMouseEvent.Call(MOUSEEVENTF_MOVE, uintptr(uint32(0xFFFFFFFF)), uintptr(uint32(0xFFFFFFFF)), 0, 0)
}
