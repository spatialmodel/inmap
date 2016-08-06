// Copyright 2012 Daniel Connelly.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rtree

import (
	"math"

	"github.com/ctessum/geom"
)

// DistError is an improper distance measurement.  It implements the error
// and is generated when a distance-related assertion fails.
type DistError geom.Point

func (err DistError) Error() string {
	return "rtreego: improper distance"
}

// Dist computes the Euclidean distance between two points p and q.
func dist(p, q geom.Point) float64 {
	sum := 0.0
	dx := p.X - q.X
	sum += dx * dx
	dx = p.Y - q.Y
	sum += dx * dx
	return math.Sqrt(sum)
}

// minDist computes the square of the distance from a point to a rectangle.
// If the point is contained in the rectangle then the distance is zero.
//
// Implemented per Definition 2 of "Nearest Neighbor Queries" by
// N. Roussopoulos, S. Kelley and F. Vincent, ACM SIGMOD, pages 71-79, 1995.
func minDist(p geom.Point, r *geom.Bounds) float64 {
	sum := 0.0
	if p.X < r.Min.X {
		d := p.X - r.Min.X
		sum += d * d
	} else if p.X > r.Max.X {
		d := p.X - r.Max.X
		sum += d * d
	} else {
		sum += 0
	}
	if p.Y < r.Min.Y {
		d := p.Y - r.Min.Y
		sum += d * d
	} else if p.Y > r.Max.Y {
		d := p.Y - r.Max.Y
		sum += d * d
	} else {
		sum += 0
	}
	return sum
}

// minMaxDist computes the minimum of the maximum distances from p to points
// on r.  If r is the bounding box of some geometric objects, then there is
// at least one object contained in r within minMaxDist(p, r) of p.
//
// Implemented per Definition 4 of "Nearest Neighbor Queries" by
// N. Roussopoulos, S. Kelley and F. Vincent, ACM SIGMOD, pages 71-79, 1995.
func minMaxDist(p geom.Point, r *geom.Bounds) float64 {
	// by definition, MinMaxDist(p, r) =
	// min{1<=k<=n}(|pk - rmk|^2 + sum{1<=i<=n, i != k}(|pi - rMi|^2))
	// where rmk and rMk are defined as follows:

	rmX := func() float64 {
		if p.X <= (r.Min.X+r.Max.X)/2 {
			return r.Min.X
		}
		return r.Max.X
	}
	rmY := func() float64 {
		if p.Y <= (r.Min.Y+r.Max.Y)/2 {
			return r.Min.Y
		}
		return r.Max.Y
	}

	rMX := func() float64 {
		if p.X >= (r.Min.X+r.Max.X)/2 {
			return r.Min.X
		}
		return r.Max.X
	}
	rMY := func() float64 {
		if p.Y >= (r.Min.Y+r.Max.Y)/2 {
			return r.Min.Y
		}
		return r.Max.Y
	}

	// This formula can be computed in linear time by precomputing
	// S = sum{1<=i<=n}(|pi - rMi|^2).

	S := 0.0
	d := p.X - rMX()
	S += d * d
	d = p.Y - rMY()
	S += d * d

	// Compute MinMaxDist using the precomputed S.
	min := math.MaxFloat64
	d1 := p.X - rMX()
	d2 := p.X - rmX()
	d = S - d1*d1 + d2*d2
	if d < min {
		min = d
	}
	d1 = p.Y - rMY()
	d2 = p.Y - rmY()
	d = S - d1*d1 + d2*d2
	if d < min {
		min = d
	}

	return min
}

// NewRect constructs and returns a pointer to a Rect given a corner point and
// the lengths of each dimension.  The point p should be the most-negative point
// on the rectangle (in every dimension) and every length should be positive.
func newRect(p, lengths geom.Point) (r *geom.Bounds, err error) {
	r = geom.NewBounds()
	r.Min = p
	r.Max.X = lengths.X + p.X
	r.Max.Y = lengths.Y + p.Y
	if lengths.X <= 0 || lengths.Y <= 0 {
		return r, DistError(lengths)
	}
	return r, nil
}

// size computes the measure of a rectangle (the product of its side lengths).
func size(r *geom.Bounds) float64 {
	return (r.Max.X - r.Min.X) * (r.Max.Y - r.Min.Y)
}

// margin computes the sum of the edge lengths of a rectangle.
func margin(r *geom.Bounds) float64 {
	return 2 * ((r.Max.X - r.Min.X) + (r.Max.Y - r.Min.Y))
}

// containsPoint tests whether p is located inside or on the boundary of r.
func containsPoint(r *geom.Bounds, p geom.Point) bool {
	// p is contained in (or on) r if and only if p <= a <= q for
	// every dimension.
	if p.X < r.Min.X || p.X > r.Max.X {
		return false
	}
	if p.Y < r.Min.Y || p.Y > r.Max.Y {
		return false
	}

	return true
}

// containsRect tests whether r2 is is located inside r1.
func containsRect(r1, r2 *geom.Bounds) bool {
	// enforced by constructor: a1 <= b1 and a2 <= b2.
	// so containment holds if and only if a1 <= a2 <= b2 <= b1
	// for every dimension.
	if r1.Min.X > r2.Min.X || r2.Max.X > r1.Max.X {
		return false
	}
	if r1.Min.Y > r2.Min.Y || r2.Max.Y > r1.Max.Y {
		return false
	}

	return true
}

func enlarge(r1, r2 *geom.Bounds) {
	if r1.Min.X > r2.Min.X {
		r1.Min.X = r2.Min.X
	}
	if r1.Max.X < r2.Max.X {
		r1.Max.X = r2.Max.X
	}
	if r1.Min.Y > r2.Min.Y {
		r1.Min.Y = r2.Min.Y
	}
	if r1.Max.Y < r2.Max.Y {
		r1.Max.Y = r2.Max.Y
	}
}

// intersect computes the intersection of two rectangles.  If no intersection
// exists, the intersection is nil.
func intersect(r1, r2 *geom.Bounds) bool {
	// There are four cases of overlap:
	//
	//     1.  a1------------b1
	//              a2------------b2
	//              p--------q
	//
	//     2.       a1------------b1
	//         a2------------b2
	//              p--------q
	//
	//     3.  a1-----------------b1
	//              a2-------b2
	//              p--------q
	//
	//     4.       a1-------b1
	//         a2-----------------b2
	//              p--------q
	//
	// Thus there are only two cases of non-overlap:
	//
	//     1. a1------b1
	//                    a2------b2
	//
	//     2.             a1------b1
	//        a2------b2
	//
	// Enforced by constructor: a1 <= b1 and a2 <= b2.  So we can just
	// check the endpoints.

	if r2.Max.X < r1.Min.X || r1.Max.X < r2.Min.X {
		return false
	}
	if r2.Max.Y < r1.Min.Y || r1.Max.Y < r2.Min.Y {
		return false
	}
	return true
}

// ToRect constructs a rectangle containing p with side lengths 2*tol.
func ToRect(p geom.Point, tol float64) *geom.Bounds {
	var r geom.Bounds
	r.Min.X = p.X - tol
	r.Max.X = p.X + tol
	r.Min.Y = p.Y - tol
	r.Max.Y = p.Y + tol
	return &r
}

func initBoundingBox(r, r1, r2 *geom.Bounds) {
	*r = *r1
	enlarge(r, r2)
}

// boundingBox constructs the smallest rectangle containing both r1 and r2.
func boundingBox(r1, r2 *geom.Bounds) *geom.Bounds {
	var r geom.Bounds
	initBoundingBox(&r, r1, r2)
	return &r
}
