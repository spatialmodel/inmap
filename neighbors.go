package inmap

import "github.com/ctessum/geom"

func (d *InMAPdata) setNeighbors(c *Cell, bboxOffset float64) {
	d.neighbors(c, bboxOffset)

	if len(c.West) == 0 {
		d.addWestBoundary(c)
	}
	if len(c.East) == 0 {
		d.addEastBoundary(c)
	}
	if len(c.North) == 0 {
		d.addNorthBoundary(c)
	}
	if len(c.South) == 0 {
		d.addSouthBoundary(c)
	}
	if len(c.Above) == 0 {
		d.addTopBoundary(c)
	}
	if c.Layer == 0 {
		c.Below = []*Cell{c}
		c.GroundLevel = []*Cell{c}
	}

	c.neighborInfo()
}

func (d *InMAPdata) neighbors(c *Cell, bboxOffset float64) {
	b := c.Bounds()

	// Horizontal
	westbox := newRect(b.Min.X-2*bboxOffset, b.Min.Y+bboxOffset,
		b.Min.X-bboxOffset, b.Max.Y-bboxOffset)
	c.West = getCells(d.index, westbox, c.Layer)
	for _, w := range c.West {
		if len(w.East) == 1 && w.East[0].Boundary {
			deleteCellFromSlice(w.East[0], &d.eastBoundary)
			deleteCellFromSlice(w.East[0], &w.East)
		}
		w.East = append(w.East, c)
		w.neighborInfo()
	}

	eastbox := newRect(b.Max.X+bboxOffset, b.Min.Y+bboxOffset,
		b.Max.X+2*bboxOffset, b.Max.Y-bboxOffset)
	c.East = getCells(d.index, eastbox, c.Layer)
	for _, e := range c.East {
		if len(e.West) == 1 && e.West[0].Boundary {
			deleteCellFromSlice(e.West[0], &d.westBoundary)
			deleteCellFromSlice(e.West[0], &e.West)
		}
		e.West = append(e.West, c)
		e.neighborInfo()
	}

	southbox := newRect(b.Min.X+bboxOffset, b.Min.Y-2*bboxOffset,
		b.Max.X-bboxOffset, b.Min.Y-bboxOffset)
	c.South = getCells(d.index, southbox, c.Layer)
	for _, s := range c.South {
		if len(s.North) == 1 && s.North[0].Boundary {
			deleteCellFromSlice(s.North[0], &d.northBoundary)
			deleteCellFromSlice(s.North[0], &s.North)
		}
		s.North = append(s.North, c)
		s.neighborInfo()
	}

	northbox := newRect(b.Min.X+bboxOffset, b.Max.Y+bboxOffset,
		b.Max.X-bboxOffset, b.Max.Y+2*bboxOffset)
	c.North = getCells(d.index, northbox, c.Layer)
	for _, n := range c.North {
		if len(n.South) == 1 && n.South[0].Boundary {
			deleteCellFromSlice(n.South[0], &d.southBoundary)
			deleteCellFromSlice(n.South[0], &n.South)
		}
		n.South = append(n.South, c)
		n.neighborInfo()
	}

	// Above
	abovebox := newRect(b.Min.X+bboxOffset, b.Min.Y+bboxOffset,
		b.Max.X-bboxOffset, b.Max.Y-bboxOffset)
	c.Above = getCells(d.index, abovebox, c.Layer+1)
	for _, a := range c.Above {
		if len(a.Below) == 1 && a.Below[0] == a {
			deleteCellFromSlice(a.Below[0], &a.Below)
		}
		a.Below = append(a.Below, c)
		a.neighborInfo()
	}

	// Below
	belowbox := newRect(b.Min.X+bboxOffset, b.Min.Y+bboxOffset,
		b.Max.X-bboxOffset, b.Max.Y-bboxOffset)
	c.Below = getCells(d.index, belowbox, c.Layer-1)
	for _, b := range c.Below {
		if len(b.Above) == 1 && b.Above[0].Boundary {
			deleteCellFromSlice(b.Above[0], &d.topBoundary)
			deleteCellFromSlice(b.Above[0], &b.Above)
		}
		b.Above = append(b.Above, c)
		b.neighborInfo()
	}

	// Ground level.
	groundlevelbox := newRect(b.Min.X+bboxOffset, b.Min.Y+bboxOffset,
		b.Max.X-bboxOffset, b.Max.Y-bboxOffset)
	c.GroundLevel = getCells(d.index, groundlevelbox, 0)

	// Find the cells that this cell is the ground level for.
	if c.Layer == 0 {
		for _, ccI := range d.index.SearchIntersect(c.Centroid().Bounds()) {
			cc := ccI.(*Cell)
			if cc.Layer > 0 {
				cc.GroundLevel = append(cc.GroundLevel, c)
				cc.neighborInfo()
			}
		}
	}
}

