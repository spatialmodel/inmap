/*
Copyright (C) 2012 the InMAP authors.
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

package aep

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/op"
	"github.com/ctessum/sparse"
)

type srgGenWorker struct {
	surrogates *rtree.Rtree
	GridCells  *GridDef
}

type srgGenWorkerInitData struct {
	Surrogates *rtree.Rtree
	GridCells  *GridDef
}

// GriddingSurrogate holds generated gridding surrogate data, and can be
// used to allocate emissions attributed to a relatively large area, such as
// a county, to the grid cells within that area.
type GriddingSurrogate struct {
	// Srg holds surrogate data associated with individual input locations.
	Srg map[string]*GriddedSrgData

	// Nx and Ny are the number of columns and rows in the grid
	Nx, Ny int
}

// ToGrid allocates the 1 unit of emissions associated with shapeID to a grid
// based on gs. It will return nil if there is no surrogate for the specified
// shapeID or if the sum of the surrogate is zero. The second returned value
// indicates whether the shape corresponding to shapeID is completely covered
// by the grid.
func (gs *GriddingSurrogate) ToGrid(shapeID string) (*sparse.SparseArray, bool) {
	srg, ok := gs.Srg[shapeID]
	if !ok {
		return nil, false
	}
	srgOut := sparse.ZerosSparse(gs.Ny, gs.Nx)
	for _, cell := range srg.Cells {
		srgOut.AddVal(cell.Weight, cell.Row, cell.Col)
	}
	sum := srgOut.Sum()
	if sum == 0 {
		return nil, false
	}
	// normalize so sum = 1 if the input shape is completely covered by the
	// grid.
	if srg.CoveredByGrid {
		srgOut.Scale(1. / sum)
	}
	return srgOut, srg.CoveredByGrid
}

// ToGrid allocates the 1 unit of emissions associated with shapeID to a grid
// based on gs. It will return nil if there is no surrogate for the specified
// shapeID or if the sum of the surrogate is zero. The second returned value
// indicates whether the shape corresponding to shapeID is completely covered
// by the grid.
func (gs *GriddedSrgData) ToGrid() (*sparse.SparseArray, bool) {
	srgOut := sparse.ZerosSparse(gs.Ny, gs.Nx)
	for _, cell := range gs.Cells {
		srgOut.AddVal(cell.Weight, cell.Row, cell.Col)
	}
	sum := srgOut.Sum()
	if sum == 0 {
		return nil, false
	}
	// normalize so sum = 1 if the input shape is completely covered by the
	// grid.
	if gs.CoveredByGrid {
		srgOut.Scale(1. / sum)
	}
	return srgOut, gs.CoveredByGrid
}

// mergeSrgs merges a number of surrogates, multiplying each of them by the
// corresponding factor.
func mergeSrgs(srgs []*GriddedSrgData, factors []float64) *GriddedSrgData {
	o := new(GriddedSrgData)
	o.Nx, o.Ny = srgs[0].Nx, srgs[0].Ny
	for i, g := range srgs {
		fac := factors[i]
		if o.InputLocation == nil {
			o.InputLocation = g.InputLocation
			o.CoveredByGrid = g.CoveredByGrid
		}
		for _, cell := range g.Cells {
			o.Cells = append(o.Cells, &GridCell{
				Row:       cell.Row,
				Col:       cell.Col,
				Weight:    cell.Weight * fac,
				Polygonal: cell.Polygonal,
			})
		}
	}
	return o
}

// GriddedSrgData holds the data for a single input shape of a gridding surrogate.
type GriddedSrgData struct {
	InputLocation        *Location
	Cells                []*GridCell
	SingleShapeSrgWeight float64
	CoveredByGrid        bool
	Nx, Ny               int
}

type srgHolder struct {
	Weight float64
	geom.Geom
}

// SurrogateFilter can be used to limit which rows in a shapefile are
// used to create a gridding surrogate.
type SurrogateFilter struct {
	Column        string
	EqualNotEqual string
	Values        []string
}

// ParseSurrogateFilter creates a new surrogate filter object from a
// SMOKE-format spatial surrogate filter definition.
func ParseSurrogateFilter(filterFunction string) *SurrogateFilter {
	if filterFunction != none && filterFunction != "" {
		srgflt := new(SurrogateFilter)
		srgflt.Values = make([]string, 0)
		var s []string
		if strings.Index(filterFunction, "!=") != -1 {
			srgflt.EqualNotEqual = "NotEqual"
			s = strings.Split(filterFunction, "!=")
		} else {
			srgflt.EqualNotEqual = "Equal"
			s = strings.Split(filterFunction, "=")
		}
		srgflt.Column = strings.TrimSpace(s[0])
		splitstr := strings.Split(s[1], ",")
		for _, val := range splitstr {
			srgflt.Values = append(srgflt.Values,
				strings.TrimSpace(val))
		}
		return srgflt
	}
	return nil
}

// createMerged creates a surrogate by creating and merging other surrogates.
func (sp *SpatialProcessor) createMerged(srg *SrgSpec, gridData *GridDef, loc *Location) (*GriddedSrgData, error) {
	mrgSrgs := make([]*GriddedSrgData, len(srg.MergeNames))
	for i, mrgName := range srg.MergeNames {
		newSrg, err := sp.SrgSpecs.GetByName(srg.Region, mrgName)
		if err != nil {
			return nil, err
		}
		// If we use the cache here it is possible to end up with a channel deadlock,
		// so we generate the surrogate from scratch here.
		data, err := sp.createSurrogate(context.Background(), &srgGrid{srg: newSrg, gridData: gridData, loc: loc})
		if err != nil {
			return nil, err
		}
		mrgSrgs[i] = data.(*GriddedSrgData)
	}
	return mergeSrgs(mrgSrgs, srg.MergeMultipliers), nil
}

// srgGrid holds a surrogate specification and a grid definition.
type srgGrid struct {
	srg      *SrgSpec
	gridData *GridDef
	loc      *Location
}

// key returns a unique key for this surrogate request.
func (s *srgGrid) key() string {
	return fmt.Sprintf("%s_%s_%s_%s", s.srg.Region, s.srg.Code, s.gridData.Name, s.loc.Key())
}

// createSurrogate creates a new gridding surrogate based on a
// surrogate specification and grid definition.
func (sp *SpatialProcessor) createSurrogate(_ context.Context, inData interface{}) (interface{}, error) {
	in := inData.(*srgGrid)
	srg := in.srg
	gridData := in.gridData
	if in.loc == nil {
		return nil, fmt.Errorf("aep.SpatialProcessor.createSurrogate: missing location: %+v", gridData)
	}
	if len(srg.MergeNames) != 0 {
		return sp.createMerged(srg, gridData, in.loc)
	}

	srgData, err := srg.getSrgData(gridData, in.loc, sp.SimplifyTolerance)
	if err != nil {
		return nil, err
	}

	// Start workers
	nprocs := runtime.GOMAXPROCS(0)
	singleShapeChan := make(chan *GriddedSrgData, nprocs*2)
	griddedSrgChan := make(chan *GriddedSrgData, nprocs*2)
	errchan := make(chan error, nprocs*2)
	workersRunning := 0
	for i := 0; i < nprocs; i++ {
		go genSrgWorker(singleShapeChan, griddedSrgChan, errchan, gridData, srgData)
		workersRunning++
	}

	singleShapeData := &GriddedSrgData{InputLocation: in.loc}
	singleShapeChan <- singleShapeData
	close(singleShapeChan)
	// wait for remaining results
	grdsrg := <-griddedSrgChan
	grdsrg.Nx = gridData.Nx
	grdsrg.Ny = gridData.Ny
	// wait for workers to finish
	for i := 0; i < workersRunning; i++ {
		err = <-errchan
		if err != nil {
			return nil, err
		}
	}
	return grdsrg, nil
}

// WriteToShp write an individual gridding surrogate to a shapefile.
func (g *GriddedSrgData) WriteToShp(s *shp.Encoder) error {
	covered := "F"
	if g.CoveredByGrid {
		covered = "T"
	}
	for _, cell := range g.Cells {
		err := s.EncodeFields(cell.Polygonal,
			cell.Row, cell.Col, g.InputLocation.Key(), cell.Weight, covered)
		if err != nil {
			return err
		}
	}
	return nil
}

// get surrogate shapes and weights. tol is a geometry simplification tolerance.
func (srg *SrgSpec) getSrgData(gridData *GridDef, inputLoc *Location, tol float64) (*rtree.Rtree, error) {
	srgShp, err := shp.NewDecoder(srg.WEIGHTSHAPEFILE)
	if err != nil {
		return nil, err
	}
	defer srgShp.Close()

	srgSR, err := srgShp.SR()
	if err != nil {
		return nil, err
	}

	srgCT, err := srgSR.NewTransform(gridData.SR)
	if err != nil {
		return nil, err
	}

	gridSrgCT, err := gridData.SR.NewTransform(srgSR)
	if err != nil {
		return nil, err
	}

	// Calculate the area of interest for our surrogate data.
	inputShapeT, err := inputLoc.Reproject(srgSR)
	if err != nil {
		return nil, err
	}
	inputShapeBounds := inputShapeT.Bounds()
	srgBounds := inputShapeBounds.Copy()
	for _, cell := range gridData.Cells {
		cellT, err := cell.Transform(gridSrgCT)
		if err != nil {
			return nil, err
		}
		b := cellT.Bounds()
		if b.Overlaps(inputShapeBounds) {
			srgBounds.Extend(b)
		}
	}

	var fieldNames []string
	if srg.FilterFunction != nil {
		fieldNames = append(fieldNames, srg.FilterFunction.Column)
	}
	if srg.WeightColumns != nil {
		fieldNames = append(fieldNames, srg.WeightColumns...)
	}
	srgData := rtree.NewTree(25, 50)
	var recGeom geom.Geom
	var data map[string]string
	var keepFeature bool
	var featureVal string
	var size float64
	var more bool
	for {
		recGeom, data, more = srgShp.DecodeRowFields(fieldNames...)
		if !more {
			break
		}

		if !recGeom.Bounds().Overlaps(srgBounds) {
			continue
		}

		if srg.FilterFunction == nil {
			keepFeature = true
		} else {
			// Determine whether this feature should be kept according to
			// the filter function.
			keepFeature = false
			featureVal = strings.TrimSpace(fmt.Sprintf("%v", data[srg.FilterFunction.Column]))
			for _, filterVal := range srg.FilterFunction.Values {
				switch srg.FilterFunction.EqualNotEqual {
				case "NotEqual":
					if featureVal != filterVal {
						keepFeature = true
					}
				default:
					if featureVal == filterVal {
						keepFeature = true
					}
				}
			}
		}
		if keepFeature && recGeom != nil {
			srgH := new(srgHolder)
			srgH.Geom, err = recGeom.Transform(srgCT)
			if err != nil {
				return srgData, err
			}
			if tol > 0 {
				switch srgH.Geom.(type) {
				case geom.Simplifier:
					srgH.Geom = srgH.Geom.(geom.Simplifier).Simplify(tol)
				}
			}
			if len(srg.WeightColumns) != 0 {
				weightval := 0.
				for i, name := range srg.WeightColumns {
					var v float64
					vstring := data[name]
					if strings.Contains(vstring, "\x00\x00\x00\x00\x00\x00") || strings.Contains(vstring, "***") || vstring == "" {
						// null value
						v = 0.
					} else {
						v, err = strconv.ParseFloat(data[name], 64)
						if err != nil {
							return srgData, fmt.Errorf("aep.getSrgData: shapefile %s column %s, %v", srg.WEIGHTSHAPEFILE, name, err)
						}
						v = math.Max(v, 0) // Get rid of any negative weights.
					}
					weightval += v * srg.WeightFactors[i]
				}
				switch srgH.Geom.(type) {
				case geom.Polygonal:
					size = srgH.Geom.(geom.Polygonal).Area()
					if size == 0. {
						if tol > 0 {
							// We probably simplified the shape down to zero area.
							continue
						} else {
							// TODO: Is it okay for input shapes to have zero area? Probably....
							continue
							//err = fmt.Errorf("Area should not equal "+
							//	"zero in %v", srg.WEIGHTSHAPEFILE)
							//return srgData, err
						}
					} else if size < 0 {
						panic(fmt.Errorf("negative area: %g, geom:%#v", size, srgH.Geom))
					}
					srgH.Weight = weightval / size
				case geom.Linear:
					size = srgH.Geom.(geom.Linear).Length()
					if size == 0. {
						err = fmt.Errorf("Length should not equal "+
							"zero in %v", srg.WEIGHTSHAPEFILE)
						return srgData, err
					}
					srgH.Weight = weightval / size
				case geom.Point:
					srgH.Weight = weightval
				default:
					err = fmt.Errorf("aep: in file %s, unsupported geometry type %#v",
						srg.WEIGHTSHAPEFILE, srgH.Geom)
					return srgData, err
				}
			} else {
				srgH.Weight = 1.
			}
			if srgH.Weight < 0. || math.IsInf(srgH.Weight, 0) ||
				math.IsNaN(srgH.Weight) {
				err = fmt.Errorf("Surrogate weight is %v, which is not acceptable.", srgH.Weight)
				return srgData, err
			} else if srgH.Weight != 0. {
				srgData.Insert(srgH)
			}
		}
	}
	if srgShp.Error() != nil {
		return nil, fmt.Errorf("in file %s, %v", srg.WEIGHTSHAPEFILE, srgShp.Error())
	}
	return srgData, nil
}

func genSrgWorker(singleShapeChan, griddedSrgChan chan *GriddedSrgData,
	errchan chan error, gridData *GridDef, srgData *rtree.Rtree) {
	var err error

	s := new(srgGenWorker)

	var data *GriddedSrgData
	first := true
	for data = range singleShapeChan {
		if first {
			d := &srgGenWorkerInitData{srgData, gridData}
			err = s.Initialize(d) // Load data (only do once)
			if err != nil {
				errchan <- err
				return
			}
			first = false
		}
		result := new(GriddedSrgData)
		err = s.Calculate(data, result)
		if err != nil {
			errchan <- err
		}
		griddedSrgChan <- result
	}
	errchan <- err
}

func (s *srgGenWorker) Initialize(data *srgGenWorkerInitData) error {
	s.surrogates = data.Surrogates
	s.GridCells = data.GridCells
	return nil
}

// Set up to allow distributed computing through RPC
func (s *srgGenWorker) Calculate(data, result *GriddedSrgData) (err error) {
	result.InputLocation = data.InputLocation

	inputGeom, err := data.InputLocation.Reproject(s.GridCells.SR)
	if err != nil {
		return err
	}

	// Figure out if inputShape is completely within the grid
	result.CoveredByGrid, err = op.Within(inputGeom, s.GridCells.Extent)
	if err != nil {
		return
	}

	var GridCells []*GridCell
	var InputShapeSrgs []*srgHolder
	GridCells, InputShapeSrgs, data.SingleShapeSrgWeight, err =
		s.intersections1(data, s.surrogates, inputGeom.(geom.Polygonal))
	if err != nil {
		return
	}

	if data.SingleShapeSrgWeight != 0. {
		result.Cells, err = s.intersections2(data, InputShapeSrgs, GridCells)
		if err != nil {
			return
		}
	}
	return
}

// Calculate the intersections between the grid cells and the input shape,
// and between the surrogate shapes and the input shape
func (s *srgGenWorker) intersections1(
	data *GriddedSrgData, surrogates *rtree.Rtree, inputGeom geom.Polygonal) (
	GridCells []*GridCell, srgs []*srgHolder,
	singleShapeSrgWeight float64, err error) {

	nprocs := runtime.GOMAXPROCS(0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Figure out which grid cells might intersect with the input shape
	inputBounds := inputGeom.Bounds()
	GridCells = make([]*GridCell, 0, 30)
	wg.Add(nprocs)
	for procnum := 0; procnum < nprocs; procnum++ {
		go func(procnum int) {
			defer wg.Done()
			var intersects bool
			for i := procnum; i < len(s.GridCells.Cells); i += nprocs {
				cell := s.GridCells.Cells[i]
				intersects = cell.Polygonal.Bounds().Overlaps(inputBounds)
				if intersects {
					mu.Lock()
					GridCells = append(GridCells, cell)
					mu.Unlock()
				}
			}
		}(procnum)
	}
	wg.Wait()

	// get all of the surrogates which intersect with the input
	// shape, and save only the intersecting parts.
	singleShapeSrgWeight = 0.
	srgs = make([]*srgHolder, 0, 500)
	wg.Add(nprocs)
	srgsWithinBounds := s.surrogates.SearchIntersect(inputBounds)
	errChan := make(chan error)
	for procnum := 0; procnum < nprocs; procnum++ {
		go func(procnum int) {
			for i := procnum; i < len(srgsWithinBounds); i += nprocs {
				srg := srgsWithinBounds[i].(*srgHolder)
				intersection := intersection(srg.Geom, inputGeom)
				if intersection == nil {
					continue
				}
				mu.Lock()
				srgs = append(srgs, &srgHolder{Weight: srg.Weight,
					Geom: intersection})
				// Add the individual surrogate weight to the total
				// weight for the input shape.
				singleShapeSrgWeight += geomWeight(srg.Weight, intersection)
				mu.Unlock()
			}
			errChan <- nil
		}(procnum)
	}
	for procnum := 0; procnum < nprocs; procnum++ {
		if err = <-errChan; err != nil {
			return
		}
	}
	return
}

// intersection calculates the intersection of g and poly
func intersection(g geom.Geom, poly geom.Polygonal) geom.Geom {
	var intersection geom.Geom
	switch g.(type) {
	case geom.Point:
		in := g.(geom.Point).Within(poly)
		if in == geom.Inside || in == geom.OnEdge {
			intersection = g
		} else {
			return nil
		}
	case geom.Polygonal:
		intersection = g.(geom.Polygonal).Intersection(poly)
	case geom.Linear:
		intersection = g.(geom.Linear).Clip(poly)
	default:
		panic(fmt.Errorf("unsupported intersection geometry type %#v", g))
	}
	return intersection
}

// geomWeight multiplies w by a relevant property of g.
func geomWeight(w float64, g geom.Geom) float64 {
	switch g.(type) {
	case geom.Polygonal:
		return w * g.(geom.Polygonal).Area()
	case geom.LineString, geom.MultiLineString:
		return w * g.(geom.Linear).Length()
	case geom.Point:
		return w
	default:
		panic(op.UnsupportedGeometryError{G: g})
	}
}

// Given the surrogate shapes that are within an input shape,
// find the surrogate shapes that are within an individual grid
// cell. This function updates the values in `GridCells`.
func (s *srgGenWorker) intersections2(data *GriddedSrgData,
	InputShapeSrgs []*srgHolder, GridCells []*GridCell) (
	result []*GridCell, err error) {

	nprocs := runtime.GOMAXPROCS(0)
	var mu sync.Mutex
	result = make([]*GridCell, 0, len(GridCells))

	errChan := make(chan error)
	for procnum := 0; procnum < nprocs; procnum++ {
		go func(procnum int) {
			for i := procnum; i < len(GridCells); i += nprocs {
				cell := GridCells[i].Copy()
				for _, srg := range InputShapeSrgs {
					intersection := intersection(srg.Geom, cell.Polygonal)
					if intersection == nil {
						continue
					}
					cell.Weight += geomWeight(srg.Weight, intersection) /
						data.SingleShapeSrgWeight
				}
				mu.Lock()
				if cell.Weight > 0. {
					result = append(result, cell)
				}
				mu.Unlock()
			}
			errChan <- nil
		}(procnum)
	}
	for procnum := 0; procnum < nprocs; procnum++ {
		if err = <-errChan; err != nil {
			return
		}
	}
	return
}
