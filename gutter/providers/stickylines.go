package providers

import (
	"image"
	"image/color"
	"regexp"
	"strings"

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
	// StickyLinesProviderID is the unique identifier for the sticky lines provider.
	StickyLinesProviderID = "stickylines"

	// DefaultMaxStickyLines is the default maximum number of sticky lines to display.
	DefaultMaxStickyLines = 5
)

// StickyLineInfo contains information about a sticky line.
type StickyLineInfo struct {
	// Line is the 0-based line number.
	Line int
	// Text is the text content of the line.
	Text string
	// Indent is the indentation level (number of leading tabs/spaces).
	Indent int
	// Type is the type of code structure (function, type, const, etc.).
	Type string
}

// StickyLinesProvider renders sticky lines that remain visible while scrolling.
// This is similar to JetBrains GoLand's "Sticky Lines" feature.
type StickyLinesProvider struct {
	// enabled indicates whether sticky lines are enabled.
	enabled bool

	// maxStickyLines is the maximum number of sticky lines to display.
	maxStickyLines int

	// stickyLines contains the currently sticky lines.
	stickyLines []StickyLineInfo

	// allLines caches all lines from the document for structure analysis.
	allLines []string

	// structureCache caches the code structure analysis results.
	structureCache []StickyLineInfo

	// clicker handles click events on sticky lines.
	clicker gesture.Click

	// pending holds sticky line click events.
	pending []StickyLineEvent

	// paragraphs caches the visible paragraphs from the last Layout call.
	paragraphs []gutter.Paragraph

	// lineHeight caches the line height from the last Layout call.
	lineHeight int

	// viewport caches the viewport from the last Layout call.
	viewport image.Rectangle

	// stickyAreaHeight is the height occupied by sticky lines.
	stickyAreaHeight int

	// scrollOffY is the Y scroll offset from the last Layout call.
	scrollOffY int

	// stickyBackgroundColor is the background color for sticky lines.
	stickyBackgroundColor gvcolor.Color

	// stickyBorderColor is the border color for sticky lines.
	stickyBorderColor gvcolor.Color

	// stickyTextColor is the text color for sticky lines.
	stickyTextColor gvcolor.Color
}

// StickyLineEvent represents a click event on a sticky line.
type StickyLineEvent struct {
	// Line is the 0-based line number that was clicked.
	Line int
	// Text is the text content of the line.
	Text string
}

// NewStickyLinesProvider creates a new sticky lines provider with default settings.
func NewStickyLinesProvider() *StickyLinesProvider {
	return &StickyLinesProvider{
		enabled:        true,
		maxStickyLines: DefaultMaxStickyLines,
		stickyLines:    make([]StickyLineInfo, 0),
		structureCache: make([]StickyLineInfo, 0),
		pending:        make([]StickyLineEvent, 0),
	}
}

// SetEnabled sets whether sticky lines are enabled.
func (p *StickyLinesProvider) SetEnabled(enabled bool) {
	p.enabled = enabled
}

// Enabled returns whether sticky lines are enabled.
func (p *StickyLinesProvider) Enabled() bool {
	return p.enabled
}

// SetMaxStickyLines sets the maximum number of sticky lines to display.
func (p *StickyLinesProvider) SetMaxStickyLines(max int) {
	if max < 1 {
		max = 1
	}
	p.maxStickyLines = max
}

// MaxStickyLines returns the maximum number of sticky lines.
func (p *StickyLinesProvider) MaxStickyLines() int {
	return p.maxStickyLines
}

// ID returns the unique identifier for this provider.
func (p *StickyLinesProvider) ID() string {
	return StickyLinesProviderID
}

// Priority returns the rendering priority. Sticky lines have priority 0,
// meaning they are rendered on top of all other content in the editor area.
func (p *StickyLinesProvider) Priority() int {
	return 0
}

// Width returns the width needed for sticky lines.
// Since sticky lines span the entire editor width, this returns 0.
func (p *StickyLinesProvider) Width(gtx layout.Context, shaper *text.Shaper, params text.Parameters, lineCount int) unit.Dp {
	return unit.Dp(0)
}

