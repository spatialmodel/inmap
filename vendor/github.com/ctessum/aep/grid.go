/*
Copyright (C) 2012-2014 Regents of the University of Minnesota.
This file is part of AEP.

AEP is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

AEP is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with AEP.  If not, see <http://www.gnu.org/licenses/>.
*/

package aep

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"bitbucket.org/ctessum/sparse"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/op"
	"github.com/ctessum/geom/proj"
	goshp "github.com/jonas-p/go-shp"
)

func init() {
	gob.Register(geom.Polygon{})
}

// GridDef specifies the grid that we are allocating the emissions to.
type GridDef struct {
	Name          string
	Nx, Ny        int
	Dx, Dy        float64
	X0, Y0        float64
	TimeZones     map[string]*sparse.SparseArray
	Cells         []*GridCell
	SR            *proj.SR
	Extent        geom.Polygonal
	IrregularGrid bool // whether the grid is a regular grid
	rtree         *rtree.Rtree
}

// GridCell defines an individual cell in a grid.
type GridCell struct {
	geom.Polygonal
	Row, Col int
	Weight   float64
	TimeZone string
}

// Copy copies a grid cell.
func (c *GridCell) Copy() *GridCell {
	o := new(GridCell)
	o.Polygonal = c.Polygonal
	o.Row, o.Col = c.Row, c.Col
	return o
}

// NewGridRegular creates a new regular grid, where all grid cells are the
// same size.
func NewGridRegular(Name string, Nx, Ny int, Dx, Dy, X0, Y0 float64,
	sr *proj.SR) (grid *GridDef) {
	grid = new(GridDef)
	grid.Name = Name
	grid.Nx, grid.Ny = Nx, Ny
	grid.Dx, grid.Dy = Dx, Dy
	grid.X0, grid.Y0 = X0, Y0
	grid.SR = sr
	grid.rtree = rtree.NewTree(25, 50)
	// Create geometry
	grid.Cells = make([]*GridCell, grid.Nx*grid.Ny)
	i := 0
	for ix := 0; ix < grid.Nx; ix++ {
		for iy := 0; iy < grid.Ny; iy++ {
			cell := new(GridCell)
			x := grid.X0 + float64(ix)*grid.Dx
			y := grid.Y0 + float64(iy)*grid.Dy
			cell.Row, cell.Col = iy, ix
			cell.Polygonal = geom.Polygon([][]geom.Point{{
				{x, y}, {x + grid.Dx, y},
				{x + grid.Dx, y + grid.Dy}, {x, y + grid.Dy}, {x, y}}})
			grid.rtree.Insert(cell)
			grid.Cells[i] = cell
			i++
		}
	}
	grid.Extent = geom.Polygon([][]geom.Point{{{X0, Y0},
		{X0 + Dx*float64(Nx), Y0},
		{X0 + Dx*float64(Nx), Y0 + Dy*float64(Ny)},
		{X0, Y0 + Dy*float64(Ny)}, {X0, Y0}}})
	return
}

// NewGridIrregular creates a new irregular grid. g is the grid geometry.
// Irregular grids have 1 column and n rows, where n is the number of
// shapes in g. inputSR is the spatial reference of g, and outputSR is the
// desired spatial reference of the grid.
func NewGridIrregular(Name string, g []geom.Polygonal, inputSR, outputSR *proj.SR) (grid *GridDef, err error) {
	grid = new(GridDef)
	grid.Name = Name
	grid.SR = outputSR
	grid.IrregularGrid = true
	grid.Cells = make([]*GridCell, len(g))
	grid.Ny = len(g)
	grid.Nx = 1
	var ct proj.Transformer
	ct, err = inputSR.NewTransform(outputSR)
	if err != nil {
		return
	}
	grid.rtree = rtree.NewTree(25, 50)
	for i, gg := range g {
		cell := new(GridCell)
		var gg2 geom.Geom
		gg2, err = gg.Transform(ct)
		if err != nil {
			return
		}
		cell.Polygonal = gg2.(geom.Polygonal)
		cell.Row = i
		grid.Cells[i] = cell

		if grid.Extent == nil {
			grid.Extent = cell.Polygonal
		} else {
			grid.Extent = grid.Extent.Union(cell.Polygonal)
		}
		grid.rtree.Insert(cell)
	}
	grid.Extent = grid.Extent.Simplify(1.e-8).(geom.Polygonal)
	return
}

