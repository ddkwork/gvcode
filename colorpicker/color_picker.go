package colorpicker

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// hslToRGB 将 HSL 转换为 RGB
func hslToRGB(h, s, l float64) (r, g, b float64) {
	if s == 0 {
		r = l
		g = l
		b = l
		return
	}

	var hue2rgb func(p, q, t float64) float64
	hue2rgb = func(p, q, t float64) float64 {
		if t < 0 {
			t += 1
		}
		if t > 1 {
			t -= 1
		}
		if t < 1.0/6.0 {
			return p + (q-p)*6*t
		}
		if t < 1.0/2.0 {
			return q
		}
		if t < 2.0/3.0 {
			return p + (q-p)*(2.0/3.0-t)*6
		}
		return p
	}

	var q float64
	if l < 0.5 {
		q = l * (1.0 + s)
	} else {
		q = l + s - l*s
	}
	p := 2.0*l - q

	r = hue2rgb(p, q, h+1.0/3.0)
	g = hue2rgb(p, q, h)
	b = hue2rgb(p, q, h-1.0/3.0)
	return
}

// ColorPickerState 颜色选择器状态
type ColorPickerState struct {
	SelectedColor color.NRGBA
	Red           uint8
	Green         uint8
	Blue          uint8
	Alpha         uint8
	Hue           float64
	Saturation    float64
	Lightness     float64
	HexInput      string
	IsOpen        bool
}

// NewColorPickerState 创建新的颜色选择器状态
func NewColorPickerState(initialColor color.NRGBA) *ColorPickerState {
	state := &ColorPickerState{
		SelectedColor: initialColor,
		Red:           initialColor.R,
		Green:         initialColor.G,
		Blue:          initialColor.B,
		Alpha:         initialColor.A,
		HexInput:      rgbToHex(initialColor.R, initialColor.G, initialColor.B),
	}
	state.updateHSL()
	return state
}

// updateHSL 从 RGB 更新 HSL 值
func (s *ColorPickerState) updateHSL() {
	r := float64(s.Red) / 255.0
	g := float64(s.Green) / 255.0
	b := float64(s.Blue) / 255.0

	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	delta := max - min

	s.Lightness = (max + min) / 2.0

	if delta == 0 {
		s.Hue = 0
		s.Saturation = 0
	} else {
		if s.Lightness < 0.5 {
			s.Saturation = delta / (max + min)
		} else {
			s.Saturation = delta / (2.0 - max - min)
		}

		var hue float64
		switch max {
		case r:
			hue = (g - b) / delta
			if g < b {
				hue += 6
			}
		case g:
			hue = (b-r)/delta + 2
		case b:
			hue = (r-g)/delta + 4
		}
		s.Hue = hue / 6.0
	}
}

// updateRGB 从 HSL 更新 RGB 值
func (s *ColorPickerState) updateRGB() {
	var r, g, b float64

	if s.Saturation == 0 {
		r = s.Lightness
		g = s.Lightness
		b = s.Lightness
	} else {
		var hue2rgb func(p, q, t float64) float64
		hue2rgb = func(p, q, t float64) float64 {
			if t < 0 {
				t += 1
			}
			if t > 1 {
				t -= 1
			}
			if t < 1.0/6.0 {
				return p + (q-p)*6*t
			}
			if t < 1.0/2.0 {
				return q
			}
			if t < 2.0/3.0 {
				return p + (q-p)*(2.0/3.0-t)*6
			}
			return p
		}

		var q float64
		if s.Lightness < 0.5 {
			q = s.Lightness * (1.0 + s.Saturation)
		} else {
			q = s.Lightness + s.Saturation - s.Lightness*s.Saturation
		}
		p := 2.0*s.Lightness - q

		r = hue2rgb(p, q, s.Hue+1.0/3.0)
		g = hue2rgb(p, q, s.Hue)
		b = hue2rgb(p, q, s.Hue-1.0/3.0)
	}

	s.Red = uint8(r * 255.0)
	s.Green = uint8(g * 255.0)
	s.Blue = uint8(b * 255.0)
	s.SelectedColor = color.NRGBA{
		R: s.Red,
		G: s.Green,
		B: s.Blue,
		A: s.Alpha,
	}
	s.HexInput = rgbToHex(s.Red, s.Green, s.Blue)
}

// SetRGB 设置 RGB 值
func (s *ColorPickerState) SetRGB(r, g, b uint8) {
	s.Red = r
	s.Green = g
	s.Blue = b
	s.SelectedColor = color.NRGBA{
		R: s.Red,
		G: s.Green,
		B: s.Blue,
		A: s.Alpha,
	}
	s.updateHSL()
}

// SetHSL 设置 HSL 值
func (s *ColorPickerState) SetHSL(h, sat, l float64) {
	s.Hue = h
	s.Saturation = sat
	s.Lightness = l
	s.updateRGB()
}

