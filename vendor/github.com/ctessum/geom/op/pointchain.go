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

package op

import (
	"github.com/ctessum/geom"
)

// Represents a connected sequence of segments. The sequence can only be extended by connecting
// new segments that share an endpoint with the chain.
type chain struct {
	closed bool
	points []geom.Point
}

func newChain(s segment) *chain {
	return &chain{
		closed: false,
		points: []geom.Point{s.start, s.end}}
}

func (c *chain) pushFront(p geom.Point) { c.points = append([]geom.Point{p}, c.points...) }
func (c *chain) pushBack(p geom.Point)  { c.points = append(c.points, p) }

// Links a segment to the chain
func (c *chain) linkSegment(s segment) bool {
	front := c.points[0]
	back := c.points[len(c.points)-1]

	switch true {
	case PointEquals(s.start, front):
		if PointEquals(s.end, back) {
			c.closed = true
		} else {
			c.pushFront(s.end)
		}
		return true
	case PointEquals(s.end, back):
		if PointEquals(s.start, front) {
			c.closed = true
		} else {
			c.pushBack(s.start)
		}
		return true
	case PointEquals(s.end, front):
		if PointEquals(s.start, back) {
			c.closed = true
		} else {
			c.pushFront(s.start)
		}
		return true
	case PointEquals(s.start, back):
		if PointEquals(s.end, front) {
			c.closed = true
		} else {
			c.pushBack(s.end)
		}
		return true
	}
	return false
}

// Links another chain onto this point chain.
func (c *chain) linkChain(other *chain) bool {

	front := c.points[0]
	back := c.points[len(c.points)-1]

	otherFront := other.points[0]
	otherBack := other.points[len(other.points)-1]

	if PointEquals(otherFront, back) {
		c.points = append(c.points, other.points[1:]...)
		goto success
		//c.points = append(c.points[:len(c.points)-1], other.points...)
		//return true
	}

	if PointEquals(otherBack, front) {
		c.points = append(other.points, c.points[1:]...)
		goto success
		//return true
	}

	if PointEquals(otherFront, front) {
		// Remove the first element, and join to reversed chain.points
		c.points = append(reversed(other.points), c.points[1:]...)
		goto success
		//return true
	}

	if PointEquals(otherBack, back) {
		c.points = append(c.points[:len(c.points)-1], reversed(other.points)...)
		goto success
		//c.points = append(other.points, reversed(c.points)...)
		//return true
	}

	return false

success:
	other.points = []geom.Point{}
	return true
}

func reversed(list []geom.Point) []geom.Point {
	length := len(list)
	other := make([]geom.Point, length)
	for i := range list {
		other[length-i-1] = list[i]
	}
	return other
}