// GetTimeZones gets the time zone of each grid cell.
func (grid *GridDef) GetTimeZones(tzFile, tzColumn string) error {

	timezones, tzsr, err := getTimeZones(tzFile, tzColumn)
	if err != nil {
		return err
	}
	grid.TimeZones = make(map[string]*sparse.SparseArray)

	ct, err := grid.SR.NewTransform(tzsr)
	if err != nil {
		return err
	}

	var lock sync.Mutex
	errChan := make(chan error)
	nprocs := runtime.GOMAXPROCS(-1)
	for proc := 0; proc < nprocs; proc++ {
		go func(proc int) {
			var err2 error
			var cellCenter geom.Geom
			for ii := proc; ii < len(grid.Cells); ii += nprocs {
				cell := grid.Cells[ii]
				// find timezone nearest to the center of the cell.
				// Need to project grid to timezone projection rather than the
				// other way around because the timezones can include the north
				// and south poles which don't convert well to other projections.
				cellCenter, err2 = op.Centroid(cell.Polygonal)
				if err2 != nil {
					errChan <- err2
					return
				}
				cellCenter, err2 = cellCenter.Transform(ct)
				if err2 != nil {
					errChan <- err2
					return
				}
				var tz string
				var foundtz bool
				for _, tzDataI := range timezones.SearchIntersect(cellCenter.Bounds()) {
					tzData := tzDataI.(*tzHolder)
					intersects := cellCenter.(geom.Point).Within(tzData.Polygonal)
					if intersects == geom.Inside || intersects == geom.OnEdge {
						if foundtz {
							panic("In spatialsrg.GetTimeZones, there is a " +
								"grid cell that overlaps with more than one timezone." +
								" This probably shouldn't be happening.")
						}
						tz = tzData.tz
						foundtz = true
					}
				}
				if !foundtz {
					//err = fmt.Errorf("In spatialsrg.GetTimeZones, there is a " +
					//	"grid cell that doesn't match any timezones")
					//return
					tz = "UTC" // The timezone shapefile doesn't include timezones
					// over the ocean, so we assume all timezones that don't have
					// tz info are UTC.
				}
				cell.TimeZone = tz
				lock.Lock()
				if _, ok := grid.TimeZones[tz]; !ok {
					grid.TimeZones[tz] = sparse.ZerosSparse(grid.Ny, grid.Nx)
				}
				grid.TimeZones[tz].Set(1., cell.Row, cell.Col)
				lock.Unlock()
			}
			errChan <- nil
		}(proc)
	}
	for procnum := 0; procnum < nprocs; procnum++ {
		if err = <-errChan; err != nil {
			return err
		}
	}
	return nil
}

type tzHolder struct {
	tz string
	geom.Polygonal
}

func getTimeZones(tzFile, tzColumn string) (*rtree.Rtree, *proj.SR, error) {
	timezones := rtree.NewTree(25, 50)

	tzShp, err := shp.NewDecoder(tzFile)
	if err != nil {
		return nil, nil, err
	}
	defer tzShp.Close()
	sr, err := tzShp.SR()
	if err != nil {
		return nil, nil, err
	}
	for {
		g, fields, more := tzShp.DecodeRowFields(tzColumn)
		if !more {
			break
		}

		tzData := new(tzHolder)
		tzData.Polygonal = g.(geom.Polygonal)
		tzData.tz = fields[tzColumn]
		timezones.Insert(tzData)
	}
	return timezones, sr, tzShp.Error()
}

// GetIndex gets the returns the row and column indices of point p in the grid.
// withinGrid is false if point (X,Y) is not within the grid. Usually
// there will be only one row and column for each point, but it the point
// lies on a shared edge among multiple grid cells, all of the overlapping
// grid cells will be returned.
func (grid *GridDef) GetIndex(p geom.Point) (rows, cols []int, withinGrid bool, err error) {
	for _, cI := range grid.rtree.SearchIntersect(p.Bounds()) {
		c := cI.(*GridCell)
		if grid.IrregularGrid && p.Within(c.Polygonal) == geom.Outside {
			continue
		}
		rows = append(rows, c.Row)
		cols = append(cols, c.Col)
	}
	if len(rows) > 0 {
		withinGrid = true
	}
	return
}

// WriteToShp writes the grid definition to a shapefile in directory outdir.
func (grid *GridDef) WriteToShp(outdir string) error {
	var err error
	for _, ext := range []string{".shp", ".prj", ".dbf", ".shx"} {
		os.Remove(filepath.Join(outdir, grid.Name+ext))
	}
	fields := make([]goshp.Field, 2)
	fields[0] = goshp.NumberField("row", 10)
	fields[1] = goshp.NumberField("col", 10)
	var shpf *shp.Encoder
	shpf, err = shp.NewEncoderFromFields(filepath.Join(outdir, grid.Name+".shp"),
		goshp.POLYGON, fields...)
	if err != nil {
		return err
	}
	for _, cell := range grid.Cells {
		data := []interface{}{cell.Row, cell.Col}
		err = shpf.EncodeFields(cell.Polygonal, data...)
		if err != nil {
			return err
		}
	}
	shpf.Close()
	return nil
}