// SetAlpha 设置 Alpha 值
func (s *ColorPickerState) SetAlpha(a uint8) {
	s.Alpha = a
	s.SelectedColor = color.NRGBA{
		R: s.Red,
		G: s.Green,
		B: s.Blue,
		A: s.Alpha,
	}
}

// rgbToHex 将 RGB 转换为十六进制字符串
func rgbToHex(r, g, b uint8) string {
	return string([]byte{
		'#',
		hexChar(r >> 4),
		hexChar(r & 0x0F),
		hexChar(g >> 4),
		hexChar(g & 0x0F),
		hexChar(b >> 4),
		hexChar(b & 0x0F),
	})
}

// hexChar 将 0-15 转换为十六进制字符
func hexChar(v uint8) byte {
	if v < 10 {
		return '0' + v
	}
	return 'A' + v - 10
}

// hexToRGB 将十六进制字符串转换为 RGB
func hexToRGB(hex string) (r, g, b uint8, err error) {
	if len(hex) == 0 || hex[0] != '#' {
		return 0, 0, 0, fmt.Errorf("hex string must start with #")
	}
	hex = hex[1:]

	var parseHex func(s string) (uint8, error)
	parseHex = func(s string) (uint8, error) {
		var v uint8
		for _, c := range s {
			v <<= 4
			switch {
			case '0' <= c && c <= '9':
				v |= uint8(c - '0')
			case 'a' <= c && c <= 'f':
				v |= uint8(c - 'a' + 10)
			case 'A' <= c && c <= 'F':
				v |= uint8(c - 'A' + 10)
			default:
				return 0, fmt.Errorf("invalid hex character: %c", c)
			}
		}
		return v, nil
	}

	switch len(hex) {
	case 3:
		r, err = parseHex(string([]byte{hex[0], hex[0]}))
		if err != nil {
			return 0, 0, 0, err
		}
		g, err = parseHex(string([]byte{hex[1], hex[1]}))
		if err != nil {
			return 0, 0, 0, err
		}
		b, err = parseHex(string([]byte{hex[2], hex[2]}))
		if err != nil {
			return 0, 0, 0, err
		}
	case 6:
		r, err = parseHex(hex[0:2])
		if err != nil {
			return 0, 0, 0, err
		}
		g, err = parseHex(hex[2:4])
		if err != nil {
			return 0, 0, 0, err
		}
		b, err = parseHex(hex[4:6])
		if err != nil {
			return 0, 0, 0, err
		}
	default:
		return 0, 0, 0, fmt.Errorf("hex string must be 3 or 6 characters long")
	}

	return r, g, b, nil
}

// ColorPicker 颜色选择器组件
type ColorPicker struct {
	State *ColorPickerState

	RedSlider   *widget.Float
	GreenSlider *widget.Float
	BlueSlider  *widget.Float
	AlphaSlider *widget.Float
	HueSlider   *widget.Float
	SatSlider   *widget.Float
	LightSlider *widget.Float
	HexEditor   *widget.Editor

	// Color selection area
	colorArea widget.Clickable

	// History colors
	HistoryColors []color.NRGBA

	// 256 color palette
	ColorPalette []color.NRGBA

	// Color palette clickables
	PaletteClickables []widget.Clickable
	HistoryClickables []widget.Clickable

	// Color history for undo functionality
	ColorHistory []color.NRGBA
	HistoryIndex int

	// Action buttons
	ConfirmButton widget.Clickable
	CancelButton  widget.Clickable
	UndoButton    widget.Clickable

	// Callback for confirm action
	OnConfirm func()
}

// NewColorPicker 创建新的颜色选择器
func NewColorPicker(initialColor color.NRGBA) *ColorPicker {
	state := NewColorPickerState(initialColor)

	// Create 256 color palette
	palette := create256ColorPalette()

	// Initialize clickables
	paletteClickables := make([]widget.Clickable, len(palette))
	historyClickables := make([]widget.Clickable, 9) // 9 history colors

	// Initialize color history with initial color
	colorHistory := []color.NRGBA{initialColor}

	return &ColorPicker{
		State:       state,
		RedSlider:   &widget.Float{Value: float32(initialColor.R)},
		GreenSlider: &widget.Float{Value: float32(initialColor.G)},
		BlueSlider:  &widget.Float{Value: float32(initialColor.B)},
		AlphaSlider: &widget.Float{Value: float32(initialColor.A)},
		HueSlider:   &widget.Float{Value: float32(state.Hue)},
		SatSlider:   &widget.Float{Value: float32(state.Saturation)},
		LightSlider: &widget.Float{Value: float32(state.Lightness)},
		HexEditor:   &widget.Editor{SingleLine: true, Submit: true},
		HistoryColors: []color.NRGBA{
			{R: 255, G: 0, B: 0, A: 255},     // Red
			{R: 0, G: 255, B: 0, A: 255},     // Green
			{R: 0, G: 0, B: 255, A: 255},     // Blue
			{R: 255, G: 255, B: 0, A: 255},   // Yellow
			{R: 255, G: 0, B: 255, A: 255},   // Magenta
			{R: 0, G: 255, B: 255, A: 255},   // Cyan
			{R: 128, G: 128, B: 128, A: 255}, // Gray
			{R: 0, G: 0, B: 0, A: 255},       // Black
			{R: 255, G: 255, B: 255, A: 255}, // White
		},
		ColorPalette:      palette,
		PaletteClickables: paletteClickables,
		HistoryClickables: historyClickables,
		ColorHistory:      colorHistory,
		HistoryIndex:      0,
	}
}

