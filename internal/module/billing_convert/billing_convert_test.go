//go:build module_billing_convert

package billing_convert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestConvertBillingCreatesUploadFiles(t *testing.T) {
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source.xlsx")

	source := excelize.NewFile()
	sheet := source.GetSheetName(source.GetActiveSheetIndex())
	if err := source.SetCellValue(sheet, "A2", "员工编号"); err != nil {
		t.Fatal(err)
	}
	if err := source.SetCellValue(sheet, "B2", "养老公司含补缴总计"); err != nil {
		t.Fatal(err)
	}
	if err := source.SetCellValue(sheet, "C2", "医疗个人含补缴总计"); err != nil {
		t.Fatal(err)
	}
	if err := source.SetCellValue(sheet, "A4", "10001"); err != nil {
		t.Fatal(err)
	}
	if err := source.SetCellValue(sheet, "B4", "10.50"); err != nil {
		t.Fatal(err)
	}
	if err := source.SetCellValue(sheet, "C4", "20.75"); err != nil {
		t.Fatal(err)
	}
	if err := source.SaveAs(sourcePath); err != nil {
		t.Fatal(err)
	}
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}

	result, err := convertBilling(billingConfig{
		sourcePath: sourcePath,
		outputPath: filepath.Join(tempDir, "output"),
		perNoCol:   "A",
		useDate:    "20260815",
		startRow:   4,
		titleRow:   2,
		mappings:   "养老公司含补缴总计=9004\n医疗个人含补缴总计=9007",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.rowCount != 2 {
		t.Fatalf("rowCount = %d, want 2", result.rowCount)
	}

	output, err := excelize.OpenFile(result.excelPath)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	rows, err := output.GetRows(output.GetSheetName(output.GetActiveSheetIndex()))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("output rows = %d, want 3", len(rows))
	}
	if strings.Join(rows[0], ",") != "PERNR,SUBTY,BEGDA,BETRG" {
		t.Fatalf("headers = %v", rows[0])
	}

	textFile, err := os.ReadFile(filepath.Join(result.outputDir, "ML1.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Split(strings.TrimSpace(string(textFile)), "\n"); len(lines) != 3 {
		t.Fatalf("text lines = %d, want 3", len(lines))
	}
}
