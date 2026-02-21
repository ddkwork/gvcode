package providers

import (
	"image"
	"image/color"
	"regexp"

	"gioui.org/f32"
	"gioui.org/gesture"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	gvcolor "github.com/oligo/gvcode/color"
	"github.com/oligo/gvcode/gutter"
)

const (
	// RunButtonProviderID is the unique identifier for the run button provider.
	RunButtonProviderID = "runbutton"

	// buttonSize is the size of the run button in dp units.
	buttonSize = 24
)

// RunButtonType represents the type of run button.
type RunButtonType int

const (
	// RunButtonNone indicates no run button should be shown.
	RunButtonNone RunButtonType = iota
	// RunButtonMain indicates a main function run button.
	RunButtonMain
	// RunButtonTest indicates a test function run button.
	RunButtonTest
)

// RunButtonProvider renders run buttons for main and test functions in the gutter.
type RunButtonProvider struct {
	// clicker handles click events on buttons.
	clicker gesture.Click

	// pending holds run events that haven't been consumed yet.
	pending []RunButtonEvent

	// paragraphs caches the visible paragraphs from the last Layout call.
	paragraphs []gutter.Paragraph

	// buttonTypes caches the button type for each line.
	buttonTypes map[int]RunButtonType

	// buttonTexts caches button labels for each line.
	buttonTexts map[int]string

	// lineHeight caches the line height from the last Layout call.
	lineHeight int

	// viewport caches the viewport from the last Layout call.
	viewport image.Rectangle
}

// NewRunButtonProvider creates a new run button provider with default settings.
func NewRunButtonProvider() *RunButtonProvider {
	return &RunButtonProvider{
		buttonTypes: make(map[int]RunButtonType),
		buttonTexts: make(map[int]string),
		paragraphs:  make([]gutter.Paragraph, 0),
	}
}

// ID returns the unique identifier for this provider.
func (p *RunButtonProvider) ID() string {
	return RunButtonProviderID
}

// Priority returns the rendering priority. Run buttons have priority 150
// meaning they are rendered between line numbers (100) and text (200+).
func (p *RunButtonProvider) Priority() int {
	return 150
}

// Width returns the fixed width needed for run buttons.
func (p *RunButtonProvider) Width(gtx layout.Context, shaper *text.Shaper, params text.Parameters, lineCount int) unit.Dp {
	return unit.Dp(buttonSize)
}

// SetLineContents sets the contents of all visible lines for analysis.
// This should be called before Layout.
func (p *RunButtonProvider) SetLineContents(lines []string, startLine int) {
	p.analyzeLines(lines, startLine)
}

