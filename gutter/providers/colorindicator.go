package providers

import (
	"image"
	"image/color"
	"strings"

	"gioui.org/gesture"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"github.com/oligo/gvcode/colorpicker"
	"github.com/oligo/gvcode/gutter"
	gestureExt "github.com/oligo/gvcode/internal/gesture"
	lt "github.com/oligo/gvcode/internal/layout"
	"golang.org/x/image/math/fixed"
)

const (
	// ColorIndicatorProviderID is the unique identifier for the color indicator provider.
	ColorIndicatorProviderID = "colorindicator"

	// indicatorSize is the size of the color indicator in dp units.
	indicatorSize = 16
)

// ColorIndicatorProvider renders color indicators in the gutter for detected colors.
type ColorIndicatorProvider struct {
	// paragraphs caches the visible paragraphs from the last Layout call.
	paragraphs []gutter.Paragraph

	// colorInfos caches color information for each line.
	colorInfos map[int][]colorpicker.ColorInfo

	// lineTexts caches the text content for each line.
	lineTexts map[int]string

	// lineHeight caches the line height from the last Layout call.
	lineHeight int

	// viewport caches the viewport from the last Layout call.
	viewport image.Rectangle

	// hoverer handles hover events.
	hoverer gestureExt.Hover

	// clicker handles click events.
	clicker gesture.Click

	// editorText is the current editor text.
	editorText string

	// isHovering indicates if the mouse is hovering over a color indicator.
	isHovering bool

	// hoveredLine is the line being hovered.
	hoveredLine int

	// hoveredColorIndex is the index of the color being hovered.
	hoveredColorIndex int

	// showPicker indicates if the color picker should be shown.
	showPicker bool

	// colorPicker is the color picker instance.
	colorPicker *colorpicker.ColorPicker

	// editorColorPicker manages the color picker integration.
	editorColorPicker *colorpicker.EditorColorPicker

	// enabled indicates whether color indicators are enabled.
	enabled bool

	// startLine is the first line that was provided in SetLineContents.
	startLine int
}

// NewColorIndicatorProvider creates a new color indicator provider.
func NewColorIndicatorProvider() *ColorIndicatorProvider {
	editorColorPicker := colorpicker.NewEditorColorPicker()

	provider := &ColorIndicatorProvider{
		colorInfos:        make(map[int][]colorpicker.ColorInfo),
		lineTexts:         make(map[int]string),
		paragraphs:        make([]gutter.Paragraph, 0),
		editorColorPicker: editorColorPicker,
		enabled:           true,
	}

	// Set the close callback to reset showPicker when the picker is closed
	editorColorPicker.SetOnClose(func() {
		provider.showPicker = false
	})

	return provider
}

// SetEnabled sets whether color indicators are enabled.
func (p *ColorIndicatorProvider) SetEnabled(enabled bool) {
	p.enabled = enabled
}

// Enabled returns whether color indicators are enabled.
func (p *ColorIndicatorProvider) Enabled() bool {
	return p.enabled
}

// ID returns the unique identifier for this provider.
func (p *ColorIndicatorProvider) ID() string {
	return ColorIndicatorProviderID
}

// Priority returns the rendering priority. Color indicators have priority 140
// meaning they are rendered between line numbers (100) and run buttons (150).
func (p *ColorIndicatorProvider) Priority() int {
	return 140
}

// Width returns the fixed width needed for color indicators.
func (p *ColorIndicatorProvider) Width(gtx layout.Context, shaper *text.Shaper, params text.Parameters, lineCount int) unit.Dp {
	if !p.enabled {
		return 0
	}
	return unit.Dp(indicatorSize + 4) // Indicator size plus padding
}

