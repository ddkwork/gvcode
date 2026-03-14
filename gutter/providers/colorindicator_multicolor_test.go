package providers

import (
	"testing"
)

func TestMultipleColorsOnSameLineRendering(t *testing.T) {
	provider := NewColorIndicatorProvider()

	// Set up a line with multiple colors
	lines := []string{
		`var colors = "#FF0000 #00FF00 #0000FF"`,
	}
	provider.SetLineContents(lines, 0)

	// Verify that we detected all 3 colors
	colors, hasColors := provider.colorInfos[0]
	if !hasColors {
		t.Fatal("Expected to find colors on line 0")
	}

	if len(colors) != 3 {
		t.Fatalf("Expected 3 colors, got %d", len(colors))
	}

	// Verify color positions
	expectedColors := []string{"#FF0000", "#00FF00", "#0000FF"}
	for i, expectedColor := range expectedColors {
		if colors[i].Original != expectedColor {
			t.Errorf("Color %d: expected %q, got %q", i, expectedColor, colors[i].Original)
		}
	}

	// Test GetColorOffsets method
	offsets := provider.GetColorOffsets()
	if len(offsets) == 0 {
		t.Fatal("Expected to find color offsets")
	}

	line0Offsets, ok := offsets[0]
	if !ok {
		t.Fatal("Expected to find color offsets on line 0")
	}

	if len(line0Offsets) != 3 {
		t.Fatalf("Expected 3 offsets on line 0, got %d", len(line0Offsets))
	}

	// Verify that offsets are in increasing order
	for i := 1; i < len(line0Offsets); i++ {
		if line0Offsets[i] <= line0Offsets[i-1] {
			t.Errorf("Offset %d should be greater than offset %d", i, i-1)
		}
	}

	t.Logf("Successfully detected %d colors on line 0", len(colors))
	for i, color := range colors {
		t.Logf("Color %d: %q at position %d", i, color.Original, color.Range.Start)
	}
	for i, offset := range line0Offsets {
		t.Logf("Offset %d: position %d", i, offset)
	}
}
