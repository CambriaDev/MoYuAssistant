package excel_split

import (
	"path/filepath"
	"strconv"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestSplitExcelFileSkipsMissingSheets(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "LZACN_LZACN030_source.xlsx")
	f := excelize.NewFile()
	if err := f.SaveAs(filePath); err != nil {
		t.Fatalf("save source workbook: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close source workbook: %v", err)
	}

	convertedSheets, err := splitExcelFile(filePath, "")
	if err != nil {
		t.Fatalf("splitExcelFile returned an error for missing sheets: %v", err)
	}
	if len(convertedSheets) != 0 {
		t.Fatalf("converted sheets = %v, want none", convertedSheets)
	}
}

func TestSplitExcelFileSkipsHeaderOnlySheets(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "LZACN_LZACN030_source.xlsx")
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", "DATA_CHANGE"); err != nil {
		t.Fatalf("rename source sheet: %v", err)
	}
	for row := 1; row <= 3; row++ {
		if err := f.SetCellValue("DATA_CHANGE", "A"+strconv.Itoa(row), "header"); err != nil {
			t.Fatalf("set header row %d: %v", row, err)
		}
	}
	if err := f.SetCellValue("DATA_CHANGE", "A4", "   "); err != nil {
		t.Fatalf("set blank data row: %v", err)
	}
	if err := f.SaveAs(filePath); err != nil {
		t.Fatalf("save source workbook: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close source workbook: %v", err)
	}

	convertedSheets, err := splitExcelFile(filePath, "")
	if err != nil {
		t.Fatalf("splitExcelFile returned an error for a header-only sheet: %v", err)
	}
	if len(convertedSheets) != 0 {
		t.Fatalf("converted sheets = %v, want none", convertedSheets)
	}
}

func TestHasDataRows(t *testing.T) {
	tests := []struct {
		name       string
		rows       [][]string
		headerRows int
		want       bool
	}{
		{
			name:       "only headers",
			rows:       [][]string{{"header"}, {"header"}},
			headerRows: 2,
			want:       false,
		},
		{
			name:       "blank rows after headers",
			rows:       [][]string{{"header"}, {"  "}, {}},
			headerRows: 1,
			want:       false,
		},
		{
			name:       "value after headers",
			rows:       [][]string{{"header"}, {"", "data"}},
			headerRows: 1,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasDataRows(tt.rows, tt.headerRows); got != tt.want {
				t.Errorf("hasDataRows() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestAddTimestampSuffix(t *testing.T) {
	got := addTimestampSuffix("LZACN0301994_Social Insurance.xlsx", "202608142012")
	want := "LZACN0301994_SocialInsurance_202608142012.xlsx"
	if got != want {
		t.Errorf("addTimestampSuffix() = %q, want %q", got, want)
	}
}

func TestFindMatchingSheetName(t *testing.T) {
	tests := []struct {
		name          string
		sheetNames    []string
		acceptedNames []string
		want          string
	}{
		{
			name:          "matches public housing fund alias",
			sheetNames:    []string{"DATA_CHANGE (cnphf)"},
			acceptedNames: []string{"DATA PHF (ViewID=284)", "DATA_CHANGE (cnphf)"},
			want:          "DATA_CHANGE (cnphf)",
		},
		{
			name:          "matches social insurance alias",
			sheetNames:    []string{"DATA_CHANGE (cnsi)"},
			acceptedNames: []string{"DATA SI (ViewId=286)", "DATA_CHANGE (cnsi)"},
			want:          "DATA_CHANGE (cnsi)",
		},
		{
			name:          "matches names ignoring case and spaces",
			sheetNames:    []string{"data_change(cnphf)"},
			acceptedNames: []string{"DATA_CHANGE (cnphf)"},
			want:          "data_change(cnphf)",
		},
		{
			name:          "returns empty for missing sheet",
			sheetNames:    []string{"Sheet1"},
			acceptedNames: []string{"DATA_CHANGE"},
			want:          "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findMatchingSheetName(tt.sheetNames, tt.acceptedNames); got != tt.want {
				t.Errorf("findMatchingSheetName() = %q, want %q", got, tt.want)
			}
		})
	}
}
