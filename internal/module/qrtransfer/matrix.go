//go:build module_qr_transfer

package qrtransfer

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	matrixColumns    = 20
	matrixCellPitch  = 12
	matrixCellSize   = 8
	matrixPadding    = 4
	matrixLabelWidth = 38
)

var (
	matrixEmptyColor   = color.NRGBA{R: 216, G: 222, B: 228, A: 255}
	matrixCurrentColor = color.NRGBA{R: 9, G: 105, B: 218, A: 255}
	matrixLabelColor   = color.NRGBA{R: 87, G: 96, B: 106, A: 255}
)

type chunkMatrix struct {
	widget.BaseWidget
	total    int
	current  int
	cells    []*canvas.Rectangle
	content  *fyne.Container
	onSelect func(int)
}

func newChunkMatrix(onSelect func(int)) *chunkMatrix {
	matrix := &chunkMatrix{
		current:  -1,
		content:  container.NewWithoutLayout(),
		onSelect: onSelect,
	}
	matrix.ExtendBaseWidget(matrix)
	return matrix
}

func (m *chunkMatrix) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(m.content)
}

func (m *chunkMatrix) SetTotal(total int) {
	m.total = total
	m.current = -1
	m.cells = make([]*canvas.Rectangle, total)
	objects := make([]fyne.CanvasObject, 0, total+(total+matrixColumns-1)/matrixColumns)
	rows := (total + matrixColumns - 1) / matrixColumns

	for row := 0; row < rows; row++ {
		y := float32(matrixPadding + row*matrixCellPitch)
		label := canvas.NewText(fmt.Sprint(row*matrixColumns+1), matrixLabelColor)
		label.Alignment = fyne.TextAlignTrailing
		label.TextSize = 9
		label.Move(fyne.NewPos(0, y))
		label.Resize(fyne.NewSize(matrixLabelWidth-matrixPadding, matrixCellPitch))
		objects = append(objects, label)

		for column := 0; column < matrixColumns; column++ {
			index := row*matrixColumns + column
			if index >= total {
				break
			}
			cell := canvas.NewRectangle(matrixEmptyColor)
			cell.Move(fyne.NewPos(
				float32(matrixPadding+matrixLabelWidth+column*matrixCellPitch),
				y,
			))
			cell.Resize(fyne.NewSize(matrixCellSize, matrixCellSize))
			m.cells[index] = cell
			objects = append(objects, cell)
		}
	}

	m.content.Objects = objects
	m.content.Resize(m.MinSize())
	m.content.Refresh()
	m.Refresh()
}

func (m *chunkMatrix) SetCurrent(index int) {
	if index < -1 || index >= m.total || index == m.current {
		return
	}
	if m.current >= 0 {
		m.cells[m.current].FillColor = matrixEmptyColor
		m.cells[m.current].Refresh()
	}
	if index >= 0 {
		m.cells[index].FillColor = matrixCurrentColor
		m.cells[index].Refresh()
	}
	m.current = index
}

func (m *chunkMatrix) Tapped(event *fyne.PointEvent) {
	index, ok := matrixIndexAt(event.Position, m.total)
	if ok && m.onSelect != nil {
		m.onSelect(index)
	}
}

func (m *chunkMatrix) MinSize() fyne.Size {
	rows := (m.total + matrixColumns - 1) / matrixColumns
	return fyne.NewSize(
		float32(matrixPadding*2+matrixLabelWidth+matrixColumns*matrixCellPitch),
		float32(matrixPadding*2+rows*matrixCellPitch),
	)
}

func matrixIndexAt(position fyne.Position, total int) (int, bool) {
	x := int(position.X) - matrixPadding - matrixLabelWidth
	y := int(position.Y) - matrixPadding
	if x < 0 || y < 0 {
		return 0, false
	}
	column := x / matrixCellPitch
	row := y / matrixCellPitch
	if column >= matrixColumns {
		return 0, false
	}
	if x%matrixCellPitch >= matrixCellSize || y%matrixCellPitch >= matrixCellSize {
		return 0, false
	}
	index := row*matrixColumns + column
	if index >= total {
		return 0, false
	}
	return index, true
}
