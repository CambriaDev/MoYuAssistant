//go:build module_excel_split

package excel_split

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
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

type ExcelSplitModule struct{}

func (m *ExcelSplitModule) Name() string        { return i18n.T("MassUpload", "MassUpload") }
func (m *ExcelSplitModule) Description() string { return i18n.T("按Sheet页拆分Excel文件", "Split Excel file by sheets") }
func (m *ExcelSplitModule) Icon() fyne.Resource { return theme.DocumentIcon() }

func (m *ExcelSplitModule) CreateUI(w fyne.Window) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(i18n.T("📑 Excel按Sheet拆分", "📑 Split Excel by Sheet"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	desc := widget.NewLabel(i18n.T("选择一个Excel文件，将自动按Sheet页拆分成多个文件。", "Select an Excel file, it will be automatically split by sheets into multiple files."))
	desc.Alignment = fyne.TextAlignCenter

	statusLabel := widget.NewLabel("")
	statusLabel.Alignment = fyne.TextAlignCenter
	
	var selectedFile string
	fileLabel := widget.NewLabel(i18n.T("未选择文件", "No file selected"))
	fileLabel.Alignment = fyne.TextAlignCenter

	selectBtn := widget.NewButtonWithIcon(i18n.T("选择Excel文件", "Select Excel File"), theme.FolderOpenIcon(), func() {
		go func() {
			filename, err := zenity.SelectFile(
				zenity.Title("Select Excel File"),
				zenity.FileFilter{Name: "Excel Files", Patterns: []string{"*.xlsx", "*.XLSX"}},
			)
			if err != nil {
				if err != zenity.ErrCanceled {
					dialog.ShowError(err, w)
				}
				return
			}
			if filename == "" {
				return
			}

			selectedFile = filename
			fileLabel.SetText(fmt.Sprintf("%s: %s", i18n.T("已选择", "Selected"), filepath.Base(selectedFile)))
			statusLabel.SetText("")
		}()
	})

	appPrefs := fyne.CurrentApp().Preferences()
	lastOutDir := appPrefs.StringWithFallback("MassUpload_OutDir", "")
	var selectedOutDir string = lastOutDir

	outDirLabel := widget.NewLabel("")
	outDirLabel.Alignment = fyne.TextAlignCenter
	if selectedOutDir == "" {
		outDirLabel.SetText(i18n.T("输出目录: (默认同源文件目录)", "Output Dir: (Default same as source)"))
	} else {
		outDirLabel.SetText(i18n.T("输出目录: ", "Output Dir: ") + selectedOutDir)
	}

	selectOutDirBtn := widget.NewButtonWithIcon(i18n.T("选择输出目录", "Select Output Dir"), theme.FolderIcon(), func() {
		go func() {
			dir, err := zenity.SelectFile(
				zenity.Title("Select Output Folder"),
				zenity.Directory(),
			)
			if err != nil {
				if err != zenity.ErrCanceled {
					dialog.ShowError(err, w)
				}
				return
			}
			if dir == "" {
				return
			}
			
			selectedOutDir = dir
			appPrefs.SetString("MassUpload_OutDir", selectedOutDir)
			outDirLabel.SetText(i18n.T("输出目录: ", "Output Dir: ") + selectedOutDir)
		}()
	})

	processBtn := widget.NewButtonWithIcon(i18n.T("开始拆分", "Start Splitting"), theme.MediaPlayIcon(), func() {
		if selectedFile == "" {
			dialog.ShowInformation(i18n.T("提示", "Info"), i18n.T("请先选择一个Excel文件", "Please select an Excel file first"), w)
			return
		}
		
		statusLabel.SetText(i18n.T("正在处理中...", "Processing..."))
		
		go func() {
			err := splitExcelFile(selectedFile, selectedOutDir)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("%s: %v", i18n.T("处理失败", "Failed"), err))
			} else {
				statusLabel.SetText(i18n.T("拆分完成！", "Split successfully!"))
			}
		}()
	})
	processBtn.Importance = widget.HighImportance

	return container.New(layout.NewVBoxLayout(),
		title,
		desc,
		layout.NewSpacer(),
		fileLabel,
		container.NewHBox(layout.NewSpacer(), selectBtn, layout.NewSpacer()),
		layout.NewSpacer(),
		outDirLabel,
		container.NewHBox(layout.NewSpacer(), selectOutDirBtn, layout.NewSpacer()),
		layout.NewSpacer(),
		container.NewHBox(layout.NewSpacer(), processBtn, layout.NewSpacer()),
		layout.NewSpacer(),
		statusLabel,
		layout.NewSpacer(),
	)
}

func (m *ExcelSplitModule) OnInit()    {}
func (m *ExcelSplitModule) OnDestroy() {}
func (m *ExcelSplitModule) Category() string { return "LonzaCN" }

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
