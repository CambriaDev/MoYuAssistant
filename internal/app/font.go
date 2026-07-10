package app

import (
	"os"
	"strings"

	"github.com/flopp/go-findfont"
)

func init() {
	// 如果用户已经手动设置了 FYNE_FONT 环境变量，则尊重用户的设置
	if os.Getenv("FYNE_FONT") != "" {
		return
	}

	// 遍历系统字体，寻找常见的中文字体并设置为 Fyne 的默认字体
	fontPaths := findfont.List()
	for _, path := range fontPaths {
		lowerPath := strings.ToLower(path)
		
		// 优先使用微软雅黑 (Windows)
		if strings.Contains(lowerPath, "msyh.ttc") || strings.Contains(lowerPath, "msyh.ttf") {
			os.Setenv("FYNE_FONT", path)
			return
		}
		// 其次是黑体 (Windows)
		if strings.Contains(lowerPath, "simhei.ttf") {
			os.Setenv("FYNE_FONT", path)
			return
		}
		// 苹方 (macOS)
		if strings.Contains(lowerPath, "pingfang.ttc") {
			os.Setenv("FYNE_FONT", path)
			return
		}
		// 文泉驿微米黑 (Linux)
		if strings.Contains(lowerPath, "wqy-microhei.ttc") {
			os.Setenv("FYNE_FONT", path)
			return
		}
	}
}
