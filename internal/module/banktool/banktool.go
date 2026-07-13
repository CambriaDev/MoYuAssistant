package banktool

import (
	"fmt"
	"image/color"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/eslider/go-xls/v2"
	"github.com/ncruces/zenity"
	"github.com/tealeg/xlsx"

	"moyu-assistant/internal/i18n"
	"moyu-assistant/internal/module"
)

func init() {
	module.Register(&BankToolModule{})
}

type BankToolModule struct {
	state *appState
}

func (m *BankToolModule) Name() string {
	return i18n.T("薪资转换", "BankTool")
}

func (m *BankToolModule) Description() string {
	return i18n.T("用于转换薪资Excel文件格式的工具", "Tool to convert payroll Excel format")
}

func (m *BankToolModule) Icon() fyne.Resource {
	return theme.DocumentIcon()
}

func (m *BankToolModule) OnInit() {
	m.state = &appState{
		outputDir: ".",
		logWidget: newReadOnlyEntry(),
	}
}

func (m *BankToolModule) OnDestroy() {
}

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
	payrollFile   string
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
// Conversion & Validation logic
// ---------------------------------------------------------------------------

func getPayrollTotals(filename string) (int, float64, error) {
	f, err := xlsx.OpenFile(filename)
	if err != nil {
		return 0, 0, err
	}
	if len(f.Sheets) == 0 {
		return 0, 0, fmt.Errorf("no sheets found in payroll file")
	}
	sheet := f.Sheets[0]
	if len(sheet.Rows) == 0 {
		return 0, 0, fmt.Errorf("payroll file is empty")
	}

	colIdx := -1
	for i, cell := range sheet.Rows[0].Cells {
		txt := cell.String()
		if strings.Contains(txt, "/559") || strings.Contains(txt, "Bank Transfer") || strings.Contains(txt, "Bank Tranfer") {
			colIdx = i
			break
		}
	}

	if colIdx == -1 {
		return 0, 0, fmt.Errorf("could not find '/559 Bank Transfer' column")
	}

	count := 0
	sum := 0.0

	// Exclude header row
	for i := 1; i < len(sheet.Rows); i++ {
		row := sheet.Rows[i]
		if len(row.Cells) <= colIdx {
			continue
		}
		
		if len(row.Cells) > 3 && strings.TrimSpace(row.Cells[3].String()) == "" {
			continue // Skip total row for count
		}

		valStr := row.Cells[colIdx].String()
		val, _ := strconv.ParseFloat(strings.TrimSpace(valStr), 64)
		if math.Abs(val) > 0.001 {
			count++
			sum += val
		}
	}
	return count, sum, nil
}

