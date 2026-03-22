package colorpicker

import (
	"image/color"
	"testing"
)

func TestColorDetection(t *testing.T) {
	detector := NewColorDetector()

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "Hex colors",
			text:     "color: #FF5733; background: #00FF00;",
			expected: 2,
		},
		{
			name:     "RGB colors",
			text:     "rgb(255, 0, 0) and rgb(0, 255, 0)",
			expected: 2,
		},
		{
			name:     "RGBA colors",
			text:     "rgba(255, 0, 0, 0.5)",
			expected: 1,
		},
		{
			name:     "HSL colors",
			text:     "hsl(120, 100%, 50%)",
			expected: 1,
		},
		{
			name:     "HSLA colors",
			text:     "hsla(120, 100%, 50%, 0.5)",
			expected: 1,
		},
		{
			name:     "Named colors",
			text:     "color: red; background: blue;",
			expected: 2,
		},
		{
			name:     "Mixed formats",
			text:     "#FF5733, rgb(255,0,0), hsl(120,100%,50%), red",
			expected: 4,
		},
		{
			name:     "No colors",
			text:     "This is just plain text without any colors",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			colors := detector.DetectColors(tt.text)
			if len(colors) != tt.expected {
				t.Errorf("Expected %d colors, got %d", tt.expected, len(colors))
			}
		})
	}
}

func TestHexColorParsing(t *testing.T) {
	tests := []struct {
		name     string
		hex      string
		expected color.NRGBA
	}{
		{
			name:     "3-digit hex",
			hex:      "#F00",
			expected: color.NRGBA{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:     "6-digit hex",
			hex:      "#FF5733",
			expected: color.NRGBA{R: 255, G: 87, B: 51, A: 255},
		},
		{
			name:     "8-digit hex",
			hex:      "#FF573380",
			expected: color.NRGBA{R: 255, G: 87, B: 51, A: 128},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewColorDetector()
			colors := detector.DetectColors(tt.hex)
			if len(colors) != 1 {
				t.Fatalf("Expected 1 color, got %d", len(colors))
			}
			if colors[0].Color != tt.expected {
				t.Errorf("Expected color %v, got %v", tt.expected, colors[0].Color)
			}
		})
	}
}

func TestRGBToHSLConversion(t *testing.T) {
	tests := []struct {
		name  string
		input color.NRGBA
	}{
		{
			name:  "Red",
			input: color.NRGBA{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name:  "Green",
			input: color.NRGBA{R: 0, G: 255, B: 0, A: 255},
		},
		{
			name:  "Blue",
			input: color.NRGBA{R: 0, G: 0, B: 255, A: 255},
		},
		{
			name:  "White",
			input: color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		},
		{
			name:  "Black",
			input: color.NRGBA{R: 0, G: 0, B: 0, A: 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := NewColorPickerState(tt.input)
			if state.Red != tt.input.R || state.Green != tt.input.G || state.Blue != tt.input.B {
				t.Errorf("RGB values not preserved: expected %v, got %v", tt.input, color.NRGBA{R: state.Red, G: state.Green, B: state.Blue, A: state.Alpha})
			}
		})
	}
}

func TestColorFormatting(t *testing.T) {
	tests := []struct {
		name     string
		color    color.NRGBA
		format   ColorFormat
		original string
		expected string
	}{
		{
			name:     "Hex format",
			color:    color.NRGBA{R: 255, G: 87, B: 51, A: 255},
			format:   ColorFormatHex6,
			original: "#FF5733",
			expected: "#FF5733",
		},
		{
			name:     "RGB format",
			color:    color.NRGBA{R: 255, G: 0, B: 0, A: 255},
			format:   ColorFormatRGB,
			original: "rgb(255, 0, 0)",
			expected: "rgb(255, 0, 0)",
		},
		{
			name:     "RGBA format",
			color:    color.NRGBA{R: 255, G: 0, B: 0, A: 128},
			format:   ColorFormatRGBA,
			original: "rgba(255, 0, 0, 0.50)",
			expected: "rgba(255, 0, 0, 0.50)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := NewColorPickerState(tt.color)
			result := state.FormatColorToString(tt.format, tt.original)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestColorPickerState(t *testing.T) {
	initialColor := color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	state := NewColorPickerState(initialColor)

	if state.SelectedColor != initialColor {
		t.Errorf("Initial color not set correctly")
	}

	newColor := color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	state.SetRGB(255, 0, 0)

	if state.SelectedColor != newColor {
		t.Errorf("Color not updated correctly: expected %v, got %v", newColor, state.SelectedColor)
	}

	state.SetAlpha(128)
	if state.Alpha != 128 {
		t.Errorf("Alpha not updated correctly: expected 128, got %d", state.Alpha)
	}
}

func TestEditorColorPicker(t *testing.T) {
	ecp := NewEditorColorPicker()
	text := "color: #FF5733; background: rgb(0, 255, 0);"

	ecp.SetEditorText(text)
	if ecp.GetEditorText() != text {
		t.Errorf("Editor text not set correctly")
	}

	previews := ecp.GetPreviews()
	if len(previews) != 2 {
		t.Errorf("Expected 2 color previews, got %d", len(previews))
	}

	changeCalled := false
	var newText string
	ecp.SetOnColorChange(func(ot, nt string, start, end int) {
		changeCalled = true
		newText = nt
		if ot != "#FF5733" {
			t.Errorf("Expected old text #FF5733, got %s", ot)
		}
	})

	ecp.HandleClick(0)
	if !ecp.IsPickerOpen() {
		t.Error("Color picker should be open after click")
	}

	picker := ecp.previewManager.GetColorPicker()
	if picker != nil {
		picker.State.SetRGB(0, 255, 0)
	}

	ecp.HandleConfirm()
	if !changeCalled {
		t.Error("Color change callback should have been called")
	}

	if ecp.IsPickerOpen() {
		t.Error("Color picker should be closed after confirm")
	}

	if newText != "#00FF00" {
		t.Errorf("Expected new text #00FF00, got %s", newText)
	}
}
