package app

import (
	"image/color"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/flopp/go-findfont"
)

var customFont fyne.Resource

func init() {
	fontPaths := findfont.List()
	var fontPath string
	for _, path := range fontPaths {
		lowerPath := strings.ToLower(path)
		// 常见中文字体
		if strings.Contains(lowerPath, "msyh.ttc") || strings.Contains(lowerPath, "msyh.ttf") ||
			strings.Contains(lowerPath, "simhei.ttf") ||
			strings.Contains(lowerPath, "pingfang.ttc") ||
			strings.Contains(lowerPath, "wqy-microhei.ttc") {
			fontPath = path
			break
		}
	}

	if fontPath != "" {
		fontBytes, err := os.ReadFile(fontPath)
		if err == nil {
			customFont = fyne.NewStaticResource("cjk.ttf", fontBytes)
		}
	}
}

// cjkTheme 包装原有的 Theme，但在请求字体时始终返回支持中文的 customFont
type cjkTheme struct {
	fallback fyne.Theme
}

func (c *cjkTheme) Font(style fyne.TextStyle) fyne.Resource {
	// 不管是 Bold、Italic 还是 Monospace，都统一使用我们的中文字体，防止部分文字变方块
	if customFont != nil {
		return customFont
	}
	return c.fallback.Font(style)
}

func (c *cjkTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return c.fallback.Color(name, variant)
}

func (c *cjkTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return c.fallback.Icon(name)
}

func (c *cjkTheme) Size(name fyne.ThemeSizeName) float32 {
	return c.fallback.Size(name)
}