// SetLineContents sets the contents of all lines for structure analysis.
// This implements the gutter.LineContentProvider interface.
func (p *StickyLinesProvider) SetLineContents(lines []string, startLine int) {
	// Only update if the content has changed
	if p.allLines == nil || len(p.allLines) != len(lines) {
		p.allLines = lines
		p.analyzeStructure()
	}
}

// analyzeStructure analyzes the code structure to identify lines that can be sticky.
// This includes functions, types, constants, variables, etc.
func (p *StickyLinesProvider) analyzeStructure() {
	if len(p.allLines) == 0 {
		p.structureCache = make([]StickyLineInfo, 0)
		return
	}

	p.structureCache = make([]StickyLineInfo, 0)

	// Regular expressions for different code structures (Go-specific patterns)
	functionPattern := regexp.MustCompile(`^\s*(func|func\s+\(\s*\w+\s*\*?\s*\w+\s*\))\s+(\w+)\s*\(`)
	typePattern := regexp.MustCompile(`^\s*type\s+(\w+)\s+(struct|interface|map|chan|func)`)
	constPattern := regexp.MustCompile(`^\s*(const|var)\s+\(`)
	importPattern := regexp.MustCompile(`^\s*import\s*\(`)
	simpleConstPattern := regexp.MustCompile(`^\s*const\s+\w+`)
	simpleVarPattern := regexp.MustCompile(`^\s*var\s+\w+`)

	for i, line := range p.allLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Calculate indentation level
		indent := p.calculateIndent(line)

		var stickyType string
		var shouldStick bool

		// Check for function declarations
		if functionPattern.MatchString(line) {
			stickyType = "function"
			shouldStick = true
		} else if typePattern.MatchString(line) {
			stickyType = "type"
			shouldStick = true
		} else if constPattern.MatchString(line) || importPattern.MatchString(line) {
			stickyType = "block"
			shouldStick = true
		} else if simpleConstPattern.MatchString(line) {
			stickyType = "const"
			shouldStick = true
		} else if simpleVarPattern.MatchString(line) {
			// Only stick top-level variables (indentation 0 or 1)
			if indent <= 1 {
				stickyType = "var"
				shouldStick = true
			}
		}

		if shouldStick {
			p.structureCache = append(p.structureCache, StickyLineInfo{
				Line:   i,
				Text:   line,
				Indent: indent,
				Type:   stickyType,
			})
		}
	}
}

// calculateIndent calculates the indentation level of a line.
func (p *StickyLinesProvider) calculateIndent(line string) int {
	indent := 0
	for _, r := range line {
		if r == ' ' {
			// Assume 4 spaces per indentation level
			indent++
		} else if r == '\t' {
			indent += 4
		} else {
			break
		}
	}
	return indent / 4
}

// Layout renders sticky lines on top of the editor content.
func (p *StickyLinesProvider) Layout(gtx layout.Context, ctx gutter.GutterContext) layout.Dimensions {
	// Cache context info for event handling
	p.paragraphs = ctx.Paragraphs
	p.lineHeight = ctx.LineHeight.Ceil()
	p.viewport = ctx.Viewport
	p.scrollOffY = ctx.Viewport.Min.Y

	// Set up colors
	p.setupColors(ctx.Colors)

	// Calculate which lines should be sticky based on current scroll position
	p.calculateStickyLines(ctx)

	// Render sticky lines (if any - though rendering is done by editor)
	if len(p.stickyLines) > 0 {
		// p.renderStickyLines is not used here, editor handles rendering
	}

	return layout.Dimensions{Size: image.Point{X: 0, Y: p.stickyAreaHeight}}
}

