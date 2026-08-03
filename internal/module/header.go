package module

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

// CreateHeader returns a standardized header for modules with a consistent gradient background.
func CreateHeader(title, subtitle string) fyne.CanvasObject {
	var startColor, endColor color.Color
	if fyne.CurrentApp().Settings().ThemeVariant() == theme.VariantDark {
		startColor = color.NRGBA{R: 144, G: 80, B: 168, A: 255} // #9050a8
		endColor = color.NRGBA{R: 75, G: 108, B: 183, A: 255}   // #4b6cb7
	} else {
		startColor = color.NRGBA{R: 90, G: 110, B: 140, A: 255}
		endColor = color.NRGBA{R: 160, G: 175, B: 190, A: 255}
	}

	headerGrad := canvas.NewLinearGradient(startColor, endColor, -45)

	headerText := canvas.NewText(title, color.White)
	headerText.TextSize = 28
	headerText.TextStyle = fyne.TextStyle{Bold: true}
	headerText.Alignment = fyne.TextAlignCenter

	subHeaderText := canvas.NewText(subtitle, color.NRGBA{255, 255, 255, 200})
	subHeaderText.TextSize = 12
	subHeaderText.Alignment = fyne.TextAlignCenter

	headerContent := container.NewVBox(
		layout.NewSpacer(),
		headerText,
		subHeaderText,
		layout.NewSpacer(),
	)

	spacerRect := canvas.NewRectangle(color.Transparent)
	spacerRect.SetMinSize(fyne.NewSize(1, 100))

	return container.NewMax(headerGrad, spacerRect, container.NewPadded(headerContent))
}