// create256ColorPalette 创建256色选择面板
func create256ColorPalette() []color.NRGBA {
	var palette []color.NRGBA

	// Basic colors
	basicColors := []color.NRGBA{
		{R: 0, G: 0, B: 0, A: 255},       // Black
		{R: 128, G: 0, B: 0, A: 255},     // Dark Red
		{R: 0, G: 128, B: 0, A: 255},     // Dark Green
		{R: 128, G: 128, B: 0, A: 255},   // Dark Yellow
		{R: 0, G: 0, B: 128, A: 255},     // Dark Blue
		{R: 128, G: 0, B: 128, A: 255},   // Dark Magenta
		{R: 0, G: 128, B: 128, A: 255},   // Dark Cyan
		{R: 192, G: 192, B: 192, A: 255}, // Light Gray
		{R: 128, G: 128, B: 128, A: 255}, // Gray
		{R: 255, G: 0, B: 0, A: 255},     // Red
		{R: 0, G: 255, B: 0, A: 255},     // Green
		{R: 255, G: 255, B: 0, A: 255},   // Yellow
		{R: 0, G: 0, B: 255, A: 255},     // Blue
		{R: 255, G: 0, B: 255, A: 255},   // Magenta
		{R: 0, G: 255, B: 255, A: 255},   // Cyan
		{R: 255, G: 255, B: 255, A: 255}, // White
	}
	palette = append(palette, basicColors...)

	// Grayscale
	for i := range 16 {
		gray := uint8(i * 16)
		palette = append(palette, color.NRGBA{R: gray, G: gray, B: gray, A: 255})
	}

	// Color gradients
	for r := range 4 {
		for g := range 4 {
			for b := range 4 {
				palette = append(palette, color.NRGBA{
					R: uint8(r * 85),
					G: uint8(g * 85),
					B: uint8(b * 85),
					A: 255,
				})
			}
		}
	}

	// More color variations
	for h := 0; h < 360; h += 30 {
		for s := 1; s <= 3; s++ {
			for l := 1; l <= 3; l++ {
				r, g, b := hslToRGB(float64(h)/360.0, float64(s)/3.0, float64(l)/4.0)
				palette = append(palette, color.NRGBA{
					R: uint8(r * 255),
					G: uint8(g * 255),
					B: uint8(b * 255),
					A: 255,
				})
			}
		}
	}

	// Limit to 256 colors
	if len(palette) > 256 {
		palette = palette[:256]
	}

	return palette
}

// Layout 渲染颜色选择器
func (cp *ColorPicker) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	// Set maximum width and height for the picker to match VS Code style
	maxWidth := gtx.Dp(unit.Dp(280))
	maxHeight := gtx.Dp(unit.Dp(700))

	gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, maxWidth)
	gtx.Constraints.Max.Y = min(gtx.Constraints.Max.Y, maxHeight)

	// Register pointer events for the entire picker area
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()

	// Draw background with rounded corners (VS Code style)
	backgroundColor := color.NRGBA{R: 48, G: 48, B: 48, A: 255}
	bounds := image.Rectangle{Max: gtx.Constraints.Max}
	roundedRect := clip.RRect{Rect: bounds, SE: 8, SW: 8, NE: 8, NW: 8}
	backgroundStack := roundedRect.Push(gtx.Ops)
	paint.Fill(gtx.Ops, backgroundColor)
	backgroundStack.Pop()

	// Draw border
	borderColor := color.NRGBA{R: 80, G: 80, B: 80, A: 255}
	borderRect := image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: gtx.Constraints.Max,
	}
	paint.FillShape(gtx.Ops, borderColor, clip.Stroke{Path: clip.RRect{Rect: borderRect, SE: 8, SW: 8, NE: 8, NW: 8}.Path(gtx.Ops), Width: 1}.Op())

	return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Start, Spacing: layout.SpaceEnd}.Layout(gtx,
			// Color preview area
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return cp.layoutColorPreview(gtx)
			}),

			// Color selection area and hue slider
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween, Alignment: layout.Start}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return cp.layoutColorSelection(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return cp.layoutHueSlider(gtx, th)
					}),
				)
			}),

			// History colors
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return cp.layoutHistoryColors(gtx)
			}),

			// 256 color palette (smaller version)
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return cp.layoutColorPalette(gtx)
			}),

			// HEX input
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return cp.layoutHexInput(gtx, th)
			}),

			// Action buttons
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween, Alignment: layout.End}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(th, &cp.UndoButton, "Undo")
						btn.Background = color.NRGBA{R: 60, G: 60, B: 60, A: 255}
						btn.Color = color.NRGBA{R: 220, G: 220, B: 220, A: 255}
						return btn.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(th, &cp.ConfirmButton, "Apply")
						btn.Background = color.NRGBA{R: 0, G: 120, B: 215, A: 255}
						btn.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
						return btn.Layout(gtx)
					}),
				)
			}),
		)
	})
}

