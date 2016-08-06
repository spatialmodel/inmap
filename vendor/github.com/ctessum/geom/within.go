package geom

import "math"

// pointInPolygonal determines whether "pt" is
// within any of the polygons in "pg".
// adapted from https://rosettacode.org/wiki/Ray-casting_algorithm#Go.
// In this version of the algorithm, points that lie on the edge of the polygon
// are considered inside.
func pointInPolygonal(pt Point, pg Polygonal) (in WithinStatus) {
	for _, poly := range pg.Polygons() {
		pgBounds := poly.ringBounds()
		tempIn := pointInPolygon(pt, poly, pgBounds)
		if tempIn == OnEdge {
			return tempIn
		} else if tempIn == Inside {
			in = in.invert()
		}
	}
	return in
}

// WithinStatus gives the status of a point relative to a polygon: whether
// it is inside, outside, or on the edge.
type WithinStatus int

// WithinStatus gives the status of a point relative to a polygon: whether
// it is inside, outside, or on the edge.
const (
	Outside WithinStatus = iota
	Inside
	OnEdge
)

func (w WithinStatus) invert() WithinStatus {
	if w == Outside {
		return Inside
	}
	return Outside
}

// pointInPolygon determines whether "pt" is
// within "pg".
// adapted from https://rosettacode.org/wiki/Ray-casting_algorithm#Go.
// pgBounds is the bounds of each ring in pg.
func pointInPolygon(pt Point, pg Polygon, pgBounds []*Bounds) (in WithinStatus) {
	for i, ring := range pg {
		if len(ring) < 3 {
			continue
		}
		if !pgBounds[i].Overlaps(NewBoundsPoint(pt)) {
			continue
		}
		// check segment between beginning and ending points
		if !ring[len(ring)-1].Equals(ring[0]) {
			if pointOnSegment(pt, ring[len(ring)-1], ring[0]) {
				return OnEdge
			}
			if rayIntersectsSegment(pt, ring[len(ring)-1], ring[0]) {
				in = in.invert()
			}
		}
		// check the rest of the segments.
		for i := 1; i < len(ring); i++ {
			if pointOnSegment(pt, ring[i-1], ring[i]) {
				return OnEdge
			}
			if rayIntersectsSegment(pt, ring[i-1], ring[i]) {
				in = in.invert()
			}
		}
	}
	return in
}

func rayIntersectsSegment(p, a, b Point) bool {
	if a.Y > b.Y {
		a, b = b, a
	}
	for p.Y == a.Y || p.Y == b.Y {
		p.Y = math.Nextafter(p.Y, math.Inf(1))
	}
	if p.Y < a.Y || p.Y > b.Y {
		return false
	}
	if a.X > b.X {
		if p.X >= a.X {
			return false
		}
		if p.X < b.X {
			return true
		}
	} else {
		if p.X > b.X {
			return false
		}
		if p.X < a.X {
			return true
		}
	}
	return (p.Y-a.Y)/(p.X-a.X) >= (b.Y-a.Y)/(b.X-a.X)
}