func processExcelFiles(state *appState) {
	filesToProcess := state.selectedFiles

	if len(filesToProcess) == 0 {
		dir := "."
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			state.appendLog(fmt.Sprintf("✗ Error reading directory: %v", err), levelError)
			return
		}
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".xlsx") && !strings.Contains(strings.ToLower(file.Name()), "payroll") {
				filesToProcess = append(filesToProcess, file.Name())
			}
		}
	}

	if len(filesToProcess) == 0 {
		state.appendLog("⚠ No valid .xlsx bank files found or selected.", levelWarning)
		return
	}

	totalBankItems := 0
	totalBankSum := 0.0

	type fileStat struct {
		label string
		count int
		sum   float64
	}
	var stats []fileStat
	
	count := 0
	for _, file := range filesToProcess {
		name := filepath.Base(file)
		ext := filepath.Ext(name)
		baseName := name[:len(name)-len(ext)]
		
		if strings.HasPrefix(baseName, "BANK_MAE_CNA_") {
			baseName = "MAE" + strings.TrimPrefix(baseName, "BANK_MAE_CNA_")
		}
		xlsName := baseName + ".xls"
		outPath := filepath.Join(state.outputDir, xlsName)
		
		state.appendLog(fmt.Sprintf("⏳ Converting %s ...", name), levelInfo)

		f, err := xlsx.OpenFile(file)
		if err != nil {
			state.appendLog(fmt.Sprintf("  ✗ Failed to open: %v", err), levelError)
			continue
		}
		
		fileCount := 0
		fileSum := 0.0

		if len(f.Sheets) > 0 {
			sheet := f.Sheets[0]
			for i := 2; i < len(sheet.Rows); i++ {
				row := sheet.Rows[i]
				if len(row.Cells) > 14 {
					valStr := row.Cells[14].String()
					val, _ := strconv.ParseFloat(strings.TrimSpace(valStr), 64)
					if math.Abs(val) > 0.001 {
						fileCount++
						fileSum += val
					}
				}
			}
		}
		
		totalBankItems += fileCount
		totalBankSum += fileSum

		label := "Unknown"
		parts := strings.Split(name, "_")
		for _, p := range parts {
			if (strings.HasPrefix(p, "LCN") || strings.HasPrefix(p, "CN")) && len(p) > 2 {
				labelCand := strings.TrimPrefix(p, "L") 
				if len(labelCand) > 2 && labelCand[2] >= '0' && labelCand[2] <= '9' {
					label = labelCand
					break
				}
			}
		}
		stats = append(stats, fileStat{label: label, count: fileCount, sum: fileSum})

		err = saveAsXLSViaExcel(file, outPath)
		if err != nil {
			state.appendLog(fmt.Sprintf("  ⚠ Excel automation failed (%v), falling back to native go-xls...", err), levelWarning)
			var cols []string
			var rows [][]string
			if len(f.Sheets) > 0 {
				sheet := f.Sheets[0]
				if len(sheet.Rows) > 0 {
					for _, cell := range sheet.Rows[0].Cells {
						cols = append(cols, cell.String())
					}
					for i := 1; i < len(sheet.Rows); i++ {
						var rowData []string
						for _, cell := range sheet.Rows[i].Cells {
							rowData = append(rowData, cell.String())
						}
						rows = append(rows, rowData)
					}
				}
			}

			tab := xls.Table{
				Columns: cols,
				Rows:    rows,
			}

			outFile, createErr := os.Create(outPath)
			if createErr != nil {
				state.appendLog(fmt.Sprintf("  ✗ Failed to create output file: %v", createErr), levelError)
				continue
			}

			err = xls.WriteXLS(outFile, tab, true)
			outFile.Close()
		}

		if err != nil {
			state.appendLog(fmt.Sprintf("  ✗ Failed to save: %v", err), levelError)
		} else {
			state.appendLog(fmt.Sprintf("  ✓ Saved: %s", outPath), levelSuccess)
			count++
		}
	}
	state.appendLog(fmt.Sprintf("☆ Completed – %d file(s) converted.", count), levelSuccess)

	if state.payrollFile != "" {
		state.appendLog("------------------------------------------", levelInfo)
		state.appendLog(fmt.Sprintf("🔍 Validating against: %s", filepath.Base(state.payrollFile)), levelHighlight)
		pCount, pSum, err := getPayrollTotals(state.payrollFile)
		if err != nil {
			state.appendLog(fmt.Sprintf("  ✗ Validation Error: %v", err), levelError)
		} else {
			state.appendLog(fmt.Sprintf("  Payroll File -> Count: %d, Sum: %.2f", pCount, pSum), levelInfo)
			state.appendLog(fmt.Sprintf("  Bank Files   -> Count: %d, Sum: %.2f", totalBankItems, totalBankSum), levelInfo)

			if pCount == totalBankItems && math.Abs(pSum-totalBankSum) < 0.01 {
				state.appendLog("  ✓ VALIDATION PASSED: Items and Amounts Match!", levelSuccess)
			} else {
				state.appendLog("  ✗ VALIDATION FAILED: Mismatch detected!", levelError)
			}
		}
	}
	
	state.appendLog("------------------------------------------", levelInfo)
	state.appendLog("📄 File breakdown:", levelHighlight)
	for _, s := range stats {
		state.appendLog(fmt.Sprintf("  [%s] Count: %d, Sum: %.2f", s.label, s.count, s.sum), levelInfo)
	}
}

// ---------------------------------------------------------------------------
// UI
// ---------------------------------------------------------------------------

