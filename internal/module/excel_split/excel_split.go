//go:build module_excel_split

package excel_split

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ncruces/zenity"
	"github.com/xuri/excelize/v2"

	"moyu-assistant/internal/i18n"
	"moyu-assistant/internal/module"
)

func init() {
	module.Register(&ExcelSplitModule{})
}

type ExcelSplitModule struct {
	state *appState
}

func (m *ExcelSplitModule) Name() string        { return i18n.T("MassUpload", "MassUpload") }
func (m *ExcelSplitModule) Description() string { return i18n.T("按Sheet页拆分Excel文件", "Split Excel file by sheets") }
func (m *ExcelSplitModule) Icon() fyne.Resource { return theme.DocumentIcon() }
func (m *ExcelSplitModule) Category() string    { return "LonzaCN" }

func (m *ExcelSplitModule) OnInit() {
	m.state = &appState{
		outputDir: "",
		logWidget: newReadOnlyEntry(),
	}
}

func (m *ExcelSplitModule) OnDestroy() {}

// ---------------------------------------------------------------------------
// Log Types & Widgets
// ---------------------------------------------------------------------------

type logLevel int

const (
	levelInfo logLevel = iota
	levelSuccess
	levelError
	levelWarning
	levelHighlight
)

type readOnlyEntry struct {
	widget.Entry
}

func newReadOnlyEntry() *readOnlyEntry {
	e := &readOnlyEntry{}
	e.ExtendBaseWidget(e)
	e.MultiLine = true
	e.Wrapping = fyne.TextWrapWord
	e.TextStyle = fyne.TextStyle{Monospace: true}
	return e
}

func (e *readOnlyEntry) TypedRune(r rune) {}
func (e *readOnlyEntry) TypedKey(k *fyne.KeyEvent) {
	switch k.Name {
	case fyne.KeyUp, fyne.KeyDown, fyne.KeyLeft, fyne.KeyRight, fyne.KeyPageUp, fyne.KeyPageDown, fyne.KeyHome, fyne.KeyEnd:
		e.Entry.TypedKey(k)
	}
}
func (e *readOnlyEntry) TypedShortcut(s fyne.Shortcut) {
	if _, ok := s.(*fyne.ShortcutCopy); ok {
		e.Entry.TypedShortcut(s)
	}
	if _, ok := s.(*fyne.ShortcutSelectAll); ok {
		e.Entry.TypedShortcut(s)
	}
}

// ---------------------------------------------------------------------------
// Application state
// ---------------------------------------------------------------------------

type appState struct {
	mu            sync.Mutex
	logWidget     *readOnlyEntry
	selectedFiles []string
	outputDir     string
}

func (s *appState) appendLog(msg string, level logLevel) {
	s.mu.Lock()
	defer s.mu.Unlock()

	prefix := ""
	switch level {
	case levelError:
		prefix = "[ERROR] "
	case levelWarning:
		prefix = "[WARN]  "
	case levelHighlight:
		prefix = "[OK]    "
	case levelInfo:
		prefix = "[INFO]  "
	case levelSuccess:
		prefix = "[SUCCESS]"
	}
	
	formattedMsg := prefix + msg

	if s.logWidget != nil {
		current := s.logWidget.Text
		if current != "" {
			current += "\n"
		}
		current += formattedMsg
		s.logWidget.SetText(current)
		s.logWidget.CursorRow = len(strings.Split(current, "\n")) - 1
	}
}

// ---------------------------------------------------------------------------
// UI
// ---------------------------------------------------------------------------

