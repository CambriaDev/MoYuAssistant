//go:build module_billing_convert

package billing_convert

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ncruces/zenity"
	"github.com/xuri/excelize/v2"

	"moyu-assistant/internal/i18n"
	"moyu-assistant/internal/module"
)

const (
	preferencePrefix = "BillingConvert_"
	defaultMappings  = "养老公司含补缴总计=9004\n养老个人含补缴总计=9003\n医疗公司含补缴总计=9008\n医疗个人含补缴总计=9007\n失业公司含补缴总计=9006\n失业个人含补缴总计=9005\n生育公司含补缴总计=9013\n工伤工公司含补缴总计=9011\n公积金公司含补缴总计=9015\n公积金个人含补缴总计=9014"
)

func init() {
	module.Register(&BillingConvertModule{})
}

type BillingConvertModule struct {
	state *appState
}

func (m *BillingConvertModule) Name() string {
	return i18n.T("BillingConvert", "BillingConvert")
}

func (m *BillingConvertModule) Description() string {
	return i18n.T("将社保公积金账单转换为批量上传文件", "Convert billing data to mass-upload files")
}

func (m *BillingConvertModule) Category() string {
	return "MaerskCN"
}

func (m *BillingConvertModule) Icon() fyne.Resource {
	return theme.DocumentIcon()
}

func (m *BillingConvertModule) OnInit() {
	m.state = &appState{logWidget: newReadOnlyEntry()}
}

func (m *BillingConvertModule) OnDestroy() {}

type appState struct {
	mu        sync.Mutex
	logWidget *readOnlyEntry
}

type readOnlyEntry struct {
	widget.Entry
}

func newReadOnlyEntry() *readOnlyEntry {
	entry := &readOnlyEntry{}
	entry.ExtendBaseWidget(entry)
	entry.MultiLine = true
	entry.Wrapping = fyne.TextWrapWord
	entry.TextStyle = fyne.TextStyle{Monospace: true}
	return entry
}

func (entry *readOnlyEntry) TypedRune(r rune) {}

func (entry *readOnlyEntry) TypedKey(event *fyne.KeyEvent) {
	switch event.Name {
	case fyne.KeyUp, fyne.KeyDown, fyne.KeyLeft, fyne.KeyRight, fyne.KeyPageUp, fyne.KeyPageDown, fyne.KeyHome, fyne.KeyEnd:
		entry.Entry.TypedKey(event)
	}
}

func (entry *readOnlyEntry) TypedShortcut(shortcut fyne.Shortcut) {
	if _, ok := shortcut.(*fyne.ShortcutCopy); ok {
		entry.Entry.TypedShortcut(shortcut)
	}
	if _, ok := shortcut.(*fyne.ShortcutSelectAll); ok {
		entry.Entry.TypedShortcut(shortcut)
	}
}

func (state *appState) appendLog(message string) {
	state.mu.Lock()
	defer state.mu.Unlock()

	if state.logWidget == nil {
		return
	}
	if state.logWidget.Text == "" {
		state.logWidget.SetText(message)
	} else {
		state.logWidget.SetText(state.logWidget.Text + "\n" + message)
	}
	state.logWidget.CursorRow = len(strings.Split(state.logWidget.Text, "\n")) - 1
}

type billingConfig struct {
	sourcePath string
	outputPath string
	perNoCol   string
	useDate    string
	startRow   int
	titleRow   int
	mappings   string
}