func (m *BankToolModule) CreateUI(w fyne.Window) fyne.CanvasObject {
	state := m.state

	headerGrad := canvas.NewLinearGradient(
		color.NRGBA{R: 90, G: 110, B: 140, A: 255},  
		color.NRGBA{R: 160, G: 175, B: 190, A: 255}, 
		-45,
	)
	headerText := canvas.NewText("BankTool", color.White)
	headerText.TextSize = 28
	headerText.TextStyle = fyne.TextStyle{Bold: true}
	headerText.Alignment = fyne.TextAlignCenter
	
	subHeaderText := canvas.NewText("Excel Format Converter", color.NRGBA{255, 255, 255, 200})
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

	inputLabel := widget.NewLabel(i18n.T("默认：当前目录下所有 .xlsx", "Default: All .xlsx in current dir"))
	inputLabel.Wrapping = fyne.TextWrapWord
	inputLabel.TextStyle = fyne.TextStyle{Italic: true}

	selectFilesBtn := widget.NewButton(i18n.T("添加文件", "Add Files"), func() {
		go func() {
			filenames, err := zenity.SelectFileMultiple(
				zenity.Title("Select Source Files"),
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
			state.selectedFiles = append(state.selectedFiles, filenames...)
			inputLabel.SetText(fmt.Sprintf(i18n.T("已选择 %d 个文件", "%d file(s) selected"), len(state.selectedFiles)))
			for _, fn := range filenames {
				state.appendLog(fmt.Sprintf("+ Queued: %s", filepath.Base(fn)), levelHighlight)
			}
		}()
	})

	addFolderBtn := widget.NewButton(i18n.T("添加文件夹", "Add Folder"), func() {
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
			files, err := ioutil.ReadDir(dir)
			if err != nil {
				state.appendLog(fmt.Sprintf("⚠ Folder read error: %v", err), levelWarning)
				return
			}
			added := 0
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".xlsx") && !strings.Contains(strings.ToLower(file.Name()), "payroll") {
					state.selectedFiles = append(state.selectedFiles, filepath.Join(dir, file.Name()))
					state.appendLog(fmt.Sprintf("+ Queued: %s", file.Name()), levelHighlight)
					added++
				}
			}
			inputLabel.SetText(fmt.Sprintf(i18n.T("已选择 %d 个文件", "%d file(s) selected"), len(state.selectedFiles)))
			if added == 0 {
				state.appendLog(fmt.Sprintf("⚠ No valid .xlsx files found in %s", filepath.Base(dir)), levelWarning)
			}
		}()
	})

	clearFilesBtn := widget.NewButton(i18n.T("清除", "Clear"), func() {
		state.selectedFiles = nil
		inputLabel.SetText(i18n.T("默认：当前目录下所有 .xlsx", "Default: All .xlsx in current dir"))
		state.appendLog("⚠ Cleared file selection.", levelWarning)
	})
	
	inputBtns := container.NewGridWithColumns(2, selectFilesBtn, addFolderBtn)
	
	inputControls := container.NewVBox(
		widget.NewLabelWithStyle(i18n.T("输入设置", "Input Settings"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, nil, nil, clearFilesBtn, inputBtns),
		inputLabel,
	)

	outputLabel := widget.NewLabel(i18n.T("默认：与输入目录相同", "Default: Same as input"))
	outputLabel.Wrapping = fyne.TextWrapWord
	outputLabel.TextStyle = fyne.TextStyle{Italic: true}

	selectOutputBtn := widget.NewButton(i18n.T("选择输出目录", "Choose Output Folder"), func() {
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
			outputLabel.SetText(filepath.Base(state.outputDir))
			state.appendLog(fmt.Sprintf("→ Output set to: %s", state.outputDir), levelInfo)
		}()
	})
	outputControls := container.NewVBox(
		widget.NewLabelWithStyle(i18n.T("输出设置", "Output Settings"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		selectOutputBtn,
		outputLabel,
	)

	payrollLabel := widget.NewLabel(i18n.T("未选择", "None selected"))
	payrollLabel.Wrapping = fyne.TextWrapWord
	payrollLabel.TextStyle = fyne.TextStyle{Italic: true}

	selectPayrollBtn := widget.NewButton(i18n.T("选择薪资文件", "Choose Payroll File"), func() {
		go func() {
			filename, err := zenity.SelectFile(
				zenity.Title("Select Payroll File"),
				zenity.FileFilter{Name: "Excel Files", Patterns: []string{"*.xlsx", "*.XLSX"}},
			)
			if err != nil {
				if err != zenity.ErrCanceled {
					state.appendLog(fmt.Sprintf("⚠ Dialog error: %v", err), levelWarning)
				}
				return
			}
			if filename == "" {
				return
			}
			state.payrollFile = filename
			payrollLabel.SetText(filepath.Base(filename))
			state.appendLog(fmt.Sprintf("→ Validation file set to: %s", filepath.Base(filename)), levelInfo)
		}()
	})

	clearPayrollBtn := widget.NewButton(i18n.T("清除", "Clear"), func() {
		state.payrollFile = ""
		payrollLabel.SetText(i18n.T("未选择", "None selected"))
		state.appendLog("⚠ Cleared validation file. Validation will be skipped.", levelWarning)
	})

	payrollControls := container.NewVBox(
		widget.NewLabelWithStyle(i18n.T("校验文件 (可选)", "Validation (Optional)"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, nil, nil, clearPayrollBtn, selectPayrollBtn),
		payrollLabel,
	)

	startBtn := widget.NewButton(i18n.T("开始转换", "START CONVERSION"), func() {
		state.appendLog("==========================================", levelInfo)
		state.appendLog("🚀 Process initiated...", levelInfo)
		go processExcelFiles(state)
	})
	startBtn.Importance = widget.HighImportance

	controlsContainer := container.NewVBox(
		inputControls,
		widget.NewSeparator(),
		outputControls,
		widget.NewSeparator(),
		payrollControls,
	)

	sidebar := container.NewBorder(
		sizedHeader,
		container.NewPadded(startBtn),
		nil, nil,
		container.NewVScroll(container.NewPadded(controlsContainer)),
	)

	logTitle := widget.NewLabelWithStyle(i18n.T("控制台输出", "Console Output"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	
	clearConsoleBtn := widget.NewButton(i18n.T("清空", "Clear"), func() {
		state.mu.Lock()
		state.mu.Unlock()
		state.logWidget.SetText("")
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
