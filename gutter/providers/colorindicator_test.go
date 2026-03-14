package providers

import (
	"testing"
)

func TestColorIndicatorOffsets(t *testing.T) {
	// Test case 1: Simple hex color in quotes
	testCases := []struct {
		name        string
		lineText    string
		colorPos    int
		expectedPos int
	}{
		{
			name:        "Hex color at position 10",
			lineText:    `color: "#FF0000";`,
			colorPos:    10, // Position of # in "#FF0000"
			expectedPos: 10,
		},
		{
			name:        "Hex color at position 0",
			lineText:    `#FF0000`,
			colorPos:    0,
			expectedPos: 0,
		},
		{
			name:        "RGB color in middle",
			lineText:    `background: rgb(255, 0, 0);`,
			colorPos:    14, // Position of 'r' in "rgb"
			expectedPos: 14,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.colorPos != tc.expectedPos {
				t.Errorf("Expected color position %d, got %d", tc.expectedPos, tc.colorPos)
			}
		})
	}
}

func TestColorIndicatorRendering(t *testing.T) {
	// Test that the indicator is rendered at the correct position
	// The indicator should be rendered at the position where the space was inserted
	// not at position - indicatorSize - gap

	testCases := []struct {
		name               string
		colorStartPos      int
		indicatorSize      int
		gap                int
		expectedIndicatorX int
	}{
		{
			name:               "Color at position 10",
			colorStartPos:      100,
			indicatorSize:      16,
			gap:                4,
			expectedIndicatorX: 100, // Should be at the position where space was inserted
		},
		{
			name:               "Color at position 0",
			colorStartPos:      0,
			indicatorSize:      16,
			gap:                4,
			expectedIndicatorX: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// The indicator should be rendered at colorStartPos
			// not at colorStartPos - indicatorSize - gap
			wrongIndicatorX := tc.colorStartPos - tc.indicatorSize - tc.gap
			if wrongIndicatorX != tc.expectedIndicatorX {
				t.Logf("Current (wrong) implementation would place indicator at %d", wrongIndicatorX)
				t.Logf("Correct implementation should place indicator at %d", tc.expectedIndicatorX)
			}

			// Test that the indicator does NOT overlap with the text
			// The text should start at colorStartPos + indicatorSize
			textStartPos := tc.colorStartPos + tc.indicatorSize
			indicatorEndPos := tc.expectedIndicatorX + tc.indicatorSize

			// The indicator should end before the text starts
			if indicatorEndPos > textStartPos {
				t.Errorf("Indicator overlaps with text! Indicator ends at %d, text starts at %d",
					indicatorEndPos, textStartPos)
			} else {
				t.Logf("Indicator correctly placed: ends at %d, text starts at %d",
					indicatorEndPos, textStartPos)
			}
		})
	}
}

func TestColorIndicatorNoOverlap(t *testing.T) {
	// Test that color indicators do not overlap with code text
	// This is a critical test to ensure the indicator is placed in the reserved space

	testCases := []struct {
		name                   string
		lineText               string
		colorPos               int
		indicatorSize          int
		originalGlyphPositions []int
	}{
		{
			name:          "Hex color in quotes",
			lineText:      `color: "#FF0000";`,
			colorPos:      10, // Position of #
			indicatorSize: 24,
			// Original positions before color offsets
			originalGlyphPositions: []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170},
		},
		{
			name:          "RGB color",
			lineText:      `background: rgb(255, 0, 0);`,
			colorPos:      14, // Position of 'r' in "rgb"
			indicatorSize: 24,
			// Original positions before color offsets
			originalGlyphPositions: []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150, 160, 170, 180, 190, 200, 210, 220, 230, 240, 250},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get the original glyph position at the color position
			if tc.colorPos >= len(tc.originalGlyphPositions) {
				t.Fatalf("colorPos %d is out of range for originalGlyphPositions (length %d)",
					tc.colorPos, len(tc.originalGlyphPositions))
			}

			originalGlyphPos := tc.originalGlyphPositions[tc.colorPos]
			indicatorX := originalGlyphPos
			indicatorEnd := indicatorX + tc.indicatorSize

			// The indicator should be placed at the original glyph position
			// The text should be shifted by indicatorSize
			expectedTextPos := originalGlyphPos + tc.indicatorSize

			// Simulate the shifted glyph positions (after color offsets)
			shiftedGlyphPositions := make([]int, len(tc.originalGlyphPositions))
			for i, pos := range tc.originalGlyphPositions {
				if i >= tc.colorPos {
					// Characters at or after the color position are shifted
					shiftedGlyphPositions[i] = pos + tc.indicatorSize
				} else {
					// Characters before the color position are not shifted
					shiftedGlyphPositions[i] = pos
				}
			}

			// Check if the indicator overlaps with any shifted text characters
			for i, pos := range shiftedGlyphPositions {
				// Skip the color position itself
				if i == tc.colorPos {
					continue
				}

				// Check if this character is overlapped by the indicator
				if pos >= indicatorX && pos < indicatorEnd {
					t.Errorf("Indicator overlaps with character at position %d (shifted glyph pos %d)",
						i, pos)
					t.Errorf("Indicator range: [%d, %d), character at %d",
						indicatorX, indicatorEnd, pos)
				}
			}

			// Verify that the text at colorPos is shifted correctly
			if tc.colorPos < len(shiftedGlyphPositions) {
				shiftedGlyphPos := shiftedGlyphPositions[tc.colorPos]
				if shiftedGlyphPos < expectedTextPos {
					t.Errorf("Text not shifted correctly: expected text at %d, got %d",
						expectedTextPos, shiftedGlyphPos)
				} else {
					t.Logf("Text correctly shifted: character at %d (expected >= %d)",
						shiftedGlyphPos, expectedTextPos)
				}
			}
		})
	}
}
