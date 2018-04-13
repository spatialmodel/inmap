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
	"encoding/csv"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"bitbucket.org/ctessum/sparse"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/requestcache"
	"github.com/ctessum/unit"
)

// SpatialProcessor spatializes emissions records.
type SpatialProcessor struct {
	SrgSpecs
	Grids []*GridDef
	GridRef

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
}

// NewSpatialProcessor creates a new spatial processor.
func NewSpatialProcessor(srgSpecs *SrgSpecs, grids []*GridDef, gridRef *GridRef, inputSR *proj.SR, matchFullSCC bool) *SpatialProcessor {
	sp := new(SpatialProcessor)
	sp.SrgSpecs = *srgSpecs
	sp.Grids = grids
	sp.GridRef = *gridRef
	sp.inputSR = inputSR
	sp.matchFullSCC = matchFullSCC

	sp.MemCacheSize = 100
	sp.MaxMergeDepth = 10
	return sp
}

func init() {
	gob.Register(GriddingSurrogate{})
}

// unmarshalGob unmarshals an interface from a byte array and fulfills
// the requirements for the Disk cache unmarshalFunc input.
func unmarshalGob(b []byte) (interface{}, error) {
	r := bytes.NewBuffer(b)
	d := gob.NewDecoder(r)
	var data GriddingSurrogate
	if err := d.Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

// marshalGob marshals an interface to a byte array and fulfills
// the requirements for the Disk cache marshalFunc input.
func marshalGob(data interface{}) ([]byte, error) {
	w := bytes.NewBuffer(nil)
	e := gob.NewEncoder(w)
	d := *data.(*interface{})
	dd := d.(*GriddingSurrogate)
	if err := e.Encode(dd); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (sp *SpatialProcessor) load() {
	if sp.DiskCachePath == "" {
		sp.cache = requestcache.NewCache(sp.createSurrogate, runtime.GOMAXPROCS(-1),
			requestcache.Deduplicate(), requestcache.Memory(sp.MemCacheSize))
	} else {
		sp.cache = requestcache.NewCache(sp.createSurrogate, runtime.GOMAXPROCS(-1),
			requestcache.Deduplicate(), requestcache.Memory(sp.MemCacheSize),
			requestcache.Disk(sp.DiskCachePath, marshalGob, unmarshalGob))
	}
}

// Spatialize takes a spatial processor (sp) and a grid index number (gi) and
// returns a gridded spatial surrogate (gridSrg) for an area emissions source,
// as well as whether the emissions source is completely covered by the grid
// (coveredByGrid) and whether it is in the grid all all (inGrid).
func (r *SourceData) Spatialize(sp *SpatialProcessor, gi int) (
	gridSrg *sparse.SparseArray, coveredByGrid, inGrid bool, err error) {

	var srgNum string
	srgNum, err = sp.GridRef.GetSrgCode(r.SCC, r.Country, r.FIPS, sp.matchFullSCC)
	if err != nil {
		return
	}
	var srgSpec *SrgSpec
	srgSpec, err = sp.SrgSpecs.GetByCode(r.Country, srgNum)
	if err != nil {
		return
	}

	gridSrg, coveredByGrid, err = sp.Surrogate(srgSpec, sp.Grids[gi], r.FIPS)
	if err != nil {
		return
	}
	if gridSrg != nil {
		inGrid = true
	}
	return
}

// Spatialize takes a spatial processor (sp) and a grid index number (gi) and
// returns a gridded spatial surrogate (gridSrg) for a point emissions source,
// as well as whether the emissions source is completely covered by the grid
// (coveredByGrid) and whether it is in the grid all all (inGrid).
func (r *PointSourceData) Spatialize(sp *SpatialProcessor, gi int) (
	gridSrg *sparse.SparseArray, coveredByGrid, inGrid bool, err error) {

	var ct proj.Transformer
	ct, err = r.SR.NewTransform(sp.Grids[gi].SR)
	if err != nil {
		return
	}

	p2, err := r.Point.Transform(ct)
	if err != nil {
		return
	}

	var rows, cols []int
	rows, cols, inGrid, err = sp.Grids[gi].GetIndex(p2.(geom.Point))
	if err != nil {
		return
	}
	// for points, inGrid and coveredByGrid are the same thing.
	coveredByGrid = inGrid
	if inGrid {
		gridSrg = sparse.ZerosSparse(sp.Grids[gi].Ny, sp.Grids[gi].Nx)
		// A point can be allocated to more than one grid cell if it lies
		// on the boundary between two cells.
		frac := 1.0 / float64(len(rows))
		for i, row := range rows {
			gridSrg.Set(frac, row, cols[i])
		}
	}
	return
}

// Spatialize takes a spatial processor (sp) and a grid index number (gi) and
// returns a gridded spatial surrogate (gridSrg) for a point emissions source,
// as well as whether the emissions source is completely covered by the grid
// (coveredByGrid) and whether it is in the grid all all (inGrid).
func (r *PointRecord) Spatialize(sp *SpatialProcessor, gi int) (
	gridSrg *sparse.SparseArray, coveredByGrid, inGrid bool, err error) {
	return r.PointSourceData.Spatialize(sp, gi)
}

// Spatialize takes a spatial processor (sp) and a grid index number (gi) and
// returns a gridded spatial surrogate (gridSrg) for a point emissions source,
// as well as whether the emissions source is completely covered by the grid
// (coveredByGrid) and whether it is in the grid all all (inGrid).
func (r *pointRecordIDA) Spatialize(sp *SpatialProcessor, gi int) (
	gridSrg *sparse.SparseArray, coveredByGrid, inGrid bool, err error) {
	return r.PointSourceData.Spatialize(sp, gi)
}

// Spatialize is added here to fulfill the Record in interace, but
// it does not contain enough information on its own to be spatialized so
// it panics if used. (It should never be used).
func (r *supplementalPointRecord) Spatialize(sp *SpatialProcessor, gi int) (
	gridSrg *sparse.SparseArray, coveredByGrid, inGrid bool, err error) {
	panic("supplementalPointRecord cannot be spatialized")
}

// GriddedEmissions returns gridded emissions of record r for a given grid index and period.
func GriddedEmissions(r Record, begin, end time.Time, sp *SpatialProcessor,
	gi int) (emis map[Pollutant]*sparse.SparseArray, units map[Pollutant]unit.Dimensions, err error) {

	var gridSrg *sparse.SparseArray
	gridSrg, _, _, err = r.Spatialize(sp, gi)
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

// Surrogate gets the specified spatial surrogate.
// It is important not to edit the returned surrogate in place, because the
// same copy is used over and over again. The second return value indicates
// whether the shape corresponding to fips is completely covered by the grid.
func (sp *SpatialProcessor) Surrogate(srgSpec *SrgSpec, grid *GridDef, fips string) (*sparse.SparseArray, bool, error) {

	sp.lazyLoad.Do(sp.load)

	s := &srgGrid{srg: srgSpec, gridData: grid}
	req := sp.cache.NewRequest(context.Background(), s, s.key())
	resultI, err := req.Result()
	if err != nil {
		return nil, false, err
	}
	result := resultI.(*GriddingSurrogate)
	srg, coveredByGrid := result.ToGrid(fips)
	if srg != nil {
		return srg, coveredByGrid, nil
	}
	// if srg was nil, try backup surrogates.
	for _, newName := range srgSpec.BackupSurrogateNames {
		newSrgSpec, err := sp.SrgSpecs.GetByName(srgSpec.Region, newName)
		if err != nil {
			return nil, false, err
		}
		s := &srgGrid{srg: newSrgSpec, gridData: grid}
		req := sp.cache.NewRequest(context.Background(), s, s.key())
		resultI, err := req.Result()
		if err != nil {
			return nil, false, err
		}
		result := resultI.(*GriddingSurrogate)
		srg, coveredByGrid := result.ToGrid(fips)
		if srg != nil {
			return srg, coveredByGrid, nil
		}
	}
	return nil, false, nil
}

type srgRequest struct {
	srgSpec    *SrgSpec
	grid       *GridDef
	data       *GriddingSurrogate
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
	return fmt.Sprintf("%s_%s_%s", s.srgSpec.Region, s.srgSpec.Code,
		s.grid.Name)
}

// repeat copies the request with a new return chan
func (s *srgRequest) repeat() *srgRequest {
	ss := newSrgRequest(s.srgSpec, s.grid)
	ss.waitInQueue = s.waitInQueue
	return ss
}

func newSrgRequest(srgSpec *SrgSpec, grid *GridDef) *srgRequest {
	d := new(srgRequest)
	d.srgSpec = srgSpec
	d.grid = grid
	d.returnChan = make(chan *srgRequest)
	d.waitInQueue = true
	return d
}

// SrgSpecs holds a group of surrogate specifications
type SrgSpecs struct {
	byName map[Country]map[string]*SrgSpec
	byCode map[Country]map[string]*SrgSpec
}

// NewSrgSpecs initializes a new SrgSpecs object.
func NewSrgSpecs() *SrgSpecs {
	s := new(SrgSpecs)
	s.byName = make(map[Country]map[string]*SrgSpec)
	s.byCode = make(map[Country]map[string]*SrgSpec)
	return s
}

// Add adds a new SrgSpec to s.
func (s *SrgSpecs) Add(ss *SrgSpec) {
	if _, ok := s.byName[ss.Region]; !ok {
		s.byName[ss.Region] = make(map[string]*SrgSpec)
		s.byCode[ss.Region] = make(map[string]*SrgSpec)
	}
	s.byName[ss.Region][ss.Name] = ss
	s.byCode[ss.Region][ss.Code] = ss
}

// GetByName gets the surrogate matching the given region and name.
func (s *SrgSpecs) GetByName(region Country, name string) (*SrgSpec, error) {
	ss, ok := s.byName[region][name]
	if ok {
		return ss, nil
	}
	return nil, fmt.Errorf("can't find surrogate for region=%s, name=%s", region, name)
}

// GetByCode gets the surrogate matching the given region and code.
func (s *SrgSpecs) GetByCode(region Country, code string) (*SrgSpec, error) {
	ss, ok := s.byCode[region][code]
	if ok {
		return ss, nil
	}
	return nil, fmt.Errorf("can't find surrogate for region=%s, code=%s", region, code)
}

// Status returns the status of the spatial surrogates in s.
func (s *SrgSpecs) Status() []Status {
	var o statuses
	for _, ss := range s.byName {
		for _, sss := range ss {
			o = append(o, sss.Status())
		}
	}
	sort.Sort(statuses(o))
	return o
}

// SrgSpec holds spatial surrogate specification information.
type SrgSpec struct {
	Region          Country
	Name            string
	Code            string
	DATASHAPEFILE   string
	DATAATTRIBUTE   string
	WEIGHTSHAPEFILE string
	Details         string

	// BackupSurrogateNames specifies names of surrogates to use if this
	// one doesn't have data for the desired location.
	BackupSurrogateNames []string

	// WeightColumns specify the fields of the surogate shapefile that
	// should be used to weight the output locations.
	WeightColumns []string

	// WeightFactors are factors by which each of the WeightColumns should
	// be multiplied.
	WeightFactors []float64

	// FilterFunction specifies which rows in the surrogate shapefile should
	// be used to create this surrogate.
	FilterFunction *SurrogateFilter

	// MergeNames specify names of other surrogates that should be combined
	// to create this surrogate.
	MergeNames []string
	// MergeMultipliers specifies multipliers associated with the surrogates
	// in MergeNames.
	MergeMultipliers []float64

	// progress specifies the progress in generating the surrogate.
	progress     float64
	progressLock sync.Mutex
	// status specifies what the surrogate generator is currently doing.
	status string
}

// Status returns information about the status of s.
func (s *SrgSpec) Status() Status {
	s.progressLock.Lock()
	o := Status{
		Name:     s.Name,
		Code:     s.Code,
		Status:   s.status,
		Progress: s.progress,
	}
	s.progressLock.Unlock()
	return o
}

const none = "NONE"

// ReadSrgSpec reads a SMOKE formatted spatial surrogate specification file.
// Results are returned as a map of surrogate specifications as indexed by
// their unique ID, which is Region+SurrogateCode. shapefileDir specifies the
// location of all the required shapefiles, and checkShapeFiles specifies whether
// to check if the required shapefiles actually exist. If checkShapeFiles is
// true, then it is okay for the shapefiles to be in any subdirectory of
// shapefileDir, otherwise all shapefiles must be in shapefileDir itself and
// not a subdirectory.
func ReadSrgSpec(fid io.Reader, shapefileDir string, checkShapefiles bool) (*SrgSpecs, error) {
	srgs := NewSrgSpecs()
	reader := csv.NewReader(fid)
	reader.Comment = '#'
	reader.TrailingComma = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("in ReadSrgSpec: %v", err)
	}
	for i := 1; i < len(records); i++ {
		record := records[i]
		srg := new(SrgSpec)
		srg.Region, err = countryFromName(record[0])
		if err != nil {
			return nil, fmt.Errorf("in ReadSrgSpec: %v", err)
		}
		srg.Name = strings.TrimSpace(record[1])
		srg.Code = record[2]
		srg.DATASHAPEFILE = record[3]
		srg.DATAATTRIBUTE = strings.TrimSpace(record[4])
		srg.WEIGHTSHAPEFILE = record[5]
		WEIGHTATTRIBUTE := record[6]
		WEIGHTFUNCTION := record[7]
		FILTERFUNCTION := record[8]
		MERGEFUNCTION := record[9]
		for i := 10; i <= 12; i++ {
			if len(record[i]) != 0 {
				srg.BackupSurrogateNames = append(srg.BackupSurrogateNames, record[i])
			}
		}
		srg.Details = record[13]

		// Parse weight function
		if WEIGHTATTRIBUTE != none && WEIGHTATTRIBUTE != "" {
			srg.WeightColumns = append(srg.WeightColumns,
				strings.TrimSpace(WEIGHTATTRIBUTE))
			srg.WeightFactors = append(srg.WeightFactors, 1.)
		}
		if WEIGHTFUNCTION != "" {
			weightfunction := strings.Split(WEIGHTFUNCTION, "+")
			for _, wf := range weightfunction {
				mulFunc := strings.Split(wf, "*")
				if len(mulFunc) == 1 {
					srg.WeightColumns = append(srg.WeightColumns,
						strings.TrimSpace(mulFunc[0]))
					srg.WeightFactors = append(srg.WeightFactors, 1.)
				} else if len(mulFunc) == 2 {
					v, err2 := strconv.ParseFloat(mulFunc[0], 64)
					if err2 != nil {
						return nil, fmt.Errorf("srgspec weight function: %v", err2)
					}
					srg.WeightColumns = append(srg.WeightColumns,
						strings.TrimSpace(mulFunc[1]))
					srg.WeightFactors = append(srg.WeightFactors, v)
				} else {
					return nil, fmt.Errorf("invalid value %s in srgspec "+
						"weighting function", wf)
				}
			}
		}

		// Parse filter function
		srg.FilterFunction = ParseSurrogateFilter(FILTERFUNCTION)

		// Parse merge function
		if MERGEFUNCTION != none && MERGEFUNCTION != "" {
			s := strings.Split(MERGEFUNCTION, "+")
			for _, s2 := range s {
				s3 := strings.Split(s2, "*")
				srg.MergeNames = append(srg.MergeNames, strings.TrimSpace(s3[1]))
				val, err2 := strconv.ParseFloat(strings.TrimSpace(s3[0]), 64)
				if err2 != nil {
					return nil, err2
				}
				srg.MergeMultipliers = append(srg.MergeMultipliers, val)
			}
		}
		if len(srg.MergeNames) == 0 {
			// If this is not a merged surrogate, setup the shapefile paths and
			// optionally check to make sure the shapefiles exist.
			if checkShapefiles {
				srg.DATASHAPEFILE, err = findFile(shapefileDir, srg.DATASHAPEFILE+".shp")
				if err != nil {
					return nil, err
				}
				srg.WEIGHTSHAPEFILE, err = findFile(shapefileDir, srg.WEIGHTSHAPEFILE+".shp")
				if err != nil {
					return nil, err
				}
			} else {
				srg.DATASHAPEFILE = filepath.Join(
					shapefileDir, srg.DATASHAPEFILE+".shp")
				srg.WEIGHTSHAPEFILE = filepath.Join(
					shapefileDir, srg.WEIGHTSHAPEFILE+".shp")
			}

			if checkShapefiles {
				shpf, err := shp.NewDecoder(srg.DATASHAPEFILE)
				if err != nil {
					return nil, err
				}
				shpf.Close()
				shpf, err = shp.NewDecoder(srg.WEIGHTSHAPEFILE)
				if err != nil {
					return nil, err
				}
				shpf.Close()
			}
		}
		srgs.Add(srg)
	}
	return srgs, nil
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
type GridRef map[Country]map[string]map[string]interface{}

// ReadGridRef reads the SMOKE gref file, which maps FIPS and SCC codes to grid surrogates
func ReadGridRef(f io.Reader) (*GridRef, error) {
	gr := make(GridRef)
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

			if _, ok := gr[country]; !ok {
				gr[country] = make(map[string]map[string]interface{})
			}
			if _, ok := gr[country][SCC]; !ok {
				gr[country][SCC] = make(map[string]interface{})
			}
			gr[country][SCC][FIPS] = srg
		}
	}
	return &gr, nil
}

// GetSrgCode returns the surrogate code appropriate for the given SCC code,
// country and FIPS.
func (gr GridRef) GetSrgCode(SCC string, c Country, FIPS string, matchFullSCC bool) (string, error) {
	var err error
	var matchedVal interface{}
	if !matchFullSCC {
		_, _, matchedVal, err = MatchCodeDouble(SCC, FIPS, gr[c])
	} else {
		_, matchedVal, err = MatchCode(FIPS, gr[c][SCC])
	}
	if err != nil {
		return "", fmt.Errorf("in aep.GridRef.GetSrgCode: %v. (SCC=%v, Country=%v, FIPS=%v)",
			err.Error(), SCC, c, FIPS)
	}
	return matchedVal.(string), nil
}

// Merge combines values in gr2 into gr. If gr2 combines any values that
// conflict with values already in gr, an error is returned.
func (gr *GridRef) Merge(gr2 GridRef) error {
	for country, d1 := range gr2 {
		if _, ok := (*gr)[country]; !ok {
			(*gr)[country] = make(map[string]map[string]interface{})
		}
		for SCC, d2 := range d1 {
			if _, ok := (*gr)[country][SCC]; !ok {
				(*gr)[country][SCC] = make(map[string]interface{})
			}
			for FIPS, code := range d2 {
				if existingCode, ok := (*gr)[country][SCC][FIPS]; ok && existingCode != code {
					return fmt.Errorf("GridRef already has code of %s for country=%s, "+
						"SCC=%s, FIPS=%s. Cannot replace with code %s.",
						existingCode, country, SCC, FIPS, code)
				}
				(*gr)[country][SCC][FIPS] = code
			}
		}
	}
	return nil
}
