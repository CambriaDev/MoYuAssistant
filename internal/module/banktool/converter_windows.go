//go:build windows

package banktool

import (
	"fmt"
	"path/filepath"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// saveAsXLSViaExcel uses MS Excel COM automation to convert the file perfectly.
// Requires MS Excel installed on the Windows system.
func saveAsXLSViaExcel(xlsxPath, xlsPath string) error {
	// Initialize COM
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		if oleErr, ok := err.(*ole.OleError); ok && oleErr.Code() == ole.S_FALSE {
			// already initialized
		} else {
			return fmt.Errorf("CoInitializeEx failed: %v", err)
		}
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("Excel.Application")
	if err != nil {
		return fmt.Errorf("could not create Excel.Application (is MS Office installed?): %v", err)
	}
	excel, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer excel.Release()

	// Hide excel window
	oleutil.PutProperty(excel, "Visible", false)
	oleutil.PutProperty(excel, "DisplayAlerts", false)

	// Make sure we quit excel
	defer oleutil.CallMethod(excel, "Quit")

	workbooks := oleutil.MustGetProperty(excel, "Workbooks").ToIDispatch()
	if workbooks == nil {
		return fmt.Errorf("failed to get Workbooks property")
	}
	defer workbooks.Release()

	absXlsx, _ := filepath.Abs(xlsxPath)
	absXls, _ := filepath.Abs(xlsPath)

	workbook, err := oleutil.CallMethod(workbooks, "Open", absXlsx)
	if err != nil {
		return fmt.Errorf("failed to open workbook %s: %v", absXlsx, err)
	}
	wbDispatch := workbook.ToIDispatch()
	defer wbDispatch.Release()
	
	// Close workbook without saving changes when done
	defer oleutil.CallMethod(wbDispatch, "Close", false)

	// 56 = xlExcel8 (Excel 97-2003 format)
	_, err = oleutil.CallMethod(wbDispatch, "SaveAs", absXls, 56)
	if err != nil {
		return fmt.Errorf("failed to SaveAs %s: %v", absXls, err)
	}

	return nil
}
