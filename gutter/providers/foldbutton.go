package providers

import (
	"image"
	"image/color"

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
	"github.com/oligo/gvcode/internal/folding"
)

const (
	// FoldButtonProviderID is the unique identifier for the fold button provider.
	FoldButtonProviderID = "foldbutton"

	// foldButtonSize is the size of the fold button in dp units.
	foldButtonSize = 12
)

// FoldButtonType represents the type of fold button to display.
type FoldButtonType int

const (
	// FoldButtonNone indicates no fold button should be shown.
	FoldButtonNone FoldButtonType = iota
	// FoldButtonCollapsed indicates a collapsed fold (show expand icon).
	FoldButtonCollapsed
	// FoldButtonExpanded indicates an expanded fold (show collapse icon).
	FoldButtonExpanded
)

// FoldButtonProvider renders fold buttons in the gutter.
type FoldButtonProvider struct {
	// foldManager manages fold regions.
	foldManager *folding.Manager

	// clicker handles click events on buttons.
	clicker gesture.Click

	// pending holds fold events that haven't been consumed yet.
	pending []FoldButtonEvent

	// paragraphs caches the visible paragraphs from the last Layout call.
	paragraphs []gutter.Paragraph

	// lineHeight caches the line height from the last Layout call.
	lineHeight int

	// viewport caches the viewport from the last Layout call.
	viewport image.Rectangle

	// buttonStates caches the button state for each line.
	buttonStates map[int]FoldButtonType

	// enabled indicates whether fold buttons are enabled.
	enabled bool
}

// FoldButtonEvent represents a click event on a fold button.
type FoldButtonEvent struct {
	// Line is the 0-based line number of the fold.
	Line int
	// IsCollapsed indicates the new state after the click.
	IsCollapsed bool
}

// NewFoldButtonProvider creates a new fold button provider.
func NewFoldButtonProvider(foldManager *folding.Manager) *FoldButtonProvider {
	return &FoldButtonProvider{
		foldManager:  foldManager,
		buttonStates: make(map[int]FoldButtonType),
		pending:      make([]FoldButtonEvent, 0),
		enabled:      true,
	}
}

// SetEnabled sets whether fold buttons are enabled.
func (p *FoldButtonProvider) SetEnabled(enabled bool) {
	p.enabled = enabled
}

// Enabled returns whether fold buttons are enabled.
func (p *FoldButtonProvider) Enabled() bool {
	return p.enabled
}

// ID returns the unique identifier for this provider.
func (p *FoldButtonProvider) ID() string {
	return FoldButtonProviderID
}

// Priority returns the rendering priority. Fold buttons have priority 120
// meaning they are rendered between line numbers (100) and run buttons (150).
func (p *FoldButtonProvider) Priority() int {
	return 120
}

// Width returns the fixed width needed for fold buttons.
func (p *FoldButtonProvider) Width(gtx layout.Context, shaper *text.Shaper, params text.Parameters, lineCount int) unit.Dp {
	if !p.enabled {
		return 0
	}
	return unit.Dp(foldButtonSize + 4) // Button size plus padding
}

// SetLineContents is called to provide line contents for analysis.
// The fold manager will analyze the lines to detect foldable regions.
func (p *FoldButtonProvider) SetLineContents(lines []string, startLine int) {
	if p.foldManager != nil {
		p.foldManager.AnalyzeLines(lines)
	}
}

// Layout renders fold buttons for visible paragraphs.
func (p *FoldButtonProvider) Layout(gtx layout.Context, ctx gutter.GutterContext) layout.Dimensions {
	if !p.enabled {
		return layout.Dimensions{}
	}

	// Cache context info for event handling
	p.paragraphs = ctx.Paragraphs
	p.lineHeight = ctx.LineHeight.Ceil()
	p.viewport = ctx.Viewport

	// Clear previous button states
	p.buttonStates = make(map[int]FoldButtonType)

	// Get all fold ranges
	foldRanges := p.foldManager.GetFoldRanges()

	// Build a map of lines that have fold buttons
	foldMap := make(map[int]*folding.FoldRange)
	for i := range foldRanges {
		foldMap[foldRanges[i].StartLine] = &foldRanges[i]
	}

	// Define colors
	buttonColor := gvcolor.MakeColor(color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xFF})
	if ctx.Colors != nil && ctx.Colors.Text.IsSet() {
		buttonColor = ctx.Colors.Text
	}

	buttonSizePx := gtx.Dp(unit.Dp(foldButtonSize))
	padding := (ctx.LineHeight.Ceil() - buttonSizePx) / 2

	// Render buttons for each visible paragraph
	for _, para := range ctx.Paragraphs {
		// Skip paragraphs outside the viewport
		if para.EndY < ctx.Viewport.Min.Y {
			continue
		}
		if para.StartY > ctx.Viewport.Max.Y {
			break
		}

		// Check if this line has a fold
		fold, hasFold := foldMap[para.Index]
		if !hasFold {
			continue
		}

		// Determine button type
		var btnType FoldButtonType
		if fold.Collapsed {
			btnType = FoldButtonCollapsed
		} else {
			btnType = FoldButtonExpanded
		}
		p.buttonStates[para.Index] = btnType

		// Calculate button position
		buttonY := para.StartY - ctx.Viewport.Min.Y + padding
		xPos := 2 // Small left padding

		// Register click handler
		pointer.CursorPointer.Add(gtx.Ops)
		clip.Rect(image.Rect(xPos, buttonY, xPos+buttonSizePx, buttonY+buttonSizePx)).Push(gtx.Ops).Pop()
		p.clicker.Add(gtx.Ops)

		// Draw the button background/border (subtle rectangle)
		btnRect := image.Rect(xPos, buttonY, xPos+buttonSizePx, buttonY+buttonSizePx)
		btnStack := clip.Rect(btnRect).Push(gtx.Ops)
		paint.ColorOp{Color: color.NRGBA{R: 0xE0, G: 0xE0, B: 0xE0, A: 0x40}}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		btnStack.Pop()

		// Draw the icon (plus or minus)
		centerX := float32(xPos + buttonSizePx/2)
		centerY := float32(buttonY + buttonSizePx/2)
		size := float32(buttonSizePx) * 0.6

		buttonColor.Op(gtx.Ops).Add(gtx.Ops)

		if btnType == FoldButtonCollapsed {
			// Draw plus sign (expand)
			drawPlus(gtx.Ops, centerX, centerY, size)
		} else {
			// Draw minus sign (collapse)
			drawMinus(gtx.Ops, centerX, centerY, size)
		}
	}

	// Process click events
	for {
		evt, ok := p.clicker.Update(gtx.Source)
		if !ok {
			break
		}
		if evt.Kind == gesture.KindClick {
			// Find which line was clicked
			clickY := int(evt.Position.Y) + ctx.Viewport.Min.Y
			line := p.hitTestLine(clickY)

			if line >= 0 {
				// Toggle the fold
				isCollapsed := p.foldManager.ToggleFold(line)
				p.pending = append(p.pending, FoldButtonEvent{
					Line:        line,
					IsCollapsed: isCollapsed,
				})
			}
		}
	}

	buttonWidth := buttonSizePx + 4
	return layout.Dimensions{Size: image.Pt(buttonWidth, 0)}
}

