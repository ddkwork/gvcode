package painter

import (
	"image"
	"math"
	"sort"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
)

// PolygonBuilder detects how many polygons can be formed from a group of rectangles, then
// extarcts vertices for each of them. To draw the polygons in Gio, call Path or Paths to
// build the clip paths.
type PolygonBuilder struct {
	// Apply minimum width to zero-width rectangles if set to true.
	expandEmpty bool
	minWidth    int
	// radius is the corner radius in pixels.
	radius float32
	// polygons holds points for the detected polygons
	polygons [][]f32.Point
}

func NewPolygonBuilder(expandEmpty bool, minWidth int, radius float32) *PolygonBuilder {
	return &PolygonBuilder{
		expandEmpty: expandEmpty,
		minWidth:    minWidth,
		radius:      radius,
	}
}

// Group groups rectangles by horizontal overlap and generates
// polygon points for each group. Returns a slice of point slices, one per group.
func (pb *PolygonBuilder) Group(rects []image.Rectangle) [][]f32.Point {
	if len(rects) == 0 {
		return nil
	}

	// Apply minimum width to zero-width rectangles
	if pb.expandEmpty {
		for i := range rects {
			if rects[i].Dx() <= 0 {
				rects[i].Max.X += pb.minWidth
			}
		}
	}

	// Sort rectangles by top Y coordinate
	sort.Slice(rects, func(i, j int) bool {
		return rects[i].Min.Y < rects[j].Min.Y
	})

	// Group rectangles by horizontal overlap
	var groups [][]image.Rectangle
	var currentGroup []image.Rectangle

	for _, rect := range rects {
		if len(currentGroup) == 0 {
			// Start first group
			currentGroup = append(currentGroup, rect)
			continue
		}

		// Check if rect overlaps horizontally with any rect in current group
		overlaps := false
		for _, groupRect := range currentGroup {
			// Check horizontal overlap
			if rect.Min.X < groupRect.Max.X && groupRect.Min.X < rect.Max.X {
				overlaps = true
				break
			}
		}

		if overlaps {
			currentGroup = append(currentGroup, rect)
		} else {
			// Start new group
			groups = append(groups, currentGroup)
			currentGroup = []image.Rectangle{rect}
		}
	}

	// Add last group
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	pb.polygons = pb.polygons[:0]
	// For each group, generate polygon points
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}

		// Ensure vertical continuity within the group
		for i := 0; i < len(group)-1; i++ {
			if group[i].Max.Y < group[i+1].Min.Y {
				group[i].Max.Y = group[i+1].Min.Y
			}
		}

		// Generate points for this group using the original logic
		points := polygonPointsForGroup(group)
		if len(points) > 0 {
			pb.polygons = append(pb.polygons, points)
		}
	}

	return pb.polygons
}

// polygonPointsForGroup generates points for a single group of rectangles
func polygonPointsForGroup(rects []image.Rectangle) []f32.Point {
	if len(rects) == 0 {
		return nil
	}

	// Build polygon points following the "staircase" outline.
	// Right edge from top to bottom, then left edge from bottom to top
	var points []f32.Point

	// Right edge: top-right of first rectangle to bottom-right of last
	for _, rect := range rects {
		points = append(points, f32.Point{X: float32(rect.Max.X), Y: float32(rect.Min.Y)})
		points = append(points, f32.Point{X: float32(rect.Max.X), Y: float32(rect.Max.Y)})
	}

	// Left edge: bottom-left of last rectangle to top-left of first (reverse order)
	for i := len(rects) - 1; i >= 0; i-- {
		rect := rects[i]
		points = append(points, f32.Point{X: float32(rect.Min.X), Y: float32(rect.Max.Y)})
		points = append(points, f32.Point{X: float32(rect.Min.X), Y: float32(rect.Min.Y)})
	}

	return points
}

