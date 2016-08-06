package geom

import (
	"math"
)

// Bounds holds the spatial extent of a geometry.
type Bounds struct {
	Min, Max Point
}

// Extend increases the extent of b1 to include b2.
func (b *Bounds) Extend(b2 *Bounds) {
	if b2 == nil {
		return
	}
	b.extendPoint(b2.Min)
	b.extendPoint(b2.Max)
}

// NewBounds initializes a new bounds object.
func NewBounds() *Bounds {
	return &Bounds{Point{X: math.Inf(1), Y: math.Inf(1)}, Point{X: math.Inf(-1), Y: math.Inf(-1)}}
}

// NewBoundsPoint creates a bounds object from a point.
func NewBoundsPoint(point Point) *Bounds {
	return &Bounds{Point{X: point.X, Y: point.Y}, Point{X: point.X, Y: point.Y}}
}

// Copy returns a copy of b.
func (b *Bounds) Copy() *Bounds {
	return &Bounds{Point{X: b.Min.X, Y: b.Min.Y}, Point{X: b.Max.X, Y: b.Max.Y}}
}

// Empty returns true if b does not contain any points.
func (b *Bounds) Empty() bool {
	return b.Max.X < b.Min.X || b.Max.Y < b.Min.Y
}

func (b *Bounds) extendPoint(point Point) *Bounds {
	b.Min.X = math.Min(b.Min.X, point.X)
	b.Min.Y = math.Min(b.Min.Y, point.Y)
	b.Max.X = math.Max(b.Max.X, point.X)
	b.Max.Y = math.Max(b.Max.Y, point.Y)
	return b
}

func (b *Bounds) extendPoints(points []Point) {
	for _, point := range points {
		b.extendPoint(point)
	}
}

func (b *Bounds) extendPointss(pointss [][]Point) {
	for _, points := range pointss {
		b.extendPoints(points)
	}
}

// Overlaps returns whether b and b2 overlap.
func (b *Bounds) Overlaps(b2 *Bounds) bool {
	return b.Min.X <= b2.Max.X && b.Min.Y <= b2.Max.Y && b.Max.X >= b2.Min.X && b.Max.Y >= b2.Min.Y
}

// Bounds returns b
func (b *Bounds) Bounds() *Bounds {
	return b
}

// Within calculates whether b is within poly.
func (b *Bounds) Within(poly Polygonal) WithinStatus {
	minIn := pointInPolygonal(b.Min, poly)
	maxIn := pointInPolygonal(b.Max, poly)
	if minIn == Outside || maxIn == Outside {
		return Outside
	}
	return Inside
}
