/*
Copyright (C) 2012-2014 the InMAP authors.
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
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ctessum/geom/proj"
	"github.com/ctessum/requestcache/v4"
	"github.com/ctessum/sparse"
	"github.com/ctessum/unit"

	// Register sqlite drivers
	_ "github.com/mattn/go-sqlite3"
)

// SpatialProcessor spatializes emissions records.
type SpatialProcessor struct {
	SrgSpecs *SrgSpecs
	Grids    []*GridDef
	GridRef  *GridRef

	// inputSR is the spatial reference of the input data. It will usually be
	// "+longlat".
	inputSR *proj.SR

	// matchFullSCC indicates whether partial SCC matches are okay.
	matchFullSCC bool

	cache    *requestcache.Cache
	lazyLoad sync.Once

	// DiskCachePath specifies a directory to cache surrogate files in. If it is
	// empty or invalid, surrogates will not be stored on the disk for later use.
	DiskCachePath string

	// MemCacheSize specifies the number of surrogates to hold in the memory cache.
	// A larger number results in potentially faster performance but more memory use.
	// The default is 100.
	MemCacheSize int

	// MaxMergeDepth is the maximum number of nested merged surrogates.
	// For example, if surrogate 100 is a combination of surrogates 110 and 120,
	// and surrogate 110 is a combination of surrogates 130 and 140, then
	// MaxMergeDepth should be set to 3, because 100 depends on 110 and 110
	// depends on 130. If MaxMergeDepth is set too low, the program may hang
	// when attempting to create a merged surrogate.
	// The default value is 10.
	MaxMergeDepth int

	// SimplifyTolerance specifies the length of features up to which to remove
	// when simplifying shapefiles for spatial surrogate creation. The default is
	// 0 (i.e., no simplification). Simplifying decreases processing time and
	// memory use. The value should be in the units of the output grid
	// (e.g., meters or degrees).
	SimplifyTolerance float64

	// SrgCellRatio is the number of surrogate shapes to process per
	// grid cell for each input shape. Larger numbers require
	// longer to compute. If srgCellRatio > 1, all surrogate
	// shapes will be processed.
	SrgCellRatio int

	MsgChan chan string
}

// NewSpatialProcessor creates a new spatial processor.
func NewSpatialProcessor(srgSpecs *SrgSpecs, grids []*GridDef, gridRef *GridRef, inputSR *proj.SR, matchFullSCC bool) *SpatialProcessor {
	sp := new(SpatialProcessor)
	sp.SrgSpecs = srgSpecs
	sp.Grids = grids
	sp.GridRef = gridRef
	sp.inputSR = inputSR
	sp.matchFullSCC = matchFullSCC

	sp.MemCacheSize = 100
	sp.MaxMergeDepth = 10
	return sp
}

type griddedSrgDataHolder GriddedSrgData

// UnmarshalBinary unmarshals the receiver from a byte array.
func (data *griddedSrgDataHolder) UnmarshalBinary(b []byte) error {
	r := bytes.NewBuffer(b)
	d := gob.NewDecoder(r)
	dd := (*GriddedSrgData)(data)
	return d.Decode(dd)
}

// MarshalBinary marshals the data to a byte array.
func (data *griddedSrgDataHolder) MarshalBinary() ([]byte, error) {
	w := bytes.NewBuffer(nil)
	e := gob.NewEncoder(w)
	if err := e.Encode((*GriddedSrgData)(data)); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (sp *SpatialProcessor) load() error {
	var err error
	sp.cache, err = newCache(sp.DiskCachePath, sp.MemCacheSize)
	return err
}

func newCache(diskCachePath string, memCacheSize int) (*requestcache.Cache, error) {
	dedup := requestcache.Deduplicate()
	mc := requestcache.Memory(memCacheSize)
	if diskCachePath == "" {
		return requestcache.NewCache(dedup, mc), nil
	} else {
		if strings.HasPrefix(diskCachePath, "gs://") {
			loc, err := url.Parse(diskCachePath)
			if err != nil {
				return nil, err
			}
			cf, err := requestcache.GoogleCloudStorage(context.TODO(), loc.Host, strings.TrimLeft(loc.Path, "/"))
			if err != nil {
				return nil, err
			}
			return requestcache.NewCache(dedup, mc, cf), nil
		} else if filepath.Ext(diskCachePath) == ".sqlite3" {
			db, err := sql.Open("sqlite3", diskCachePath)
			if err != nil {
				return nil, err
			}
			cf, err := requestcache.SQL(context.Background(), db)
			if err != nil {
				return nil, err
			}
			return requestcache.NewCache(dedup, mc, cf), nil
		} else {
			if err := os.MkdirAll(diskCachePath, os.ModePerm); err != nil {
				return nil, err
			}
			return requestcache.NewCache(dedup, mc,
				requestcache.Disk(diskCachePath)), nil
		}
	}
}

// RecordGridded represents an emissions record that can be allocated
// to a spatial grid.
type RecordGridded interface {
	Record

	// Parent returns the record that this record was created from.
	Parent() Record

	// GridFactors returns the normalized fractions of emissions in each
	// grid cell (gridSrg) of grid index gi.
	// coveredByGrid indicates whether the emissions source is completely
	// covered by the grid, and inGrid indicates whether it is in the
	// grid at all.
	GridFactors(gi int) (gridSrg *sparse.SparseArray, coveredByGrid, inGrid bool, err error)

	// GriddedEmissions returns gridded emissions of the receiver for a given grid index and period.
	GriddedEmissions(begin, end time.Time, gi int) (
		emis map[Pollutant]*sparse.SparseArray, units map[Pollutant]unit.Dimensions, err error)
}

// GridRecord returns a record that can allocate the emissions to a grid.
func (sp *SpatialProcessor) GridRecord(r Record) RecordGridded {
	return &recordGridded{
		Record: r,
		sp:     sp,
	}
}

type recordGridded struct {
	Record
	sp *SpatialProcessor
}

// GridFactors returns the normalized fractions of emissions in each
// grid cell (gridSrg) of grid index gi.
// coveredByGrid indicates whether the emissions source is completely
// covered by the grid, and inGrid indicates whether it is in the
// grid at all.
func (r *recordGridded) GridFactors(gi int) (
	gridSrg *sparse.SparseArray, coveredByGrid, inGrid bool, err error) {

	loc := r.Location()

	if ra, ok := r.Record.(RecordSpatialSurrogate); ok {
		// If this record has a spatial surrogate, use it.
		srgSpec, err := ra.SurrogateSpecification()
		if err != nil {
			return nil, false, false, err
		}
		gridSrg, coveredByGrid, err = r.sp.Surrogate(srgSpec, r.sp.Grids[gi], loc)
		if err != nil {
			return nil, false, false, err
		}
		if gridSrg != nil {
			inGrid = true
		}
		return gridSrg, coveredByGrid, inGrid, nil
	}
	return r.sp.recordToGrid(loc, r.sp.Grids[gi])
}

// recordToGrid allocates the reciever to the specifed grid
// without any spatial surrogate.
func (sp *SpatialProcessor) recordToGrid(loc *Location, grid *GridDef) (gridSrg *sparse.SparseArray, coveredByGrid, inGrid bool, err error) {
	// Otherwise, directly allocate emissions to grid.
	gridLoc, err := loc.Reproject(grid.SR)
	if err != nil {
		return
	}
	var rows, cols []int
	var fracs []float64
	rows, cols, fracs, inGrid, coveredByGrid = grid.GetIndex(gridLoc)
	if inGrid {
		gridSrg = sparse.ZerosSparse(grid.Ny, grid.Nx)
		// A point can be allocated to more than one grid cell if it lies
		// on the boundary between two cells.
		for i, row := range rows {
			gridSrg.Set(fracs[i], row, cols[i])
		}
	}
	return
}

// GriddedEmissions returns gridded emissions of the receiver for a given grid index and period.
func (r *recordGridded) GriddedEmissions(begin, end time.Time, gi int) (
	emis map[Pollutant]*sparse.SparseArray, units map[Pollutant]unit.Dimensions, err error) {

	var gridSrg *sparse.SparseArray
	gridSrg, _, _, err = r.GridFactors(gi)
	if err != nil || gridSrg == nil {
		return
	}

	emis = make(map[Pollutant]*sparse.SparseArray)
	units = make(map[Pollutant]unit.Dimensions)
	periodEmis := r.PeriodTotals(begin, end)
	for pol, data := range periodEmis {
		emis[pol] = gridSrg.ScaleCopy(data.Value())
		units[pol] = data.Dimensions()
	}
	return
}

// Parent returns the record that this record was created from.
func (r *recordGridded) Parent() Record { return r.Record }

// Surrogate gets the specified spatial surrogate.
// It is important not to edit the returned surrogate in place, because the
// same copy is used over and over again. The second return value indicates
// whether the shape corresponding to fips is completely covered by the grid.
func (sp *SpatialProcessor) Surrogate(srgSpec SrgSpec, grid *GridDef, loc *Location) (*sparse.SparseArray, bool, error) {
	var err error
	sp.lazyLoad.Do(func() {
		err = sp.load()
	})
	if err != nil {
		return nil, false, err
	}

	s := &srgGrid{srg: srgSpec, gridData: grid, loc: loc, sp: sp, msgChan: sp.MsgChan}
	req := sp.cache.NewRequestRecursive(context.Background(), s)
	result := new(GriddedSrgData)
	if err := req.Result((*griddedSrgDataHolder)(result)); err != nil {
		return nil, false, err
	}
	srg, coveredByGrid := result.ToGrid()
	if srg != nil {
		return srg, coveredByGrid, nil
	}
	// if srg was nil, try backup surrogates.
	for _, newName := range srgSpec.backupSurrogateNames() {
		newSrgSpec, err := sp.SrgSpecs.GetByName(srgSpec.region(), newName)
		if err != nil {
			return nil, false, err
		}
		s := &srgGrid{srg: newSrgSpec, gridData: grid, loc: loc, sp: sp, msgChan: sp.MsgChan}
		result := new(GriddedSrgData)
		req := sp.cache.NewRequestRecursive(context.Background(), s)
		if err := req.Result((*griddedSrgDataHolder)(result)); err != nil {
			return nil, false, err
		}
		srg, coveredByGrid := result.ToGrid()
		if srg != nil {
			return srg, coveredByGrid, nil
		}
	}
	// If there are no backup surrogates, allocate emissions evenly across
	// all relevant grid cells.
	srg, coveredByGrid, _, err = sp.recordToGrid(loc, grid)
	return srg, coveredByGrid, err
}

type srgRequest struct {
	srgSpec    SrgSpec
	grid       *GridDef
	err        error
	returnChan chan *srgRequest

	// Usually we use channel to make sure only one surrogate is getting created
	// at a time. This avoids duplicate work if 2 records request the same surrogate
	// at the same time. However, surrogates that are being created to merge with
	// other surrogates need to skip the queue to avoid a channel lock.
	waitInQueue bool
}

// key returns a unique key for this surrogate request.
func (s *srgRequest) key() string {
	return fmt.Sprintf("%s_%s_%s", s.srgSpec.region(), s.srgSpec.code(),
		s.grid.Name)
}

// repeat copies the request with a new return chan
func (s *srgRequest) repeat() *srgRequest {
	ss := newSrgRequest(s.srgSpec, s.grid)
	ss.waitInQueue = s.waitInQueue
	return ss
}

func newSrgRequest(srgSpec SrgSpec, grid *GridDef) *srgRequest {
	d := new(srgRequest)
	d.srgSpec = srgSpec
	d.grid = grid
	d.returnChan = make(chan *srgRequest)
	d.waitInQueue = true
	return d
}

// SrgSpecs holds a group of surrogate specifications
type SrgSpecs struct {
	byName map[Country]map[string]SrgSpec
	byCode map[Country]map[string]SrgSpec
}

// NewSrgSpecs initializes a new SrgSpecs object.
func NewSrgSpecs() *SrgSpecs {
	s := new(SrgSpecs)
	s.byName = make(map[Country]map[string]SrgSpec)
	s.byCode = make(map[Country]map[string]SrgSpec)
	return s
}

// Add adds a new SrgSpec to s.
func (s *SrgSpecs) Add(ss SrgSpec) {
	if _, ok := s.byName[ss.region()]; !ok {
		s.byName[ss.region()] = make(map[string]SrgSpec)
		s.byCode[ss.region()] = make(map[string]SrgSpec)
	}
	s.byName[ss.region()][ss.name()] = ss
	s.byCode[ss.region()][ss.code()] = ss
}

// AddAll adds all surrogates in srgs to the receiver.
func (s *SrgSpecs) AddAll(srgs *SrgSpecs) {
	for _, sp := range srgs.byName {
		for _, srg := range sp {
			s.Add(srg)
		}
	}
}

// GetByName gets the surrogate matching the given region and name.
func (s *SrgSpecs) GetByName(region Country, name string) (SrgSpec, error) {
	ss, ok := s.byName[region][name]
	if ok {
		return ss, nil
	}
	return nil, fmt.Errorf("can't find surrogate for region=%s, name=%s", region, name)
}

// GetByCode gets the surrogate matching the given region and code.
func (s *SrgSpecs) GetByCode(region Country, code string) (SrgSpec, error) {
	ss, ok := s.byCode[region][code]
	if ok {
		return ss, nil
	}
	return nil, fmt.Errorf("can't find surrogate for region=%s, code=%s", region, code)
}

// findFile finds a file in dir or any of its subdirectories.
func findFile(dir, file string) (string, error) {
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}

	var fullPath string
	var found bool
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || found {
			return nil
		}
		if info.Name() == file {
			fullPath = path
			found = true
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("could not find file %s within directory %s", file, dir)
	}
	return fullPath, nil
}

// GridRef specifies the grid surrogates the correspond with combinations of
// country (first map), SCC (second map), and FIPS or spatial ID (third map).
type GridRef struct {
	data          map[Country]map[string]map[string]interface{}
	sccExactMatch bool
}

// ReadGridRef reads the SMOKE gref file, which maps FIPS and SCC codes to grid surrogates.
// sccExactMatch specifies whether SCC codes must match exactly, or if partial
// matches are allowed.
func ReadGridRef(f io.Reader, sccExactMatch bool) (*GridRef, error) {
	gr := &GridRef{
		data:          make(map[Country]map[string]map[string]interface{}),
		sccExactMatch: sccExactMatch,
	}
	buf := bufio.NewReader(f)
	for {
		record, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, fmt.Errorf("in ReadGridRef: %v \nrecord= %s",
					err.Error(), record)
			}
		}
		// Get rid of comments at end of line.
		if i := strings.Index(record, "!"); i != -1 {
			record = record[0:i]
		}

		if record[0] != '#' && record[0] != '\n' {
			splitLine := strings.Split(record, ";")
			SCC := splitLine[1]
			if len(SCC) == 8 {
				// TODO: make this work with different types of codes; i.e. some sort of
				// fuzzy matching instead of just adding 2 zeros.
				SCC = "00" + SCC
			}
			var country Country
			FIPS := splitLine[0]
			if len(FIPS) == 6 {
				country = getCountryFromID(FIPS[0:1])
				FIPS = FIPS[1:]
			} else {
				country = Country(0)
			}
			srg := strings.Trim(splitLine[2], "\"\n ")

			if _, ok := gr.data[country]; !ok {
				gr.data[country] = make(map[string]map[string]interface{})
			}
			if _, ok := gr.data[country][SCC]; !ok {
				gr.data[country][SCC] = make(map[string]interface{})
			}
			gr.data[country][SCC][FIPS] = srg
		}
	}
	return gr, nil
}

// GetSrgCode returns the surrogate code appropriate for the given SCC code,
// country and FIPS.
func (gr GridRef) GetSrgCode(SCC string, c Country, FIPS string) (string, error) {
	var err error
	var matchedVal interface{}
	if !gr.sccExactMatch {
		_, _, matchedVal, err = MatchCodeDouble(SCC, FIPS, gr.data[c])
	} else {
		_, matchedVal, err = MatchCode(FIPS, gr.data[c][SCC])
	}
	if err != nil {
		return "", fmt.Errorf("in aep.GridRef.GetSrgCode: %v. (SCC=%v, Country=%v, FIPS=%v)",
			err.Error(), SCC, c, FIPS)
	}
	return matchedVal.(string), nil
}

// Merge combines values in gr2 into gr. If gr2 combines any values that
// conflict with values already in gr, an error is returned.
func (gr *GridRef) Merge(gr2 *GridRef) error {
	for country, d1 := range gr2.data {
		if _, ok := gr.data[country]; !ok {
			gr.data[country] = make(map[string]map[string]interface{})
		}
		for SCC, d2 := range d1 {
			if _, ok := gr.data[country][SCC]; !ok {
				gr.data[country][SCC] = make(map[string]interface{})
			}
			for FIPS, code := range d2 {
				if existingCode, ok := gr.data[country][SCC][FIPS]; ok && existingCode != code {
					return fmt.Errorf("GridRef already has code of %s for country=%s, "+
						"SCC=%s, FIPS=%s. Cannot replace with code %s.",
						existingCode, country, SCC, FIPS, code)
				}
				gr.data[country][SCC][FIPS] = code
			}
		}
	}
	return nil
}
