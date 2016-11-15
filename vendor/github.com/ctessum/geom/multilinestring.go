package geom

import "math"

// MultiLineString is a holder for multiple related LineStrings.
type MultiLineString []LineString

// Bounds gives the rectangular extents of the MultiLineString.
func (ml MultiLineString) Bounds() *Bounds {
	b := NewBounds()
	for _, l := range ml {
		b.Extend(l.Bounds())
	}
	return b
}

// Length calculates the combined length of the linestrings in ml.
func (ml MultiLineString) Length() float64 {
	length := 0.
	for _, l := range ml {
		length += l.Length()
	}
	return length
}

// Within calculates whether ml is completely within p or on its edge.
func (ml MultiLineString) Within(p Polygonal) WithinStatus {
	for _, l := range ml {
		if l.Within(p) == Outside {
			return Outside
		}
	}
	return Inside
}

// Distance calculates the shortest distance from p to the MultiLineString.
func (ml MultiLineString) Distance(p Point) float64 {
	d := math.Inf(1)
	for _, l := range ml {
		lDist := l.Distance(p)
		d = math.Min(d, lDist)
	}
	return d
}