func (m *ExcelSplitModule) CreateUI(w fyne.Window) fyne.CanvasObject {
	state := m.state

	appPrefs := fyne.CurrentApp().Preferences()
	state.outputDir = appPrefs.StringWithFallback("MassUpload_OutDir", "")

	var startColor, endColor color.Color
	if fyne.CurrentApp().Settings().ThemeVariant() == theme.VariantDark {
		startColor = color.NRGBA{R: 144, G: 80, B: 168, A: 255} // #9050a8
		endColor = color.NRGBA{R: 75, G: 108, B: 183, A: 255}   // #4b6cb7
	} else {
		startColor = color.NRGBA{R: 90, G: 110, B: 140, A: 255}
		endColor = color.NRGBA{R: 160, G: 175, B: 190, A: 255}
	}

	headerGrad := canvas.NewLinearGradient(startColor, endColor, -45)
	headerText := canvas.NewText("MassUpload", color.White)
	headerText.TextSize = 28
	headerText.TextStyle = fyne.TextStyle{Bold: true}
	headerText.Alignment = fyne.TextAlignCenter
	
	subHeaderText := canvas.NewText(i18n.T("Excel Format Converter", "Excel Format Converter"), color.NRGBA{255, 255, 255, 200})
	subHeaderText.TextSize = 12
	subHeaderText.Alignment = fyne.TextAlignCenter

	headerContent := container.NewVBox(
		layout.NewSpacer(),
		headerText,
		subHeaderText,
		layout.NewSpacer(),
	)
	headerContainer := container.NewMax(headerGrad, container.NewPadded(headerContent))
	sizedHeader := container.New(layout.NewGridWrapLayout(fyne.NewSize(320, 100)), headerContainer)

	inputLabel := widget.NewLabel(i18n.T("未选择文件", "No file selected"))
	inputLabel.Wrapping = fyne.TextWrapWord
	inputLabel.TextStyle = fyne.TextStyle{Italic: true}
	inputLabel.Alignment = fyne.TextAlignLeading

	selectFilesBtn := widget.NewButtonWithIcon(i18n.T("添加文件", "Add Files"), theme.FolderOpenIcon(), func() {
		go func() {
			filenames, err := zenity.SelectFileMultiple(
				zenity.Title("Select Excel Files"),
				zenity.FileFilter{Name: "Excel Files", Patterns: []string{"*.xlsx", "*.XLSX"}},
			)
			if err != nil {
				if err != zenity.ErrCanceled {
					state.appendLog(fmt.Sprintf("⚠ Dialog error: %v", err), levelWarning)
				}
				return
			}
			if len(filenames) == 0 {
				return
			}

			added := 0
			for _, filename := range filenames {
				state.selectedFiles = append(state.selectedFiles, filename)
				added++
				state.appendLog(fmt.Sprintf("+ Queued: %s", filepath.Base(filename)), levelHighlight)
			}
			inputLabel.SetText(fmt.Sprintf(i18n.T("已选择 %d 个文件", "Selected %d file(s)"), len(state.selectedFiles)))
		}()
	})

	addFolderBtn := widget.NewButtonWithIcon(i18n.T("添加文件夹", "Add Folder"), theme.FolderIcon(), func() {
		go func() {
			dir, err := zenity.SelectFile(
				zenity.Title("Select Source Folder"),
				zenity.Directory(),
			)
			if err != nil {
				if err != zenity.ErrCanceled {
					state.appendLog(fmt.Sprintf("⚠ Dialog error: %v", err), levelWarning)
				}
				return
			}
			if dir == "" {
				return
			}
			files, err := os.ReadDir(dir)
			if err != nil {
				state.appendLog(fmt.Sprintf("⚠ Folder read error: %v", err), levelWarning)
				return
			}
			added := 0
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".xlsx") && strings.HasPrefix(file.Name(), "LZACN_") {
					state.selectedFiles = append(state.selectedFiles, filepath.Join(dir, file.Name()))
					state.appendLog(fmt.Sprintf("+ Queued from dir: %s", file.Name()), levelHighlight)
					added++
				}
			}
			inputLabel.SetText(fmt.Sprintf(i18n.T("已选择 %d 个文件", "Selected %d file(s)"), len(state.selectedFiles)))
			if added == 0 {
				state.appendLog(fmt.Sprintf("⚠ No LZACN_ *.xlsx found in %s", filepath.Base(dir)), levelWarning)
			}
		}()
	})

	clearBtn := widget.NewButtonWithIcon(i18n.T("清除", "Clear"), theme.ContentClearIcon(), func() {
		state.selectedFiles = nil
		inputLabel.SetText(i18n.T("未选择文件", "No file selected"))
		state.appendLog("Cleared selected files.", levelInfo)
	})

	outDirLabel := widget.NewLabel("")
	outDirLabel.Alignment = fyne.TextAlignLeading
	if state.outputDir == "" {
		outDirLabel.SetText(i18n.T("输出目录: (默认同源文件目录)", "Output Dir: (Default same as source)"))
	} else {
		outDirLabel.SetText(i18n.T("输出目录: ", "Output Dir: ") + state.outputDir)
	}

	selectOutDirBtn := widget.NewButtonWithIcon(i18n.T("选择输出目录", "Select Output Dir"), theme.FolderIcon(), func() {
		go func() {
			dir, err := zenity.SelectFile(
				zenity.Title("Select Output Folder"),
				zenity.Directory(),
			)
			if err != nil {
				if err != zenity.ErrCanceled {
					state.appendLog(fmt.Sprintf("⚠ Dialog error: %v", err), levelWarning)
				}
				return
			}
			if dir == "" {
				return
			}
			
			state.outputDir = dir
			appPrefs.SetString("MassUpload_OutDir", state.outputDir)
			outDirLabel.SetText(i18n.T("输出目录: ", "Output Dir: ") + state.outputDir)
			state.appendLog(fmt.Sprintf("Set output directory to: %s", dir), levelInfo)
		}()
	})

	var processBtn *widget.Button
	processBtn = widget.NewButtonWithIcon(i18n.T("开始转换", "Start Conversion"), theme.MediaPlayIcon(), func() {
		if len(state.selectedFiles) == 0 {
			dialog.ShowInformation(i18n.T("提示", "Info"), i18n.T("请先选择Excel文件或目录", "Please select Excel files or directory"), w)
			return
		}
		
		processBtn.Disable()
		state.appendLog(fmt.Sprintf("Starting conversion for %d files...", len(state.selectedFiles)), levelInfo)
		
		go func() {
			successCount := 0
			failCount := 0
			for i, file := range state.selectedFiles {
				state.appendLog(fmt.Sprintf("Processing (%d/%d): %s", i+1, len(state.selectedFiles), filepath.Base(file)), levelInfo)
				err := splitExcelFile(file, state.outputDir)
				if err != nil {
					state.appendLog(fmt.Sprintf("Failed %s: %v", filepath.Base(file), err), levelError)
					failCount++
				} else {
					state.appendLog(fmt.Sprintf("Successfully processed %s", filepath.Base(file)), levelSuccess)
					successCount++
				}
			}
			state.appendLog(fmt.Sprintf("Done! Success: %d, Failed: %d", successCount, failCount), levelInfo)
			processBtn.Enable()
		}()
	})
	processBtn.Importance = widget.HighImportance

	inputBtns := container.NewGridWithColumns(2, selectFilesBtn, addFolderBtn)
	
	inputControls := container.NewVBox(
		widget.NewLabelWithStyle(i18n.T("输入设置", "Input Settings"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, nil, nil, clearBtn, inputBtns),
		inputLabel,
	)

	outputControls := container.NewVBox(
		widget.NewLabelWithStyle(i18n.T("输出设置", "Output Settings"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		selectOutDirBtn,
		outDirLabel,
	)

	controlsContainer := container.NewVBox(
		inputControls,
		widget.NewSeparator(),
		outputControls,
	)

	sidebar := container.NewBorder(
		sizedHeader,
		container.NewPadded(processBtn),
		nil, nil,
		container.NewVScroll(container.NewPadded(controlsContainer)),
	)

	logTitle := widget.NewLabelWithStyle(i18n.T("控制台输出", "Console Output"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	
	clearConsoleBtn := widget.NewButton(i18n.T("清空", "Clear"), func() {
		state.mu.Lock()
		state.logWidget.SetText("")
		state.mu.Unlock()
	})
	
	logHeader := container.NewBorder(nil, nil, logTitle, clearConsoleBtn)
	
	logArea := container.NewBorder(
		container.NewVBox(logHeader, widget.NewSeparator()),
		nil, nil, nil,
		state.logWidget,
	)

	split := container.NewHSplit(
		sidebar,
		container.NewPadded(logArea),
	)
	split.Offset = 0.35 

	return split
}

// ---------------------------------------------------------------------------
// Processing Logic
// ---------------------------------------------------------------------------

type SheetConfig struct {
	TemplateFile       string
	SourceHeaderRows   int
	TemplateHeaderRows int
}

func splitExcelFile(filePath string, outDir string) error {
	baseName := filepath.Base(filePath)
	re := regexp.MustCompile(`^LZACN_LZACN(\d{3})`)
	matches := re.FindStringSubmatch(baseName)
	if len(matches) < 2 {
		return fmt.Errorf("文件名不符合要求，必须以 LZACN_LZACN<LCC> 开头")
	}
	lcc := matches[1]

	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %v", err)
	}
	defer f.Close()

	sheetConfigs := map[string]SheetConfig{
		"DATA_CHANGE":                 {TemplateFile: "LZA000001998TPEv005_PayElementUpload.xlsx", SourceHeaderRows: 3, TemplateHeaderRows: 2},
		"DATA_CHANGE (report only)":   {TemplateFile: "LZA000001991_Corrections.xlsx", SourceHeaderRows: 2, TemplateHeaderRows: 2},
		"DATA_CHANGE(cnit)":           {TemplateFile: "LZA000001994_Income Tax_2026_08_03_02_20_43.xlsx", SourceHeaderRows: 1, TemplateHeaderRows: 1},
		"DATA PHF (ViewID=284)":       {TemplateFile: "LZA000001994_Public Housing Fund_2026_08_03_02_20_42.xlsx", SourceHeaderRows: 1, TemplateHeaderRows: 1},
		"DATA SI (ViewId=286)":        {TemplateFile: "LZA000001994_Social Insurance_2026_08_03_02_20_43.xlsx", SourceHeaderRows: 1, TemplateHeaderRows: 1},
	}

	dir := outDir
	if dir == "" {
		dir = filepath.Dir(filePath)
	}

	for sheetName, config := range sheetConfigs {
		// check if sheet exists in source file using fuzzy match
		var actualSheetName string
		expectedFuzzy := strings.ToLower(strings.ReplaceAll(sheetName, " ", ""))
		for _, name := range f.GetSheetList() {
			actualFuzzy := strings.ToLower(strings.ReplaceAll(name, " ", ""))
			if actualFuzzy == expectedFuzzy {
				actualSheetName = name
				break
			}
		}

		if actualSheetName == "" {
			return fmt.Errorf("源文件缺少必要的Sheet: %s", sheetName)
		}

		rows, err := f.GetRows(actualSheetName)
		if err != nil {
			return fmt.Errorf("读取Sheet %s 失败: %v", actualSheetName, err)
		}

		if len(rows) <= config.SourceHeaderRows {
			continue // No data, skip
		}

		templatePath := filepath.Join("assets", "template", "LZACN", config.TemplateFile)
		tmplFile, err := excelize.OpenFile(templatePath)
		if err != nil {
			return fmt.Errorf("打开模板文件 %s 失败: %v", config.TemplateFile, err)
		}

		activeSheet := tmplFile.GetSheetName(tmplFile.GetActiveSheetIndex())

		for rIdx := config.SourceHeaderRows; rIdx < len(rows); rIdx++ {
			row := rows[rIdx]
			targetRowIdx := (rIdx - config.SourceHeaderRows) + config.TemplateHeaderRows + 1
			for cIdx, colCell := range row {
				cellName, _ := excelize.CoordinatesToCellName(cIdx+1, targetRowIdx)
				tmplFile.SetCellValue(activeSheet, cellName, colCell)
			}
		}

		outFileName := strings.Replace(config.TemplateFile, "LZA00000", "LZACN"+lcc, 1)
		outFilePath := filepath.Join(dir, outFileName)
		
		if err := tmplFile.SaveAs(outFilePath); err != nil {
			tmplFile.Close()
			return fmt.Errorf("保存文件 %s 失败: %v", outFileName, err)
		}
		tmplFile.Close()
	}

	return nil
}
