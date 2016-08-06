package geom

import "math"

func similar(a, b, e float64) bool {
	return math.Abs(a-b) < e
}

func pointSimilar(p1, p2 Point, e float64) bool {
	return similar(p1.X, p2.X, e) && similar(p1.Y, p2.Y, e)
}

func pointsSimilar(p1s, p2s []Point, e float64) bool {
	if len(p1s) != len(p2s) {
		return false
	}
	for i, n := 0, len(p1s); i < n; i++ {
		if !pointSimilar(p1s[i], p2s[i], e) {
			return false
		}
	}
	return true
}

func pointssSimilar(p1ss, p2ss [][]Point, e float64) bool {
	if len(p1ss) != len(p2ss) {
		return false
	}
	for i, n := 0, len(p1ss); i < n; i++ {
		if !pointsSimilar(p1ss[i], p2ss[i], e) {
			return false
		}
	}
	return true
}

// Similar determines whether two geometries are similar within tolerance.
func (p Point) Similar(g Geom, tolerance float64) bool {
	switch g.(type) {
	case Point:
		return pointSimilar(p, g.(Point), tolerance)
	default:
		return false
	}
}

// Similar determines whether two geometries are similar within tolerance.
func (mp MultiPoint) Similar(g Geom, tolerance float64) bool {
	switch g.(type) {
	case MultiPoint:
		return pointsSimilar(mp, g.(MultiPoint), tolerance)
	default:
		return false
	}
}

// Similar determines whether two geometries are similar within tolerance.
// If two lines contain the same points but in different directions it will
// return false.
func (l LineString) Similar(g Geom, tolerance float64) bool {
	switch g.(type) {
	case LineString:
		return pointsSimilar(l, g.(LineString), tolerance)
	default:
		return false
	}
}

// Similar determines whether two geometries are similar within tolerance.
// If ml and g have the similar linestrings but in a different order, it
// will return true.
func (ml MultiLineString) Similar(g Geom, tolerance float64) bool {
	switch g.(type) {
	case MultiLineString:
		ml2 := g.(MultiLineString)
		indices := make([]int, len(ml2))
		for i := range ml2 {
			indices[i] = i
		}
		for _, l := range ml {
			matched := false
			for ii, i := range indices {
				if l.Similar(ml2[i], tolerance) { // we found a match
					matched = true
					// remove index i from futher consideration.
					if ii == len(indices)-1 {
						indices = indices[0:ii]
					} else {
						indices = append(indices[0:ii], indices[ii+1:len(indices)]...)
					}
					break
				}
			}
			if !matched {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// Similar determines whether two geometries are similar within tolerance.
// If ml and g have the similar polygons but in a different order, it
// will return true.
func (mp MultiPolygon) Similar(g Geom, tolerance float64) bool {
	switch g.(type) {
	case MultiPolygon:
		mp2 := g.(MultiPolygon)
		indices := make([]int, len(mp2))
		for i := range mp2 {
			indices[i] = i
		}
		for _, l := range mp {
			matched := false
			for ii, i := range indices {
				if l.Similar(mp2[i], tolerance) { // we found a match
					matched = true
					// remove index i from futher consideration.
					if ii == len(indices)-1 {
						indices = indices[0:ii]
					} else {
						indices = append(indices[0:ii], indices[ii+1:len(indices)]...)
					}
					break
				}
			}
			if !matched {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// Similar determines whether two geometries are similar within tolerance.
// If p and g have the same points with the same winding direction, but a
// different starting point, it will return true. If they have the same
// rings but in a different order, it will return true. If the rings have the same
// points but different winding directions, it will return false.
func (p Polygon) Similar(g Geom, tolerance float64) bool {
	switch g.(type) {
	case Polygon:
		p2 := g.(Polygon)
		indices := make([]int, len(p2))
		for i := range p2 {
			indices[i] = i
		}
		for _, r1 := range p {
			matched := false
			for ii, i := range indices {
				if ringSimilar(r1, p2[i], tolerance) { // we found a match
					matched = true
					// remove index i from futher consideration.
					if ii == len(indices)-1 {
						indices = indices[0:ii]
					} else {
						indices = append(indices[0:ii], indices[ii+1:len(indices)]...)
					}
					break
				}
			}
			if !matched {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// Similar determines whether two bounds are similar within tolerance.
func (b *Bounds) Similar(g Geom, tolerance float64) bool {
	switch g.(type) {
	case *Bounds:
		b2 := g.(*Bounds)
		return pointSimilar(b.Min, b2.Min, tolerance) && pointSimilar(b.Max, b2.Max, tolerance)
	default:
		return false
	}
}

// Similar determines whether two geometries collections are similar within tolerance.
// If gc and g have the same geometries
// but in a different order, it will return true.
func (gc GeometryCollection) Similar(g Geom, tolerance float64) bool {
	switch g.(type) {
	case GeometryCollection:
		gc2 := g.(GeometryCollection)
		indices := make([]int, len(gc2))
		for i := range gc2 {
			indices[i] = i
		}
		for _, gc1 := range gc {
			matched := false
			for ii, i := range indices {
				if gc1.Similar(gc2[i], tolerance) { // we found a match
					matched = true
					// remove index i from futher consideration.
					if ii == len(indices)-1 {
						indices = indices[0:ii]
					} else {
						indices = append(indices[0:ii], indices[ii+1:len(indices)]...)
					}
					break
				}
			}
			if !matched {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func ringSimilar(a, b []Point, e float64) bool {
	if len(a) != len(b) {
		return false
	}
	ia := minPt(a)
	ib := minPt(b)
	for i := 0; i < len(a); i++ {
		if !pointSimilar(a[ia], b[ib], e) {
			return false
		}
		ia = nextPt(ia, len(a))
		ib = nextPt(ib, len(b))
	}
	return true
}

// ring iterator function
func nextPt(i, l int) int {
	if i == l-2 { // Skip the last point that matches the first point.
		return 0
	}
	return i + 1
}

// find bottom-most of leftmost points, to have fixed anchor
func minPt(c []Point) int {
	min := 0
	for j, p := range c {
		if p.X < c[min].X || p.X == c[min].X && p.Y < c[min].Y {
			min = j
		}
	}
	return min
}
