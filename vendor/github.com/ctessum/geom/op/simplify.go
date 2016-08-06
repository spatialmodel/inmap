package op

import (
	"fmt"
	"time"

	"github.com/ctessum/geom"
)

// Simplify simplifies a line, multiline, polygon, or multipolygon
// by removing points according to the tolerance parameter,
// while ensuring that the resulting shape is not self intersecting
// (but only if the input shape is not self intersecting).
//
// It is based on the algorithm:
// J. L. G. Pallero, Robust line simplification on the plane.
// Comput. Geosci. 61, 152–159 (2013).
func Simplify(g geom.Geom, tolerance float64) (geom.Geom, error) {
	if g == nil {
		return nil, nil
	}

	var err error
	resultChan := make(chan geom.Geom)
	go func() {
		// Run the simplifier, while checking for an infinite loop.
		var gg geom.Geom
		gg, err = simplify(g, tolerance)
		resultChan <- gg
	}()
	timer := time.NewTimer(1 * time.Minute)
	select {
	case result := <-resultChan:
		timer.Stop()
		return result, err
	case <-timer.C:
		return nil, newSimplifyInfiniteLoopError(g)
	}
}

func simplify(g geom.Geom, tolerance float64) (geom.Geom, error) {
	switch g.(type) {
	case geom.Point, geom.MultiPoint:
		return g, nil
	case geom.Polygon:
		p := g.(geom.Polygon)
		var out geom.Polygon = make([][]geom.Point, len(p))
		for i, r := range p {
			out[i] = simplifyCurve(r, p, tolerance)
		}
		return out, nil
	case geom.MultiPolygon:
		mp := g.(geom.MultiPolygon)
		var out geom.MultiPolygon = make([]geom.Polygon, len(mp))
		for i, p := range mp {
			o, err := Simplify(p, tolerance)
			if err != nil {
				return out, err
			}
			if o != nil {
				out[i] = o.(geom.Polygon)
			}
		}
		return out, nil
	case geom.LineString:
		l := g.(geom.LineString)
		out := geom.LineString(simplifyCurve(l, [][]geom.Point{}, tolerance))
		return out, nil
	case geom.MultiLineString:
		ml := g.(geom.MultiLineString)
		var out geom.MultiLineString = make([]geom.LineString, len(ml))
		for i, l := range ml {
			o, err := Simplify(l, tolerance)
			if err != nil {
				return out, err
			}
			if o != nil {
				out[i] = o.(geom.LineString)
			}
		}
		return out, nil
	default:
		return nil, newUnsupportedGeometryError(g)
	}
}

func simplifyCurve(curve []geom.Point,
	otherCurves [][]geom.Point, tol float64) []geom.Point {
	out := make([]geom.Point, 0, len(curve))

	i := 0
	out = append(out, curve[i])
	for {
		breakTime := false
		for j := i + 2; j < len(curve); j++ {
			breakTime2 := false
			for k := i + 1; k < j; k++ {
				d := distPointToSegment(curve[k], curve[i], curve[j])
				if d > tol {
					// we have found a candidate point to keep
					for {
						// Make sure this simplifcation doesn't cause any self
						// intersections.
						if segMakesNotSimple(curve[i], curve[j-1],
							[][]geom.Point{out[0 : len(out)-1]}) ||
							segMakesNotSimple(curve[i], curve[j-1],
								[][]geom.Point{curve[j:]}) ||
							segMakesNotSimple(curve[i], curve[j-1],
								otherCurves) {
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

func segMakesNotSimple(segStart, segEnd geom.Point, paths [][]geom.Point) bool {
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

// SimplifyInfiniteLoopError indicates that the Simplify function has fallen
// into an infinite loop.
type SimplifyInfiniteLoopError struct {
	g geom.Geom
}

func newSimplifyInfiniteLoopError(g geom.Geom) SimplifyInfiniteLoopError {
	return SimplifyInfiniteLoopError{g: g}
}

func (e SimplifyInfiniteLoopError) Error() string {
	return fmt.Sprintf(
		"function op.Simplify appears to have fallen into an "+
			"infinite loop. \n\ngeometry=%#v", e.g)
}
