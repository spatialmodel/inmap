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

import "github.com/ctessum/geom"

func (d *InMAP) setNeighbors(c *Cell, bboxOffset float64) {
	d.neighbors(c, bboxOffset)
	d.setBoundaryNeighbors(c)
	if c.Layer == 0 {
		c.below = []*Cell{c}
		c.groundLevel = []*Cell{c}
	}
	c.neighborInfo()
}

func (d *InMAP) setBoundaryNeighbors(c *Cell) {
	if len(c.west) == 0 {
		d.addWestBoundary(c)
	}
	if len(c.east) == 0 {
		d.addEastBoundary(c)
	}
	if len(c.north) == 0 {
		d.addNorthBoundary(c)
	}
	if len(c.south) == 0 {
		d.addSouthBoundary(c)
	}
	if len(c.above) == 0 {
		d.addTopBoundary(c)
	}
}

func (d *InMAP) neighbors(c *Cell, bboxOffset float64) {
	b := c.Bounds()

	// Horizontal
	westbox := newRect(b.Min.X-2*bboxOffset, b.Min.Y+bboxOffset,
		b.Min.X-bboxOffset, b.Max.Y-bboxOffset)
	c.west = getCells(d.index, westbox, c.Layer)
	for _, w := range c.west {
		if len(w.east) == 1 && w.east[0].boundary {
			deleteCellFromSlice(w.east[0], &d.eastBoundary)
			deleteCellFromSlice(w.east[0], &w.east)
		}
		w.east = append(w.east, c)
		w.neighborInfo()
	}

	eastbox := newRect(b.Max.X+bboxOffset, b.Min.Y+bboxOffset,
		b.Max.X+2*bboxOffset, b.Max.Y-bboxOffset)
	c.east = getCells(d.index, eastbox, c.Layer)
	for _, e := range c.east {
		if len(e.west) == 1 && e.west[0].boundary {
			deleteCellFromSlice(e.west[0], &d.westBoundary)
			deleteCellFromSlice(e.west[0], &e.west)
		}
		e.west = append(e.west, c)
		e.neighborInfo()
	}

	southbox := newRect(b.Min.X+bboxOffset, b.Min.Y-2*bboxOffset,
		b.Max.X-bboxOffset, b.Min.Y-bboxOffset)
	c.south = getCells(d.index, southbox, c.Layer)
	for _, s := range c.south {
		if len(s.north) == 1 && s.north[0].boundary {
			deleteCellFromSlice(s.north[0], &d.northBoundary)
			deleteCellFromSlice(s.north[0], &s.north)
		}
		s.north = append(s.north, c)
		s.neighborInfo()
	}

	northbox := newRect(b.Min.X+bboxOffset, b.Max.Y+bboxOffset,
		b.Max.X-bboxOffset, b.Max.Y+2*bboxOffset)
	c.north = getCells(d.index, northbox, c.Layer)
	for _, n := range c.north {
		if len(n.south) == 1 && n.south[0].boundary {
			deleteCellFromSlice(n.south[0], &d.southBoundary)
			deleteCellFromSlice(n.south[0], &n.south)
		}
		n.south = append(n.south, c)
		n.neighborInfo()
	}

	// Above
	abovebox := newRect(b.Min.X+bboxOffset, b.Min.Y+bboxOffset,
		b.Max.X-bboxOffset, b.Max.Y-bboxOffset)
	c.above = getCells(d.index, abovebox, c.Layer+1)
	for _, a := range c.above {
		if len(a.below) == 1 && a.below[0] == a {
			deleteCellFromSlice(a.below[0], &a.below)
		}
		a.below = append(a.below, c)
		a.neighborInfo()
	}

	// Below
	belowbox := newRect(b.Min.X+bboxOffset, b.Min.Y+bboxOffset,
		b.Max.X-bboxOffset, b.Max.Y-bboxOffset)
	c.below = getCells(d.index, belowbox, c.Layer-1)
	for _, b := range c.below {
		if len(b.above) == 1 && b.above[0].boundary {
			deleteCellFromSlice(b.above[0], &d.topBoundary)
			deleteCellFromSlice(b.above[0], &b.above)
		}
		b.above = append(b.above, c)
		b.neighborInfo()
	}

	// Ground level.
	groundlevelbox := newRect(b.Min.X+bboxOffset, b.Min.Y+bboxOffset,
		b.Max.X-bboxOffset, b.Max.Y-bboxOffset)
	c.groundLevel = getCells(d.index, groundlevelbox, 0)

	// Find the cells that this cell is the ground level for.
	if c.Layer == 0 {
		for _, ccI := range d.index.SearchIntersect(c.Centroid().Bounds()) {
			cc := ccI.(*Cell)
			if cc.Layer > 0 {
				cc.groundLevel = append(cc.groundLevel, c)
				cc.neighborInfo()
			}
		}
	}
}