// Calculate center-to-center cell distance,
// fractions of grid cell covered by each neighbor
// and harmonic mean staggered-grid diffusivities.
func (c *Cell) neighborInfo() {
	c.DxPlusHalf = make([]float64, len(c.East))
	c.EastFrac = make([]float64, len(c.East))
	c.KxxEast = make([]float64, len(c.East))
	for i, e := range c.East {
		c.DxPlusHalf[i] = (c.Dx + e.Dx) / 2.
		c.EastFrac[i] = min(e.Dy/c.Dy, 1.)
		c.KxxEast[i] = harmonicMean(c.Kxxyy, e.Kxxyy)
		e.DxMinusHalf = append(e.DxMinusHalf, c.DxPlusHalf[i])
		e.WestFrac = append(e.WestFrac, c.EastFrac[i])
		e.KxxWest = append(e.KxxWest, c.KxxEast[i])
	}
	c.DxMinusHalf = make([]float64, len(c.West))
	c.WestFrac = make([]float64, len(c.West))
	c.KxxWest = make([]float64, len(c.West))
	for i, w := range c.West {
		c.DxMinusHalf[i] = (c.Dx + w.Dx) / 2.
		c.WestFrac[i] = min(w.Dy/c.Dy, 1.)
		c.KxxWest[i] = harmonicMean(c.Kxxyy, w.Kxxyy)
		w.DxPlusHalf = append(w.DxPlusHalf, c.DxMinusHalf[i])
		w.EastFrac = append(w.EastFrac, c.WestFrac[i])
		w.KxxEast = append(w.KxxEast, c.KxxWest[i])
	}
	c.DyPlusHalf = make([]float64, len(c.North))
	c.NorthFrac = make([]float64, len(c.North))
	c.KyyNorth = make([]float64, len(c.North))
	for i, n := range c.North {
		c.DyPlusHalf[i] = (c.Dy + n.Dy) / 2.
		c.NorthFrac[i] = min(n.Dx/c.Dx, 1.)
		c.KyyNorth[i] = harmonicMean(c.Kxxyy, n.Kxxyy)
		n.DyMinusHalf = append(n.DyMinusHalf, c.DyPlusHalf[i])
		n.SouthFrac = append(n.SouthFrac, c.NorthFrac[i])
		n.KyySouth = append(n.KyySouth, c.KyyNorth[i])
	}
	c.DyMinusHalf = make([]float64, len(c.South))
	c.SouthFrac = make([]float64, len(c.South))
	c.KyySouth = make([]float64, len(c.South))
	for i, s := range c.South {
		c.DyMinusHalf[i] = (c.Dy + s.Dy) / 2.
		c.SouthFrac[i] = min(s.Dx/c.Dx, 1.)
		c.KyySouth[i] = harmonicMean(c.Kxxyy, s.Kxxyy)
		s.DyPlusHalf = append(s.DyPlusHalf, c.DyMinusHalf[i])
		s.NorthFrac = append(s.NorthFrac, c.SouthFrac[i])
		s.KyyNorth = append(s.KyyNorth, c.KyySouth[i])
	}
	c.DzPlusHalf = make([]float64, len(c.Above))
	c.AboveFrac = make([]float64, len(c.Above))
	c.KzzAbove = make([]float64, len(c.Above))
	for i, a := range c.Above {
		c.DzPlusHalf[i] = (c.Dz + a.Dz) / 2.
		c.AboveFrac[i] = min((a.Dx*a.Dy)/(c.Dx*c.Dy), 1.)
		c.KzzAbove[i] = harmonicMean(c.Kzz, a.Kzz)
		a.DzMinusHalf = append(a.DzMinusHalf, c.DzPlusHalf[i])
		a.BelowFrac = append(a.BelowFrac, c.AboveFrac[i])
		a.KzzBelow = append(a.KzzBelow, c.KzzAbove[i])
	}
	c.DzMinusHalf = make([]float64, len(c.Below))
	c.BelowFrac = make([]float64, len(c.Below))
	c.KzzBelow = make([]float64, len(c.Below))
	for i, b := range c.Below {
		c.DzMinusHalf[i] = (c.Dz + b.Dz) / 2.
		c.BelowFrac[i] = min((b.Dx*b.Dy)/(c.Dx*c.Dy), 1.)
		c.KzzBelow[i] = harmonicMean(c.Kzz, b.Kzz)
		b.DzPlusHalf = append(b.DzPlusHalf, c.DzMinusHalf[i])
		b.AboveFrac = append(b.AboveFrac, c.BelowFrac[i])
		b.KzzAbove = append(b.KzzAbove, c.KzzBelow[i])
	}
	c.GroundLevelFrac = make([]float64, len(c.GroundLevel))
	for i, g := range c.GroundLevel {
		c.GroundLevelFrac[i] = min((g.Dx*g.Dy)/(c.Dx*c.Dy), 1.)
	}
}