// layoutColorPreview 渲染颜色预览
func (cp *ColorPicker) layoutColorPreview(gtx layout.Context) layout.Dimensions {
	previewHeight := gtx.Dp(unit.Dp(40))
	paint.FillShape(gtx.Ops, cp.State.SelectedColor, clip.Rect(image.Rectangle{
		Max: image.Point{X: gtx.Constraints.Max.X, Y: previewHeight},
	}).Op())
	return layout.Dimensions{Size: image.Point{X: gtx.Constraints.Max.X, Y: previewHeight}}
}

// layoutColorSelection 渲染颜色选择区域
func (cp *ColorPicker) layoutColorSelection(gtx layout.Context) layout.Dimensions {
	selectionSize := gtx.Dp(unit.Dp(140))
	bounds := image.Rectangle{Max: image.Point{X: selectionSize, Y: selectionSize}}

	// Draw color gradient with rounded corners (VS Code style)
	roundedRect := clip.RRect{Rect: bounds, SE: 4, SW: 4, NE: 4, NW: 4}
	roundedStack := roundedRect.Push(gtx.Ops)

	// Draw a simplified gradient - use larger blocks instead of individual pixels
	resolution := 14 // Use 14x14 grid instead of 140x140
	blockSize := selectionSize / resolution

	// Draw color gradient with larger blocks
	for y := range resolution {
		lightness := float64(y) / float64(resolution)
		for x := range resolution {
			saturation := float64(x) / float64(resolution)
			r, g, b := hslToRGB(float64(cp.HueSlider.Value), saturation, lightness)
			col := color.NRGBA{
				R: uint8(r * 255),
				G: uint8(g * 255),
				B: uint8(b * 255),
				A: 255,
			}
			blockRect := image.Rectangle{
				Min: image.Point{X: x * blockSize, Y: y * blockSize},
				Max: image.Point{X: (x + 1) * blockSize, Y: (y + 1) * blockSize},
			}
			paint.FillShape(gtx.Ops, col, clip.Rect(blockRect).Op())
		}
	}

	roundedStack.Pop()

	// Handle click events
	if cp.colorArea.Clicked(gtx) {
		// For simplicity, we'll just use the center of the color area
		// In a real implementation, we would track the actual pointer position
		saturation := 0.5
		lightness := 0.5
		r, g, b := hslToRGB(float64(cp.HueSlider.Value), saturation, lightness)
		cp.State.Red = uint8(r * 255)
		cp.State.Green = uint8(g * 255)
		cp.State.Blue = uint8(b * 255)
		cp.State.updateHSL()
		cp.RedSlider.Value = float32(cp.State.Red)
		cp.GreenSlider.Value = float32(cp.State.Green)
		cp.BlueSlider.Value = float32(cp.State.Blue)
		cp.SatSlider.Value = float32(cp.State.Saturation)
		cp.LightSlider.Value = float32(cp.State.Lightness)
		cp.State.HexInput = rgbToHex(cp.State.Red, cp.State.Green, cp.State.Blue)
	}

	// Register clickable area
	pointer.CursorPointer.Add(gtx.Ops)
	cp.colorArea.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Point{X: selectionSize, Y: selectionSize}}
	})
	roundedRect.Push(gtx.Ops).Pop()

	// Draw selection marker (VS Code style)
	if cp.State.Saturation > 0 && cp.State.Lightness > 0 {
		markerX := int(float64(selectionSize) * cp.State.Saturation)
		markerY := int(float64(selectionSize) * cp.State.Lightness)
		markerSize := gtx.Dp(unit.Dp(10))

		// Draw crosshair marker
		crossSize := gtx.Dp(unit.Dp(16))

		// Horizontal line
		horizontalLine := image.Rectangle{
			Min: image.Point{X: markerX - crossSize/2, Y: markerY},
			Max: image.Point{X: markerX + crossSize/2, Y: markerY + 1},
		}
		paint.FillShape(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, clip.Rect(horizontalLine).Op())

		// Vertical line
		verticalLine := image.Rectangle{
			Min: image.Point{X: markerX, Y: markerY - crossSize/2},
			Max: image.Point{X: markerX + 1, Y: markerY + crossSize/2},
		}
		paint.FillShape(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, clip.Rect(verticalLine).Op())

		// Black border for crosshair
		horizontalLineBorder := image.Rectangle{
			Min: image.Point{X: markerX - crossSize/2 - 1, Y: markerY - 1},
			Max: image.Point{X: markerX + crossSize/2 + 1, Y: markerY + 2},
		}
		paint.FillShape(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, clip.Rect(horizontalLineBorder).Op())

		verticalLineBorder := image.Rectangle{
			Min: image.Point{X: markerX - 1, Y: markerY - crossSize/2 - 1},
			Max: image.Point{X: markerX + 2, Y: markerY + crossSize/2 + 1},
		}
		paint.FillShape(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, clip.Rect(verticalLineBorder).Op())

		// Center circle - use simple rectangle instead
		markerRect := image.Rectangle{
			Min: image.Point{X: markerX - markerSize/2, Y: markerY - markerSize/2},
			Max: image.Point{X: markerX + markerSize/2, Y: markerY + markerSize/2},
		}
		markerStack := clip.Rect(markerRect).Push(gtx.Ops)
		paint.Fill(gtx.Ops, color.NRGBA{255, 255, 255, 255})
		markerStack.Pop()

		// Draw black border
		borderRect := image.Rectangle{
			Min: image.Point{X: markerX - markerSize/2 - 1, Y: markerY - markerSize/2 - 1},
			Max: image.Point{X: markerX + markerSize/2 + 1, Y: markerY + markerSize/2 + 1},
		}
		borderStack := clip.Rect(borderRect).Push(gtx.Ops)
		paint.Fill(gtx.Ops, color.NRGBA{0, 0, 0, 255})
		borderStack.Pop()
	}

	return layout.Dimensions{Size: image.Point{X: selectionSize, Y: selectionSize}}
}

