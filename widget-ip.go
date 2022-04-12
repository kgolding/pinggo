package main

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type IPWidget struct {
	widget.Label
	IP         string
	active     bool
	lastChange time.Time
}

func newIPWidget(ip string) *IPWidget {
	b := &IPWidget{
		IP:         ip,
		active:     true,
		lastChange: time.Now(),
	}
	b.ExtendBaseWidget(b)
	b.SetText(ip)
	b.Alignment = fyne.TextAlignCenter
	return b
}

func (b *IPWidget) SetActive(v bool) {
	b.active = v
	b.Refresh()
}

func (b *IPWidget) CreateRenderer() fyne.WidgetRenderer {
	return &IPWidgetRenderer{
		WidgetRenderer: b.Label.CreateRenderer(),
		bg:             canvas.NewRectangle(theme.ShadowColor()),
		w:              b,
	}
}

type IPWidgetRenderer struct {
	fyne.WidgetRenderer
	bg *canvas.Rectangle
	w  *IPWidget
}

func (r *IPWidgetRenderer) Layout(s fyne.Size) {
	halfPad := theme.Padding() / 2
	r.bg.Move(fyne.NewPos(halfPad, theme.Padding()))
	r.bg.Resize(s.Subtract(fyne.NewSize(theme.Padding(), theme.Padding())))

	r.WidgetRenderer.Layout(s)
}

func (r *IPWidgetRenderer) Objects() []fyne.CanvasObject {
	return append([]fyne.CanvasObject{r.bg}, r.WidgetRenderer.Objects()...)
}

func (r *IPWidgetRenderer) Refresh() {
	r.bg.FillColor = theme.ShadowColor()
	if r.w.active {
		r.bg.FillColor = theme.FocusColor()
	}
	r.WidgetRenderer.Refresh()
}
