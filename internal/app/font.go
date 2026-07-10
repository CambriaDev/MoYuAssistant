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
	fontPath := findPreferredFont(fontPaths)

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

func findPreferredFont(fontPaths []string) string {
	preferredNames := []string{
		"simhei.ttf",
		"simfang.ttf",
		"simkai.ttf",
		"simsunb.ttf",
		"msyh.ttf",
		"msyhbd.ttf",
		"msyhl.ttf",
		"pingfang.ttf",
	}

	for _, preferredName := range preferredNames {
		for _, path := range fontPaths {
			if strings.EqualFold(filepath.Base(path), preferredName) {
				return path
			}
		}
	}

	for _, path := range fontPaths {
		lowerPath := strings.ToLower(path)
		if strings.HasSuffix(lowerPath, ".ttc") || strings.HasSuffix(lowerPath, ".otc") {
			continue
		}

		if strings.Contains(lowerPath, "msyh") ||
			strings.Contains(lowerPath, "simhei") ||
			strings.Contains(lowerPath, "simfang") ||
			strings.Contains(lowerPath, "simkai") ||
			strings.Contains(lowerPath, "simsun") ||
			strings.Contains(lowerPath, "pingfang") ||
			strings.Contains(lowerPath, "noto sans cjk") ||
			strings.Contains(lowerPath, "sourcehansans") ||
			strings.Contains(lowerPath, "sarasa") ||
			strings.Contains(lowerPath, "wqy") {
			return path
		}
	}

	return ""
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