// layoutHueSlider 渲染色相滑块
func (cp *ColorPicker) layoutHueSlider(gtx layout.Context, th *material.Theme) layout.Dimensions {
	sliderWidth := gtx.Dp(unit.Dp(20))
	sliderHeight := gtx.Dp(unit.Dp(140))
	bounds := image.Rectangle{Max: image.Point{X: sliderWidth, Y: sliderHeight}}

	// Draw hue gradient with rounded corners (VS Code style)
	roundedRect := clip.RRect{Rect: bounds, SE: 4, SW: 4, NE: 4, NW: 4}
	roundedStack := roundedRect.Push(gtx.Ops)

	// Draw hue gradient with simplified blocks
	resolution := 14
	blockHeight := sliderHeight / resolution

	for y := range resolution {
		hue := float64(y) / float64(resolution)
		r, g, b := hslToRGB(hue, 1.0, 0.5)
		col := color.NRGBA{
			R: uint8(r * 255),
			G: uint8(g * 255),
			B: uint8(b * 255),
			A: 255,
		}
		blockRect := image.Rectangle{
			Min: image.Point{X: 0, Y: y * blockHeight},
			Max: image.Point{X: sliderWidth, Y: (y + 1) * blockHeight},
		}
		paint.FillShape(gtx.Ops, col, clip.Rect(blockRect).Op())
	}

	roundedStack.Pop()

	// Update hue slider
	if cp.HueSlider.Update(gtx) {
		cp.State.Hue = float64(cp.HueSlider.Value)
		cp.State.updateRGB()
		cp.RedSlider.Value = float32(cp.State.Red)
		cp.GreenSlider.Value = float32(cp.State.Green)
		cp.BlueSlider.Value = float32(cp.State.Blue)
		cp.SatSlider.Value = float32(cp.State.Saturation)
		cp.LightSlider.Value = float32(cp.State.Lightness)
		cp.State.HexInput = rgbToHex(cp.State.Red, cp.State.Green, cp.State.Blue)
	}

	// Draw hue selection marker (VS Code style)
	markerY := int(float64(sliderHeight) * cp.State.Hue)
	markerSize := gtx.Dp(unit.Dp(12))

	// White inner rectangle
	markerRect := image.Rectangle{
		Min: image.Point{X: 0, Y: markerY - markerSize/2},
		Max: image.Point{X: markerSize, Y: markerY + markerSize/2},
	}
	markerStack := clip.Rect(markerRect).Push(gtx.Ops)
	paint.Fill(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	markerStack.Pop()

	// Black border
	borderRect := image.Rectangle{
		Min: image.Point{X: -1, Y: markerY - markerSize/2 - 1},
		Max: image.Point{X: markerSize + 1, Y: markerY + markerSize/2 + 1},
	}
	borderStack := clip.Rect(borderRect).Push(gtx.Ops)
	paint.Fill(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	borderStack.Pop()

	// Layout slider with proper spacing (VS Code style)
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X = sliderWidth
			gtx.Constraints.Max.Y = sliderHeight
			return material.Slider(th, cp.HueSlider).Layout(gtx)
		})
	}))
}

