package geom

import "math"

// Area returns the area of p. The function works correctly for polygons with
// holes, regardless of the winding order of the holes, but will give the wrong
// result for self-intersecting polygons.
func (p Polygon) Area() float64 {
	a := 0.

	// Calculate the bounds of all the rings.
	bounds := make([]*Bounds, len(p))
	for i, r := range p {
		b := NewBounds()
		b.extendPoints(r)
		bounds[i] = b
	}

	for i, r := range p {
		a += area(r, i, p, bounds)
	}
	return a
}

// area calculates the area of r, where r is a ring within p.
// It returns a negative value if r represents a hole in p.
// It is adapted from http://www.mathopenref.com/coordpolygonarea2.html
// to allow arbitrary winding order. bounds is the bounds of each ring in p.
func area(r []Point, i int, p Polygon, bounds []*Bounds) float64 {
	if len(r) < 2 {
		return 0
	}
	highI := len(r) - 1
	A := (r[highI].X +
		r[0].X) * (r[0].Y - r[highI].Y)
	for ii := 0; ii < highI; ii++ {
		A += (r[ii].X +
			r[ii+1].X) * (r[ii+1].Y - r[ii].Y)
	}
	A = math.Abs(A / 2.)
	// check whether all of the points on this ring are inside
	// the polygon.
	if len(p) == 1 {
		return A // This is not a hole.
	}
	pWithoutRing := make(Polygon, len(p))
	copy(pWithoutRing, p)
	pWithoutRing = Polygon(append(pWithoutRing[:i], pWithoutRing[i+1:]...))
	boundsWithoutRing := make([]*Bounds, len(p))
	copy(boundsWithoutRing, bounds)
	boundsWithoutRing = append(boundsWithoutRing[:i], boundsWithoutRing[i+1:]...)

	for _, pp := range r {
		in := pointInPolygon(pp, pWithoutRing, boundsWithoutRing)
		if in == OnEdge {
			continue // It is not clear whether this is a hole or not.
		} else if in == Outside {
			return A // This is not a hole.
		}
		return -A // This is a hole
	}

	// All of the points on this ring are on the edge of the polygon. In this
	// case we check if this ring exactly matches, and therefore cancels out,
	// any of the other rings.
	matches := 0
	for _, rr := range pWithoutRing {
		if pointsSimilar(r, rr, 0) {
			matches++
		}
	}
	if matches%2 == 1 {
		return 0 // There is an odd number of matches so the area cancels out.
	}
	// If we get here there is an even number of matches. If the polygon is not
	// self-intersecting (only self-touching) that means this is a hole.
	// The algorithm is not guaranteed to work with self-intersecting polygons.
	return -A // This is a hole
}

func (p Polygon) ringBounds() []*Bounds {
	bounds := make([]*Bounds, len(p))
	for i, r := range p {
		pgBounds := NewBounds()
		pgBounds.extendPoints(r)
		bounds[i] = pgBounds
	}
	return bounds
}

// see http://www.mathopenref.com/coordpolygonarea2.html
func signedarea(polygon []Point) float64 {
	if len(polygon) < 2 {
		return 0
	}
	highI := len(polygon) - 1
	A := (polygon[highI].X +
		polygon[0].X) * (polygon[0].Y - polygon[highI].Y)
	for i := 0; i < highI; i++ {
		A += (polygon[i].X +
			polygon[i+1].X) * (polygon[i+1].Y - polygon[i].Y)
	}
	return A / 2.
}

// Centroid calculates the centroid of p, from
// wikipedia: http://en.wikipedia.org/wiki/Centroid#Centroid_of_polygon.
// The polygon can have holes, but each ring must be closed (i.e.,
// p[0] == p[n-1], where the ring has n points) and must not be
// self-intersecting.
// The algorithm will not check to make sure the holes are
// actually inside the outer rings.
// This has not been thoroughly tested.
func (p Polygon) Centroid() Point {
	var A, xA, yA float64
	for _, r := range p {
		a := signedarea(r)
		cx, cy := 0., 0.
		if r[len(r)-1] != r[0] {
			r = append(r, r[0])
		}
		for i := 0; i < len(r)-1; i++ {
			cx += (r[i].X + r[i+1].X) *
				(r[i].X*r[i+1].Y - r[i+1].X*r[i].Y)
			cy += (r[i].Y + r[i+1].Y) *
				(r[i].X*r[i+1].Y - r[i+1].X*r[i].Y)
		}
		cx /= 6 * a
		cy /= 6 * a
		A += a
		xA += cx * a
		yA += cy * a
	}
	return Point{X: xA / A, Y: yA / A}
}
