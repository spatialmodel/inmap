/*
Copyright © 2017 the InMAP authors.
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

package aeputil

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/ctessum/sparse"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/unit"
	"github.com/spatialmodel/inmap/emissions/aep"
)

// SpatialConfig holds emissions spatialization configuration information.
type SpatialConfig struct {
	// SrgSpecSMOKE gives the location of the SMOKE-formatted
	// surrogate specification file, if any.
	SrgSpecSMOKE string

	// SrgSpecOSM gives the location of the OSM-formatted
	// surrogate specification file, if any.
	SrgSpecOSM string

	// PostGISURL specifies the URL to use to connect to a PostGIS database
	// with the OpenStreetMap data loaded. The URL should be in the format:
	// postgres://username:password@hostname:port/databasename".
	//
	// The OpenStreetMap data can be loaded into the database using the
	// osm2pgsql program, for example with the command:
	// osm2pgsql -l --hstore-all --hstore-add-index --database=databasename --host=hostname --port=port --username=username --create planet_latest.osm.pbf
	//
	// The -l and --hstore-all flags for the osm2pgsql command are both necessary,
	// and the PostGIS database should have the "hstore" extension installed before
	// loading the data.
	PostGISURL string

	// SrgShapefileDirectory gives the location of the directory holding
	// the shapefiles used for creating spatial surrogates.
	SrgShapefileDirectory string

	// SCCExactMatch specifies whether SCC codes must match exactly when processing
	// emissions.
	SCCExactMatch bool

	// GridRef specifies the locations of the spatial surrogate gridding
	// reference files used for processing the NEI.
	GridRef []string

	// OutputSR specifies the output spatial reference in Proj4 format.
	OutputSR string

	// InputSR specifies the input emissions spatial reference in Proj4 format.
	InputSR string

	// SimplifyTolerance is the tolerance for simplifying spatial surrogate
	// geometry, in units of OutputSR.
	SimplifyTolerance float64

	// SpatialCache specifies the location for storing spatial emissions
	// data for quick access. If this is left empty, no cache will be used.
	SpatialCache string

	// SrgDataCache specifies the location for caching spatial surrogate input data.
	// If it is empty, the input surrogate data will be stored in SpatialCache.
	SrgDataCache string

	// MaxCacheEntries specifies the maximum number of emissions and concentrations
	// surrogates to hold in a memory cache. Larger numbers can result in faster
	// processing but increased memory usage.
	MaxCacheEntries int

	// GridCells specifies the geometry of the spatial grid.
	GridCells []geom.Polygonal

	// GridName specifies a name for the grid which is used in the names
	// of intermediate and output files.
	// Changes to the geometry of the grid must be accompanied by either a
	// a change in GridName or the deletion of all the files in the
	// SpatialCache directory.
	GridName string

	loadOnce sync.Once
	sp       *aep.SpatialProcessor
}

var _ Iterator = &SpatialIterator{} // Ensure that SpatialIterator fulfills the Iterator interface.

// Iterator creates a SpatialIterator from the given parent iterator
// for the given gridIndex.
func (c *SpatialConfig) Iterator(parent Iterator, gridIndex int) *SpatialIterator {
	si := &SpatialIterator{
		parent:    parent,
		c:         c,
		gridIndex: gridIndex,
		emis:      make(map[aep.Pollutant]*sparse.SparseArray),
		units:     make(map[aep.Pollutant]unit.Dimensions),
		ungridded: make(map[aep.Pollutant]*unit.Unit),
		inChan:    make(chan recordErr, 100),
		outChan:   make(chan recordGriddedErr, 100),
	}

	// Read all records from the parent
	// and send them for asynchronous processing.
	go func() {
		for {
			rec, err := si.parent.Next()
			if err == io.EOF {
				close(si.inChan)
				return
			}
			si.inChan <- recordErr{Record: rec, err: err}
		}
	}()

	wg := sync.WaitGroup{}
	nprocs := runtime.GOMAXPROCS(-1)
	wg.Add(nprocs)
	for i := 0; i < nprocs; i++ {
		go func() {
			for recErr := range si.inChan {
				recGridded, err := si.processRecord(recErr)
				si.outChan <- recordGriddedErr{RecordGridded: recGridded, err: err}
			}
			wg.Done()
		}()
	}
	go func() {
		// Close channel after processing is finished.
		wg.Wait()
		close(si.outChan)
	}()
	return si
}

type recordErr struct {
	aep.Record
	err error
}
type recordGriddedErr struct {
	aep.RecordGridded
	err error
}

// SpatialIterator is an Iterator that spatializes the records that it
// processes.
type SpatialIterator struct {
	parent    Iterator
	c         *SpatialConfig
	gridIndex int

	inChan  chan recordErr
	outChan chan recordGriddedErr

	emis      map[aep.Pollutant]*sparse.SparseArray // Gridded emissions
	units     map[aep.Pollutant]unit.Dimensions
	ungridded map[aep.Pollutant]*unit.Unit // Emissions before gridding

	mx sync.Mutex
}

// processRecord allocates one record to a grid.
func (si *SpatialIterator) processRecord(r recordErr) (aep.RecordGridded, error) {
	if r.err != nil {
		return nil, r.err
	}
	var err error
	si.c.loadOnce.Do(func() {
		si.c.sp, err = si.c.setupSpatialProcessor()
	})
	if err != nil {
		return nil, err
	}

	rec := r.Record
	// Add a spatial surrogate for records that are polygons,
	// if SrgSpecs and a GridRef have been specified.
	loc := rec.Location()
	if _, ok := loc.Geom.(geom.Polygonal); ok && si.c.sp.SrgSpecs != nil && si.c.sp.GridRef != nil {
		if _, err := si.c.sp.GridRef.GetSrgCode(rec.GetSCC(), rec.GetCountry(), rec.GetFIPS()); err == nil {
			// Only add spatial surrogate if there is one available for this record.
			rec = si.c.sp.AddSurrogate(rec)
		}
	}

	recG := si.c.sp.GridRecord(rec)

	srg, _, inGrid, err := recG.GridFactors(si.gridIndex)
	if err != nil {
		return nil, err
	}
	if !inGrid {
		return recG, nil
	}
	t := rec.Totals()
	si.mx.Lock()
	for p, totalEmis := range t {
		spatialEmis := srg.ScaleCopy(totalEmis.Value())
		if _, ok := si.emis[p]; !ok {
			si.ungridded[p] = totalEmis.Clone()
			si.emis[p] = spatialEmis
			si.units[p] = totalEmis.Dimensions()
		} else {
			si.emis[p].AddSparse(spatialEmis)
			if !si.units[p].Matches(totalEmis.Dimensions()) {
				return nil, fmt.Errorf("aeputil.SpatialIterator: inconsistent units for pollutant %v: %v != %v",
					p, si.units[p], totalEmis.Dimensions())
			}
			si.ungridded[p].Add(totalEmis)
		}
	}
	si.mx.Unlock()
	return recG, nil
}

// NextGridded returns a spatialized a record from the parent iterator.
func (si *SpatialIterator) NextGridded() (aep.RecordGridded, error) {
	out, ok := <-si.outChan
	if !ok {
		return nil, io.EOF
	}
	return out.RecordGridded, out.err
}

// Next returns a spatialized a record from the parent iterator
// to fulfill the iterator interface.
func (si *SpatialIterator) Next() (aep.Record, error) {
	return si.NextGridded()
}

// SpatialTotals returns spatial arrays of the total emissions
// for each pollutant, as well as their units.
func (si *SpatialIterator) SpatialTotals() (emissions map[aep.Pollutant]*sparse.SparseArray, units map[aep.Pollutant]unit.Dimensions) {
	return si.emis, si.units
}

// SpatialProcessor returns the spatial processor associated with the
// receiver.
func (c *SpatialConfig) SpatialProcessor() (*aep.SpatialProcessor, error) {
	var err error
	c.loadOnce.Do(func() {
		c.sp, err = c.setupSpatialProcessor()
	})
	if err != nil {
		return nil, err
	}
	return c.sp, nil
}

func readSrgSpec(srgSpecSMOKEPath, srgSpecOSMPath, postGISURL, srgShapefileDirectory string, sccExactMatch bool, diskCachePath string, memCacheEntries int) (*aep.SrgSpecs, error) {
	srgSpecs := aep.NewSrgSpecs()
	if srgSpecSMOKEPath != "" {
		f, err := os.Open(os.ExpandEnv(srgSpecSMOKEPath))
		if err != nil {
			return nil, fmt.Errorf("aep: opening SMOKE surrogate specification: %v", err)
		}
		srgSpecsTemp, err := aep.ReadSrgSpecSMOKE(f, os.ExpandEnv(srgShapefileDirectory), sccExactMatch, diskCachePath, memCacheEntries)
		if err != nil {
			return nil, err
		}
		if err = f.Close(); err != nil {
			return nil, err
		}
		srgSpecs.AddAll(srgSpecsTemp)
	}
	if srgSpecOSMPath != "" {
		f, err := os.Open(os.ExpandEnv(srgSpecOSMPath))
		if err != nil {
			return nil, fmt.Errorf("aep: opening OSM surrogate specification: %v", err)
		}
		srgSpecsTemp, err := aep.ReadSrgSpecOSM(context.Background(), f, postGISURL)
		if err != nil {
			return nil, err
		}
		if err = f.Close(); err != nil {
			return nil, err
		}
		srgSpecs.AddAll(srgSpecsTemp)
	}
	return srgSpecs, nil
}

func readGridRef(paths []string, sccExactMatch bool) (*aep.GridRef, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	var gridRef *aep.GridRef
	for _, gf := range paths {
		f, err := os.Open(os.ExpandEnv(gf))
		if err != nil {
			return nil, fmt.Errorf("aep: opening GridRef file: %v", err)
		}
		gridRefTemp, err2 := aep.ReadGridRef(f, sccExactMatch)
		if err2 != nil {
			return nil, err2
		}
		if err = f.Close(); err != nil {
			return nil, err
		}
		if gridRef == nil {
			gridRef = gridRefTemp
		} else {
			err = gridRef.Merge(gridRefTemp)
			if err != nil {
				return nil, err
			}
		}
	}
	return gridRef, nil
}

// setupSpatialProcessor reads in the necessary information to initialize
// a processor for spatializing emissions, and then does so.
func (c *SpatialConfig) setupSpatialProcessor() (*aep.SpatialProcessor, error) {
	if c.GridCells == nil {
		return nil, fmt.Errorf("aeputil: GridCells must be specified for spatial processor")
	}

	var cacheLoc string
	if c.SrgDataCache != "" {
		cacheLoc = c.SrgDataCache
	} else {
		cacheLoc = c.SpatialCache
	}

	srgSpecs, err := readSrgSpec(c.SrgSpecSMOKE, c.SrgSpecOSM, c.PostGISURL, c.SrgShapefileDirectory, c.SCCExactMatch, cacheLoc, c.MaxCacheEntries)
	if err != nil {
		return nil, err
	}

	gridRef, err := readGridRef(c.GridRef, c.SCCExactMatch)
	if err != nil {
		return nil, err
	}

	outSR, err := proj.Parse(os.ExpandEnv(c.OutputSR))
	if err != nil {
		return nil, err
	}
	inSR, err := proj.Parse(os.ExpandEnv(c.InputSR))
	if err != nil {
		return nil, err
	}
	grid, err := aep.NewGridIrregular(c.GridName, c.GridCells, outSR, outSR)
	if err != nil {
		return nil, err
	}
	sp := aep.NewSpatialProcessor(srgSpecs, []*aep.GridDef{grid}, gridRef, inSR, c.SCCExactMatch)
	sp.DiskCachePath = c.SpatialCache
	sp.SimplifyTolerance = c.SimplifyTolerance

	// Set up logging.
	sp.MsgChan = make(chan string)
	logTick := time.Tick(2 * time.Second)
	go func() {
		for msg := range sp.MsgChan {
			select {
			case <-logTick:
				log.Println(msg)
			default:
				runtime.Gosched()
			}
		}
	}()

	return sp, nil
}

type spatialReport struct {
	si *SpatialIterator
}

func (sr *spatialReport) Totals() map[aep.Pollutant]*unit.Unit {
	emis, units := sr.si.SpatialTotals()
	o := make(map[aep.Pollutant]*unit.Unit)
	for p, e := range emis {
		o[p] = unit.New(e.Sum(), units[p])
	}
	return o
}
func (sr *spatialReport) DroppedTotals() map[aep.Pollutant]*unit.Unit {
	griddedTotals := sr.Totals()
	o := make(map[aep.Pollutant]*unit.Unit)
	for p, v := range sr.si.ungridded {
		if v2, ok := griddedTotals[p]; ok {
			o[p] = unit.Sub(v, v2)
		} else {
			o[p] = v.Clone()
		}
	}
	return o
}
func (sr *spatialReport) Group() string { return "" }
func (sr *spatialReport) Name() string  { return "Spatial" }

// Report returns an emissions report on the records that have been
// processed by this iterator.
func (si *SpatialIterator) Report() *aep.InventoryReport {
	return &aep.InventoryReport{Data: []aep.Totaler{&spatialReport{si: si}}}
}
