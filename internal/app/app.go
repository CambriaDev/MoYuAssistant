package app

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"moyu-assistant/internal/i18n"
	"moyu-assistant/internal/module"
)

const (
	appID = "MoYuAssistant"
)

// Run initializes and starts the application.
func Run() {
	appTitle := i18n.T("摸鱼助手", "MoYu Assistant")

	a := fyneapp.NewWithID(appID)
	a.SetIcon(newTrayIcon())
	applyTheme(a)

	w := a.NewWindow(appTitle)
	w.Resize(fyne.NewSize(800, 560))
	w.CenterOnScreen()
	setupMenu(a, w)

	// Setup system tray (minimize-to-tray behavior)
	setupTray(a, w, appTitle)

	// Initialize all registered modules
	modules := module.All()
	for _, m := range modules {
		m.OnInit()
	}

	// Build and set the main UI
	w.SetContent(buildMainUI(w, modules))
	w.ShowAndRun()

	// Cleanup on exit
	for _, m := range modules {
		m.OnDestroy()
	}
}

func buildMainUI(w fyne.Window, modules []module.Module) fyne.CanvasObject {
	if len(modules) == 0 {
		return buildEmptyState()
	}

	// Categorize modules
	var jigglerModIdx int = -1
	topLevelMods := []int{}
	catMap := make(map[string][]int)
	var catOrder []string // to preserve order of categories

	for i, m := range modules {
		cat := m.Category()
		if cat == "" {
			if m.Name() == i18n.T("假装在线", "Fake Online") {
				jigglerModIdx = i
			} else {
				topLevelMods = append(topLevelMods, i)
			}
		} else {
			if _, exists := catMap[cat]; !exists {
				catOrder = append(catOrder, cat)
			}
			catMap[cat] = append(catMap[cat], i)
		}
	}

	// Cache UIs to avoid recreating them on every selection
	uiCache := make(map[int]fyne.CanvasObject)
	for i, m := range modules {
		uiCache[i] = m.CreateUI(w)
	}

	// Right side content area
	contentArea := container.NewMax()

	// Initial selection - first available module
	if len(modules) > 0 {
		contentArea.Objects = []fyne.CanvasObject{uiCache[0]}
	}

	// Tree functions
	tree := widget.NewTree(
		// ChildUIDs
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			if uid == "" {
				var rootChildren []widget.TreeNodeID
				// Jiggler at the top
				if jigglerModIdx != -1 {
					rootChildren = append(rootChildren, fmt.Sprintf("mod:%d", jigglerModIdx))
				}
				// Top-level categories
				for _, cat := range catOrder {
					rootChildren = append(rootChildren, "cat:"+cat)
				}
				// Top-level modules
				for _, modIdx := range topLevelMods {
					rootChildren = append(rootChildren, fmt.Sprintf("mod:%d", modIdx))
				}
				return rootChildren
			}
			if strings.HasPrefix(uid, "cat:") {
				cat := strings.TrimPrefix(uid, "cat:")
				var children []widget.TreeNodeID
				for _, modIdx := range catMap[cat] {
					children = append(children, fmt.Sprintf("mod:%d", modIdx))
				}
				return children
			}
			return nil
		},
		// IsBranch
		func(uid widget.TreeNodeID) bool {
			return uid == "" || strings.HasPrefix(uid, "cat:")
		},
		// CreateNode
		func(branch bool) fyne.CanvasObject {
			// Using icon and label
			icon := widget.NewIcon(theme.FolderIcon())
			label := widget.NewLabel("Node")
			return container.NewHBox(icon, label)
		},
		// UpdateNode
		func(uid widget.TreeNodeID, branch bool, node fyne.CanvasObject) {
			box := node.(*fyne.Container)
			icon := box.Objects[0].(*widget.Icon)
			label := box.Objects[1].(*widget.Label)
			
			if strings.HasPrefix(uid, "cat:") {
				catName := strings.TrimPrefix(uid, "cat:")
				label.SetText(catName)
				icon.SetResource(theme.FolderIcon())
			} else if strings.HasPrefix(uid, "mod:") {
				var modIdx int
				fmt.Sscanf(uid, "mod:%d", &modIdx)
				label.SetText(modules[modIdx].Name())
				icon.SetResource(modules[modIdx].Icon())
			}
		},
	)

	tree.OnSelected = func(uid widget.TreeNodeID) {
		if strings.HasPrefix(uid, "mod:") {
			var modIdx int
			fmt.Sscanf(uid, "mod:%d", &modIdx)
			contentArea.Objects = []fyne.CanvasObject{uiCache[modIdx]}
			contentArea.Refresh()
		}
	}
	
	// Open all categories by default
	for _, cat := range catOrder {
		tree.OpenBranch("cat:" + cat)
	}
	
	// Pre-select first module
	if jigglerModIdx != -1 {
		tree.Select(fmt.Sprintf("mod:%d", jigglerModIdx))
	} else if len(modules) > 0 {
		tree.Select(fmt.Sprintf("mod:%d", 0))
	}

	split := container.NewHSplit(tree, contentArea)
	split.Offset = 0.25 // 25% width for the menu

	return split
}

// buildEmptyState creates the UI shown when no modules are compiled in.
func buildEmptyState() fyne.CanvasObject {
	title := widget.NewLabelWithStyle(
		i18n.T("📦 没有加载任何功能模块", "📦 No functional modules loaded"),
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	hint := widget.NewLabelWithStyle(
		i18n.T("请使用 build tags 编译所需模块，例如：\ngo build -tags \"module_clock module_todo\" -o moyu.exe .", "Please compile with build tags, e.g.:\ngo build -tags \"module_clock module_todo\" -o moyu.exe ."),
		fyne.TextAlignCenter,
		fyne.TextStyle{Monospace: true},
	)

	return container.New(layout.NewCenterLayout(),
		container.NewVBox(title, widget.NewSeparator(), hint),
	)
}