// setupColors sets up the colors for sticky lines based on the context.
func (p *StickyLinesProvider) setupColors(colors *gutter.GutterColors) {
	if colors != nil {
		// Use background color with slight opacity for sticky background
		if colors.Background.IsSet() {
			bg := colors.Background.NRGBA()
			// Add slight transparency
			bg.A = 0xD0
			p.stickyBackgroundColor = gvcolor.MakeColor(bg)
		} else {
			p.stickyBackgroundColor = gvcolor.MakeColor(color.NRGBA{R: 0xF0, G: 0xF0, B: 0xF0, A: 0xD0})
		}

		// Use border color from text color
		if colors.Text.IsSet() {
			border := colors.Text.NRGBA()
			border.A = 0x40
			p.stickyBorderColor = gvcolor.MakeColor(border)
		} else {
			p.stickyBorderColor = gvcolor.MakeColor(color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x40})
		}

		// Use text color for sticky text
		if colors.Text.IsSet() {
			p.stickyTextColor = colors.Text
		} else {
			p.stickyTextColor = gvcolor.MakeColor(color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF})
		}
	} else {
		p.stickyBackgroundColor = gvcolor.MakeColor(color.NRGBA{R: 0xF0, G: 0xF0, B: 0xF0, A: 0xD0})
		p.stickyBorderColor = gvcolor.MakeColor(color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x40})
		p.stickyTextColor = gvcolor.MakeColor(color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF})
	}
}

// calculateStickyLines determines which lines should be sticky based on scroll position.
func (p *StickyLinesProvider) calculateStickyLines(ctx gutter.GutterContext) {
	if !p.enabled || len(p.structureCache) == 0 {
		p.stickyLines = p.stickyLines[:0]
		p.stickyAreaHeight = 0
		return
	}

	// Find the first visible paragraph
	firstVisibleLine := -1
	for _, para := range ctx.Paragraphs {
		if para.EndY >= ctx.Viewport.Min.Y && para.StartY <= ctx.Viewport.Max.Y {
			firstVisibleLine = para.Index
			break
		}
	}

	if firstVisibleLine == -1 {
		p.stickyLines = p.stickyLines[:0]
		p.stickyAreaHeight = 0
		return
	}

	// Find all structure lines that are above or at the first visible line
	p.stickyLines = p.stickyLines[:0]
	for _, info := range p.structureCache {
		if info.Line <= firstVisibleLine {
			p.stickyLines = append(p.stickyLines, info)
		} else {
			break
		}
	}

	// Limit to max sticky lines
	if len(p.stickyLines) > p.maxStickyLines {
		p.stickyLines = p.stickyLines[len(p.stickyLines)-p.maxStickyLines:]
	}

	// Calculate sticky area height
	p.stickyAreaHeight = len(p.stickyLines) * p.lineHeight
}

