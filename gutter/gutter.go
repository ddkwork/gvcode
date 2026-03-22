package gutter

import (
	"image"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	gvcolor "github.com/oligo/gvcode/color"
	"golang.org/x/image/math/fixed"
)

const (
	// LineNumberProviderID is the unique identifier for the line number provider.
	LineNumberProviderID = "linenumber"
)

// GutterProvider defines the interface for components that render content
// in the gutter area of the editor. Providers are rendered left-to-right
// sorted by their priority (lower priority = closer to text).
type GutterProvider interface {
	// ID returns a unique identifier for this provider.
	ID() string

	// Priority determines the rendering order. Lower values are rendered
	// closer to the text area (rightmost in the gutter).
	Priority() int

	// Width returns the width needed by this provider for the given context.
	// The lineCount parameter indicates the total number of lines in the document,
	// which can be used to calculate width (e.g., for line numbers).
	Width(gtx layout.Context, shaper *text.Shaper, params text.Parameters, lineCount int) unit.Dp

	// Layout renders the gutter content for the visible lines.
	Layout(gtx layout.Context, ctx GutterContext) layout.Dimensions
}

// LineContentProvider is an optional interface that GutterProviders can implement
// to receive the contents of visible lines for analysis.
type LineContentProvider interface {
	GutterProvider
	// SetLineContents sets the contents of all visible lines for analysis.
	// The startLine parameter indicates the absolute line number of the first line in the slice.
	SetLineContents(lines []string, startLine int)
}

// GutterContext provides the context needed for gutter providers to render
// their content. It includes information about the visible area, line metadata,
// and colors.
type GutterContext struct {
	// Shaper is the text shaper used for rendering text.
	Shaper *text.Shaper

	// TextParams contains the text parameters for shaping.
	TextParams text.Parameters

	// Viewport is the visible area in document coordinates.
	Viewport image.Rectangle

	// Paragraphs contains metadata for visible lines.
	Paragraphs []Paragraph

	// CurrentLine is the line number where the caret is located.
	// It is -1 if the selection spans multiple lines.
	CurrentLine int

	// LineHeight is the calculated line height in fixed-point format.
	LineHeight fixed.Int26_6

	// Colors provides the color scheme for gutter rendering.
	Colors *GutterColors

	// LineNumberWidth is the width of the line number column in pixels.
	// This is set by the gutter manager when a line number provider is present.
	LineNumberWidth int
}

// Paragraph contains metadata about a paragraph (logical line) in the document.
type Paragraph struct {
	// StartY is the baseline Y coordinate of the first screen line in the paragraph.
	StartY int

	// EndY is the baseline Y coordinate of the last screen line in the paragraph.
	EndY int

	// Ascent is the distance from baseline to the top of glyphs.
	Ascent fixed.Int26_6

	// Descent is the distance from baseline to the bottom of glyphs.
	Descent fixed.Int26_6

	// Runes is the number of runes in this paragraph.
	Runes int

	// RuneOff is the rune offset of the first rune in this paragraph.
	RuneOff int

	// Index is the 0-based line number of this paragraph.
	Index int
}

// GutterColors defines the color scheme for gutter rendering.
type GutterColors struct {
	// Text is the default text color for gutter content.
	Text gvcolor.Color

	// TextHighlight is the text color for highlighted content (e.g., current line number).
	TextHighlight gvcolor.Color

	// Background is the background color for the gutter area.
	Background gvcolor.Color

	// LineHighlight is the color used to highlight the current line background.
	LineHighlight gvcolor.Color

	// Custom contains provider-specific colors, keyed by provider ID or custom name.
	Custom map[string]gvcolor.Color
}

// LineHighlighter is an optional interface that GutterProviders can implement
// to specify lines that should be highlighted with a background color.
// The Editor will paint these highlights spanning the full editor width.
type LineHighlighter interface {
	// HighlightedLines returns the lines that should be highlighted.
	// This is called after Layout to collect highlights from all providers.
	HighlightedLines() []LineHighlight
}

// LineHighlight specifies a line to be highlighted with a background color.
type LineHighlight struct {
	// Line is the 0-based line index to highlight.
	Line int

	// Color is the background color for the highlight.
	Color gvcolor.Color
}

// RunButtonEvent represents a click event on a run button in the gutter.
type RunButtonEvent struct {
	// ButtonType is the type of button that was clicked.
	ButtonType int

	// Line is the 0-based line number where the button was clicked.
	Line int

	// ButtonText is the text content of the line containing the button.
	ButtonText string
}

// RunButtonType constants
const (
	RunButtonNone = iota
	RunButtonMain
	RunButtonTest
)