// SetLineContents is called to provide line contents for color detection.
func (p *ColorIndicatorProvider) SetLineContents(lines []string, startLine int) {
	if !p.enabled {
		return
	}

	var fullText strings.Builder
	for i, line := range lines {
		fullText.WriteString(line)
		if i < len(lines)-1 {
			fullText.WriteString("\n")
		}
	}

	p.editorText = fullText.String()
	p.startLine = startLine

	p.lineTexts = make(map[int]string)
	for i, line := range lines {
		p.lineTexts[startLine+i] = line
	}

	detector := colorpicker.NewColorDetector()
	allColors := detector.DetectColors(fullText.String())

	p.colorInfos = make(map[int][]colorpicker.ColorInfo)
	for _, colorInfo := range allColors {
		lineNum := 0
		charCount := 0
		for i, line := range lines {
			if colorInfo.Range.Start >= charCount && colorInfo.Range.Start < charCount+len(line) {
				lineNum = startLine + i
				break
			}
			charCount += len(line) + 1
		}

		p.colorInfos[lineNum] = append(p.colorInfos[lineNum], colorInfo)
	}

	// Update editor color picker
	p.editorColorPicker.SetEditorText(fullText.String())
}

// Layout renders color indicators for gutter (disabled - now rendering in text area).
func (p *ColorIndicatorProvider) Layout(gtx layout.Context, ctx gutter.GutterContext) layout.Dimensions {
	// Cache context info for event handling
	p.paragraphs = ctx.Paragraphs
	p.lineHeight = ctx.LineHeight.Ceil()
	p.viewport = ctx.Viewport

	// Process hover events
	evt, ok := p.hoverer.Update(gtx)
	if ok && evt.Kind == gestureExt.KindHovered {
		// Find which color indicator is being hovered
		clickY := evt.Position.Y + ctx.Viewport.Min.Y
		clickX := evt.Position.X
		line := p.hitTestLine(clickY)

		if line >= 0 {
			colorIndex := p.hitTestColorIndex(line, clickX, ctx)
			if colorIndex >= 0 {
				p.hoveredLine = line
				p.hoveredColorIndex = colorIndex
				// Don't auto-open on hover, only on click
			} else {
				p.hoveredLine = -1
				p.hoveredColorIndex = -1
			}
		} else {
			p.hoveredLine = -1
			p.hoveredColorIndex = -1
		}
	} else if ok && evt.Kind == gestureExt.KindCancelled {
		// Don't close picker when hover is cancelled if picker is open
		// Only close when user explicitly closes it (e.g., clicks elsewhere or presses ESC)
		// This prevents the picker from disappearing immediately after opening
	}

	// Don't render in gutter anymore - render in text area instead
	return layout.Dimensions{Size: image.Pt(0, 0)}
}

// HandleClick implements the InteractiveGutter interface.
func (p *ColorIndicatorProvider) HandleClick(line int, source pointer.Source, numClicks int, modifiers key.Modifiers) bool {
	colors, hasColors := p.colorInfos[line]
	if !hasColors || len(colors) == 0 {
		return false
	}

	p.hoveredLine = line
	p.hoveredColorIndex = 0
	p.showPicker = true
	p.editorColorPicker.HandleClick(0)

	return true
}

// HandleHover implements the InteractiveGutter interface.
func (p *ColorIndicatorProvider) HandleHover(line int) *gutter.HoverInfo {
	colors, hasColors := p.colorInfos[line]
	if !hasColors || len(colors) == 0 {
		return nil
	}

	return &gutter.HoverInfo{
		Text: "Click to open color picker",
	}
}

// GetEditorColorPicker returns the editor color picker instance.
func (p *ColorIndicatorProvider) GetEditorColorPicker() gutter.ColorPickerLayout {
	return p.editorColorPicker
}

// ShowColorPicker returns whether the color picker should be shown.
func (p *ColorIndicatorProvider) ShowColorPicker() bool {
	return p.showPicker
}

func (p *ColorIndicatorProvider) CloseColorPicker() {
	p.showPicker = false
	p.editorColorPicker.HandleCancel()
}

