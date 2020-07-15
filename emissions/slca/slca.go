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
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"sync"

	"github.com/spatialmodel/inmap/emissions/aep"
	"github.com/spatialmodel/inmap/emissions/aep/aeputil"
	"github.com/spatialmodel/inmap/epi"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/requestcache"
	"github.com/spatialmodel/inmap/sr"
)

// DB is a holder for an SLCA database.
type DB struct {
	LCADB

	*http.ServeMux

	// Chemical, spatial, and temporal (CST) configuration
	CSTConfig *CSTConfig
}

// CSTConfig holds Chemical, spatial, and temporal (CST) configuration
type CSTConfig struct {
	// PolTrans translates between GREET pollutants and InMAP pollutants.
	// Format is map[GREET pollution]InMAP pollutant. There must be an
	// entry for each GREET pollutant. GREET pollutants that do not have
	// a corresponding InMAP species should be set equal to "none".
	PolTrans map[string]string

	// SRFile gives the location of the InMAP SR matrix data files in a
	// map where each key is an identifying name of an air quality model
	// and each value is the path to an SR matrix file.
	SRFiles map[string]string

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

	// FugitiveDustAdjustment specifies the path to the files---one for
	// each air quality model--that contain
	// grid-cell specific fugitive dust adjustment factors.
	FugitiveDustAdjustment map[string]string

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
	// surrogates. Set to "00000" by default.
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

	speciator             *aep.Speciator
	speciateOnce          sync.Once
	loadSROnce            sync.Once
	loadSpatialOnce       sync.Once
	loadConcentrationOnce sync.Once
	loadHealthOnce        sync.Once
	loadPopulationOnce    sync.Once
	loadEvalConcOnce      sync.Once
	loadEvalEmisOnce      sync.Once
	loadEvalHealthOnce    sync.Once
	loadCROnce            sync.Once
	loadGeometryOnce      sync.Once

	// NEI emissions data cache
	emisCache struct {
		mx sync.Mutex

		// year and aqm identify the setup we currently have loaded.
		year int
		aqm  string

		// emissions records: Format: map[Year][sector][]records
		emisRecords map[string][]aep.RecordGridded
	}

	// sr matrix data cache. These fields should never be accessed
	// directly, instead use c.srSetup.
	srCache struct {
		mx sync.Mutex

		// aqm identifies the setup we currently have loaded.
		aqm string

		sr            *sr.Reader
		spatialConfig *aeputil.SpatialConfig
		gridIndex     *rtree.Rtree
	}

	// hr holds the registered hazard ratio functions.
	hr map[string]epi.HRer
}

// setup sets up the chemical, spatial, and temporal configuration, where
// hr specifies the hazard ratio functions that should be included.
func (c *CSTConfig) Setup(hr ...epi.HRer) error {
	expandEnv(reflect.ValueOf(c).Elem())

	if c.DefaultFIPS == "" {
		c.DefaultFIPS = "00000"
	}

	for k, v := range c.SRFiles {
		c.SRFiles[k] = os.ExpandEnv(v)
	}

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
	c.hr = make(map[string]epi.HRer)
	for _, h := range hr {
		c.hr[h.Name()] = h
	}
	return nil
}

type gridIndex struct {
	geom.Polygonal
	i int
}

func init() {
	gob.Register([]geom.Polygonal{})
	gob.Register(geom.Polygon{})
}

// Geometry returns the air quality model grid cell geometry when
// given an identifier for which air quality model to use.
func (c *CSTConfig) Geometry(aqm string) ([]geom.Polygonal, error) {
	c.loadGeometryOnce.Do(func() {
		c.geometryCache = loadCacheOnce(c.geometry, 1, 1, c.SpatialCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	req := c.geometryCache.NewRequest(context.Background(), aqm, "geometry_"+aqm)
	iface, err := req.Result()
	if err != nil {
		return nil, err
	}
	return iface.([]geom.Polygonal), nil
}

// geometry returns the air quality model grid cell geometry.
func (c *CSTConfig) geometry(ctx context.Context, aqmI interface{}) (interface{}, error) {
	spatialConfig, _, _, err := c.srSetup(aqmI.(string))
	if err != nil {
		return nil, err
	}
	return spatialConfig.GridCells, nil
}

// srSetup returns a spatial processing configuration for the
// specified air quality model.
func (c *CSTConfig) srSetup(aqm string) (*aeputil.SpatialConfig, *sr.Reader, *rtree.Rtree, error) {
	c.srCache.mx.Lock()
	defer c.srCache.mx.Unlock()

	if aqm == "" {
		return nil, nil, nil, errors.New("air quality model is not specified")
	}

	if aqm == c.srCache.aqm {
		// If the air quality model we want is already loaded, return it.
		return c.srCache.spatialConfig, c.srCache.sr, c.srCache.gridIndex, nil
	}

	// Make a copy of the spatial configuration to allow the
	// use of multiple grids.
	c.srCache.spatialConfig = &aeputil.SpatialConfig{
		SrgSpecSMOKE:          c.SpatialConfig.SrgSpecSMOKE,
		SrgSpecOSM:            c.SpatialConfig.SrgSpecOSM,
		SrgShapefileDirectory: c.SpatialConfig.SrgShapefileDirectory,
		SCCExactMatch:         c.SpatialConfig.SCCExactMatch,
		GridRef:               c.SpatialConfig.GridRef,
		OutputSR:              c.SpatialConfig.OutputSR,
		InputSR:               c.SpatialConfig.InputSR,
		SimplifyTolerance:     c.SpatialConfig.SimplifyTolerance,
		SpatialCache:          c.SpatialConfig.SpatialCache,
		MaxCacheEntries:       c.SpatialConfig.MaxCacheEntries,
		GridName:              aqm,
	}

	srFile, ok := c.SRFiles[aqm]
	if !ok {
		return nil, nil, nil, fmt.Errorf("air quality model `%s` is not included in config.SRFiles; valid aqms include %v", aqm, c.SRFiles)
	}

	f, err := os.Open(srFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("slca: opening sr matrix file: %w", err)
	}
	c.srCache.sr, err = sr.NewReader(f)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("slca: opening sr matrix file: %w", err)
	}
	if c.SRCacheSize != 0 {
		c.srCache.sr.CacheSize = c.SRCacheSize
	}
	c.srCache.spatialConfig.GridCells = c.srCache.sr.Geometry()
	c.srCache.gridIndex = rtree.NewTree(25, 50)
	for i, g := range c.srCache.spatialConfig.GridCells {
		c.srCache.gridIndex.Insert(gridIndex{Polygonal: g, i: i})
	}
	c.srCache.aqm = aqm
	return c.srCache.spatialConfig, c.srCache.sr, c.srCache.gridIndex, nil
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
