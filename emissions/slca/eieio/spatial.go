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

package eieio

import (
	"bytes"
	"context"
	"encoding/gob"
	"os"
	"reflect"
	"sync"

	"github.com/ctessum/requestcache"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/emissions/aep"
	"github.com/spatialmodel/inmap/epi"
	"gonum.org/v1/gonum/mat"

	"github.com/spatialmodel/inmap/emissions/slca"
	"github.com/spatialmodel/inmap/emissions/slca/eieio/eieiorpc"
)

// SpatialEIO implements a spatial EIO LCA model.
type SpatialEIO struct {
	EIO
	slca.CSTConfig

	// EIEIOCache specfies the path to the directory to be used to
	// cache spatial information.
	EIEIOCache string

	// MemCacheSize is the size of the memory cache for emissions, concentration,
	// and health spatial results.
	MemCacheSize int

	// SCCs are the source codes for emissions sources in the model.
	SCCs []slca.SCC

	// sccIndex maps SCC codes to indices in the SCCs list.
	sccIndex map[slca.SCC]int

	// sccMap provides a mapping between the SCC codes ('SCCs' above)
	// and IO industries, where the outer index is the SCC code  and the
	// inner index is the IO industry.
	sccMap [][]int

	// SpatialRefs are a list of spatial references
	// corresponding to the SCCs.
	SpatialRefs []slca.SpatialRef

	sccDescriptions map[string]string

	totalRequirementsSCC    map[Year]*mat.Dense
	domesticRequirementsSCC map[Year]*mat.Dense
	importRequirementsSCC   map[Year]*mat.Dense

	loadEmissionsOnce        sync.Once
	loadConcentrationsOnce   sync.Once
	loadHealthOnce           sync.Once
	loadEFOnce               sync.Once
	loadConcOnce             sync.Once
	loadHealthFactorsOnce    sync.Once
	emissionsCache           *requestcache.Cache
	concentrationsCache      *requestcache.Cache
	healthCache              *requestcache.Cache
	emissionFactorCache      *requestcache.Cache
	concentrationFactorCache *requestcache.Cache
	healthFactorCache        *requestcache.Cache

	// hr holds a registry of hazard ratio functions.
	hr map[string]epi.HRer
}

// domesticProduction calculates total domestic economic production.
func (e *SpatialEIO) domesticProductionSCC(year Year) (*mat.VecDense, error) {
	demand, err := e.FinalDemand(context.TODO(), &eieiorpc.FinalDemandInput{
		FinalDemandType: eieiorpc.FinalDemandType_AllDemand,
		Year:            int32(year),
		Location:        eieiorpc.Location_Domestic,
	})
	if err != nil {
		return nil, err
	}
	return e.economicImpactsSCC(array2vec(demand.Data), year, Domestic)
}

type aqmPolYear struct {
	aqm  string
	pol  slca.Pollutant
	year Year
}

// SpatialConfig holds configuration information for performing
// spatial EIO LCA.
type SpatialConfig struct {
	SCCMapFile         string
	SCCDescriptionFile string

	Config     Config
	SpatialEIO SpatialEIO
}

// NewSpatial creates a new SpatialEIO variable.
func NewSpatial(c *SpatialConfig, hr ...epi.HRer) (*SpatialEIO, error) {
	if err := c.SpatialEIO.CSTConfig.Setup(hr...); err != nil {
		return nil, err
	}
	// expand environment variables.
	expandEnv(reflect.ValueOf(c).Elem())
	eio, err := New(&c.Config)
	if err != nil {
		return nil, err
	}
	c.SpatialEIO.EIO = *eio

	if err := c.SpatialEIO.loadSCCMap(c.SCCMapFile); err != nil {
		return nil, err
	}
	s := &c.SpatialEIO

	s.totalRequirementsSCC = make(map[Year]*mat.Dense)
	s.domesticRequirementsSCC = make(map[Year]*mat.Dense)
	s.importRequirementsSCC = make(map[Year]*mat.Dense)
	for year := range s.EIO.totalRequirements {
		s.totalRequirementsSCC[year], err = s.requirementsSCC(s.EIO.totalRequirements[year])
		if err != nil {
			return nil, err
		}
		s.domesticRequirementsSCC[year], err = s.requirementsSCC(s.EIO.domesticRequirements[year])
		if err != nil {
			return nil, err
		}

		imports := new(mat.Dense)
		imports.Sub(s.totalRequirementsSCC[year], s.domesticRequirementsSCC[year])
		s.importRequirementsSCC[year] = imports
	}

	f, err := os.Open(c.SCCDescriptionFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s.sccDescriptions, err = aep.SCCDescription(f)
	if err != nil {
		return nil, err
	}

	s.hr = make(map[string]epi.HRer)
	for _, r := range hr {
		s.hr[r.Name()] = r
	}

	return s, nil
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

// unmarshalGob unmarshals an interface from a byte array and fulfills
// the requirements for the Disk cache unmarshalFunc input.
func unmarshalGob(b []byte) (interface{}, error) {
	r := bytes.NewBuffer(b)
	d := gob.NewDecoder(r)
	var data []*inmap.EmisRecord
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
	dd := d.([]*inmap.EmisRecord)
	if err := e.Encode(dd); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

// matrixMarshal converts a matrix to a byte array for storing in a cache.
func matrixMarshal(data interface{}) ([]byte, error) {
	i := data.(*interface{})
	m := (*i).(*mat.Dense)
	return m.MarshalBinary()
}

// matrixMarshal converts a byte array to a matrix after storing it in a cache.
func matrixUnmarshal(b []byte) (interface{}, error) {
	m := new(mat.Dense)
	err := m.UnmarshalBinary(b)
	return m, err
}

// matrixMarshal converts a matrix to a byte array for storing in a cache.
func vectorMarshal(data interface{}) ([]byte, error) {
	i := data.(*interface{})
	m := (*i).(*mat.VecDense)
	return m.MarshalBinary()
}

// matrixMarshal converts a byte array to a matrix after storing it in a cache.
func vectorUnmarshal(b []byte) (interface{}, error) {
	m := new(mat.VecDense)
	m.Reset()
	err := m.UnmarshalBinary(b)
	return m, err
}
