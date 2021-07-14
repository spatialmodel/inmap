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
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/requestcache/v4"
	"github.com/ctessum/sparse"
	"github.com/spatialmodel/inmap/internal/hash"
)

type srgGenWorker struct {
	surrogates SearchIntersecter
	GridCells  *GridDef

	// srgCellRatio is the number of surrogate shapes to process per
	// grid cell for each input shape. Larger numbers require
	// longer to compute. If srgCellRatio > 1, all surrogate
	// shapes will be processed.
	srgCellRatio int
}

type srgGenWorkerInitData struct {
	Surrogates SearchIntersecter
	GridCells  *GridDef
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
func (sp *SpatialProcessor) createMerged(srg SrgSpec, gridData *GridDef, loc *Location, result requestcache.Result) error {
	mrgSrgs := make([]*GriddedSrgData, len(srg.mergeNames()))
	for i, mrgName := range srg.mergeNames() {
		newSrg, err := sp.SrgSpecs.GetByName(srg.region(), mrgName)
		if err != nil {
			return err
		}
		sg := &srgGrid{srg: newSrg, gridData: gridData, loc: loc, sp: sp}
		req := sp.cache.NewRequestRecursive(context.Background(), sg)
		data := new(GriddedSrgData)
		if err := req.Result((*griddedSrgDataHolder)(data)); err != nil {
			return err
		}
		mrgSrgs[i] = data
	}
	res := mergeSrgs(mrgSrgs, srg.mergeMultipliers())
	resH := result.(*griddedSrgDataHolder)
	res2 := (*griddedSrgDataHolder)(res)
	*resH = *res2
	return nil
}

// srgGrid holds a surrogate specification and a grid definition.
type srgGrid struct {
	srg      SrgSpec
	gridData *GridDef
	loc      *Location
	sp       *SpatialProcessor
	msgChan  chan string
}

func (sg *srgGrid) Key() string {
	return fmt.Sprintf("surrogate_%s%s_%s_%s", sg.srg.region(), sg.srg.code(),
		sg.gridData.Name, sg.loc.String())
}

// Run creates a new gridding surrogate based on a
// surrogate specification and grid definition.
func (sg *srgGrid) Run(_ context.Context, _ *requestcache.Cache, res requestcache.Result) error {
	srg := sg.srg
	gridData := sg.gridData
	sp := sg.sp
	loc := sg.loc
	if loc == nil {
		return fmt.Errorf("aep.SpatialProcessor.createSurrogate: missing location: %+v", gridData)
	}
	if len(srg.mergeNames()) != 0 {
		return sp.createMerged(srg, gridData, loc, res)
	}
	if sg.msgChan != nil {
		sg.msgChan <- fmt.Sprintf("creating surrogate `%s` for location %s", srg.name(), loc)
	}

	srgData, err := srg.getSrgData(gridData, loc, sp.SimplifyTolerance)
	if err != nil {
		return err
	}

	// Start workers
	nprocs := runtime.GOMAXPROCS(0)
	singleShapeChan := make(chan *GriddedSrgData, nprocs*2)
	griddedSrgChan := make(chan *GriddedSrgData, nprocs*2)
	errchan := make(chan error, nprocs*2)
	workersRunning := 0
	for i := 0; i < nprocs; i++ {
		go genSrgWorker(singleShapeChan, griddedSrgChan, errchan, gridData, srgData, sg.sp.SrgCellRatio)
		workersRunning++
	}

	singleShapeData := &GriddedSrgData{InputLocation: loc}
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
			return err
		}
	}
	rr := res.(*griddedSrgDataHolder)
	grdsrg2 := (*griddedSrgDataHolder)(grdsrg)
	*rr = *grdsrg2
	return nil
}

// WriteToShp write an individual gridding surrogate to a shapefile.
func (g *GriddedSrgData) WriteToShp(file string) error {
	covered := "F"
	if g.CoveredByGrid {
		covered = "T"
	}
	s, err := shp.NewEncoder(file, struct {
		geom.Polygon
		Row, Col int
		InputID  string
		Weight   float64
		Covered  string
	}{})
	if err != nil {
		return fmt.Errorf("aep: creating shapefile to write gridding surrogate: %v", err)
	}

	for _, cell := range g.Cells {
		err := s.EncodeFields(cell.Polygonal,
			cell.Row, cell.Col, hash.Hash(g.InputLocation), cell.Weight, covered)
		if err != nil {
			return err
		}
	}
	s.Close()
	return nil
}

