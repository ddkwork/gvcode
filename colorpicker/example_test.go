package colorpicker

import (
	"image/color"

	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// ExampleColorPicker demonstrates how to use the color picker
func ExampleColorPicker() {
	// Create a new color picker with an initial color
	picker := NewColorPicker(color.NRGBA{R: 255, G: 0, B: 0, A: 255})

	// Create a window
	w := &app.Window{}
	w.Option(app.Title("Color Picker Example"))
	w.Option(app.Size(unit.Dp(400), unit.Dp(500)))

	// Create a theme
	th := material.NewTheme()

	// Run the window
	var ops op.Ops
	for {
		e := w.Event()
		switch e := e.(type) {
		case app.DestroyEvent:
			return
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			// Layout the color picker
			picker.Layout(gtx, th)
			// Update the color picker state
			picker.Update(gtx)
			e.Frame(gtx.Ops)
		}
	}
}