// dereferenceNeighbors removes any references to this cell that exist in its
// neighbors.
func (c *Cell) dereferenceNeighbors(d *InMAPdata) {
	for _, w := range c.West {
		if w.Boundary {
			deleteCellFromSlice(w, &d.westBoundary)
		} else {
			deleteCellFromSlice(c, &w.East, &w.EastFrac, &w.KxxEast, &w.DxPlusHalf)
		}
	}
	for _, e := range c.East {
		if e.Boundary {
			deleteCellFromSlice(e, &d.eastBoundary)
		} else {
			deleteCellFromSlice(c, &e.West, &e.WestFrac, &e.KxxWest, &e.DxMinusHalf)
		}
	}
	for _, s := range c.South {
		if s.Boundary {
			deleteCellFromSlice(s, &d.southBoundary)
		} else {
			deleteCellFromSlice(c, &s.North, &s.NorthFrac, &s.KyyNorth, &s.DyPlusHalf)
		}
	}
	for _, n := range c.North {
		if n.Boundary {
			deleteCellFromSlice(n, &d.northBoundary)
		} else {
			deleteCellFromSlice(c, &n.South, &n.SouthFrac, &n.KyySouth, &n.DyMinusHalf)
		}
	}
	for _, b := range c.Below {
		deleteCellFromSlice(c, &b.Above, &b.AboveFrac, &b.KzzAbove, &b.DzPlusHalf)
	}
	for _, a := range c.Above {
		if a.Boundary {
			deleteCellFromSlice(a, &d.topBoundary)
		} else {
			deleteCellFromSlice(c, &a.Below, &a.BelowFrac, &a.KzzBelow, &a.DzMinusHalf)
		}
	}

	// Dereference the cells that this cell is the ground level for.
	if c.Layer == 0 {
		for _, ccI := range d.index.SearchIntersect(c.Centroid().Bounds()) {
			cc := ccI.(*Cell)
			if cc.Layer > 0 {
				deleteCellFromSlice(c, &cc.GroundLevel, &cc.GroundLevelFrac)
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
