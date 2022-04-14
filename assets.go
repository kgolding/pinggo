package main

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed Icon.png
var iconPng []byte

var resIconPng = fyne.NewStaticResource("Icon", iconPng)
