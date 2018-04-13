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
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.*/

package slca

import (
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"sync"

	"github.com/spatialmodel/inmap/emissions/aep"
	"github.com/spatialmodel/inmap/emissions/aep/aeputil"

	"bitbucket.org/ctessum/cdf"
	"github.com/BurntSushi/toml"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/requestcache"
	"github.com/spatialmodel/inmap/sr"
)

// DB is a holder for an SLCA database.
type DB struct {
	LCADB

	// Chemical, spatial, and temporal (CST) configuration
	CSTConfig *CSTConfig
}

// LoadDB loads the LCA database and chemical, spatial, and
// temporal (CST) information.
func LoadDB(lca LCADB, cstConfigFile io.Reader) (*DB, error) {
	db := new(DB)
	db.LCADB = lca

	if _, err := toml.DecodeReader(cstConfigFile, db.CSTConfig); err != nil {
		return nil, err
	}
	if err := db.CSTConfig.Setup(); err != nil {
		return nil, err
	}

	return db, nil
}

// CSTConfig holds Chemical, spatial, and temporal (CST) configuration
type CSTConfig struct {
	// PolTrans translates between GREET pollutants and InMAP pollutants.
	// Format is map[GREET pollution]InMAP pollutant. There must be an
	// entry for each GREET pollutant. GREET pollutants that do not have
	// a corresponding InMAP species should be set equal to "none".
	PolTrans map[string]string

	// SRFile gives the location of the InMAP SR matrix data file.
	SRFile string

	// SRCacheSize specifies the number of SR records to hold in an in-memory
	// cache to speed up air pollution computations. 1 GB of RAM is required for
	// approximately every 1,000 records.
	SRCacheSize int

	// CensusFile specifies the location of the shapefile holding
	// population counts for each analysis year.
	CensusFile map[string]string
	censusFile map[int]string

	// CensusPopColumns specifies the names of the population attribute
	// columns in CensusFile.
	CensusPopColumns []string

	// MortalityRateFile specifies the location of the shapefile holding
	// baseline mortality rate information for each analysis year.
	MortalityRateFile map[string]string
	mortalityRateFile map[int]string

	// MortalityRateColumns maps population groups to fields in the mortality rate
	// shapefile containing their respective the mortality rates.
	MortalityRateColumns map[string]string

	// InventoryConfig specifies the configuration of the emissions inventory
	// data used for spatial surrogates.
	InventoryConfig aeputil.InventoryConfig

	// FugitiveDustSector specifies the name of the sectors, if any,
	// that should have a fugitive dust adjustment applied to them.
	FugitiveDustSectors []string

	// FugitiveDustAdjustment specifies the path to the file that contains
	// grid-cell specific fugitive dust adjustment factors.
	FugitiveDustAdjustment string

	// InventoryConfig specifies the configuration of the emissions inventory
	// data used for air quality model evaluation and calculating average
	// concentration response.
	EvaluationInventoryConfig aeputil.InventoryConfig

	// AdditionalEmissionsShapefilesForEvaluation specifies additional emissions
	// shapefiles to be used for model evaluation. This field will be removed
	// when a better way of doing biogenic emissions is completed. Units = tons/year.
	AdditionalEmissionsShapefilesForEvaluation []string

	// SpatialConfig specifies the spatial configuration used for
	// creating spatial surrogates and for air quality model evaluation and
	// calculating average concentration response.
	SpatialConfig aeputil.SpatialConfig

	// NEIData holds information needed for processing the NEI.
	NEIData struct {
		// These variables specify the locations of files used for
		// chemical speciation.
		SpecRef, SpecRefCombo, SpeciesProperties, GasProfile   string
		GasSpecies, OtherGasSpecies, PMSpecies, MechAssignment string
		MolarWeight, SpeciesInfo                               string

		// ChemicalMechanism specifies which chemical mechanism to
		// use for speciation.
		ChemicalMechanism string

		// MassSpeciation specifies whether to use mass speciation.
		// If false, speciation will convert values to moles.
		MassSpeciation bool

		SCCExactMatch bool
	}

	// NEIBaseYear specifies the year the input NEI emissions data is for,
	// for use in scaling emissions for other years.
	NEIBaseYear int

	// NEITrends specifies the file holding trends in NEI  emissions, downloadable
	// from https://www.epa.gov/air-emissions-inventories/air-pollutant-emissions-trends-data.
	NEITrends string

	// SCCReference specifies the file holding cross references between SCC
	// codes and tier 1 summary codes, for use in scaling emissions for other
	// years, downloadable from https://ofmpub.epa.gov/sccsearch/.
	SCCReference string

	// ConcentrationCache specifies the location for storing concentration
	// data for quick access. If this is left empty, no cache will be used.
	ConcentrationCache string

	// HealthCache specifies the location for storing concentration
	// data for quick access. If this is left empty, no cache will be used.
	HealthCache string

	// SpatialCache specifies the location for storing spatial emissions
	// data for quick access. If this is left empty, no cache will be used.
	SpatialCache string

	// MaxCacheEntries specifies the maximum number of emissions and concentrations
	// surrogates to hold in a memory cache. Larger numbers can result in faster
	// processing but increased memory usage.
	MaxCacheEntries int

	// concRequestCache is a cache for inmap
	// concentration surrogates.
	concRequestCache *requestcache.Cache

	// DefaultFIPS specifies the default FIPS code to use when retrieving gridding
	// surrogates.
	DefaultFIPS string

	// spatializeRequestCache is a cache for spatialized
	// emissions surrogates.
	spatializeRequestCache *requestcache.Cache

	// healthRequestCache is a cache for spatialized health impacts.
	healthRequestCache *requestcache.Cache

	// crRequestCache is a cache for average concentration response values.
	crRequestCache *requestcache.Cache

	// popRequestCache is a cache for gridded population counts.
	popRequestCache *requestcache.Cache

	// evalEmisRequestCache is a cache for gridded emissions predictions for
	// model evaluation.
	evalEmisRequestCache *requestcache.Cache
	// evalConcRequestCache is a cache for gridded concentration predictions for
	// model evaluation.
	evalConcRequestCache *requestcache.Cache
	// evalHealthRequestCache is a cache for gridded healths predictions for
	// model evaluation.
	evalHealthRequestCache *requestcache.Cache
	// geometryCache is a cache for grid cell geometry.
	geometryCache *requestcache.Cache

	sp                    *aep.SpatialProcessor
	speciator             *aep.Speciator
	speciateOnce          sync.Once
	loadSROnce            sync.Once
	loadInventoryOnce     sync.Once
	loadSpatialOnce       sync.Once
	loadConcentrationOnce sync.Once
	loadHealthOnce        sync.Once
	loadPopulationOnce    sync.Once
	loadEvalConcOnce      sync.Once
	loadEvalEmisOnce      sync.Once
	loadEvalHealthOnce    sync.Once
	loadCROnce            sync.Once
	loadGeometryOnce      sync.Once

	// NEI emissions data. Format: map[Year][sector][]records
	emis map[int]map[string][]aep.Record

	sr *sr.Reader

	gridIndex *rtree.Rtree
}

