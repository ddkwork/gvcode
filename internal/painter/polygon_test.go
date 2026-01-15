package painter

import (
	"image"
	"testing"

	"gioui.org/f32"
)

func TestIsRightAngle(t *testing.T) {
	tests := []struct {
		name     string
		p1, p2, p3 f32.Point
		want     bool
	}{
		{
			name: "90 degree angle",
			p1:   f32.Point{X: 0, Y: 0},
			p2:   f32.Point{X: 0, Y: 1},
			p3:   f32.Point{X: 1, Y: 1},
			want: true, // L shape: up then right
		},
		{
			name: "180 degree (straight)",
			p1:   f32.Point{X: 0, Y: 0},
			p2:   f32.Point{X: 0, Y: 1},
			p3:   f32.Point{X: 0, Y: 2},
			want: false, // Straight line
		},
		{
			name: "45 degree angle",
			p1:   f32.Point{X: 0, Y: 0},
			p2:   f32.Point{X: 0, Y: 1},
			p3:   f32.Point{X: 1, Y: 2},
			want: false, // Diagonal
		},
		{
			name: "zero length vectors",
			p1:   f32.Point{X: 0, Y: 0},
			p2:   f32.Point{X: 0, Y: 0},
			p3:   f32.Point{X: 1, Y: 1},
			want: false, // Degenerate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRightAngle(tt.p1, tt.p2, tt.p3)
			if got != tt.want {
				t.Errorf("isRightAngle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPolygonPoints(t *testing.T) {
	tests := []struct {
		name         string
		rects        []image.Rectangle
		expandEmpty  bool
		minWidth     int
		wantPoints   []f32.Point
	}{
		{
			name:        "empty",
			rects:       nil,
			expandEmpty: false,
			minWidth:    0,
			wantPoints:  nil,
		},
		{
			name: "single rectangle",
			rects: []image.Rectangle{
				{Min: image.Pt(10, 20), Max: image.Pt(50, 40)},
			},
			expandEmpty: false,
			minWidth:    0,
			wantPoints: []f32.Point{
				{X: 50, Y: 20}, // top-right
				{X: 50, Y: 40}, // bottom-right
				{X: 10, Y: 40}, // bottom-left
				{X: 10, Y: 20}, // top-left
			},
		},
		{
			name: "two rectangles stacked without gap",
			rects: []image.Rectangle{
				{Min: image.Pt(10, 20), Max: image.Pt(50, 40)},
				{Min: image.Pt(10, 40), Max: image.Pt(50, 60)},
			},
			expandEmpty: false,
			minWidth:    0,
			wantPoints: []f32.Point{
				// right edge top to bottom
				{X: 50, Y: 20}, // rect0 top-right
				{X: 50, Y: 40}, // rect0 bottom-right
				{X: 50, Y: 40}, // rect1 top-right (same point)
				{X: 50, Y: 60}, // rect1 bottom-right
				// left edge bottom to top
				{X: 10, Y: 60}, // rect1 bottom-left
				{X: 10, Y: 40}, // rect1 top-left
				{X: 10, Y: 40}, // rect0 bottom-left
				{X: 10, Y: 20}, // rect0 top-left
			},
		},
		{
			name: "two rectangles with gap filled",
			rects: []image.Rectangle{
				{Min: image.Pt(10, 20), Max: image.Pt(50, 40)},
				{Min: image.Pt(10, 50), Max: image.Pt(50, 70)}, // gap of 10 pixels
			},
			expandEmpty: false,
			minWidth:    0,
			wantPoints: []f32.Point{
				// right edge top to bottom
				{X: 50, Y: 20}, // rect0 top-right
				{X: 50, Y: 40}, // rect0 bottom-right
				{X: 50, Y: 50}, // rect1 top-right
				{X: 50, Y: 70}, // rect1 bottom-right
				// left edge bottom to top
				{X: 10, Y: 70}, // rect1 bottom-left
				{X: 10, Y: 50}, // rect1 top-left
				{X: 10, Y: 40}, // rect0 bottom-left
				{X: 10, Y: 20}, // rect0 top-left
			},
		},
		{
			name: "two rectangles with different x positions",
			rects: []image.Rectangle{
				{Min: image.Pt(10, 20), Max: image.Pt(30, 40)},
				{Min: image.Pt(40, 40), Max: image.Pt(70, 60)}, // shifted right
			},
			expandEmpty: false,
			minWidth:    0,
			wantPoints: []f32.Point{
				// right edge top to bottom
				{X: 30, Y: 20}, // rect0 top-right
				{X: 30, Y: 40}, // rect0 bottom-right
				{X: 70, Y: 40}, // rect1 top-right
				{X: 70, Y: 60}, // rect1 bottom-right
				// left edge bottom to top
				{X: 40, Y: 60}, // rect1 bottom-left
				{X: 40, Y: 40}, // rect1 top-left
				{X: 10, Y: 40}, // rect0 bottom-left
				{X: 10, Y: 20}, // rect0 top-left
			},
		},
		{
			name: "zero-width rectangle with expandEmpty false",
			rects: []image.Rectangle{
				{Min: image.Pt(10, 20), Max: image.Pt(10, 40)},
			},
			expandEmpty: false,
			minWidth:    5,
			wantPoints: []f32.Point{
				{X: 10, Y: 20}, // zero width stays zero
				{X: 10, Y: 40},
				{X: 10, Y: 40},
				{X: 10, Y: 20},
			},
		},
		{
			name: "zero-width rectangle with expandEmpty true",
			rects: []image.Rectangle{
				{Min: image.Pt(10, 20), Max: image.Pt(10, 40)},
			},
			expandEmpty: true,
			minWidth:    5,
			wantPoints: []f32.Point{
				{X: 10, Y: 20}, // zero width (expansion ignored by polygonPointsForGroup)
				{X: 10, Y: 40},
				{X: 10, Y: 40},
				{X: 10, Y: 20},
			},
		},
		{
			name: "multiple zero-width rectangles with expandEmpty true",
			rects: []image.Rectangle{
				{Min: image.Pt(10, 20), Max: image.Pt(10, 40)},
				{Min: image.Pt(10, 40), Max: image.Pt(10, 60)},
			},
			expandEmpty: true,
			minWidth:    5,
			wantPoints: []f32.Point{
				// right edge (expansion ignored by polygonPointsForGroup)
				{X: 10, Y: 20},
				{X: 10, Y: 40},
				{X: 10, Y: 40},
				{X: 10, Y: 60},
				// left edge
				{X: 10, Y: 60},
				{X: 10, Y: 40},
				{X: 10, Y: 40},
				{X: 10, Y: 20},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := polygonPointsForGroup(tt.rects)
			if len(got) != len(tt.wantPoints) {
				t.Errorf("selectionPolygonPoints() returned %d points, want %d", len(got), len(tt.wantPoints))
				return
			}
			for i := range got {
				if got[i] != tt.wantPoints[i] {
					t.Errorf("point[%d] = %v, want %v", i, got[i], tt.wantPoints[i])
				}
			}
		})
	}
}

func TestCornersToRound(t *testing.T) {
	tests := []struct {
		name       string
		rects      []image.Rectangle
		radius     float32
		wantRounds []bool // expected rounded corners for each vertex, in order of cleaned points
	}{
		{
			name: "single rectangle, small radius",
			rects: []image.Rectangle{
				{Min: image.Pt(10, 20), Max: image.Pt(50, 40)},
			},
			radius:     2.0,
			wantRounds: []bool{true, true, true, true}, // all four corners rounded
		},
		{
			name: "single rectangle, radius larger than edges",
			rects: []image.Rectangle{
				{Min: image.Pt(10, 20), Max: image.Pt(15, 25)}, // small 5x5 rectangle
			},
			radius:     10.0,
			wantRounds: []bool{false, false, false, false}, // edges too short
		},
		{
			name: "two rectangles stacked, small radius",
			rects: []image.Rectangle{
				{Min: image.Pt(10, 20), Max: image.Pt(50, 40)},
				{Min: image.Pt(10, 40), Max: image.Pt(50, 60)},
			},
			radius:     2.0,
			// After duplicate removal, points: top-right0, bottom-right0, bottom-right1, bottom-left1, top-left1, top-left0
			// Corners: bottom-right0 (interior straight), bottom-right1 (right angle), bottom-left1 (right angle), top-left1 (interior straight), top-left0 (right angle), top-right0 (right angle)
			// Only external right-angle corners rounded
			wantRounds: []bool{true, false, true, true, false, true}, // top-right0, bottom-right1, bottom-left1, top-left0
		},
		{
			name: "rectangle at Y=0 (first line)",
			rects: []image.Rectangle{
				{Min: image.Pt(10, 0), Max: image.Pt(50, 20)}, // Y starts at 0
			},
			radius:     4.0,
			wantRounds: []bool{true, true, true, true}, // all corners should be rounded
		},
		{
			name: "rectangle at X=0,Y=0 (first line, left edge)",
			rects: []image.Rectangle{
				{Min: image.Pt(0, 0), Max: image.Pt(50, 20)}, // X and Y start at 0
			},
			radius:     4.0,
			wantRounds: []bool{true, true, true, true}, // all corners should be rounded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			points := polygonPointsForGroup(tt.rects)
			got := cornersToRound(points, tt.radius)
			if len(got) != len(tt.wantRounds) {
				t.Errorf("cornersToRound() returned %d corners, want %d", len(got), len(tt.wantRounds))
				t.Logf("points: %v", points)
				return
			}
			for i := range got {
				if got[i] != tt.wantRounds[i] {
					t.Errorf("corner[%d] rounded = %v, want %v", i, got[i], tt.wantRounds[i])
					t.Logf("points: %v", points)
					break
				}
			}
		})
	}
}
func TestPolygonGroupsForRects(t *testing.T) {
	tests := []struct {
		name         string
		rects        []image.Rectangle
		expandEmpty  bool
		minWidth     int
		wantGroupCount int // expected number of groups
	}{
		{
			name: "empty line between text lines",
			rects: []image.Rectangle{
				{Min: image.Pt(20, 0), Max: image.Pt(100, 20)},   // text line 1
				{Min: image.Pt(0, 20), Max: image.Pt(0, 40)},     // empty line (zero width)
				{Min: image.Pt(40, 40), Max: image.Pt(120, 60)},  // text line 2 (different X)
			},
			expandEmpty: true,
			minWidth:    6,
			wantGroupCount: 3, // each rectangle forms separate group due to no horizontal overlap
		},
		{
			name: "disconnected rectangles",
			rects: []image.Rectangle{
				{Min: image.Pt(10, 0), Max: image.Pt(50, 20)},
				{Min: image.Pt(100, 50), Max: image.Pt(150, 70)}, // far away, not connected
			},
			expandEmpty: false,
			minWidth:    0,
			wantGroupCount: 2, // should be two separate polygons
		},
		{
			name: "stacked rectangles with same X",
			rects: []image.Rectangle{
				{Min: image.Pt(10, 0), Max: image.Pt(50, 20)},
				{Min: image.Pt(10, 20), Max: image.Pt(50, 40)},
				{Min: image.Pt(10, 40), Max: image.Pt(50, 60)},
			},
			expandEmpty: false,
			minWidth:    0,
			wantGroupCount: 1, // all overlap horizontally, one polygon
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := PolygonGroupsForRects(tt.rects, tt.expandEmpty, tt.minWidth)
			if len(groups) != tt.wantGroupCount {
				t.Errorf("PolygonGroupsForRects() returned %d groups, want %d", len(groups), tt.wantGroupCount)
				t.Logf("groups: %v", groups)
			}
		})
	}
}
