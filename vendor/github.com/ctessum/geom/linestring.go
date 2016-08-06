package geom

import "math"

// LineString is a number of points that make up a path or line.
type LineString []Point

// Bounds gives the rectangular extents of the LineString.
func (l LineString) Bounds() *Bounds {
	b := NewBounds()
	b.extendPoints(l)
	return b
}

// Length calculates the length of l.
func (l LineString) Length() float64 {
	length := 0.
	for i := 0; i < len(l)-1; i++ {
		p1 := l[i]
		p2 := l[i+1]
		length += math.Hypot(p2.X-p1.X, p2.Y-p1.Y)
	}
	return length
}

// Within calculates whether l is completely within p or touching its edge.
func (l LineString) Within(p Polygonal) WithinStatus {
	for _, pp := range l {
		if pointInPolygonal(pp, p) == Outside {
			return Outside
		}
	}
	return Inside
}