func (p *ColorIndicatorProvider) RenderInTextArea(gtx layout.Context, ctx gutter.GutterContext, gutterWidth int) {
	for _, para := range ctx.Paragraphs {
		colors, hasColors := p.colorInfos[para.Index]
		if !hasColors || len(colors) == 0 {
			continue
		}

		if para.EndY < ctx.Viewport.Min.Y {
			continue
		}
		if para.StartY > ctx.Viewport.Max.Y {
			break
		}

		lineText := p.getLineText(para.Index)
		if lineText == "" {
			continue
		}

		lineOffset := 0
		if p.startLine > 0 {
			continue
		}
		for i := p.startLine; i < para.Index; i++ {
			prevLineText := p.getLineText(i)
			if prevLineText == "" {
				lineOffset += 1
			} else {
				lineOffset += len(prevLineText) + 1
			}
		}

		var layoutLine *lt.Line
		if para.Index < len(ctx.LayoutLines) {
			layoutLine = &ctx.LayoutLines[para.Index]
		}

		// Collect glyph positions from the layout line
		glyphPositions := make([]int, len(lineText)+1)
		if layoutLine != nil {
			currentPos := 0
			for i, glyph := range layoutLine.Glyphs {
				// Map glyph position to character positions
				runesInGlyph := int(glyph.Runes)
				// Use original glyph positions (before color offsets) for indicator placement
				// The text layout has already inserted space at the original positions,
				// so we need to render indicators at those original positions
				originalX := fixed.Int26_6(0)
				if i < len(layoutLine.OriginalGlyphPositions) {
					originalX = layoutLine.OriginalGlyphPositions[i]
				} else {
					originalX = glyph.X
				}
				for i := 0; i < runesInGlyph && currentPos < len(lineText); i++ {
					glyphPositions[currentPos] = originalX.Floor()
					currentPos++
				}
			}
			// Fill remaining positions with the last known position
			if currentPos < len(glyphPositions) {
				lastPos := 0
				if currentPos > 0 {
					lastPos = glyphPositions[currentPos-1]
				}
				for currentPos < len(glyphPositions) {
					glyphPositions[currentPos] = lastPos
					currentPos++
				}
			}
		} else {
			// Fallback: layout text to get glyph positions
			params := ctx.TextParams
			params.MinWidth = 0
			params.MaxLines = 1
			ctx.Shaper.LayoutString(params, lineText)

			currentPos := 0
			for {
				glyph, ok := ctx.Shaper.NextGlyph()
				if !ok {
					break
				}
				runesInGlyph := int(glyph.Runes)
				for i := 0; i < runesInGlyph && currentPos < len(lineText); i++ {
					glyphPositions[currentPos] = glyph.X.Floor()
					currentPos++
				}
			}
			// Fill remaining positions with the last known position
			if currentPos < len(glyphPositions) {
				lastPos := 0
				if currentPos > 0 {
					lastPos = glyphPositions[currentPos-1]
				}
				for currentPos < len(glyphPositions) {
					glyphPositions[currentPos] = lastPos
					currentPos++
				}
			}
		}

		charToGlyphIndex := make([]int, len(lineText)+1)
		currentCharPos := 0
		for glyphIdx, glyph := range layoutLine.Glyphs {
			runesInGlyph := int(glyph.Runes)
			for i := 0; i < runesInGlyph && currentCharPos < len(lineText); i++ {
				charToGlyphIndex[currentCharPos] = glyphIdx
				currentCharPos++
			}
		}
		if currentCharPos < len(charToGlyphIndex) {
			lastGlyphIdx := 0
			if currentCharPos > 0 {
				lastGlyphIdx = charToGlyphIndex[currentCharPos-1]
			}
			for currentCharPos < len(charToGlyphIndex) {
				charToGlyphIndex[currentCharPos] = lastGlyphIdx
				currentCharPos++
			}
		}

		accumulatedOffset := 0

		for i, colorInfo := range colors {
			baselineY := para.StartY - ctx.Viewport.Min.Y
			ascent := para.Ascent.Ceil()
			descent := para.Descent.Ceil()
			textHeight := ascent + descent

			indicatorHeight := textHeight
			indicatorWidth := textHeight
			indicatorY := baselineY - ascent

			colorStartPos := colorInfo.Range.Start - lineOffset

			if colorStartPos < 0 || colorStartPos >= len(lineText) {
				continue
			}

			var indicatorX int
			if colorStartPos < len(charToGlyphIndex) {
				glyphIdx := charToGlyphIndex[colorStartPos]
				if glyphIdx < len(layoutLine.OriginalGlyphPositions) {
					indicatorX = layoutLine.OriginalGlyphPositions[glyphIdx].Floor() + accumulatedOffset
				} else if glyphIdx < len(layoutLine.Glyphs) {
					indicatorX = layoutLine.Glyphs[glyphIdx].X.Floor() + accumulatedOffset
				} else {
					indicatorX = 0
				}
			} else {
				indicatorX = 0
			}

			indicatorRect := image.Rect(indicatorX, indicatorY, indicatorX+indicatorWidth, indicatorY+indicatorHeight)

			pointer.CursorPointer.Add(gtx.Ops)
			stack := clip.Rect(indicatorRect).Push(gtx.Ops)
			p.clicker.Add(gtx.Ops)

			for {
				evt, ok := p.clicker.Update(gtx.Source)
				if !ok {
					break
				}
				if evt.Kind == gesture.KindClick {
					p.HandleClick(para.Index, evt.Source, evt.NumClicks, evt.Modifiers)
				}
			}

			stack.Pop()

			paint.FillShape(gtx.Ops, colorInfo.Color, clip.Rect(indicatorRect).Op())

			if p.hoveredLine == para.Index && p.hoveredColorIndex == i {
				borderColor := color.NRGBA{R: 0, G: 120, B: 215, A: 255}
				borderWidth := 2
				borderRect := image.Rect(
					indicatorX-borderWidth,
					indicatorY-borderWidth,
					indicatorX+indicatorWidth+borderWidth,
					indicatorY+indicatorHeight+borderWidth,
				)
				paint.FillShape(gtx.Ops, borderColor, clip.Rect(borderRect).Op())
			}

			accumulatedOffset += indicatorWidth
		}
	}
}