// Layout renders run buttons for visible paragraphs.
func (p *RunButtonProvider) Layout(gtx layout.Context, ctx gutter.GutterContext) layout.Dimensions {
	// Cache context info for event handling
	p.paragraphs = ctx.Paragraphs
	p.lineHeight = ctx.LineHeight.Ceil()
	p.viewport = ctx.Viewport

	// Define colors for different button types
	mainColor := gvcolor.MakeColor(color.NRGBA{R: 0x4C, G: 0xAF, B: 0x50, A: 0xFF}) // Green
	testColor := gvcolor.MakeColor(color.NRGBA{R: 0x21, G: 0x96, B: 0xF3, A: 0xFF}) // Blue

	if ctx.Colors != nil && ctx.Colors.Custom != nil {
		if c, ok := ctx.Colors.Custom["runbutton.main"]; ok {
			mainColor = c
		}
		if c, ok := ctx.Colors.Custom["runbutton.test"]; ok {
			testColor = c
		}
	}

	// Render buttons for each visible paragraph
	buttonSizePx := gtx.Dp(unit.Dp(buttonSize))

	for _, para := range ctx.Paragraphs {
		// Skip paragraphs outside the viewport
		if para.EndY < ctx.Viewport.Min.Y {
			continue
		}
		if para.StartY > ctx.Viewport.Max.Y {
			break
		}

		btnType, hasButton := p.buttonTypes[para.Index]
		if !hasButton || btnType == RunButtonNone {
			continue
		}

		// Calculate button position (to the right of line numbers)
		// The button should align with the line's baseline, not center
		buttonY := para.StartY - ctx.Viewport.Min.Y
		// Position button to the right of line numbers with a small gap
		gapPx := 4 // Small gap between line numbers and run button
		xPos := ctx.LineNumberWidth + gapPx

		// Calculate triangle size and position
		triangleSize := float32(buttonSizePx) * 0.5 // Make triangle smaller
		centerX := float32(xPos) + float32(buttonSizePx)/2
		centerY := float32(buttonY)

		// Register click handler using clip (use full button area for easier clicking)
		pointer.CursorPointer.Add(gtx.Ops)
		clip.Rect(image.Rect(xPos, buttonY, xPos+buttonSizePx, buttonY+p.lineHeight)).Push(gtx.Ops).Pop()
		p.clicker.Add(gtx.Ops)

		// Choose color based on button type
		var btnColor gvcolor.Color
		if btnType == RunButtonMain {
			btnColor = mainColor
		} else if btnType == RunButtonTest {
			btnColor = testColor
		}

		// Draw triangle (play button)
		m := op.Record(gtx.Ops)
		btnColor.Op(gtx.Ops).Add(gtx.Ops)

		// Create triangle path (equilateral triangle pointing right)
		// Position triangle centered vertically on the line
		var path clip.Path
		path.Begin(gtx.Ops)
		// Move to left vertex
		path.MoveTo(f32.Pt(centerX-triangleSize/2, centerY-triangleSize/2))
		// Line to right vertex
		path.LineTo(f32.Pt(centerX+triangleSize/2, centerY))
		// Line to bottom vertex
		path.LineTo(f32.Pt(centerX-triangleSize/2, centerY+triangleSize/2))
		// Close path
		path.Close()

		// Fill the triangle
		outline := clip.Outline{Path: path.End()}
		stack := outline.Op().Push(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		stack.Pop()

		m.Stop().Add(gtx.Ops)
	}

	return layout.Dimensions{Size: image.Pt(buttonSizePx, 0)}
}

// HandleClick implements the InteractiveGutter interface.
func (p *RunButtonProvider) HandleClick(line int, source pointer.Source, numClicks int, modifiers key.Modifiers) bool {
	btnType, hasButton := p.buttonTypes[line]
	if !hasButton || btnType == RunButtonNone {
		return false
	}

	// Generate run button event
	p.pending = append(p.pending, RunButtonEvent{
		ButtonType: btnType,
		Line:       line,
		ButtonText: p.buttonTexts[line],
	})

	return true
}

// HandleHover implements the InteractiveGutter interface.
func (p *RunButtonProvider) HandleHover(line int) *gutter.HoverInfo {
	btnType, hasButton := p.buttonTypes[line]
	if !hasButton || btnType == RunButtonNone {
		return nil
	}

	var text string
	if btnType == RunButtonMain {
		text = "Run main function"
	} else if btnType == RunButtonTest {
		text = "Run test function"
	}

	return &gutter.HoverInfo{
		Text: text,
	}
}

// analyzeLines analyzes line contents to determine if they should have run buttons.
func (p *RunButtonProvider) analyzeLines(lines []string, startLine int) {
	// Patterns for detecting main and test functions
	mainPattern := regexp.MustCompile(`^func\s+main\s*\(`)
	testPattern := regexp.MustCompile(`^func\s+Test\w+\s*\(`)
	benchmarkPattern := regexp.MustCompile(`^func\s+Benchmark\w+\s*\(`)

	// Clear previous button types
	p.buttonTypes = make(map[int]RunButtonType)
	p.buttonTexts = make(map[int]string)

	for i, line := range lines {
		line = trimLine(line)
		absoluteLine := startLine + i

		// Check for main function
		if mainPattern.MatchString(line) {
			p.buttonTypes[absoluteLine] = RunButtonMain
			p.buttonTexts[absoluteLine] = line
			continue
		}

		// Check for test function
		if testPattern.MatchString(line) {
			p.buttonTypes[absoluteLine] = RunButtonTest
			p.buttonTexts[absoluteLine] = line
			continue
		}

		// Check for benchmark function
		if benchmarkPattern.MatchString(line) {
			p.buttonTypes[absoluteLine] = RunButtonTest
			p.buttonTexts[absoluteLine] = line
		}
	}
}

// trimLine removes leading/trailing whitespace and comments from a line.
func trimLine(line string) string {
	// Remove comments
	for i := 0; i < len(line); i++ {
		if line[i] == '/' && i+1 < len(line) && line[i+1] == '/' {
			line = line[:i]
			break
		}
	}
	return trimSpace(line)
}

// trimSpace trims whitespace from a string.
func trimSpace(s string) string {
	// Simple implementation, can be optimized
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// GetPendingEvents returns pending run button events and clears the pending list.
func (p *RunButtonProvider) GetPendingEvents() []RunButtonEvent {
	events := p.pending
	p.pending = p.pending[:0]
	return events
}

// RunButtonEvent represents a click event on a run button.
type RunButtonEvent struct {
	// ButtonType is the type of button that was clicked.
	ButtonType RunButtonType

	// Line is the 0-based line number where the button was clicked.
	Line int

	// ButtonText is the text content of the line containing the button.
	ButtonText string
}
