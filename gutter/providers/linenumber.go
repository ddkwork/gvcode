package providers

import (
	"image"
	"strconv"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	gvcolor "github.com/oligo/gvcode/color"
	"github.com/oligo/gvcode/gutter"
	"golang.org/x/image/math/fixed"
)

const (
	// defaultMinDigits is the minimum number of digits to reserve space for.
	defaultMinDigits = 4
)

// LineNumberProvider renders line numbers in the gutter.
type LineNumberProvider struct {
	// minDigits is the minimum number of digits to reserve space for.
	// Defaults to 4 (supports up to 9999 lines without width change).
	minDigits int

	// cachedWidth stores the calculated width for the current line count.
	cachedWidth unit.Dp

	// cachedLines stores the line count used to calculate cachedWidth.
	cachedLines int

	// currentLine caches the current line from the last Layout call for HighlightedLines.
	currentLine int

	// lineHighlightColor caches the highlight color from the last Layout call.
	lineHighlightColor gvcolor.Color

	// hasCurrentLine indicates whether there is a valid current line to highlight.
	hasCurrentLine bool
}

// NewLineNumberProvider creates a new line number provider with default settings.
func NewLineNumberProvider() *LineNumberProvider {
	return &LineNumberProvider{
		minDigits: defaultMinDigits,
	}
}

// NewLineNumberProviderWithMinDigits creates a new line number provider with
// a custom minimum digit count.
func NewLineNumberProviderWithMinDigits(minDigits int) *LineNumberProvider {
	if minDigits < 1 {
		minDigits = 1
	}
	return &LineNumberProvider{
		minDigits: minDigits,
	}
}

// ID returns the unique identifier for this provider.
func (p *LineNumberProvider) ID() string {
	return gutter.LineNumberProviderID
}

// Priority returns the rendering priority. Line numbers have a high priority (100)
// meaning they are rendered closest to the text (rightmost in the gutter).
func (p *LineNumberProvider) Priority() int {
	return 100
}

// Width calculates the width needed to display line numbers.
func (p *LineNumberProvider) Width(gtx layout.Context, shaper *text.Shaper, params text.Parameters, lineCount int) unit.Dp {
	// Ensure at least minDigits worth of space
	maxLines := max(lineCount, p.minLinesForDigits())

	// Use cached value if line count hasn't changed significantly
	if p.cachedLines == maxLines && p.cachedWidth > 0 {
		return p.cachedWidth
	}

	width := p.getMaxLineNumWidth(shaper, params, maxLines)
	p.cachedWidth = unit.Dp(float32(width.Ceil()) / gtx.Metric.PxPerDp)
	p.cachedLines = maxLines

	return p.cachedWidth
}

// minLinesForDigits returns the minimum line count to ensure minDigits worth of space.
func (p *LineNumberProvider) minLinesForDigits() int {
	result := 1
	for i := 0; i < p.minDigits; i++ {
		result *= 10
	}
	return result
}

// getMaxLineNumWidth calculates the pixel width needed to display a line number.
func (p *LineNumberProvider) getMaxLineNumWidth(shaper *text.Shaper, params text.Parameters, lineCount int) fixed.Int26_6 {
	params.MinWidth = 0
	shaper.LayoutString(params, strconv.Itoa(lineCount))

	var width fixed.Int26_6
	for {
		g, ok := shaper.NextGlyph()
		if !ok {
			break
		}
		width += g.Advance
	}

	return width
}

// Layout renders the line numbers for visible paragraphs.
func (p *LineNumberProvider) Layout(gtx layout.Context, ctx gutter.GutterContext) layout.Dimensions {
	// Cache current line info for HighlightedLines
	p.hasCurrentLine = ctx.CurrentLine >= 0
	p.currentLine = ctx.CurrentLine
	if ctx.Colors != nil {
		p.lineHighlightColor = ctx.Colors.LineHighlight
	}

	if len(ctx.Paragraphs) == 0 {
		return layout.Dimensions{}
	}

	// Prepare text parameters for right-aligned line numbers
	params := ctx.TextParams
	params.Alignment = text.End
	params.MinWidth = gtx.Constraints.Max.X
	params.MaxLines = 1

	// Create material operations for text colors
	textMaterial := p.createColorOp(gtx.Ops, ctx.Colors.Text)
	highlightMaterial := p.createColorOp(gtx.Ops, ctx.Colors.TextHighlight)

	var dims layout.Dimensions
	glyphs := make([]text.Glyph, 0)

	for _, para := range ctx.Paragraphs {
		// Skip paragraphs outside the viewport
		if para.EndY < ctx.Viewport.Min.Y {
			continue
		}
		if para.StartY > ctx.Viewport.Max.Y {
			break
		}

		// Shape the line number (1-based)
		lineNum := para.Index + 1
		ctx.Shaper.LayoutString(params, strconv.Itoa(lineNum))
		glyphs = glyphs[:0]

		var bounds image.Rectangle
		visible := false

		for {
			g, ok := ctx.Shaper.NextGlyph()
			if !ok {
				break
			}

			// Visibility check based on glyph position
			if para.StartY+g.Descent.Ceil() < ctx.Viewport.Min.Y {
				break
			}
			if para.StartY-g.Ascent.Ceil() > ctx.Viewport.Max.Y {
				break
			}

			bounds.Min.X = min(bounds.Min.X, g.X.Floor())
			bounds.Min.Y = min(bounds.Min.Y, int(g.Y)-g.Ascent.Floor())
			bounds.Max.X = max(bounds.Max.X, (g.X + g.Advance).Ceil())
			bounds.Max.Y = max(bounds.Max.Y, int(g.Y)+g.Descent.Ceil())

			glyphs = append(glyphs, g)
			visible = true
		}

		if !visible || len(glyphs) == 0 {
			continue
		}

		dims.Size = image.Point{
			X: max(bounds.Dx(), dims.Size.X),
			Y: dims.Size.Y + bounds.Dy(),
		}

		// Transform to the correct position
		yPos := float32(para.StartY - ctx.Viewport.Min.Y)
		trans := op.Affine(f32.Affine2D{}.Offset(
			f32.Point{X: float32(glyphs[0].X.Floor()), Y: yPos},
		)).Push(gtx.Ops)

		// Draw the glyph
		path := ctx.Shaper.Shape(glyphs)
		outline := clip.Outline{Path: path}.Op().Push(gtx.Ops)

		// Use highlight color for current line
		if ctx.CurrentLine == para.Index {
			highlightMaterial.Add(gtx.Ops)
		} else {
			textMaterial.Add(gtx.Ops)
		}

		paint.PaintOp{}.Add(gtx.Ops)
		outline.Pop()
		trans.Pop()
	}

	return dims
}

// createColorOp creates a paint operation for the given color.
func (p *LineNumberProvider) createColorOp(ops *op.Ops, c gvcolor.Color) op.CallOp {
	m := op.Record(ops)
	paint.ColorOp{Color: c.NRGBA()}.Add(ops)
	return m.Stop()
}

// HighlightedLines returns the current line to be highlighted.
// This implements the gutter.LineHighlighter interface.
func (p *LineNumberProvider) HighlightedLines() []gutter.LineHighlight {
	if !p.hasCurrentLine || !p.lineHighlightColor.IsSet() {
		return nil
	}

	return []gutter.LineHighlight{
		{
			Line:  p.currentLine,
			Color: p.lineHighlightColor,
		},
	}
}
