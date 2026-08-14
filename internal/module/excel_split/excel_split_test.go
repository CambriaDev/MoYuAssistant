package excel_split

import (
	"path/filepath"
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