// layoutHistoryColors 渲染历史颜色
func (cp *ColorPicker) layoutHistoryColors(gtx layout.Context) layout.Dimensions {
	colorSize := gtx.Dp(unit.Dp(20))
	spacing := gtx.Dp(unit.Dp(4))
	rowSize := 8 // 8 colors per row

	var children []layout.FlexChild
	clickablesLen := len(cp.HistoryClickables)
	for i, historyColor := range cp.HistoryColors {
		if i >= clickablesLen {
			break
		}
		if i > 0 && i%rowSize == 0 {
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(spacing)}.Layout))
		}

		i := i // Capture variable for closure
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			// Handle click events
			if cp.HistoryClickables[i].Clicked(gtx) {
				cp.State.SetRGB(historyColor.R, historyColor.G, historyColor.B)
				cp.updateRGBSliders()
				cp.updateHSLSliders()
				cp.updateHexEditor()
			}

			return layout.Inset{Right: unit.Dp(spacing), Bottom: unit.Dp(spacing)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				bounds := image.Rectangle{Max: image.Point{X: colorSize, Y: colorSize}}
				// Register clickable area
				pointer.CursorPointer.Add(gtx.Ops)
				cp.HistoryClickables[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Dimensions{Size: image.Point{X: colorSize, Y: colorSize}}
				})

				// Draw color with rounded corners
				roundedRect := clip.RRect{Rect: bounds, SE: 2, SW: 2, NE: 2, NW: 2}
				roundedStack := roundedRect.Push(gtx.Ops)
				paint.Fill(gtx.Ops, historyColor)
				roundedStack.Pop()

				// Draw border
				borderColor := color.NRGBA{R: 0, G: 0, B: 0, A: 128}
				borderRect := image.Rectangle{
					Min: image.Point{X: 0, Y: 0},
					Max: image.Point{X: colorSize, Y: colorSize},
				}
				paint.FillShape(gtx.Ops, borderColor, clip.Rect(borderRect).Op())

				// Check if this color is the currently selected color
				if historyColor.R == cp.State.Red && historyColor.G == cp.State.Green && historyColor.B == cp.State.Blue {
					// Draw selection border
					selectedBorderColor := color.NRGBA{R: 0, G: 120, B: 215, A: 255}
					selectedBorderRect := image.Rectangle{
						Min: image.Point{X: -2, Y: -2},
						Max: image.Point{X: colorSize + 2, Y: colorSize + 2},
					}
					paint.FillShape(gtx.Ops, selectedBorderColor, clip.Rect(selectedBorderRect).Op())
				}

				return layout.Dimensions{Size: image.Point{X: colorSize, Y: colorSize}}
			})
		}))
	}

	return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceEnd}.Layout(gtx, children...)
}

// layoutColorPalette 渲染256色选择面板
func (cp *ColorPicker) layoutColorPalette(gtx layout.Context) layout.Dimensions {
	colorSize := gtx.Dp(unit.Dp(16))
	spacing := gtx.Dp(unit.Dp(2))
	rowSize := 16 // 16 colors per row

	var children []layout.FlexChild
	clickablesLen := len(cp.PaletteClickables)
	for i, paletteColor := range cp.ColorPalette {
		if i >= clickablesLen {
			break
		}
		if i > 0 && i%rowSize == 0 {
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(spacing)}.Layout))
		}

		i := i // Capture variable for closure
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			// Handle click events
			if cp.PaletteClickables[i].Clicked(gtx) {
				cp.State.SetRGB(paletteColor.R, paletteColor.G, paletteColor.B)
				cp.updateRGBSliders()
				cp.updateHSLSliders()
				cp.updateHexEditor()
			}

			return layout.Inset{Right: unit.Dp(spacing), Bottom: unit.Dp(spacing)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				bounds := image.Rectangle{Max: image.Point{X: colorSize, Y: colorSize}}
				// Register clickable area
				pointer.CursorPointer.Add(gtx.Ops)
				cp.PaletteClickables[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Dimensions{Size: image.Point{X: colorSize, Y: colorSize}}
				})

				// Draw color
				stack := clip.Rect(bounds).Push(gtx.Ops)
				paint.Fill(gtx.Ops, paletteColor)
				stack.Pop()

				// Check if this color is the currently selected color
				if paletteColor.R == cp.State.Red && paletteColor.G == cp.State.Green && paletteColor.B == cp.State.Blue {
					// Draw selection border
					selectedBorderColor := color.NRGBA{R: 0, G: 120, B: 215, A: 255}
					selectedBorderRect := image.Rectangle{
						Min: image.Point{X: -1, Y: -1},
						Max: image.Point{X: colorSize + 1, Y: colorSize + 1},
					}
					paint.FillShape(gtx.Ops, selectedBorderColor, clip.Rect(selectedBorderRect).Op())
				}

				return layout.Dimensions{Size: image.Point{X: colorSize, Y: colorSize}}
			})
		}))
	}

	return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceEnd}.Layout(gtx, children...)
}