// renderStickyLines renders the sticky lines at the top of the viewport.
func (p *StickyLinesProvider) renderStickyLines(gtx layout.Context, ctx gutter.GutterContext) {
	lineHeightPx := p.lineHeight

	// Draw background and text for each sticky line
	for i, sticky := range p.stickyLines {
		stickyY := i * lineHeightPx

		// Draw background
		bgRect := image.Rect(0, stickyY, gtx.Constraints.Max.X, stickyY+lineHeightPx)
		bgStack := clip.Rect(bgRect).Push(gtx.Ops)
		paint.ColorOp{Color: p.stickyBackgroundColor.NRGBA()}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		bgStack.Pop()

		// Draw border at bottom
		if i < len(p.stickyLines)-1 {
			borderRect := image.Rect(0, stickyY+lineHeightPx-1, gtx.Constraints.Max.X, stickyY+lineHeightPx)
			borderStack := clip.Rect(borderRect).Push(gtx.Ops)
			paint.ColorOp{Color: p.stickyBorderColor.NRGBA()}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			borderStack.Pop()
		}

		// Register click handler
		pointer.CursorPointer.Add(gtx.Ops)
		clip.Rect(bgRect).Push(gtx.Ops).Pop()
		p.clicker.Add(gtx.Ops)

		// Draw text
		params := ctx.TextParams
		params.MinWidth = 0
		params.MaxLines = 1

		// Trim whitespace for display
		displayText := strings.TrimLeft(sticky.Text, "\t")
		displayText = strings.TrimRight(displayText, " \t\r\n")

		ctx.Shaper.LayoutString(params, displayText)

		glyphs := make([]text.Glyph, 0)
		var bounds image.Rectangle

		for {
			g, ok := ctx.Shaper.NextGlyph()
			if !ok {
				break
			}

			bounds.Min.X = min(bounds.Min.X, g.X.Floor())
			bounds.Min.Y = min(bounds.Min.Y, int(g.Y)-g.Ascent.Floor())
			bounds.Max.X = max(bounds.Max.X, (g.X + g.Advance).Ceil())
			bounds.Max.Y = max(bounds.Max.Y, int(g.Y)+g.Descent.Ceil())

			glyphs = append(glyphs, g)
		}

		if len(glyphs) > 0 {
			// Transform to the correct position
			yPos := float32(stickyY) + float32(lineHeightPx)/2
			trans := op.Affine(f32.Affine2D{}.Offset(
				f32.Point{X: float32(glyphs[0].X.Floor()) + 8, Y: yPos},
			)).Push(gtx.Ops)

			// Draw the glyphs
			path := ctx.Shaper.Shape(glyphs)
			outline := clip.Outline{Path: path}.Op().Push(gtx.Ops)

			paint.ColorOp{Color: p.stickyTextColor.NRGBA()}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)

			outline.Pop()
			trans.Pop()
		}
	}
}

// GetPendingEvents returns pending sticky line events and clears the pending list.
func (p *StickyLinesProvider) GetPendingEvents() []StickyLineEvent {
	events := p.pending
	p.pending = p.pending[:0]
	return events
}

// StickyLinesHeight returns the height occupied by sticky lines.
func (p *StickyLinesProvider) StickyLinesHeight() int {
	return p.stickyAreaHeight
}

// HandleClick handles click events on sticky lines.
// Since sticky lines are rendered over the editor, not in the gutter,
// this method is not used. Click handling is done via HandleStickyLineClick.
func (p *StickyLinesProvider) HandleClick(line int, source pointer.Source, numClicks int, modifiers key.Modifiers) bool {
	// Sticky lines are not in the gutter area, so this is not used
	return false
}

// HandleHover handles hover events on sticky lines.
// Since sticky lines are rendered over the editor, not in the gutter,
// this method is not used.
func (p *StickyLinesProvider) HandleHover(line int) *gutter.HoverInfo {
	// Sticky lines are not in the gutter area, so this is not used
	return nil
}

// HandleStickyLineClick handles a click on a sticky line.
// The y parameter is the Y coordinate within the sticky lines area.
func (p *StickyLinesProvider) HandleStickyLineClick(y int) bool {
	if p.stickyAreaHeight == 0 || len(p.stickyLines) == 0 {
		return false
	}

	// Find which sticky line was clicked (y coordinate)
	stickyLineIndex := y / p.lineHeight

	if stickyLineIndex >= 0 && stickyLineIndex < len(p.stickyLines) {
		// Generate sticky line event
		info := p.stickyLines[stickyLineIndex]
		p.pending = append(p.pending, StickyLineEvent{
			Line: info.Line,
			Text: info.Text,
		})
		return true
	}

	return false
}

// GetStickyLinesInfo returns the current sticky lines and their total height.
// This is used by the Editor to render sticky lines.
func (p *StickyLinesProvider) GetStickyLinesInfo() ([]struct {
	Line   int
	Text   string
	Indent int
	Type   string
}, int,
) {
	result := make([]struct {
		Line   int
		Text   string
		Indent int
		Type   string
	}, len(p.stickyLines))

	for i, info := range p.stickyLines {
		result[i] = struct {
			Line   int
			Text   string
			Indent int
			Type   string
		}{
			Line:   info.Line,
			Text:   info.Text,
			Indent: info.Indent,
			Type:   info.Type,
		}
	}

	return result, p.stickyAreaHeight
}
