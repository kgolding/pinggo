package main

import (
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Declare conformity with Layout interface
var _ fyne.Layout = (*gridWrapJustifiedLayout)(nil)

type gridWrapJustifiedLayout struct {
	CellSize fyne.Size
	colCount int
	rowCount int
}

// NewGridWrapJustifiedLayout returns a new GridWrapJustifiedLayout instance
func NewGridWrapJustifiedLayout(size fyne.Size) fyne.Layout {
	return &gridWrapJustifiedLayout{size, 1, 1}
}

// Layout is called to pack all child objects into a specified size.
// For a GridWrapJustifiedLayout this will attempt to lay all the child objects in a row
// and wrap to a new row if the size is not large enough. Cells are made wider to provide
// a horizontal full justification look
func (g *gridWrapJustifiedLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	g.colCount = 1
	g.rowCount = 1

	if size.Width > g.CellSize.Width {
		g.colCount = int(math.Floor(float64(size.Width+theme.Padding()) / float64(g.CellSize.Width+theme.Padding())))
	}

	// Local copy of CellSize
	jsize := g.CellSize
	jsize.Width = (size.Width - theme.Padding()*float32(g.colCount)) / float32(g.colCount)

	i, x, y := 0, float32(0), float32(0)
	for _, child := range objects {
		if !child.Visible() {
			continue
		}

		child.Move(fyne.NewPos(x, y))
		child.Resize(jsize)

		if (i+1)%g.colCount == 0 {
			x = 0
			y += jsize.Height + theme.Padding()
			if i > 0 {
				g.rowCount++
			}
		} else {
			x += jsize.Width + theme.Padding()
		}
		i++
	}
}

// MinSize finds the smallest size that satisfies all the child objects.
// For a GridWrapLayout this is simply the specified cellsize as a single column
// layout has no padding. The returned size does not take into account the number
// of columns as this layout re-flows dynamically.
func (g *gridWrapJustifiedLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(g.CellSize.Width,
		(g.CellSize.Height*float32(g.rowCount))+(float32(g.rowCount-1)*theme.Padding()))
}