// Calculate center-to-center cell distance,
// fractions of grid cell covered by each neighbor
// and harmonic mean staggered-grid diffusivities.
func (c *Cell) neighborInfo() {
	c.dxPlusHalf = make([]float64, len(c.east))
	c.eastFrac = make([]float64, len(c.east))
	c.kxxEast = make([]float64, len(c.east))
	for i, e := range c.east {
		c.dxPlusHalf[i] = (c.Dx + e.Dx) / 2.
		c.eastFrac[i] = min(e.Dy/c.Dy, 1.)
		c.kxxEast[i] = harmonicMean(c.Kxxyy, e.Kxxyy)
		e.dxMinusHalf = append(e.dxMinusHalf, c.dxPlusHalf[i])
		e.westFrac = append(e.westFrac, c.eastFrac[i])
		e.kxxWest = append(e.kxxWest, c.kxxEast[i])
	}
	c.dxMinusHalf = make([]float64, len(c.west))
	c.westFrac = make([]float64, len(c.west))
	c.kxxWest = make([]float64, len(c.west))
	for i, w := range c.west {
		c.dxMinusHalf[i] = (c.Dx + w.Dx) / 2.
		c.westFrac[i] = min(w.Dy/c.Dy, 1.)
		c.kxxWest[i] = harmonicMean(c.Kxxyy, w.Kxxyy)
		w.dxPlusHalf = append(w.dxPlusHalf, c.dxMinusHalf[i])
		w.eastFrac = append(w.eastFrac, c.westFrac[i])
		w.kxxEast = append(w.kxxEast, c.kxxWest[i])
	}
	c.dyPlusHalf = make([]float64, len(c.north))
	c.northFrac = make([]float64, len(c.north))
	c.kyyNorth = make([]float64, len(c.north))
	for i, n := range c.north {
		c.dyPlusHalf[i] = (c.Dy + n.Dy) / 2.
		c.northFrac[i] = min(n.Dx/c.Dx, 1.)
		c.kyyNorth[i] = harmonicMean(c.Kxxyy, n.Kxxyy)
		n.dyMinusHalf = append(n.dyMinusHalf, c.dyPlusHalf[i])
		n.southFrac = append(n.southFrac, c.northFrac[i])
		n.kyySouth = append(n.kyySouth, c.kyyNorth[i])
	}
	c.dyMinusHalf = make([]float64, len(c.south))
	c.southFrac = make([]float64, len(c.south))
	c.kyySouth = make([]float64, len(c.south))
	for i, s := range c.south {
		c.dyMinusHalf[i] = (c.Dy + s.Dy) / 2.
		c.southFrac[i] = min(s.Dx/c.Dx, 1.)
		c.kyySouth[i] = harmonicMean(c.Kxxyy, s.Kxxyy)
		s.dyPlusHalf = append(s.dyPlusHalf, c.dyMinusHalf[i])
		s.northFrac = append(s.northFrac, c.southFrac[i])
		s.kyyNorth = append(s.kyyNorth, c.kyySouth[i])
	}
	c.dzPlusHalf = make([]float64, len(c.above))
	c.aboveFrac = make([]float64, len(c.above))
	c.kzzAbove = make([]float64, len(c.above))
	for i, a := range c.above {
		c.dzPlusHalf[i] = (c.Dz + a.Dz) / 2.
		c.aboveFrac[i] = min((a.Dx*a.Dy)/(c.Dx*c.Dy), 1.)
		c.kzzAbove[i] = harmonicMean(c.Kzz, a.Kzz)
		a.dzMinusHalf = append(a.dzMinusHalf, c.dzPlusHalf[i])
		a.belowFrac = append(a.belowFrac, c.aboveFrac[i])
		a.kzzBelow = append(a.kzzBelow, c.kzzAbove[i])
	}
	c.dzMinusHalf = make([]float64, len(c.below))
	c.belowFrac = make([]float64, len(c.below))
	c.kzzBelow = make([]float64, len(c.below))
	for i, b := range c.below {
		c.dzMinusHalf[i] = (c.Dz + b.Dz) / 2.
		c.belowFrac[i] = min((b.Dx*b.Dy)/(c.Dx*c.Dy), 1.)
		c.kzzBelow[i] = harmonicMean(c.Kzz, b.Kzz)
		b.dzPlusHalf = append(b.dzPlusHalf, c.dzMinusHalf[i])
		b.aboveFrac = append(b.aboveFrac, c.belowFrac[i])
		b.kzzAbove = append(b.kzzAbove, c.kzzBelow[i])
	}
	c.groundLevelFrac = make([]float64, len(c.groundLevel))
	for i, g := range c.groundLevel {
		c.groundLevelFrac[i] = min((g.Dx*g.Dy)/(c.Dx*c.Dy), 1.)
	}
}

