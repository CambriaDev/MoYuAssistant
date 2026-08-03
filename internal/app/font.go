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
// 同时为暗色主题提供基于 #9050a8 和 #4b6cb7 的统一配色方案
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
	// 仅对暗色主题应用自定义配色，亮色主题保持默认
	if variant != 0 { // 0 == dark
		return c.fallback.Color(name, variant)
	}

	// 品牌色基准:
	//   Purple #9050a8  -> R:144 G:80  B:168
	//   Blue   #4b6cb7  -> R:75  G:108 B:183
	// 以下各色均由这两个颜色调整饱和度/亮度派生而来
	switch name {
	// ── Primary: 主紫色，用于高亮按钮、选中态 ──
	case "primary":
		return color.NRGBA{R: 144, G: 80, B: 168, A: 255} // #9050a8

	// ── Focus: 聚焦态，稍亮的紫 ──
	case "focus":
		return color.NRGBA{R: 160, G: 100, B: 185, A: 255} // 亮紫

	// ── Selection: 选中高亮，低透明度的品牌蓝紫 ──
	case "selection":
		return color.NRGBA{R: 110, G: 90, B: 150, A: 80}

	// ── Hover: 鼠标悬停，非常淡的紫调 ──
	case "hover":
		return color.NRGBA{R: 100, G: 80, B: 140, A: 50}

	// ── Pressed: 按下效果 ──
	case "pressed":
		return color.NRGBA{R: 100, G: 80, B: 140, A: 80}

	// ── Button: 普通按钮底色，深蓝紫 ──
	case "button":
		return color.NRGBA{R: 55, G: 50, B: 75, A: 255} // 深蓝紫灰

	// ── Disabled button: 禁用态按钮 ──
	case "disabledButton":
		return color.NRGBA{R: 45, G: 42, B: 58, A: 255}

	// ── Background: 主背景色，极深的偏紫灰 ──
	case "background":
		return color.NRGBA{R: 24, G: 22, B: 30, A: 255} // 极深紫灰

	// ── Separator: 分割线，稍亮一点的紫灰 ──
	case "separator":
		return color.NRGBA{R: 55, G: 50, B: 70, A: 255}

	// ── InputBackground: 输入框背景 ──
	case "inputBackground":
		return color.NRGBA{R: 32, G: 30, B: 42, A: 255}

	// ── InputBorder: 输入框边框，蓝调 ──
	case "inputBorder":
		return color.NRGBA{R: 65, G: 75, B: 105, A: 255}

	// ── HeaderBackground: 列表/集合表头 ──
	case "headerBackground":
		return color.NRGBA{R: 38, G: 35, B: 50, A: 255}

	// ── MenuBackground: 菜单背景 ──
	case "menuBackground":
		return color.NRGBA{R: 30, G: 28, B: 40, A: 255}

	// ── OverlayBackground: 弹出层/对话框背景 ──
	case "overlayBackground":
		return color.NRGBA{R: 35, G: 32, B: 46, A: 255}

	// ── ScrollBar: 滚动条 ──
	case "scrollBar":
		return color.NRGBA{R: 80, G: 70, B: 110, A: 150}

	// ── Shadow: 阴影 ──
	case "shadow":
		return color.NRGBA{R: 10, G: 8, B: 18, A: 100}

	// ── Hyperlink: 超链接，用品牌蓝 ──
	case "hyperlink":
		return color.NRGBA{R: 100, G: 140, B: 210, A: 255} // 亮蓝

	// ── Foreground on primary: 主色上的前景 ──
	case "foregroundOnPrimary":
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}

	// ── Success: 成功色，偏绿但带一点蓝调以和品牌色协调 ──
	case "success":
		return color.NRGBA{R: 70, G: 180, B: 140, A: 255}

	// ── Warning: 警告色，暖色但不刺眼 ──
	case "warning":
		return color.NRGBA{R: 210, G: 170, B: 80, A: 255}

	// ── Error: 错误色 ──
	case "error":
		return color.NRGBA{R: 200, G: 80, B: 90, A: 255}

	// ── Placeholder: 占位文字，柔和灰紫 ──
	case "placeholder":
		return color.NRGBA{R: 110, G: 100, B: 130, A: 255}

	// ── Disabled: 禁用前景色 ──
	case "disabled":
		return color.NRGBA{R: 85, G: 80, B: 100, A: 255}
	}

	return c.fallback.Color(name, variant)
}

func (c *cjkTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return c.fallback.Icon(name)
}

func (c *cjkTheme) Size(name fyne.ThemeSizeName) float32 {
	return c.fallback.Size(name)
}

