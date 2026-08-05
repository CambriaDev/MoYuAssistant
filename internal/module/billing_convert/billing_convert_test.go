//go:build module_billing_convert

package billing_convert

import (
	"fmt"
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
	if err := source.SetCellValue(sheet, "A5", ""); err != nil {
		t.Fatal(err)
	}
	if err := source.SetCellValue(sheet, "B5", "30.00"); err != nil {
		t.Fatal(err)
	}
	if err := source.SetCellValue(sheet, "A6", "10002A"); err != nil {
		t.Fatal(err)
	}
	if err := source.SetCellValue(sheet, "B6", "40.00"); err != nil {
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
	if result.outputDir != tempDir {
		t.Fatalf("outputDir = %q, want source directory %q", result.outputDir, tempDir)
	}

	output, err := excelize.OpenFile(result.excelPath)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	if sheetNames := output.GetSheetList(); strings.Join(sheetNames, ",") != "Sheet1,ML1" {
		t.Fatalf("sheet names = %v, want [Sheet1 ML1]", sheetNames)
	}
	allRows, err := output.GetRows("Sheet1")
	if err != nil {
		t.Fatal(err)
	}
	if len(allRows) != 3 {
		t.Fatalf("all-data rows = %d, want 3", len(allRows))
	}
	rows, err := output.GetRows("ML1")
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
	assertTextMatchesSheet(t, string(textFile), rows)
}

func TestIsNumericEmployeeNumber(t *testing.T) {
	for _, testCase := range []struct {
		value string
		want  bool
	}{
		{value: "10001", want: true},
		{value: "00123", want: true},
		{value: "", want: false},
		{value: " ", want: false},
		{value: "10001A", want: false},
		{value: "100.01", want: false},
	} {
		if got := isNumericEmployeeNumber(testCase.value); got != testCase.want {
			t.Errorf("isNumericEmployeeNumber(%q) = %t, want %t", testCase.value, got, testCase.want)
		}
	}
}

func TestConvertBillingSplitsSheetsAndTextFiles(t *testing.T) {
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source.xlsx")
	source := excelize.NewFile()
	sheet := source.GetSheetName(source.GetActiveSheetIndex())
	for cell, value := range map[string]string{
		"A2": "员工编号",
		"B2": "养老公司含补缴总计",
	} {
		if err := source.SetCellValue(sheet, cell, value); err != nil {
			t.Fatal(err)
		}
	}
	for index := 0; index <= chunkSize; index++ {
		row := index + 4
		if err := source.SetCellValue(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("%05d", index+1)); err != nil {
			t.Fatal(err)
		}
		if err := source.SetCellValue(sheet, fmt.Sprintf("B%d", row), "10.00"); err != nil {
			t.Fatal(err)
		}
	}
	if err := source.SaveAs(sourcePath); err != nil {
		t.Fatal(err)
	}
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}

	result, err := convertBilling(billingConfig{
		sourcePath: sourcePath,
		perNoCol:   "A",
		useDate:    "20260815",
		startRow:   4,
		titleRow:   2,
		mappings:   "养老公司含补缴总计=9004",
	})
	if err != nil {
		t.Fatal(err)
	}

	output, err := excelize.OpenFile(result.excelPath)
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	if sheetNames := output.GetSheetList(); strings.Join(sheetNames, ",") != "Sheet1,ML1,ML2" {
		t.Fatalf("sheet names = %v, want [Sheet1 ML1 ML2]", sheetNames)
	}
	allRows, err := output.GetRows("Sheet1")
	if err != nil {
		t.Fatal(err)
	}
	if len(allRows) != chunkSize+2 {
		t.Fatalf("all-data rows = %d, want %d", len(allRows), chunkSize+2)
	}
	for sheetName, wantRows := range map[string]int{"ML1": chunkSize + 1, "ML2": 2} {
		rows, err := output.GetRows(sheetName)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != wantRows {
			t.Fatalf("%s rows = %d, want %d", sheetName, len(rows), wantRows)
		}
		text, err := os.ReadFile(filepath.Join(result.outputDir, sheetName+".txt"))
		if err != nil {
			t.Fatal(err)
		}
		assertTextMatchesSheet(t, string(text), rows)
	}
}

func assertTextMatchesSheet(t *testing.T, text string, rows [][]string) {
	t.Helper()
	var expected strings.Builder
	for _, row := range rows {
		for len(row) < 4 {
			row = append(row, "")
		}
		expected.WriteString(strings.Join(row[:4], "\t"))
		expected.WriteByte('\n')
	}
	if text != expected.String() {
		t.Fatal("text file content does not match its worksheet")
	}
}
