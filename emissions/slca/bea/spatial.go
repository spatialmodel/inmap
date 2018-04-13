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

package bea

import (
	"bytes"
	"encoding/gob"
	"io"
	"os"
	"reflect"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/ctessum/requestcache"
	"github.com/spatialmodel/inmap"
	"gonum.org/v1/gonum/mat"

	"github.com/spatialmodel/inmap/emissions/slca"
)

// SpatialEIO implements a spatial EIO LCA model.
type SpatialEIO struct {
	EIO
	CSTConfig slca.CSTConfig

	// SpatialCache specfies the path to the directory to be used to
	// cache spatial information.
	SpatialCache string

	// SpatialRefs are a list of spatial references
	// corresponding to the EIO industries.
	SpatialRefs map[Year][]*slca.SpatialRef

	loadEFOnce               sync.Once
	loadConcOnce             sync.Once
	loadHealthOnce           sync.Once
	emissionFactorCache      *requestcache.Cache
	concentrationFactorCache *requestcache.Cache
	healthFactorCache        *requestcache.Cache
}

// domesticProduction calculates total domestic economic production.
func (e *EIO) domesticProduction(year Year) (*mat.VecDense, error) {
	demand, err := e.FinalDemand(All, nil, year, Domestic)
	if err != nil {
		return nil, err
	}
	return e.EconomicImpacts(demand, year, Domestic)
}

type polYear struct {
	pol  slca.Pollutant
	year Year
}

// SpatialConfig holds configuration information for performing
// spatial EIO LCA.
type SpatialConfig struct {
	SpatialRefFile string

	Config     Config
	SpatialEIO SpatialEIO
}

// NewSpatial creates a new SpatialEIO variable.
func NewSpatial(r io.Reader) (*SpatialEIO, error) {
	c := new(SpatialConfig)
	if _, err := toml.DecodeReader(r, c); err != nil {
		return nil, err
	}
	if err := c.SpatialEIO.CSTConfig.Setup(); err != nil {
		return nil, err
	}
	// expand environment variables.
	expandEnv(reflect.ValueOf(c).Elem())
	eio, err := New(&c.Config)
	if err != nil {
		return nil, err
	}
	c.SpatialEIO.EIO = *eio

	c.SpatialEIO.SpatialRefs = make(map[Year][]*slca.SpatialRef)
	for _, year := range c.Config.Years {
		c.SpatialEIO.SpatialRefs[year], err = NEISpatialRefs(c.SpatialRefFile, year, eio)
		if err != nil {
			return nil, err
		}

		// Multiply light-duty vehicle emissions by 0.05 to account for
		// fact that most vehicle emissions are non-transactional.
		// TODO: Figure out a less-tricky way to do this.
		lightDutyIndex, err := c.SpatialEIO.EIO.IndustryIndex("Couriers and messengers")
		if err != nil {
			return nil, err
		}
		ref := c.SpatialEIO.SpatialRefs[year][lightDutyIndex]
		for i := range ref.SCCFractions {
			ref.SCCFractions[i] *= 0.05
		}

	}
	return &c.SpatialEIO, nil
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
	m := mat.NewDense(0, 0, nil)
	err := m.UnmarshalBinary(b)
	return m, err
}
