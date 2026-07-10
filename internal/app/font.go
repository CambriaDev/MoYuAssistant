package app

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"github.com/flopp/go-findfont"

	"moyu-assistant/internal/i18n"
)

var customFont fyne.Resource

func init() {
	fontPaths := findfont.List()
	var fontPath string
	for _, path := range fontPaths {
		lowerPath := strings.ToLower(path)
		if strings.Contains(lowerPath, "msyh.ttc") || strings.Contains(lowerPath, "msyh.ttf") ||
			strings.Contains(lowerPath, "simhei.ttf") || strings.Contains(lowerPath, "simsun.ttc") ||
			strings.Contains(lowerPath, "simkai.ttf") ||
			strings.Contains(lowerPath, "pingfang.ttc") ||
			strings.Contains(lowerPath, "wqy-microhei.ttc") {
			fontPath = path
			break
		}
	}

	if fontPath != "" {
		fmt.Println("[MoYuAssistant] 找到中文字体 / Found CJK font:", fontPath)
		fontBytes, err := os.ReadFile(fontPath)
		if err == nil {
			fileName := filepath.Base(fontPath)
			customFont = fyne.NewStaticResource(fileName, fontBytes)
			fmt.Println("[MoYuAssistant] 成功加载字体资源 / Successfully loaded font:", fileName)
			i18n.UseEnglish = false
		} else {
			fmt.Println("[MoYuAssistant] 读取字体文件失败 / Failed to read font file:", err)
			i18n.UseEnglish = true
		}
	} else {
		fmt.Println("[MoYuAssistant] 未找到中文字体，切换至英文模式 / No CJK font found, falling back to English UI.")
		i18n.UseEnglish = true
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