// layoutRGBSliders 渲染 RGB 滑块
func (cp *ColorPicker) layoutRGBSliders(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, material.Label(th, th.TextSize, "R: ").Layout)
				}),
				layout.Flexed(1, material.Slider(th, cp.RedSlider).Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, material.Label(th, th.TextSize, fmt.Sprintf("%d", int(cp.RedSlider.Value))).Layout)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, material.Label(th, th.TextSize, "G: ").Layout)
				}),
				layout.Flexed(1, material.Slider(th, cp.GreenSlider).Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, material.Label(th, th.TextSize, fmt.Sprintf("%d", int(cp.GreenSlider.Value))).Layout)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, material.Label(th, th.TextSize, "B: ").Layout)
				}),
				layout.Flexed(1, material.Slider(th, cp.BlueSlider).Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, material.Label(th, th.TextSize, fmt.Sprintf("%d", int(cp.BlueSlider.Value))).Layout)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, material.Label(th, th.TextSize, "A: ").Layout)
				}),
				layout.Flexed(1, material.Slider(th, cp.AlphaSlider).Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, material.Label(th, th.TextSize, fmt.Sprintf("%d", int(cp.AlphaSlider.Value))).Layout)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, material.Label(th, th.TextSize, "H: ").Layout)
				}),
				layout.Flexed(1, material.Slider(th, cp.HueSlider).Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, material.Label(th, th.TextSize, fmt.Sprintf("%d", int(cp.HueSlider.Value*360))).Layout)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, material.Label(th, th.TextSize, "S: ").Layout)
				}),
				layout.Flexed(1, material.Slider(th, cp.SatSlider).Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, material.Label(th, th.TextSize, fmt.Sprintf("%d%%", int(cp.SatSlider.Value*100))).Layout)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, material.Label(th, th.TextSize, "L: ").Layout)
				}),
				layout.Flexed(1, material.Slider(th, cp.LightSlider).Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, material.Label(th, th.TextSize, fmt.Sprintf("%d%%", int(cp.LightSlider.Value*100))).Layout)
				}),
			)
		}),
	)
}

// layoutHexInput 渲染十六进制输入框
func (cp *ColorPicker) layoutHexInput(gtx layout.Context, th *material.Theme) layout.Dimensions {
	// Update hex editor text if it doesn't match the current color
	if cp.HexEditor.Text() != cp.State.HexInput {
		cp.updateHexEditor()
	}

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, material.Label(th, th.TextSize, "Hex:").Layout)
		}),
		layout.Flexed(1, material.Editor(th, cp.HexEditor, "#RRGGBB").Layout),
	)
}

// Update 更新颜色选择器状态
func (cp *ColorPicker) Update(gtx layout.Context) {
	// Update RGB sliders
	if cp.RedSlider.Update(gtx) {
		cp.State.Red = uint8(cp.RedSlider.Value)
		cp.State.updateHSL()
		cp.SatSlider.Value = float32(cp.State.Saturation)
		cp.LightSlider.Value = float32(cp.State.Lightness)
		cp.State.HexInput = rgbToHex(cp.State.Red, cp.State.Green, cp.State.Blue)
		cp.updateHexEditor()
	}

	if cp.GreenSlider.Update(gtx) {
		cp.State.Green = uint8(cp.GreenSlider.Value)
		cp.State.updateHSL()
		cp.SatSlider.Value = float32(cp.State.Saturation)
		cp.LightSlider.Value = float32(cp.State.Lightness)
		cp.State.HexInput = rgbToHex(cp.State.Red, cp.State.Green, cp.State.Blue)
		cp.updateHexEditor()
	}

	if cp.BlueSlider.Update(gtx) {
		cp.State.Blue = uint8(cp.BlueSlider.Value)
		cp.State.updateHSL()
		cp.SatSlider.Value = float32(cp.State.Saturation)
		cp.LightSlider.Value = float32(cp.State.Lightness)
		cp.State.HexInput = rgbToHex(cp.State.Red, cp.State.Green, cp.State.Blue)
		cp.updateHexEditor()
	}

	if cp.AlphaSlider.Update(gtx) {
		cp.State.Alpha = uint8(cp.AlphaSlider.Value)
	}

	// Update HSL sliders
	if cp.HueSlider.Update(gtx) {
		cp.State.Hue = float64(cp.HueSlider.Value)
		cp.State.updateRGB()
		cp.RedSlider.Value = float32(cp.State.Red)
		cp.GreenSlider.Value = float32(cp.State.Green)
		cp.BlueSlider.Value = float32(cp.State.Blue)
		cp.State.HexInput = rgbToHex(cp.State.Red, cp.State.Green, cp.State.Blue)
		cp.updateHexEditor()
	}

	if cp.SatSlider.Update(gtx) {
		cp.State.Saturation = float64(cp.SatSlider.Value)
		cp.State.updateRGB()
		cp.RedSlider.Value = float32(cp.State.Red)
		cp.GreenSlider.Value = float32(cp.State.Green)
		cp.BlueSlider.Value = float32(cp.State.Blue)
		cp.State.HexInput = rgbToHex(cp.State.Red, cp.State.Green, cp.State.Blue)
		cp.updateHexEditor()
	}

	if cp.LightSlider.Update(gtx) {
		cp.State.Lightness = float64(cp.LightSlider.Value)
		cp.State.updateRGB()
		cp.RedSlider.Value = float32(cp.State.Red)
		cp.GreenSlider.Value = float32(cp.State.Green)
		cp.BlueSlider.Value = float32(cp.State.Blue)
		cp.State.HexInput = rgbToHex(cp.State.Red, cp.State.Green, cp.State.Blue)
		cp.updateHexEditor()
	}

	// Update hex editor
	hexText := cp.HexEditor.Text()
	if len(hexText) > 0 {
		// Ensure hex text starts with #
		if hexText[0] != '#' {
			hexText = "#" + hexText
			cp.HexEditor.SetText(hexText)
		}

		// Only process valid hex codes
		r, g, b, err := hexToRGB(hexText)
		if err == nil {
			// Update color state if hex is valid
			cp.State.SetRGB(r, g, b)
			cp.State.updateHSL()
			cp.RedSlider.Value = float32(cp.State.Red)
			cp.GreenSlider.Value = float32(cp.State.Green)
			cp.BlueSlider.Value = float32(cp.State.Blue)
			cp.HueSlider.Value = float32(cp.State.Hue)
			cp.SatSlider.Value = float32(cp.State.Saturation)
			cp.LightSlider.Value = float32(cp.State.Lightness)
		}
	}

	// Handle button clicks
	if cp.ConfirmButton.Clicked(gtx) {
		cp.AddToHistory(cp.State.SelectedColor)
		if cp.OnConfirm != nil {
			cp.OnConfirm()
		}
	}

	if cp.CancelButton.Clicked(gtx) {
	}

	if cp.UndoButton.Clicked(gtx) {
	}
}