// dereferenceNeighbors removes any references to this cell that exist in its
// neighbors.
func (c *Cell) dereferenceNeighbors(d *InMAP) {
	for _, w := range c.west {
		if w.boundary {
			deleteCellFromSlice(w, &d.westBoundary)
		} else {
			deleteCellFromSlice(c, &w.east, &w.eastFrac, &w.kxxEast, &w.dxPlusHalf)
		}
	}
	for _, e := range c.east {
		if e.boundary {
			deleteCellFromSlice(e, &d.eastBoundary)
		} else {
			deleteCellFromSlice(c, &e.west, &e.westFrac, &e.kxxWest, &e.dxMinusHalf)
		}
	}
	for _, s := range c.south {
		if s.boundary {
			deleteCellFromSlice(s, &d.southBoundary)
		} else {
			deleteCellFromSlice(c, &s.north, &s.northFrac, &s.kyyNorth, &s.dyPlusHalf)
		}
	}
	for _, n := range c.north {
		if n.boundary {
			deleteCellFromSlice(n, &d.northBoundary)
		} else {
			deleteCellFromSlice(c, &n.south, &n.southFrac, &n.kyySouth, &n.dyMinusHalf)
		}
	}
	for _, b := range c.below {
		deleteCellFromSlice(c, &b.above, &b.aboveFrac, &b.kzzAbove, &b.dzPlusHalf)
	}
	for _, a := range c.above {
		if a.boundary {
			deleteCellFromSlice(a, &d.topBoundary)
		} else {
			deleteCellFromSlice(c, &a.below, &a.belowFrac, &a.kzzBelow, &a.dzMinusHalf)
		}
	}

	// Dereference the cells that this cell is the ground level for.
	if c.Layer == 0 {
		for _, ccI := range d.index.SearchIntersect(c.Centroid().Bounds()) {
			cc := ccI.(*Cell)
			if cc.Layer > 0 {
				deleteCellFromSlice(c, &cc.groundLevel, &cc.groundLevelFrac)
			}
		}
	}

}

// deleteCellFromSlice deletes a cell from a slice of cells and from a
// set of matching slices of float64s.
func deleteCellFromSlice(c *Cell, a *[]*Cell, aFrac ...*[]float64) {
	for i, ac := range *a {
		if c == ac {
			(*a)[i] = (*a)[len(*a)-1]
			(*a)[len(*a)-1] = nil
			*a = (*a)[:len(*a)-1]

			for _, af := range aFrac {
				*af = append((*af)[:i], (*af)[i+1:]...)
			}
		}
	}
}

func newRect(xmin, ymin, xmax, ymax float64) *geom.Bounds {
	return &geom.Bounds{
		Min: geom.Point{X: xmin, Y: ymin},
		Max: geom.Point{X: xmax, Y: ymax},
	}
}
