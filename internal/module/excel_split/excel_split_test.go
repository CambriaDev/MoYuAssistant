package excel_split

import (
	"os"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestSplitExcelFile(t *testing.T) {
	// change working directory to project root
	os.Chdir("../../..")
	f, _ := excelize.OpenFile("test/LZACN_LZACN0301999_ Mass Upload --CN03-20260717.xlsx")
	t.Logf("Sheet list: %v", f.GetSheetList())
	err := splitExcelFile("test/LZACN_LZACN0301999_ Mass Upload --CN03-20260717.xlsx", "")
	if err != nil {
		t.Fatalf("splitExcelFile failed: %v", err)
	}
}
