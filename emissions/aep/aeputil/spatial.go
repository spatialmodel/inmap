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
	"fmt"
	"os"
	"sync"

	"github.com/ctessum/sparse"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/unit"
	"github.com/spatialmodel/inmap/emissions/aep"
)

// SpatialConfig holds emissions spatialization configuration information.
type SpatialConfig struct {
	// SrgSpec gives the location of the surrogate specification file.
	SrgSpec string

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

	// InputSR specifies the input spatial reference in Proj4 format.
	InputSR string

	// SimplifyTolerance is the tolerance for simplifying spatial surrogate
	// geometry, in units of OutputSR.
	SimplifyTolerance float64

	// SpatialCache specifies the location for storing spatial emissions
	// data for quick access. If this is left empty, no cache will be used.
	SpatialCache string

	// MaxCacheEntries specifies the maximum number of emissions and concentrations
	// surrogates to hold in a memory cache. Larger numbers can result in faster
	// processing but increased memory usage.
	MaxCacheEntries int

	// GridCells specifies the geometry of the spatial grid.
	GridCells []geom.Polygonal

	// GridName specifies a name for the grid which is used in the names
	// of intermediate and output files.
	GridName string

	loadOnce sync.Once
	sp       *aep.SpatialProcessor
}

var _ Iterator = &SpatialIterator{} // Ensure that SpatialIterator fulfills the Iterator interface.

// Iterator creates a SpatialIterator from the given parent iterator
// for the given gridIndex.
func (c *SpatialConfig) Iterator(parent Iterator, gridIndex int) *SpatialIterator {
	return &SpatialIterator{
		parent:    parent,
		c:         c,
		gridIndex: gridIndex,
		emis:      make(map[aep.Pollutant]*sparse.SparseArray),
		units:     make(map[aep.Pollutant]unit.Dimensions),
		ungridded: make(map[aep.Pollutant]*unit.Unit),
	}
}

// SpatialIterator is an Iterator that spatializes the records that it
// processes.
type SpatialIterator struct {
	parent    Iterator
	c         *SpatialConfig
	gridIndex int

	emis      map[aep.Pollutant]*sparse.SparseArray // Gridded emissions
	units     map[aep.Pollutant]unit.Dimensions
	ungridded map[aep.Pollutant]*unit.Unit // Emissions before gridding
}

// Next spatializes a record from the
func (si *SpatialIterator) Next() (aep.Record, error) {
	var err error
	si.c.loadOnce.Do(func() {
		si.c.sp, err = si.c.setupSpatialProcessor()
	})
	if err != nil {
		return nil, err
	}

	rec, err := si.parent.Next()
	if err != nil {
		return nil, err
	}

	recG := si.c.sp.GridRecord(rec)

	srg, _, inGrid, err := recG.GridFactors(si.gridIndex)
	if err != nil {
		return nil, err
	}
	if !inGrid {
		return rec, nil
	}
	t := rec.Totals()
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
	return rec, nil
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

func readSrgSpec(srgSpecPath, srgShapefileDirectory string, sccExactMatch bool) (*aep.SrgSpecs, error) {
	f, err := os.Open(os.ExpandEnv(srgSpecPath))
	if err != nil {
		return nil, err
	}
	srgSpecs, err := aep.ReadSrgSpec(f, os.ExpandEnv(srgShapefileDirectory), sccExactMatch)
	if err != nil {
		return nil, err
	}
	if err = f.Close(); err != nil {
		return nil, err
	}
	return srgSpecs, nil
}

func readGridRef(paths []string, sccExactMatch bool) (*aep.GridRef, error) {
	var gridRef *aep.GridRef
	for _, gf := range paths {
		f, err := os.Open(os.ExpandEnv(gf))
		if err != nil {
			return nil, err
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

	srgSpecs, err := readSrgSpec(c.SrgSpec, c.SrgShapefileDirectory, c.SCCExactMatch)
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
