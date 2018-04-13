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

	"bitbucket.org/ctessum/sparse"

	"github.com/spatialmodel/inmap/emissions/aep"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/unit"
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

// SpatializeTotal returns spatial arrays of the total emissions in recs.
// for each pollutant and each of the spatial grids. The returned values are
// the emissions and their units.
func (c *SpatialConfig) SpatializeTotal(recs ...aep.Record) (map[aep.Pollutant][]*sparse.SparseArray, map[aep.Pollutant]unit.Dimensions, error) {
	var err error
	c.loadOnce.Do(func() {
		c.sp, err = c.setupSpatialProcessor()
	})
	if err != nil {
		return nil, nil, err
	}

	const gi = 0 // Currently, we're only setting things up for one grid.
	emis := make(map[aep.Pollutant][]*sparse.SparseArray)
	units := make(map[aep.Pollutant]unit.Dimensions)
	for _, rec := range recs {
		srg, _, inGrid, err := rec.Spatialize(c.sp, gi)
		if err != nil {
			return nil, nil, err
		}
		if !inGrid {
			continue
		}
		t := rec.Totals()
		for p, totalEmis := range t {
			spatialEmis := srg.ScaleCopy(totalEmis.Value())
			if _, ok := emis[p]; !ok {
				emis[p] = []*sparse.SparseArray{spatialEmis}
				units[p] = totalEmis.Dimensions()
			} else {
				emis[p][0].AddSparse(spatialEmis)
				if !units[p].Matches(totalEmis.Dimensions()) {
					return nil, nil, fmt.Errorf("aeputil.SpatializeTotal: inconsistent units for pollutant %v: %v != %v",
						p, units[p], totalEmis.Dimensions())
				}
			}
		}
	}
	return emis, units, nil
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

// setupSpatialProcessor reads in the necessary information to initialize
// a processor for spatializing emissions, and then does so.
func (c *SpatialConfig) setupSpatialProcessor() (*aep.SpatialProcessor, error) {
	if c.GridCells == nil {
		return nil, fmt.Errorf("aeputil: GridCells must be specified for spatial processor")
	}
	f, err := os.Open(os.ExpandEnv(c.SrgSpec))
	if err != nil {
		return nil, err
	}
	srgSpecs, err := aep.ReadSrgSpec(f, os.ExpandEnv(c.SrgShapefileDirectory), true)
	if err != nil {
		return nil, err
	}
	if err = f.Close(); err != nil {
		return nil, err
	}

	var gridRef *aep.GridRef
	for _, gf := range c.GridRef {
		f, err = os.Open(os.ExpandEnv(gf))
		if err != nil {
			return nil, err
		}
		gridRefTemp, err2 := aep.ReadGridRef(f)
		if err2 != nil {
			return nil, err2
		}
		if err = f.Close(); err != nil {
			return nil, err
		}
		if gridRef == nil {
			gridRef = gridRefTemp
		} else {
			err = gridRef.Merge(*gridRefTemp)
			if err != nil {
				return nil, err
			}
		}
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