// updateRGBSliders 更新 RGB 滑块值
func (cp *ColorPicker) updateRGBSliders() {
	cp.RedSlider.Value = float32(cp.State.Red)
	cp.GreenSlider.Value = float32(cp.State.Green)
	cp.BlueSlider.Value = float32(cp.State.Blue)
	cp.AlphaSlider.Value = float32(cp.State.Alpha)
}

// updateHSLSliders 更新 HSL 滑块值
func (cp *ColorPicker) updateHSLSliders() {
	cp.HueSlider.Value = float32(cp.State.Hue)
	cp.SatSlider.Value = float32(cp.State.Saturation)
	cp.LightSlider.Value = float32(cp.State.Lightness)
}

// updateHexEditor 更新十六进制编辑器
func (cp *ColorPicker) updateHexEditor() {
	cp.HexEditor.SetText(cp.State.HexInput)
}

// AddToHistory 添加颜色到历史记录
func (cp *ColorPicker) AddToHistory(color color.NRGBA) {
	// Add new color to history
	cp.ColorHistory = append(cp.ColorHistory[:cp.HistoryIndex+1], color)
	cp.HistoryIndex = len(cp.ColorHistory) - 1

	// Limit history size to 20 entries
	if len(cp.ColorHistory) > 20 {
		cp.ColorHistory = cp.ColorHistory[len(cp.ColorHistory)-20:]
		cp.HistoryIndex = len(cp.ColorHistory) - 1
	}
}

// Undo 撤销到上一个颜色
func (cp *ColorPicker) Undo() {
	if cp.HistoryIndex > 0 {
		cp.HistoryIndex--
		color := cp.ColorHistory[cp.HistoryIndex]
		cp.State.SetRGB(color.R, color.G, color.B)
		cp.updateRGBSliders()
		cp.updateHSLSliders()
		cp.updateHexEditor()
	}
}

// Redo 重做颜色选择
func (cp *ColorPicker) Redo() {
	if cp.HistoryIndex < len(cp.ColorHistory)-1 {
		cp.HistoryIndex++
		color := cp.ColorHistory[cp.HistoryIndex]
		cp.State.SetRGB(color.R, color.G, color.B)
		cp.updateRGBSliders()
		cp.updateHSLSliders()
		cp.updateHexEditor()
	}
}

// FormatColorToString 将颜色格式化为字符串（保持原始格式）
func (s *ColorPickerState) FormatColorToString(originalFormat ColorFormat, original string) string {
	switch originalFormat {
	case ColorFormatHex3, ColorFormatHex4, ColorFormatHex6, ColorFormatHex8:
		return rgbToHex(s.Red, s.Green, s.Blue)
	case ColorFormatRGB:
		return fmt.Sprintf("rgb(%d, %d, %d)", s.Red, s.Green, s.Blue)
	case ColorFormatRGBA:
		return fmt.Sprintf("rgba(%d, %d, %d, %.2f)", s.Red, s.Green, s.Blue, float64(s.Alpha)/255.0)
	case ColorFormatHSL:
		return fmt.Sprintf("hsl(%d, %d%%, %d%%)", int(s.Hue*360), int(s.Saturation*100), int(s.Lightness*100))
	case ColorFormatHSLA:
		return fmt.Sprintf("hsla(%d, %d%%, %d%%, %.2f)", int(s.Hue*360), int(s.Saturation*100), int(s.Lightness*100), float64(s.Alpha)/255.0)
	case ColorFormatNamed:
		return original
	default:
		return rgbToHex(s.Red, s.Green, s.Blue)
	}
}