func genSrgWorker(singleShapeChan, griddedSrgChan chan *GriddedSrgData,
	errchan chan error, gridData *GridDef, srgData SearchIntersecter, srgCellRatio int) {
	var err error

	s := new(srgGenWorker)
	s.srgCellRatio = srgCellRatio

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
	within := inputGeom.(geom.Withiner).Within(s.GridCells.Extent)
	result.CoveredByGrid = within == geom.Inside || within == geom.OnEdge

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

type SearchIntersecter interface {
	SearchIntersect(*geom.Bounds) []geom.Geom
}

// Calculate the intersections between the grid cells and the input shape,
// and between the surrogate shapes and the input shape
func (s *srgGenWorker) intersections1(
	data *GriddedSrgData, surrogates SearchIntersecter, inputGeom geom.Polygonal) (
	GridCells []*GridCell, srgs []*srgHolder,
	singleShapeSrgWeight float64, err error) {

	nprocs := runtime.GOMAXPROCS(0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Figure out which grid cells might intersect with the input shape
	inputBounds := inputGeom.Bounds()
	GridCells = make([]*GridCell, 0, 30)
	for _, gcI := range s.GridCells.rtree.SearchIntersect(inputBounds) {
		GridCells = append(GridCells, gcI.(*GridCell))
	}

	// get all of the surrogates which intersect with the input
	// shape, and save only the intersecting parts.
	singleShapeSrgWeight = 0.
	srgs = make([]*srgHolder, 0, 500)
	wg.Add(nprocs)
	srgsWithinBounds := s.surrogates.SearchIntersect(inputBounds)

	// We want to limit the number of surrogates we're processing
	// so that it's about 10 per grid cell.
	srgMod := 1
	if s.srgCellRatio > 0 {
		if srgRatio := len(srgsWithinBounds) / len(GridCells); srgRatio > s.srgCellRatio {
			srgMod = srgRatio / s.srgCellRatio
			if srgMod < 1 {
				srgMod = 1
			}
		}
	}

	inputGeomArea := inputGeom.Area()
	errChan := make(chan error)
	for procnum := 0; procnum < nprocs; procnum++ {
		go func(procnum int) {
			for i := procnum; i < len(srgsWithinBounds); i += nprocs {
				if i%srgMod != 0 {
					continue
				}
				srg := srgsWithinBounds[i].(*srgHolder)
				intersection := intersection(srg.Geom, inputGeom, inputGeomArea)
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
func intersection(g geom.Geom, poly geom.Polygonal, polyArea float64) geom.Geom {
	switch g.(type) {
	case geom.Point, geom.MultiPoint:
		o := make(geom.MultiPoint, 0, g.Len())
		ptsF := g.Points()
		for i := 0; i < g.Len(); i++ {
			pt := ptsF()
			in := pt.Within(poly)
			if in == geom.Inside || in == geom.OnEdge {
				o = append(o, pt)
			}
		}
		if len(o) > 0 {
			return o
		}
		return nil
	case geom.Polygonal:
		p := g.(geom.Polygonal)
		if a := p.Area(); a > 0 && a < polyArea/50 {
			// If p is small compared to poly, and the centroid of p is
			// within poly, return p.
			if in := p.Centroid().Within(poly); in == geom.Inside {
				return p
			}
			return nil
		}
		return p.Intersection(poly)
	case geom.Linear:
		l := g.(geom.Linear)
		if length := l.Length(); length > 0 && length < math.Sqrt(polyArea)/50 {
			// If l is small compared to poly, and the first point in l is within
			// poly, return l.
			if in := l.Points()().Within(poly); in == geom.Inside {
				return l
			}
			return nil
		}
		return l.Clip(poly)
	default:
		panic(fmt.Errorf("unsupported intersection geometry type %#v", g))
	}
}

// geomWeight multiplies w by a relevant property of g.
func geomWeight(w float64, g geom.Geom) float64 {
	switch g.(type) {
	case geom.Polygonal:
		return w * g.(geom.Polygonal).Area()
	case geom.LineString, geom.MultiLineString:
		return w * g.(geom.Linear).Length()
	case geom.Point, geom.MultiPoint:
		return w * float64(g.Len())
	default:
		panic(fmt.Errorf("invalid geometry type %T", g))
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
				cellArea := cell.Area()
				for _, srg := range InputShapeSrgs {
					intersection := intersection(srg.Geom, cell.Polygonal, cellArea)
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

// RecordSpatialSurrogate describes emissions that need to be allocated to a grid
// using a spatial surrogate.
type RecordSpatialSurrogate interface {
	Record

	// Parent returns the record that this record was created from.
	Parent() Record

	// SurrogateSpecification returns the specification of the spatial surrogate
	// associated with an area emissions source.
	SurrogateSpecification() (SrgSpec, error)
}

// AddSurrogate adds a spatial surrogate to a record to increase its
// spatial resolution.
func (sp *SpatialProcessor) AddSurrogate(r Record) RecordSpatialSurrogate {
	return &recordSpatialSurrogate{Record: r, sp: sp}
}

type recordSpatialSurrogate struct {
	Record
	sp *SpatialProcessor
}

// SurrogateSpecification returns the specification of the spatial surrogate
// associated with an area emissions source.
func (r *recordSpatialSurrogate) SurrogateSpecification() (SrgSpec, error) {
	srgNum, err := r.sp.GridRef.GetSrgCode(r.Record.GetSCC(), r.Record.GetCountry(), r.Record.GetFIPS())
	if err != nil {
		return nil, err
	}
	return r.sp.SrgSpecs.GetByCode(r.Record.GetCountry(), srgNum)
}

// Parent returns the record that this record was created from.
func (r *recordSpatialSurrogate) Parent() Record { return r.Record }

// unmarshalSrgHolders unmarshals an interface from a byte array and fulfills
// the requirements for the Disk cache unmarshalFunc input.
func unmarshalSrgHolders(b []byte) (interface{}, error) {
	r := bytes.NewBuffer(b)
	d := gob.NewDecoder(r)
	var data []*srgHolder
	if err := d.Decode(&data); err != nil {
		return nil, err
	}
	o := readSrgDataOutput{
		srgs:  data,
		index: rtree.NewTree(25, 50),
	}
	for _, s := range data {
		o.index.Insert(s)
	}
	return o, nil
}

// marshalSrgHolders marshals an interface to a byte array and fulfills
// the requirements for the Disk cache marshalFunc input.
func marshalSrgHolders(data interface{}) ([]byte, error) {
	w := bytes.NewBuffer(nil)
	e := gob.NewEncoder(w)
	d := *data.(*interface{})
	dd := d.(readSrgDataOutput)
	if err := e.Encode(dd.srgs); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}