// Path creates a path with rounded corners from polygon points.
func (pb *PolygonBuilder) Path(gtx layout.Context, points []f32.Point) clip.PathSpec {
	if len(points) < 3 {
		return clip.PathSpec{}
	}

	// Determine which corners should be rounded (using original points)
	roundedCorners := cornersToRound(points, pb.radius)

	// Remove duplicate consecutive points
	cleanPoints := make([]f32.Point, 0, len(points))
	for i, pt := range points {
		if i == 0 || pt != points[i-1] {
			cleanPoints = append(cleanPoints, pt)
		}
	}
	points = cleanPoints

	if len(points) < 3 {
		return clip.PathSpec{}
	}

	// Remove duplicate closing point if present (first == last)
	if len(points) > 0 && points[len(points)-1] == points[0] {
		points = points[:len(points)-1]
	}

	if len(points) < 3 {
		return clip.PathSpec{}
	}

	if len(roundedCorners) != len(points) {
		// This shouldn't happen, but fall back to original logic
		roundedCorners = nil
	}

	path := clip.Path{}
	path.Begin(gtx.Ops)

	// Start at first point
	path.MoveTo(points[0])

	// Process each edge and corner
	for i := 0; i < len(points); i++ {
		p1 := points[i]
		p2 := points[(i+1)%len(points)]
		p3 := points[(i+2)%len(points)]

		// Calculate vectors for the corner at p2
		v1 := f32.Point{X: p2.X - p1.X, Y: p2.Y - p1.Y}
		v2 := f32.Point{X: p3.X - p2.X, Y: p3.Y - p2.Y}

		// Normalize vectors
		len1 := float32(math.Sqrt(float64(v1.X*v1.X + v1.Y*v1.Y)))
		len2 := float32(math.Sqrt(float64(v2.X*v2.X + v2.Y*v2.Y)))

		if len1 <= 0 || len2 <= 0 {
			// Degenerate case, draw straight to p2
			path.LineTo(p2)
			continue
		}

		v1n := f32.Point{X: v1.X / len1, Y: v1.Y / len1}
		v2n := f32.Point{X: v2.X / len2, Y: v2.Y / len2}

		// Determine if this corner should be rounded
		var canRound bool
		if roundedCorners != nil {
			cornerIdx := (i + 1) % len(points)
			canRound = roundedCorners[cornerIdx]
		} else {
			// Fallback to original logic
			isRightAngle := isRightAngle(p1, p2, p3)
			canRound = isRightAngle && len1 > pb.radius && len2 > pb.radius
		}

		if canRound {
			// Calculate points where rounded corner starts and ends
			cornerStart := f32.Point{
				X: p2.X - v1n.X*pb.radius,
				Y: p2.Y - v1n.Y*pb.radius,
			}
			cornerEnd := f32.Point{
				X: p2.X + v2n.X*pb.radius,
				Y: p2.Y + v2n.Y*pb.radius,
			}

			// Draw line to where rounded corner starts
			path.LineTo(cornerStart)
			// Draw rounded corner with quadratic Bézier
			path.QuadTo(p2, cornerEnd)
		} else {
			// Draw line to the corner point
			path.LineTo(p2)
		}
	}

	// Close the path (should already be at start point)
	path.Close()

	return path.End()
}

func (pb *PolygonBuilder) Paths(gtx layout.Context) []clip.PathSpec {
	paths := make([]clip.PathSpec, 0, len(pb.polygons))
	for _, points := range pb.polygons {
		if len(points) >= 3 {
			path := pb.Path(gtx, points)
			paths = append(paths, path)
		}
	}

	return paths
}

// isRightAngle checks if three points form approximately a right angle.
// Returns true if the angle at p2 is close to 90 degrees.
func isRightAngle(p1, p2, p3 f32.Point) bool {
	v1 := f32.Point{X: p2.X - p1.X, Y: p2.Y - p1.Y}
	v2 := f32.Point{X: p3.X - p2.X, Y: p3.Y - p2.Y}

	len1 := float32(math.Sqrt(float64(v1.X*v1.X + v1.Y*v1.Y)))
	len2 := float32(math.Sqrt(float64(v2.X*v2.X + v2.Y*v2.Y)))

	if len1 <= 0 || len2 <= 0 {
		return false
	}

	v1n := f32.Point{X: v1.X / len1, Y: v1.Y / len1}
	v2n := f32.Point{X: v2.X / len2, Y: v2.Y / len2}

	dot := v1n.X*v2n.X + v1n.Y*v2n.Y
	// dot product close to 0 means 90-degree angle
	return math.Abs(float64(dot)) < 0.1
}

// cornersToRound returns a slice indicating which corners should be rounded.
// The returned slice has same length as cleaned points (without duplicate closing).
// Each entry corresponds to vertex i (corner at points[i]).
func cornersToRound(points []f32.Point, radius float32) []bool {
	if len(points) < 3 {
		return nil
	}

	// Remove duplicate consecutive points
	cleanPoints := make([]f32.Point, 0, len(points))
	for i, pt := range points {
		if i == 0 || pt != points[i-1] {
			cleanPoints = append(cleanPoints, pt)
		}
	}
	points = cleanPoints

	// Remove duplicate closing point if present (first == last)
	if len(points) > 0 && points[len(points)-1] == points[0] {
		points = points[:len(points)-1]
	}

	if len(points) < 3 {
		return nil
	}

	result := make([]bool, len(points))

	for i := 0; i < len(points); i++ {
		p1 := points[i]
		p2 := points[(i+1)%len(points)]
		p3 := points[(i+2)%len(points)]

		// Calculate vectors for the corner at p2
		v1 := f32.Point{X: p2.X - p1.X, Y: p2.Y - p1.Y}
		v2 := f32.Point{X: p3.X - p2.X, Y: p3.Y - p2.Y}

		// Normalize vectors
		len1 := float32(math.Sqrt(float64(v1.X*v1.X + v1.Y*v1.Y)))
		len2 := float32(math.Sqrt(float64(v2.X*v2.X + v2.Y*v2.Y)))

		if len1 <= 0 || len2 <= 0 {
			result[(i+1)%len(points)] = false
			continue
		}

		// Check if this is approximately a 90-degree corner
		isRightAngle := isRightAngle(p1, p2, p3)

		// Check if we have enough length for rounded corner
		canRound := isRightAngle && len1 > radius && len2 > radius
		result[(i+1)%len(points)] = canRound
	}

	return result
}