// drawPlus draws a plus sign at the given position.
func drawPlus(ops *op.Ops, centerX, centerY, size float32) {
	// Horizontal line
	hLine := image.Rect(
		int(centerX-size/2),
		int(centerY-1),
		int(centerX+size/2),
		int(centerY+1),
	)
	hStack := clip.Rect(hLine).Push(ops)
	paint.PaintOp{}.Add(ops)
	hStack.Pop()

	// Vertical line
	vLine := image.Rect(
		int(centerX-1),
		int(centerY-size/2),
		int(centerX+1),
		int(centerY+size/2),
	)
	vStack := clip.Rect(vLine).Push(ops)
	paint.PaintOp{}.Add(ops)
	vStack.Pop()
}

// drawMinus draws a minus sign at the given position.
func drawMinus(ops *op.Ops, centerX, centerY, size float32) {
	// Horizontal line only
	hLine := image.Rect(
		int(centerX-size/2),
		int(centerY-1),
		int(centerX+size/2),
		int(centerY+1),
	)
	hStack := clip.Rect(hLine).Push(ops)
	paint.PaintOp{}.Add(ops)
	hStack.Pop()
}

// hitTestLine determines which logical line corresponds to a Y coordinate.
func (p *FoldButtonProvider) hitTestLine(y int) int {
	if len(p.paragraphs) == 0 {
		return -1
	}

	for _, para := range p.paragraphs {
		expandedStartY, expandedEndY := p.expandBounds(para)
		if y >= expandedStartY && y <= expandedEndY {
			return para.Index
		}
	}

	return -1
}

// expandBounds expands a paragraph's vertical bounds.
func (p *FoldButtonProvider) expandBounds(para gutter.Paragraph) (startY, endY int) {
	ascent := para.Ascent.Ceil()
	descent := para.Descent.Ceil()
	glyphHeight := ascent + descent
	lineHeightPx := p.lineHeight

	leading := 0
	if lineHeightPx > glyphHeight {
		leading = lineHeightPx - glyphHeight
	}

	leadingTop := leading / 2
	leadingBottom := leading - leadingTop

	return para.StartY - ascent - leadingTop, para.EndY + descent + leadingBottom
}

// HandleClick implements the InteractiveGutter interface.
func (p *FoldButtonProvider) HandleClick(line int, source pointer.Source, numClicks int, modifiers key.Modifiers) bool {
	// Check if this line has a fold button
	if _, hasButton := p.buttonStates[line]; !hasButton {
		return false
	}

	// Toggle the fold
	isCollapsed := p.foldManager.ToggleFold(line)
	p.pending = append(p.pending, FoldButtonEvent{
		Line:        line,
		IsCollapsed: isCollapsed,
	})

	return true
}

// HandleHover implements the InteractiveGutter interface.
func (p *FoldButtonProvider) HandleHover(line int) *gutter.HoverInfo {
	if _, hasButton := p.buttonStates[line]; !hasButton {
		return nil
	}

	fold := p.foldManager.GetFoldAtLine(line)
	if fold == nil {
		return nil
	}

	var text string
	if fold.Collapsed {
		text = "Expand " + fold.Type.String()
	} else {
		text = "Collapse " + fold.Type.String()
	}
	if fold.Name != "" {
		text += " (" + fold.Name + ")"
	}

	return &gutter.HoverInfo{
		Text: text,
	}
}

// GetPendingEvents returns pending fold button events and clears the pending list.
func (p *FoldButtonProvider) GetPendingEvents() []FoldButtonEvent {
	events := p.pending
	p.pending = p.pending[:0]
	return events
}

// GetFoldManager returns the underlying fold manager.
func (p *FoldButtonProvider) GetFoldManager() *folding.Manager {
	return p.foldManager
}
