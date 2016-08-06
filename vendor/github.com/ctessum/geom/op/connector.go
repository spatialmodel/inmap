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
	"math"
)

type outputType int

const (
	outputPolygons outputType = iota
	outputLines
	outputPoints
)

// Holds intermediate results (pointChains) of the clipping operation and forms them into
// the final polygon.
type connector struct {
	subject, clipping geom.Polygon
	outType           outputType
	openPolys         []chain
	closedPolys       []chain
}

func (c *connector) add(s segment) {

	// if outputting lines, only keep segments coincident with the
	// subject linestrings.
	if c.outType == outputLines {
		keepSeg := false
		for _, ls := range c.subject {
			for i := 0; i < len(ls)-1; i++ {
				if pointOnSegment(s.start, ls[i], ls[i+1]) &&
					pointOnSegment(s.end, ls[i], ls[i+1]) {
					keepSeg = true
					break
				}
			}
			if keepSeg {
				break
			}
		}
		if !keepSeg {
			return
		}
	}

	// j iterates through the openPolygon chains.
	for j := range c.openPolys {
		chain := &c.openPolys[j]
		if !chain.linkSegment(s) {
			continue
		}

		if chain.closed {
			if len(chain.points) == 2 {
				// We tried linking the same segment (but flipped end and start) to
				// a chain. (i.e. chain was <p0, p1>, we tried linking Segment(p1, p0)
				// so the chain was closed illegally.
				chain.closed = false
				return
			}
			// move the chain from openPolys to closedPolys
			c.closedPolys = append(c.closedPolys, c.openPolys[j])
			c.openPolys = append(c.openPolys[:j], c.openPolys[j+1:]...)
			return
		}

		// !chain.closed
		k := len(c.openPolys)
		for i := j + 1; i < k; i++ {
			// Try to connect this open link to the rest of the chains.
			// We won't be able to connect this to any of the chains preceding this one
			// because we know that linkSegment failed on those.
			if chain.linkChain(&c.openPolys[i]) {
				// delete
				c.openPolys = append(c.openPolys[:i], c.openPolys[i+1:]...)
				return
			}
		}
		return
	}

	// The segment cannot be connected with any open polygon
	c.openPolys = append(c.openPolys, *newChain(s))
}

func (c *connector) toShape() geom.Geom {
	// Check for empty result
	if (len(c.closedPolys) == 0 ||
		(len(c.closedPolys) == 1 && len(c.closedPolys[0].points) == 0)) &&
		(len(c.openPolys) == 0 ||
			(len(c.openPolys) == 1 && len(c.openPolys[0].points) == 0)) {
		return nil
	}

	switch c.outType {
	case outputPolygons:
		var poly geom.Polygon
		poly = make([][]geom.Point, len(c.closedPolys))
		for i, chain := range c.closedPolys {
			poly[i] = make([]geom.Point, len(chain.points)+1)
			for j, p := range chain.points {
				poly[i][j] = p
			}
			// close the ring as per OGC standard
			poly[i][len(chain.points)] = poly[i][0]
		}
		// fix winding directions
		FixOrientation(poly)
		return geom.Geom(poly)

	case outputLines:
		// Because we're dealing with linestrings and not polygons,
		// copy the openPolys to the output instead of the closedPolys
		var outline geom.MultiLineString
		outline = make([]geom.LineString, len(c.openPolys))
		for i, chain := range c.openPolys {
			outline[i] = make([]geom.Point, len(chain.points))
			for j, p := range chain.points {
				outline[i][j] = p
			}
		}
		return geom.Geom(outline)

	case outputPoints:
		// only keep points coincident with both subject and clip
		var outpt geom.MultiPoint
		outpt = make([]geom.Point, 0)
		for a, chain := range c.closedPolys {
			for b, p := range chain.points {
				for _, ls1 := range c.subject {
					for i := 0; i < len(ls1)-1; i++ {
						if pointOnSegment(p, ls1[i], ls1[i+1]) {
							for _, ls2 := range c.clipping {
								for j := 0; j < len(ls2)-1; j++ {
									if pointOnSegment(p, ls2[j], ls2[j+1]) {
										outpt = append(outpt, p)
									}
									// remove point
									c.closedPolys[a].points[b] =
										geom.Point{math.NaN(), math.NaN()}
								}
							}
						}
					}
				}
			}
		}
		return outpt
	default:
		panic("This shouldn't happen!")
		return nil
	}
}
