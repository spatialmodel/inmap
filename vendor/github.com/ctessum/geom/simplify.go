package geom

import "math"

// Simplifier is an interface for types that can be simplified.
type Simplifier interface {
	Simplify(tolerance float64) Geom
}

// Simplify simplifies p
// by removing points according to the tolerance parameter,
// while ensuring that the resulting shape is not self intersecting
// (but only if the input shape is not self intersecting). Self-intersecting
// polygons may cause the algorithm to fall into an infinite loop.
//
// It is based on the algorithm:
// J. L. G. Pallero, Robust line simplification on the plane.
// Comput. Geosci. 61, 152–159 (2013).
func (p Polygon) Simplify(tolerance float64) Geom {
	var out Polygon = make([][]Point, len(p))
	for i, r := range p {
		out[i] = simplifyCurve(r, p, tolerance)
	}
	return out
}

// Simplify simplifies mp
// by removing points according to the tolerance parameter,
// while ensuring that the resulting shape is not self intersecting
// (but only if the input shape is not self intersecting). Self-intersecting
// polygons may cause the algorithm to fall into an infinite loop.
//
// It is based on the algorithm:
// J. L. G. Pallero, Robust line simplification on the plane.
// Comput. Geosci. 61, 152–159 (2013).
func (mp MultiPolygon) Simplify(tolerance float64) Geom {
	out := make(MultiPolygon, len(mp))
	for i, p := range mp {
		out[i] = p.Simplify(tolerance).(Polygon)
	}
	return out
}

// Simplify simplifies l
// by removing points according to the tolerance parameter,
// while ensuring that the resulting shape is not self intersecting
// (but only if the input shape is not self intersecting).
//
// It is based on the algorithm:
// J. L. G. Pallero, Robust line simplification on the plane.
// Comput. Geosci. 61, 152–159 (2013).
func (l LineString) Simplify(tolerance float64) Geom {
	return LineString(simplifyCurve(l, [][]Point{}, tolerance))
}

// Simplify simplifies ml
// by removing points according to the tolerance parameter,
// while ensuring that the resulting shape is not self intersecting
// (but only if the input shape is not self intersecting).
//
// It is based on the algorithm:
// J. L. G. Pallero, Robust line simplification on the plane.
// Comput. Geosci. 61, 152–159 (2013).
func (ml MultiLineString) Simplify(tolerance float64) Geom {
	out := make(MultiLineString, len(ml))
	for i, l := range ml {
		out[i] = l.Simplify(tolerance).(LineString)
	}
	return out
}

func simplifyCurve(curve []Point,
	otherCurves [][]Point, tol float64) []Point {
	out := make([]Point, 0, len(curve))

	if len(curve) == 0 {
		return nil
	}

	i := 0
	for {
		out = append(out, curve[i])
		breakTime := false
		for j := i + 2; j < len(curve); j++ {
			breakTime2 := false
			for k := i + 1; k < j; k++ {
				d := distPointToSegment(curve[k], curve[i], curve[j])
				if d > tol {
					// we have found a candidate point to keep
					for {
						// Make sure this simplification doesn't cause any self
						// intersections.
						if j > i+2 &&
							(segMakesNotSimple(curve[i], curve[j-1], [][]Point{out[0:i]}) ||
								segMakesNotSimple(curve[i], curve[j-1], [][]Point{curve[j:]}) ||
								segMakesNotSimple(curve[i], curve[j-1], otherCurves)) {
							j--
						} else {
							i = j - 1
							out = append(out, curve[i])
							breakTime2 = true
							break
						}
					}
				}
				if breakTime2 {
					break
				}
			}
			if j == len(curve)-1 {
				// Add last point regardless of distance.
				out = append(out, curve[j])
				breakTime = true
			}
		}
		if breakTime {
			break
		}
	}
	return out
}

func segMakesNotSimple(segStart, segEnd Point, paths [][]Point) bool {
	seg1 := segment{segStart, segEnd}
	for _, p := range paths {
		for i := 0; i < len(p)-1; i++ {
			seg2 := segment{p[i], p[i+1]}
			if seg1.start == seg2.start || seg1.end == seg2.end ||
				seg1.start == seg2.end || seg1.end == seg2.start {
				// colocated endpoints are not a problem here
				return false
			}
			numIntersections, _, _ := findIntersection(seg1, seg2)
			if numIntersections > 0 {
				return true
			}
		}
	}
	return false
}

// pointOnSegment calculates whether point p is exactly on the finite line segment
// defined by points l1 and l2.
func pointOnSegment(p, l1, l2 Point) bool {
	if (p.X < l1.X && p.X < l2.X) || (p.X > l1.X && p.X > l2.X) ||
		(p.Y < l1.Y && p.Y < l2.Y) || (p.Y > l1.Y && p.Y > l2.Y) {
		return false
	}
	d1 := pointSubtract(l1, p)
	d2 := pointSubtract(l2, l1)

	// If the two slopes are the same, then the point is on the line
	if (d1.X == 0 && d2.X == 0) || d1.Y/d1.X == d2.Y/d2.X {
		return true
	}
	return false
}

// dist_Point_to_Segment(): get the distance of a point to a segment
//     Input:  a Point P and a Segment S (in any dimension)
//     Return: the shortest distance from P to S
// from http://geomalgorithms.com/a02-_lines.html
func distPointToSegment(p, segStart, segEnd Point) float64 {
	v := pointSubtract(segEnd, segStart)
	w := pointSubtract(p, segStart)

	c1 := dot(w, v)
	if c1 <= 0. {
		return d(p, segStart)
	}

	c2 := dot(v, v)
	if c2 <= c1 {
		return d(p, segEnd)
	}

	b := c1 / c2
	pb := Point{segStart.X + b*v.X, segStart.Y + b*v.Y}
	return d(p, pb)
}

func pointSubtract(p1, p2 Point) Point {
	return Point{X: p1.X - p2.X, Y: p1.Y - p2.Y}
}

// dot product
func dot(u, v Point) float64 { return u.X*v.X + u.Y*v.Y }

// norm = length of  vector
func norm(v Point) float64 { return math.Sqrt(dot(v, v)) }

// distance = norm of difference
func d(u, v Point) float64 { return norm(pointSubtract(u, v)) }