// getLineText returns the text content for a specific line
func (p *ColorIndicatorProvider) getLineText(lineIndex int) string {
	// Return the cached line text
	if lineText, hasLineText := p.lineTexts[lineIndex]; hasLineText {
		return lineText
	}
	return ""
}

// hitTestLine determines which logical line corresponds to a Y coordinate.
func (p *ColorIndicatorProvider) hitTestLine(y int) int {
	if len(p.paragraphs) == 0 {
		return -1
	}

	for _, para := range p.paragraphs {
		if y >= para.StartY && y <= para.EndY {
			return para.Index
		}
	}

	return -1
}

// hitTestColorIndex determines which color indicator on a line corresponds to an X coordinate.
func (p *ColorIndicatorProvider) hitTestColorIndex(line int, x int, ctx gutter.GutterContext) int {
	colors, hasColors := p.colorInfos[line]
	if !hasColors || len(colors) == 0 {
		return -1
	}

	indicatorSizePx := 16 // Fixed size in pixels for hit testing
	gapPx := 2

	for i := range colors {
		xPos := gapPx + i*(indicatorSizePx+gapPx)
		if x >= xPos && x <= xPos+indicatorSizePx {
			return i
		}
	}

	return -1
}

// GetColorOffsets returns the character offsets where color indicators should be inserted.
// Returns a map from line number to a slice of character offsets.
func (p *ColorIndicatorProvider) GetColorOffsets() map[int][]int {
	offsets := make(map[int][]int)
	for line, colors := range p.colorInfos {
		// Calculate line offset for converting global positions to line-relative positions
		lineOffset := 0
		for i := range line {
			prevLineText := p.getLineText(i)
			if prevLineText == "" {
				lineOffset += 1 // +1 for newline even for empty lines
			} else {
				lineOffset += len(prevLineText) + 1 // +1 for newline
			}
		}

		for _, colorInfo := range colors {
			colorStartPos := colorInfo.Range.Start - lineOffset
			if colorStartPos >= 0 {
				offsets[line] = append(offsets[line], colorStartPos)
			}
		}
	}
	return offsets
}

// GetIndicatorWidth returns the width of the color indicator in pixels.
func (p *ColorIndicatorProvider) GetIndicatorWidth(gtx layout.Context) int {
	// Return a reasonable default width for text layout
	// The actual indicator size will be calculated dynamically based on text height
	return gtx.Dp(unit.Dp(18)) // 18dp is a reasonable default for most font sizes
}
