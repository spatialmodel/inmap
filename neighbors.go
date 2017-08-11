/*
Copyright © 2013 the InMAP authors.
This file is part of InMAP.

InMAP is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

InMAP is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package inmap

import (
	"math"

	"github.com/ctessum/geom"
)

func (d *InMAP) setNeighbors(c *Cell, m Mechanism) {
	d.neighbors(c)
	d.setBoundaryNeighbors(c, m)
}

func (d *InMAP) setBoundaryNeighbors(c *Cell, m Mechanism) {
	if c.west.len() == 0 {
		d.addWestBoundary(c, m)
	}
	if c.east.len() == 0 {
		d.addEastBoundary(c, m)
	}
	if c.north.len() == 0 {
		d.addNorthBoundary(c, m)
	}
	if c.south.len() == 0 {
		d.addSouthBoundary(c, m)
	}
	if c.above.len() == 0 {
		d.addTopBoundary(c, m)
	}
}

func (d *InMAP) neighbors(c *Cell) {
	b := c.Bounds()

	// Horizontal
	westbox := newNeighborRect(b, west)
	c.west = getCells(d.index, westbox, c.Layer)
	for _, w := range *c.west {
		if w.east.len() == 1 && (*w.east)[0].boundary {
			d.eastBoundary.delete((*w.east)[0])
			w.east.delete((*w.east)[0])
		}
		w.east.add(c)
		neighborInfoEastWest(w, w.Cell.east.ref(c))
	}

	eastbox := newNeighborRect(b, east)
	c.east = getCells(d.index, eastbox, c.Layer)
	for _, e := range *c.east {
		if e.west.len() == 1 && (*e.west)[0].boundary {
			d.westBoundary.delete((*e.west)[0])
			e.west.delete((*e.west)[0])
		}
		e.west.add(c)
		neighborInfoEastWest(e, e.Cell.west.ref(c))
	}

	southbox := newNeighborRect(b, south)
	c.south = getCells(d.index, southbox, c.Layer)
	for _, s := range *c.south {
		if s.north.len() == 1 && (*s.north)[0].boundary {
			d.northBoundary.delete((*s.north)[0])
			s.north.delete((*s.north)[0])
		}
		s.north.add(c)
		neighborInfoSouthNorth(s, s.Cell.north.ref(c))
	}

	northbox := newNeighborRect(b, north)
	c.north = getCells(d.index, northbox, c.Layer)
	for _, n := range *c.north {
		if n.south.len() == 1 && (*n.south)[0].boundary {
			d.southBoundary.delete((*n.south)[0])
			n.south.delete((*n.south)[0])
		}
		n.south.add(c)
		neighborInfoSouthNorth(n, n.Cell.south.ref(c))
	}

	// Above
	abovebelowbox := newNeighborRect(b, aboveBelow)
	c.above = getCells(d.index, abovebelowbox, c.Layer+1)
	for _, a := range *c.above {
		if a.below.len() == 1 && (*a.below)[0] == a {
			a.below.delete((*a.below)[0])
		}
		a.below.add(c)
		neighborInfoAboveBelow(a, a.Cell.below.ref(c))
	}

	// Below
	c.below = getCells(d.index, abovebelowbox, c.Layer-1)
	for _, b := range *c.below {
		if b.above.len() == 1 && (*b.above)[0].boundary {
			d.topBoundary.delete((*b.above)[0])
			b.above.delete((*b.above)[0])
		}
		b.above.add(c)
		neighborInfoAboveBelow(b, b.Cell.above.ref(c))
	}
	if c.Layer == 0 {
		ref := (*c).below.add(c) // Reflective boundary at ground level.
		neighborInfoBoundaryTopBottom(ref)
	}

	// Ground level.
	c.groundLevel = getCells(d.index, abovebelowbox, 0)
	for _, g := range *c.groundLevel {
		neighborInfoGroundLevel(c, g)
	}
	// Find the cells that this cell is the ground level for.
	if c.Layer == 0 {
		for _, ccI := range d.index.SearchIntersect(abovebelowbox) {
			cc := ccI.(*Cell)
			if cc.Layer > 0 {
				cc.groundLevel.add(c)
				neighborInfoGroundLevel(cc, cc.groundLevel.ref(c))
			}
		}
	}
}

// neighborInfoEastWest calculates information about the relationship
// between two cells that neighbor in the east-west direction, where
// cr1 is the first cell's reference to the second cell, and
// cr2 is the second cell's reference to the first cell.
func neighborInfoEastWest(cr1, cr2 *cellRef) {
	cr1.info = &neighborInfo{
		centerDistance: (cr2.Dx + cr1.Dx) / 2,
		coverFrac:      min(cr1.Dy/cr2.Dy, 1.),
		diff:           harmonicMean(cr2.Kxxyy, cr1.Kxxyy),
	}
	cr2.info = &neighborInfo{
		centerDistance: cr1.info.centerDistance,
		coverFrac:      min(cr2.Dy/cr1.Dy, 1.),
		diff:           cr1.info.diff,
	}
}

// neighborInfoBoundaryEastWest holds information about the relationship
// between a cell on the east-west edge of the domain and the boundary.
func neighborInfoBoundaryEastWest(cr *cellRef) {
	cr.info = &neighborInfo{
		centerDistance: cr.Dx,
		coverFrac:      1.,
		diff:           cr.Kxxyy,
	}
}

// neighborInfoSouthNorth calculates information about the relationship
// between two cells that neighbor in the south-north direction, where
// cr1 is the first cell's reference to the second cell, and
// cr2 is the second cell's reference to the first cell.
func neighborInfoSouthNorth(cr1, cr2 *cellRef) {
	cr1.info = &neighborInfo{
		centerDistance: (cr2.Dy + cr1.Dy) / 2,
		coverFrac:      min(cr1.Dx/cr2.Dx, 1.),
		diff:           harmonicMean(cr2.Kxxyy, cr1.Kxxyy),
	}
	cr2.info = &neighborInfo{
		centerDistance: cr1.info.centerDistance,
		coverFrac:      min(cr2.Dx/cr1.Dx, 1.),
		diff:           cr1.info.diff,
	}
}

// neighborInfoBoundaryEastWest holds information about the relationship
// between a cell on the north-south edge of the domain and the boundary.
func neighborInfoBoundarySouthNorth(cr *cellRef) {
	cr.info = &neighborInfo{
		centerDistance: cr.Dy,
		coverFrac:      1.,
		diff:           cr.Kxxyy,
	}
}

// neighborInfoAboveBelow calculates information about the relationship
// between two cells that neighbor in the up-down direction, where
// cr1 is the first cell's reference to the second cell, and
// cr2 is the second cell's reference to the first cell.
func neighborInfoAboveBelow(cr1, cr2 *cellRef) {
	cr1.info = &neighborInfo{
		centerDistance: (cr2.Dz + cr1.Dz) / 2,
		coverFrac:      min((cr1.Dx*cr1.Dy)/(cr2.Dx*cr2.Dy), 1.),
		diff:           harmonicMean(cr2.Kzz, cr1.Kzz),
	}
	cr2.info = &neighborInfo{
		centerDistance: cr1.info.centerDistance,
		coverFrac:      min((cr2.Dx*cr2.Dy)/(cr1.Dx*cr1.Dy), 1.),
		diff:           cr1.info.diff,
	}
}

// neighborInfoBoundaryEastWest holds information about the relationship
// between a cell on the Top edge of the domain and the boundary.
func neighborInfoBoundaryTopBottom(cr *cellRef) {
	cr.info = &neighborInfo{
		centerDistance: cr.Dz,
		coverFrac:      1.,
		diff:           cr.Kzz,
	}
}

// neighborInfoAboveBelow calculates information about the relationship
// between two cells where cr is a reference to a cell that
// is above c when c is at ground level.
func neighborInfoGroundLevel(c *Cell, cr *cellRef) {
	cr.info = &neighborInfo{
		coverFrac: min((cr.Dx*cr.Dy)/(c.Dx*c.Dy), 1.),
	}
}

// dereferenceNeighbors removes any references to this cell that exist in its
// neighbors.
func (c *Cell) dereferenceNeighbors(d *InMAP) {
	for _, w := range *c.west {
		if w.boundary {
			d.westBoundary.deleteCell(w.Cell)
		} else {
			w.east.deleteCell(c)
		}
	}
	for _, e := range *c.east {
		if e.boundary {
			d.eastBoundary.deleteCell(e.Cell)
		} else {
			e.west.deleteCell(c)
		}
	}
	for _, s := range *c.south {
		if s.boundary {
			d.southBoundary.deleteCell(s.Cell)
		} else {
			s.north.deleteCell(c)
		}
	}
	for _, n := range *c.north {
		if n.boundary {
			d.northBoundary.deleteCell(n.Cell)
		} else {
			n.south.deleteCell(c)
		}
	}
	if c.Layer != 0 { // We don't worry about dereferencing below ground level cells.
		for _, b := range *c.below {
			b.above.deleteCell(c)
		}
	}
	for _, a := range *c.above {
		if a.boundary {
			d.topBoundary.deleteCell(a.Cell)
		} else {
			a.below.deleteCell(c)
		}
	}

	// Dereference the cells that this cell is the ground level for.
	if c.Layer == 0 {
		r := newNeighborRect(c.Bounds(), aboveBelow)
		for _, ccI := range d.index.SearchIntersect(r) {
			cc := ccI.(*Cell)
			if cc.Layer > 0 {
				cc.groundLevel.deleteCell(c)
			}
		}
	}

}

// neighborAlignment specifies the desired alignment of the neighbors that
// are being looked for.
type neighborAlignment int

const (
	west neighborAlignment = iota
	east
	north
	south
	aboveBelow
)

// newNeighborRect returns a rectangle that should overlap all of the neighbors
// of a cell with the given alignment a.
func newNeighborRect(b *geom.Bounds, a neighborAlignment) *geom.Bounds {

	// bboxOffset is a number significantly less than the smallest grid size
	// but not small enough to be confused with zero.
	const (
		bboxOffset = 1.e-10
	)

	o := new(geom.Bounds)

	offsetX := math.Abs(b.Min.X+b.Max.X) / 2 * bboxOffset
	offsetY := math.Abs(b.Min.Y+b.Max.Y) / 2 * bboxOffset

	if offsetX == 0 {
		offsetX = bboxOffset * math.Abs(b.Max.X)
	}
	if offsetY == 0 {
		offsetY = bboxOffset * math.Abs(b.Max.Y)
	}

	// Set x extents
	switch a {
	case west:
		o.Min.X = b.Min.X - 2*offsetX
		o.Max.X = b.Min.X - offsetX
	case east:
		o.Min.X = b.Max.X + offsetX
		o.Max.X = b.Max.X + 2*offsetX
	default:
		o.Min.X = b.Min.X + offsetX
		o.Max.X = b.Max.X - offsetX
	}

	// Set y extents
	switch a {
	case south:
		o.Min.Y = b.Min.Y - 2*offsetY
		o.Max.Y = b.Min.Y - offsetY
	case north:
		o.Min.Y = b.Max.Y + offsetY
		o.Max.Y = b.Max.Y + 2*offsetY
	default:
		o.Min.Y = b.Min.Y + offsetY
		o.Max.Y = b.Max.Y - offsetY
	}
	return o
}