// setup sets up the chemical, spatial, and temporal configuration.
func (c *CSTConfig) Setup() error {
	expandEnv(reflect.ValueOf(c).Elem())

	c.censusFile = make(map[int]string)
	for ys, f := range c.CensusFile {
		yi, err := strconv.ParseInt(ys, 10, 64)
		if err != nil {
			return fmt.Errorf("slca: CensusFile year '%s' is not valid: %s", ys, err)
		}
		c.censusFile[int(yi)] = os.ExpandEnv(f)
	}
	c.mortalityRateFile = make(map[int]string)
	for ys, f := range c.MortalityRateFile {
		yi, err := strconv.ParseInt(ys, 10, 64)
		if err != nil {
			return fmt.Errorf("slca: MortalityRateFile year '%s' is not valid: %s", ys, err)
		}
		c.mortalityRateFile[int(yi)] = os.ExpandEnv(f)
	}
	for i, f := range c.AdditionalEmissionsShapefilesForEvaluation {
		c.AdditionalEmissionsShapefilesForEvaluation[i] = os.ExpandEnv(f)
	}
	return nil
}

type gridIndex struct {
	geom.Polygonal
	i int
}

func init() {
	gob.Register([]geom.Polygonal{})
}

// Geometry returns the air quality model grid cell geometry.
func (c *CSTConfig) Geometry() ([]geom.Polygonal, error) {
	c.loadGeometryOnce.Do(func() {
		if c.SpatialCache == "" {
			c.geometryCache = requestcache.NewCache(c.geometry, runtime.GOMAXPROCS(-1),
				requestcache.Deduplicate(), requestcache.Memory(1))
		} else {
			c.geometryCache = requestcache.NewCache(c.geometry, runtime.GOMAXPROCS(-1),
				requestcache.Deduplicate(), requestcache.Memory(1),
				requestcache.Disk(c.SpatialCache, requestcache.MarshalGob, requestcache.UnmarshalGob),
			)
		}
	})
	req := c.geometryCache.NewRequest(context.Background(), nil, "geometry")
	iface, err := req.Result()
	if err != nil {
		return nil, err
	}
	return iface.([]geom.Polygonal), nil

}

// geometry returns the air quality model grid cell geometry.
func (c *CSTConfig) geometry(ctx context.Context, _ interface{}) (interface{}, error) {
	if err := c.lazyLoadSR(); err != nil {
		return nil, err
	}
	return c.SpatialConfig.GridCells, nil
}

func (c *CSTConfig) lazyLoadSR() error {
	var err error
	c.loadSROnce.Do(func() {
		var f cdf.ReaderWriterAt
		f, err = os.Open(c.SRFile)
		if err != nil {
			return
		}
		c.sr, err = sr.NewReader(f)
		if err != nil {
			return
		}
		if c.SRCacheSize != 0 {
			c.sr.CacheSize = c.SRCacheSize
		}
		c.SpatialConfig.GridCells = c.sr.Geometry()
		c.gridIndex = rtree.NewTree(25, 50)
		for i, g := range c.SpatialConfig.GridCells {
			c.gridIndex.Insert(gridIndex{Polygonal: g, i: i})
		}
	})
	if err != nil {
		return fmt.Errorf("slca: opening SR matrix: %v", err)
	}
	return nil
}

// expandEnv expands the environment variables in v.
func expandEnv(v reflect.Value) {
	t := v.Type()
	if !v.CanSet() {
		return
	}
	switch t.Kind() {
	case reflect.String:
		v.SetString(os.ExpandEnv(v.String()))
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			expandEnv(v.Index(i))
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			expandEnv(v.MapIndex(key))
		}
	case reflect.Ptr:
		expandEnv(v.Elem())
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			expandEnv(v.Field(i))
		}
	}
}
