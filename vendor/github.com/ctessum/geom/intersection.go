package geom

import "math"

var nanPoint Point

func init() {
	nanPoint = Point{X: math.NaN(), Y: math.NaN()}
}

// Modified from package github.com/akavel/polyclip-go.
// Copyright (c) 2011 Mateusz Czapliński (Go port)
// Copyright (c) 2011 Mahir Iqbal (as3 version)
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// based on http://code.google.com/p/as3polyclip/ (MIT licensed)
// and code by Martínez et al: http://wwwdi.ujaen.es/~fmartin/bool_op.html (public domain)
func findIntersection(seg0, seg1 segment) (int, Point, Point) {
	pi0 := nanPoint
	pi1 := nanPoint
	p0 := seg0.start
	d0 := Point{seg0.end.X - p0.X, seg0.end.Y - p0.Y}
	p1 := seg1.start
	d1 := Point{seg1.end.X - p1.X, seg1.end.Y - p1.Y}
	sqrEpsilon := 0. // was 1e-3 earlier
	E := Point{p1.X - p0.X, p1.Y - p0.Y}
	kross := d0.X*d1.Y - d0.Y*d1.X
	sqrKross := kross * kross
	sqrLen0 := lengthToOrigin(d0)
	sqrLen1 := lengthToOrigin(d1)

	if sqrKross > sqrEpsilon*sqrLen0*sqrLen1 {
		// lines of the segments are not parallel
		s := (E.X*d1.Y - E.Y*d1.X) / kross
		if s < 0 || s > 1 {
			return 0, Point{}, Point{}
		}
		t := (E.X*d0.Y - E.Y*d0.X) / kross
		if t < 0 || t > 1 {
			return 0, nanPoint, nanPoint
		}
		// intersection of lines is a point an each segment [MC: ?]
		pi0.X = p0.X + s*d0.X
		pi0.Y = p0.Y + s*d0.Y

		// [MC: commented fragment removed]

		return 1, pi0, nanPoint
	}

	// lines of the segments are parallel
	sqrLenE := lengthToOrigin(E)
	kross = E.X*d0.Y - E.Y*d0.X
	sqrKross = kross * kross
	if sqrKross > sqrEpsilon*sqrLen0*sqrLenE {
		// lines of the segment are different
		return 0, nanPoint, nanPoint
	}

	// Lines of the segment are the same. Need to test for overlap of segments.
	// s0 = Dot (D0, E) * sqrLen0
	s0 := (d0.X*E.X + d0.Y*E.Y) / sqrLen0
	// s1 = s0 + Dot (D0, D1) * sqrLen0
	s1 := s0 + (d0.X*d1.X+d0.Y*d1.Y)/sqrLen0
	smin := math.Min(s0, s1)
	smax := math.Max(s0, s1)
	w := make([]float64, 0, 2)
	imax := findIntersection2(0.0, 1.0, smin, smax, &w)

	if imax > 0 {
		pi0.X = p0.X + w[0]*d0.X
		pi0.Y = p0.Y + w[0]*d0.Y

		// [MC: commented fragment removed]

		if imax > 1 {
			pi1.X = p0.X + w[1]*d0.X
			pi1.Y = p0.Y + w[1]*d0.Y
		}
	}

	return imax, pi0, pi1
}

func findIntersection2(u0, u1, v0, v1 float64, w *[]float64) int {
	if u1 < v0 || u0 > v1 {
		return 0
	}
	if u1 == v0 {
		*w = append(*w, u1)
		return 1
	}

	// u1 > v0

	if u0 == v1 {
		*w = append(*w, u0)
		return 1
	}

	// u0 < v1

	if u0 < v0 {
		*w = append(*w, v0)
	} else {
		*w = append(*w, u0)
	}
	if u1 > v1 {
		*w = append(*w, v1)
	} else {
		*w = append(*w, u1)
	}
	return 2
}

// Length returns distance from p to point (0, 0).
func lengthToOrigin(p Point) float64 {
	return math.Sqrt(p.X*p.X + p.Y*p.Y)
}

// Used to represent an edge of a polygon.
type segment struct {
	start, end Point
}
