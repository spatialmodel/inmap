package geom

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
