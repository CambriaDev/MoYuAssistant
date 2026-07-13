//go:build !windows

package banktool

import (
	"fmt"
)

// saveAsXLSViaExcel is a stub for non-Windows platforms.
func saveAsXLSViaExcel(xlsxPath, xlsPath string) error {
	return fmt.Errorf("native Excel conversion is only supported on Windows")
}
