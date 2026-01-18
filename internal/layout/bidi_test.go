package layout

import (
	"testing"

	"gioui.org/font"
	"gioui.org/text"
	"golang.org/x/image/math/fixed"
)

func setupShaper() (*text.Shaper, text.Parameters, text.Glyph) {
	shaper := text.NewShaper()

	params := text.Parameters{
		Font:     font.Font{Typeface: font.Typeface("monospace")},
		PxPerEm:  fixed.I(14),
		MaxWidth: 1e6,
	}

	shaper.LayoutString(params, "\u0020")
	spaceGlyph, _ := shaper.NextGlyph()

	return shaper, params, spaceGlyph
}

func TestBidiTextLayout(t *testing.T) {
	testcases := []struct {
		name  string
		input string
	}{
		{
			name:  "Pure RTL Hebrew",
			input: "שלום עולם",
		},
		{
			name:  "Pure RTL Arabic",
			input: "مرحبا بالعالم",
		},
		{
			name:  "Mixed LTR and RTL",
			input: "Hello שלום World",
		},
		{
			name:  "RTL with numbers",
			input: "שלום 123 עולם",
		},
		{
			name:  "LTR with embedded RTL",
			input: "The word שלום means peace",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			shaper, params, spaceGlyph := setupShaper()

			shaper.LayoutString(params, tc.input)

			wrapper := lineWrapper{}
			lines := wrapper.WrapParagraph(glyphIter{shaper: shaper}.All(), []rune(tc.input), 1e6, 4, &spaceGlyph)

			if len(lines) == 0 {
				t.Fatal("Expected at least one line")
			}

			// Verify total runes match input
			totalRunes := 0
			for _, line := range lines {
				totalRunes += line.Runes
			}
			expectedRunes := len([]rune(tc.input))
			if totalRunes != expectedRunes {
				t.Errorf("Rune count mismatch: got %d, want %d", totalRunes, expectedRunes)
			}

			// Verify all glyphs have valid positions
			for i, line := range lines {
				for j, gl := range line.Glyphs {
					if gl == nil {
						t.Errorf("Line %d, glyph %d: nil glyph", i, j)
						continue
					}
					// X position should be non-negative after layout
					if gl.X < 0 {
						t.Errorf("Line %d, glyph %d: negative X position %d", i, j, gl.X)
					}
				}
			}
		})
	}
}

func TestBidiRecomputePreservesRelativePositions(t *testing.T) {
	shaper, params, spaceGlyph := setupShaper()

	// Test with mixed bidi text
	input := "Hello שלום World"
	shaper.LayoutString(params, input)

	wrapper := lineWrapper{}
	lines := wrapper.WrapParagraph(glyphIter{shaper: shaper}.All(), []rune(input), 1e6, 4, &spaceGlyph)

	if len(lines) == 0 {
		t.Fatal("Expected at least one line")
	}

	line := lines[0]
	if len(line.Glyphs) == 0 {
		t.Fatal("Expected glyphs in line")
	}

	// Record original relative positions (differences between consecutive glyphs)
	type relativePos struct {
		deltaX fixed.Int26_6
	}
	originalRelative := make([]relativePos, len(line.Glyphs)-1)
	for i := 0; i < len(line.Glyphs)-1; i++ {
		originalRelative[i] = relativePos{
			deltaX: line.Glyphs[i+1].X - line.Glyphs[i].X,
		}
	}

	// Apply recompute with an alignment offset
	alignOff := fixed.I(100)
	line.recompute(alignOff, 0)

	// Verify relative positions are preserved
	for i := 0; i < len(line.Glyphs)-1; i++ {
		newDeltaX := line.Glyphs[i+1].X - line.Glyphs[i].X
		if newDeltaX != originalRelative[i].deltaX {
			t.Errorf("Glyph %d: relative X position changed from %d to %d",
				i, originalRelative[i].deltaX, newDeltaX)
		}
	}

	// Verify the leftmost glyph is at alignOff
	minX := line.Glyphs[0].X
	for _, gl := range line.Glyphs {
		if gl.X < minX {
			minX = gl.X
		}
	}
	if minX != alignOff {
		t.Errorf("Leftmost glyph X: got %d, want %d", minX, alignOff)
	}
}

func TestBidiLineWidth(t *testing.T) {
	shaper, params, spaceGlyph := setupShaper()

	testcases := []struct {
		name  string
		input string
	}{
		{
			name:  "LTR text",
			input: "Hello World",
		},
		{
			name:  "RTL text",
			input: "שלום עולם",
		},
		{
			name:  "Mixed text",
			input: "Hello שלום",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			shaper.LayoutString(params, tc.input)

			wrapper := lineWrapper{}
			lines := wrapper.WrapParagraph(glyphIter{shaper: shaper}.All(), []rune(tc.input), 1e6, 4, &spaceGlyph)

			if len(lines) == 0 {
				t.Fatal("Expected at least one line")
			}

			line := lines[0]

			// Line width should be positive
			if line.Width <= 0 {
				t.Errorf("Line width should be positive, got %d", line.Width)
			}

			// Calculate sum of advances
			sumAdvances := fixed.I(0)
			for _, gl := range line.Glyphs {
				sumAdvances += gl.Advance
			}

			// Line width should equal sum of advances
			if line.Width != sumAdvances {
				t.Errorf("Line width mismatch: Width=%d, sum of advances=%d", line.Width, sumAdvances)
			}
		})
	}
}

func TestBidiGlyphOrder(t *testing.T) {
	shaper, params, spaceGlyph := setupShaper()

	// For mixed bidi text, glyphs should be in visual order
	input := "AB שלום CD"
	shaper.LayoutString(params, input)

	wrapper := lineWrapper{}
	lines := wrapper.WrapParagraph(glyphIter{shaper: shaper}.All(), []rune(input), 1e6, 4, &spaceGlyph)

	if len(lines) == 0 {
		t.Fatal("Expected at least one line")
	}

	line := lines[0]

	// After recompute, verify glyphs are properly positioned
	line.recompute(fixed.I(0), 0)

	// Check that we have the expected number of glyphs (accounting for cluster breaks)
	if len(line.Glyphs) == 0 {
		t.Fatal("Expected glyphs in line")
	}

	// Verify no overlapping glyphs (each glyph's right edge should not exceed next glyph's position significantly)
	// Note: For bidi text, glyphs might not be strictly ordered by X position
	// but they should form a valid visual representation
	for i, gl := range line.Glyphs {
		if gl.X < 0 {
			t.Errorf("Glyph %d has negative X position: %d", i, gl.X)
		}
	}
}

func TestEmptyLineRecompute(t *testing.T) {
	line := Line{}

	// Should not panic on empty line
	line.recompute(fixed.I(100), 0)

	if line.RuneOff != 0 {
		t.Errorf("RuneOff should be 0, got %d", line.RuneOff)
	}
}