func (m *BillingConvertModule) CreateUI(window fyne.Window) fyne.CanvasObject {
	prefs := fyne.CurrentApp().Preferences()
	config := loadConfig(prefs)

	sourceEntry := widget.NewEntry()
	sourceEntry.SetText(config.sourcePath)
	sourceEntry.Disable()

	selectSourceButton := widget.NewButtonWithIcon(i18n.T("选择账单文件", "Select Billing File"), theme.FolderOpenIcon(), func() {
		go func() {
			filename, err := zenity.SelectFile(
				zenity.Title(i18n.T("选择账单文件", "Select Billing File")),
				zenity.FileFilter{Name: "Excel Files", Patterns: []string{"*.xlsx", "*.XLSX"}},
			)
			if err != nil {
				if err != zenity.ErrCanceled {
					m.state.appendLog(fmt.Sprintf("选择文件失败: %v", err))
				}
				return
			}
			if filename == "" {
				return
			}
			sourceEntry.SetText(filename)
		}()
	})

	outputEntry := widget.NewEntry()
	outputEntry.SetText(config.outputPath)
	outputEntry.SetPlaceHolder(i18n.T("默认与输入文件相同的目录", "Default: same directory as source file"))

	selectOutputButton := widget.NewButtonWithIcon(i18n.T("选择输出目录", "Select Output Folder"), theme.FolderIcon(), func() {
		go func() {
			dir, err := zenity.SelectFile(
				zenity.Title(i18n.T("选择输出目录", "Select Output Folder")),
				zenity.Directory(),
			)
			if err != nil {
				if err != zenity.ErrCanceled {
					m.state.appendLog(fmt.Sprintf("选择输出目录失败: %v", err))
				}
				return
			}
			if dir != "" {
				outputEntry.SetText(dir)
			}
		}()
	})

	perNoEntry := widget.NewEntry()
	perNoEntry.SetText(config.perNoCol)
	useDateEntry := widget.NewEntry()
	useDateEntry.SetText(config.useDate)
	startRowEntry := widget.NewEntry()
	startRowEntry.SetText(strconv.Itoa(config.startRow))
	titleRowEntry := widget.NewEntry()
	titleRowEntry.SetText(strconv.Itoa(config.titleRow))
	mappingsEntry := widget.NewMultiLineEntry()
	mappingsEntry.SetText(config.mappings)
	mappingsEntry.SetMinRowsVisible(8)

	var convertButton *widget.Button
	convertButton = widget.NewButtonWithIcon(i18n.T("开始转换", "Start Conversion"), theme.MediaPlayIcon(), func() {
		current, err := readConfig(sourceEntry.Text, outputEntry.Text, perNoEntry.Text, useDateEntry.Text, startRowEntry.Text, titleRowEntry.Text, mappingsEntry.Text)
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		saveConfig(prefs, current)
		convertButton.Disable()
		m.state.appendLog(i18n.T("开始转换...", "Starting conversion..."))

		go func() {
			result, err := convertBilling(current)
			if err != nil {
				m.state.appendLog(fmt.Sprintf("%s: %v", i18n.T("转换失败", "Conversion failed"), err))
				dialog.ShowError(err, window)
			} else {
				m.state.appendLog(fmt.Sprintf("%s: %s", i18n.T("转换完成，输出目录", "Conversion complete, output directory"), result.outputDir))
				m.state.appendLog(fmt.Sprintf("%s: %s", i18n.T("Excel 文件", "Excel file"), filepath.Base(result.excelPath)))
				m.state.appendLog(fmt.Sprintf("%s: %d", i18n.T("生成数据行数", "Generated data rows"), result.rowCount))
				dialog.ShowInformation(i18n.T("转换完成", "Conversion Complete"), fmt.Sprintf("%s\n%s", i18n.T("输出目录", "Output directory")+": "+result.outputDir, i18n.T("生成数据行数", "Generated rows")+fmt.Sprintf(": %d", result.rowCount)), window)
			}
			convertButton.Enable()
		}()
	})
	convertButton.Importance = widget.HighImportance

	form := container.NewVBox(
		widget.NewLabelWithStyle(i18n.T("输入文件", "Source File"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, nil, nil, selectSourceButton, sourceEntry),
		widget.NewLabelWithStyle(i18n.T("输出设置", "Output Settings"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, nil, nil, selectOutputButton, outputEntry),
		widget.NewLabel(i18n.T("员工编号列", "PerNo Column")),
		perNoEntry,
		widget.NewLabel(i18n.T("生效日期（YYYYMMDD）", "Use Date (YYYYMMDD)")),
		useDateEntry,
		container.NewGridWithColumns(2,
			container.NewVBox(widget.NewLabel(i18n.T("数据起始行", "Data Start Row")), startRowEntry),
			container.NewVBox(widget.NewLabel(i18n.T("表头行", "Header Row")), titleRowEntry),
		),
		widget.NewLabelWithStyle(i18n.T("表头到上传代码映射（每行：表头=代码）", "Header-to-code Mapping (one Header=Code per line)"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		mappingsEntry,
	)

	header := module.CreateHeader("BillingConvert", i18n.T("MaerskCN 社保公积金批量上传转换", "MaerskCN Billing Mass Upload Converter"))
	controlArea := container.NewBorder(header, container.NewPadded(convertButton), nil, nil, container.NewVScroll(container.NewPadded(form)))
	logTitle := widget.NewLabelWithStyle(i18n.T("转换日志", "Conversion Log"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	clearLogButton := widget.NewButton(i18n.T("清空", "Clear"), func() {
		m.state.mu.Lock()
		m.state.logWidget.SetText("")
		m.state.mu.Unlock()
	})
	logArea := container.NewBorder(
		container.NewVBox(container.NewBorder(nil, nil, logTitle, clearLogButton), widget.NewSeparator()),
		nil, nil, nil,
		m.state.logWidget,
	)

	split := container.NewHSplit(controlArea, container.NewPadded(logArea))
	split.Offset = 0.42
	return split
}

func loadConfig(prefs fyne.Preferences) billingConfig {
	return billingConfig{
		sourcePath: prefs.String(preferencePrefix + "SourcePath"),
		outputPath: prefs.String(preferencePrefix + "OutputPath"),
		perNoCol:   prefs.StringWithFallback(preferencePrefix+"PerNoColumn", "B"),
		useDate:    prefs.StringWithFallback(preferencePrefix+"UseDate", time.Now().Format("200601")+"15"),
		startRow:   prefs.IntWithFallback(preferencePrefix+"StartRow", 4),
		titleRow:   prefs.IntWithFallback(preferencePrefix+"TitleRow", 2),
		mappings:   prefs.StringWithFallback(preferencePrefix+"Mappings", defaultMappings),
	}
}

func readConfig(sourcePath, outputPath, perNoCol, useDate, startRow, titleRow, mappings string) (billingConfig, error) {
	if strings.TrimSpace(sourcePath) == "" {
		return billingConfig{}, fmt.Errorf("请选择账单 Excel 文件")
	}
	if strings.TrimSpace(perNoCol) == "" {
		return billingConfig{}, fmt.Errorf("请输入员工编号列")
	}
	if _, err := excelize.ColumnNameToNumber(strings.ToUpper(strings.TrimSpace(perNoCol))); err != nil {
		return billingConfig{}, fmt.Errorf("员工编号列无效: %w", err)
	}
	if len(strings.TrimSpace(useDate)) != 8 {
		return billingConfig{}, fmt.Errorf("生效日期必须为 YYYYMMDD 格式")
	}
	if _, err := time.Parse("20060102", strings.TrimSpace(useDate)); err != nil {
		return billingConfig{}, fmt.Errorf("生效日期无效: %w", err)
	}

	start, err := strconv.Atoi(strings.TrimSpace(startRow))
	if err != nil || start < 1 {
		return billingConfig{}, fmt.Errorf("数据起始行必须是大于 0 的整数")
	}
	title, err := strconv.Atoi(strings.TrimSpace(titleRow))
	if err != nil || title < 1 {
		return billingConfig{}, fmt.Errorf("表头行必须是大于 0 的整数")
	}
	if len(parseMappings(mappings)) == 0 {
		return billingConfig{}, fmt.Errorf("请至少配置一项有效的表头到代码映射")
	}

	return billingConfig{
		sourcePath: strings.TrimSpace(sourcePath),
		outputPath: strings.TrimSpace(outputPath),
		perNoCol:   strings.ToUpper(strings.TrimSpace(perNoCol)),
		useDate:    strings.TrimSpace(useDate),
		startRow:   start,
		titleRow:   title,
		mappings:   strings.TrimSpace(mappings),
	}, nil
}

func saveConfig(prefs fyne.Preferences, config billingConfig) {
	prefs.SetString(preferencePrefix+"SourcePath", config.sourcePath)
	prefs.SetString(preferencePrefix+"OutputPath", config.outputPath)
	prefs.SetString(preferencePrefix+"PerNoColumn", config.perNoCol)
	prefs.SetString(preferencePrefix+"UseDate", config.useDate)
	prefs.SetInt(preferencePrefix+"StartRow", config.startRow)
	prefs.SetInt(preferencePrefix+"TitleRow", config.titleRow)
	prefs.SetString(preferencePrefix+"Mappings", config.mappings)
}

type conversionResult struct {
	outputDir string
	excelPath string
	rowCount  int
}

func convertBilling(config billingConfig) (conversionResult, error) {
	source, err := excelize.OpenFile(config.sourcePath)
	if err != nil {
		return conversionResult{}, fmt.Errorf("打开源 Excel 文件失败: %w", err)
	}
	defer source.Close()

	sheets := source.GetSheetList()
	if len(sheets) == 0 {
		return conversionResult{}, fmt.Errorf("源 Excel 文件没有工作表")
	}
	sheet := sheets[0]
	rows, err := source.GetRows(sheet)
	if err != nil {
		return conversionResult{}, fmt.Errorf("读取源数据失败: %w", err)
	}
	if len(rows) < config.titleRow {
		return conversionResult{}, fmt.Errorf("源文件没有第 %d 行表头", config.titleRow)
	}

	mappings := parseMappings(config.mappings)
	codeByColumn := make(map[int]string)
	for column, title := range rows[config.titleRow-1] {
		if code, ok := mappings[strings.TrimSpace(title)]; ok {
			codeByColumn[column] = code
		}
	}
	if len(codeByColumn) == 0 {
		return conversionResult{}, fmt.Errorf("未找到与映射匹配的源文件表头")
	}

	perNoColumn, _ := excelize.ColumnNameToNumber(config.perNoCol)
	perNoColumn--

	output := excelize.NewFile()
	defer output.Close()
	outputSheet := output.GetSheetName(output.GetActiveSheetIndex())
	headers := []string{"PERNR", "SUBTY", "BEGDA", "BETRG"}
	for index, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(index+1, 1)
		if err := output.SetCellValue(outputSheet, cell, header); err != nil {
			return conversionResult{}, fmt.Errorf("写入输出表头失败: %w", err)
		}
	}

	targetRow := 2
	for rowIndex := config.startRow - 1; rowIndex < len(rows); rowIndex++ {
		row := rows[rowIndex]
		if len(row) <= perNoColumn || !isNumericEmployeeNumber(row[perNoColumn]) {
			continue
		}
		for column, code := range codeByColumn {
			value := ""
			if len(row) > column {
				value = row[column]
			}
			values := []string{row[perNoColumn], code, config.useDate, value}
			for index, value := range values {
				cell, _ := excelize.CoordinatesToCellName(index+1, targetRow)
				if err := output.SetCellValue(outputSheet, cell, value); err != nil {
					return conversionResult{}, fmt.Errorf("写入第 %d 行失败: %w", targetRow, err)
				}
			}
			targetRow++
		}
	}

	outputDir := config.outputPath
	if outputDir == "" {
		outputDir = filepath.Dir(config.sourcePath)
	} else if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Clean(outputDir)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return conversionResult{}, fmt.Errorf("创建输出目录失败: %w", err)
	}

	excelPath := filepath.Join(outputDir, fmt.Sprintf("MaerskLine%s.xlsx", time.Now().Format("200601021504")))
	if err := output.SaveAs(excelPath); err != nil {
		return conversionResult{}, fmt.Errorf("保存 Excel 文件失败: %w", err)
	}
	if err := writeTextChunks(output, outputSheet, outputDir, targetRow-1); err != nil {
		return conversionResult{}, err
	}

	return conversionResult{outputDir: outputDir, excelPath: excelPath, rowCount: targetRow - 2}, nil
}

func parseMappings(raw string) map[string]string {
	mappings := make(map[string]string)
	for _, line := range strings.Split(raw, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && strings.TrimSpace(parts[1]) != "" {
			mappings[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return mappings
}

func isNumericEmployeeNumber(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func writeTextChunks(output *excelize.File, sheet, outputDir string, rowCount int) error {
	rows, err := output.GetRows(sheet)
	if err != nil {
		return fmt.Errorf("读取转换结果失败: %w", err)
	}
	if len(rows) == 0 {
		return fmt.Errorf("转换结果缺少表头")
	}

	const chunkSize = 5000
	for offset := 0; offset < rowCount; offset += chunkSize {
		end := offset + chunkSize
		if end > rowCount {
			end = rowCount
		}
		file, err := os.Create(filepath.Join(outputDir, fmt.Sprintf("ML%d.txt", offset/chunkSize+1)))
		if err != nil {
			return fmt.Errorf("创建文本文件失败: %w", err)
		}

		_, writeErr := file.WriteString(strings.Join(rows[0], "\t") + "\n")
		for _, row := range rows[offset+1 : end+1] {
			for len(row) < 4 {
				row = append(row, "")
			}
			if writeErr == nil {
				_, writeErr = file.WriteString(strings.Join(row[:4], "\t") + "\n")
			}
		}
		closeErr := file.Close()
		if writeErr != nil {
			return fmt.Errorf("写入文本文件失败: %w", writeErr)
		}
		if closeErr != nil {
			return fmt.Errorf("关闭文本文件失败: %w", closeErr)
		}
	}
	return nil
}
