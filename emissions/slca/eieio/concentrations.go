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
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/spatialmodel/inmap/emissions/slca/eieio/eieiorpc"
	"github.com/spatialmodel/inmap/internal/hash"

	"github.com/ctessum/requestcache"
	"gonum.org/v1/gonum/mat"
)

type concRequest struct {
	demand     *mat.VecDense
	industries *Mask
	pol        Pollutant
	year       Year
	loc        Location
	aqm        string
}

// Concentrations returns spatially-explicit pollutant concentrations caused by the
// specified economic demand. emitters
// specifies the emitters concentrations should be calculated for.
// If emitters == nil, combined concentrations for all emitters are calculated.
func (e *SpatialEIO) Concentrations(ctx context.Context, request *eieiorpc.ConcentrationInput) (*eieiorpc.Vector, error) {
	e.loadConcentrationsOnce.Do(func() {
		var c string
		if e.EIEIOCache != "" {
			c = e.EIEIOCache + "/individual"
		}
		e.concentrationsCache = loadCacheOnce(func(ctx context.Context, request interface{}) (interface{}, error) {
			r := request.(*concRequest)
			return e.concentrations(ctx, r.demand, r.industries, r.aqm, r.pol, r.year, r.loc) // Actually calculate the concentrations.
		}, 1, e.MemCacheSize, c, vectorMarshal, vectorUnmarshal)
	})
	req := &concRequest{
		demand:     rpc2vec(request.Demand),
		industries: rpc2mask(request.Emitters),
		pol:        Pollutant(request.Pollutant),
		year:       Year(request.Year),
		loc:        Location(request.Location),
		aqm:        request.AQM,
	}
	rr := e.concentrationsCache.NewRequest(ctx, req, "conc_"+hash.Hash(req))
	resultI, err := rr.Result()
	if err != nil {
		return nil, err
	}
	return vec2rpc(resultI.(*mat.VecDense)), nil
}

// concentrations returns spatially-explicit pollutant concentrations caused by the
// specified economic demand. industries
// specifies the industries emissions should be calculated for.
// If industries == nil, combined emissions for all industries are calculated.
func (e *SpatialEIO) concentrations(ctx context.Context, demand *mat.VecDense, industries *Mask, aqm string, pol Pollutant, year Year, loc Location) (*mat.VecDense, error) {
	cf, err := e.concentrationFactors(ctx, aqm, pol, year)
	if err != nil {
		return nil, err
	}

	activity, err := e.economicImpactsSCC(demand, year, loc)
	if err != nil {
		return nil, err
	}

	if industries != nil {
		// Set activity in industries we're not interested in to zero.
		industries.Mask(activity)
	}

	r, _ := cf.Dims()
	conc := mat.NewVecDense(r, nil)
	conc.MulVec(cf, activity)
	return conc, nil
}

// ConcentrationMatrix returns spatially- and industry-explicit pollution concentrations caused by the
// specified economic demand. In the result matrix, the rows represent air quality
// model grid cells and the columns represent emitters.
func (e *SpatialEIO) ConcentrationMatrix(ctx context.Context, request *eieiorpc.ConcentrationMatrixInput) (*eieiorpc.Matrix, error) {
	cf, err := e.concentrationFactors(ctx, request.AQM, Pollutant(request.Pollutant), Year(request.Year)) // rows = grid cells, cols = industries
	if err != nil {
		return nil, err
	}

	activity, err := e.economicImpactsSCC(array2vec(request.Demand.Data), Year(request.Year), Location(request.Location)) // rows = industries
	if err != nil {
		return nil, err
	}

	r, c := cf.Dims()
	conc := mat.NewDense(r, c, nil)
	conc.Apply(func(_, j int, v float64) float64 {
		// Multiply each emissions factor column by the corresponding activity row.
		return v * activity.At(j, 0)
	}, cf)
	return mat2rpc(conc), nil
}

//go:generate stringer -type=Pollutant

// Pollutant specifies types of airborne pollutant concentrations (not emissions).
type Pollutant int

// These pollutants are PM2.5 and its main components.
const (
	PNH4 Pollutant = iota
	PNO3
	PSO4
	SOA
	PrimaryPM25
	TotalPM25
)

type concAQMPolYear struct {
	pol  Pollutant
	year Year
	aqm  string
}

// loadCacheOnce inititalizes a request cache.
func loadCacheOnce(f requestcache.ProcessFunc, workers, memCacheSize int, cacheLoc string, marshal func(interface{}) ([]byte, error), unmarshal func([]byte) (interface{}, error)) *requestcache.Cache {
	if cacheLoc == "" {
		return requestcache.NewCache(f, workers, requestcache.Deduplicate(),
			requestcache.Memory(memCacheSize))
	} else if strings.HasPrefix(cacheLoc, "http") {
		return requestcache.NewCache(f, workers, requestcache.Deduplicate(),
			requestcache.Memory(memCacheSize), requestcache.HTTP(cacheLoc, unmarshal))
	} else if strings.HasPrefix(cacheLoc, "gs://") {
		loc, err := url.Parse(cacheLoc)
		if err != nil {
			panic(err)
		}
		cf, err := requestcache.GoogleCloudStorage(context.TODO(), loc.Host, strings.TrimLeft(loc.Path, "/"), marshal, unmarshal)
		if err != nil {
			panic(err)
		}
		return requestcache.NewCache(f, workers, requestcache.Deduplicate(),
			requestcache.Memory(memCacheSize), cf)
	}
	return requestcache.NewCache(f, workers, requestcache.Deduplicate(),
		requestcache.Memory(memCacheSize), requestcache.Disk(cacheLoc, marshal, unmarshal))
}

// concentrationFactors returns spatially-explicit pollutant concentrations per unit of economic
// production for each industry. In the result matrix, the rows represent
// air quality model grid cells and the columns represent industries.
func (e *SpatialEIO) concentrationFactors(ctx context.Context, aqm string, pol Pollutant, year Year) (*mat.Dense, error) {
	e.loadConcOnce.Do(func() {
		e.concentrationFactorCache = loadCacheOnce(e.concentrationFactorsWorker, 1, 1, e.EIEIOCache, matrixMarshal, matrixUnmarshal)
	})
	key := fmt.Sprintf("concentrationFactors_%s_%v_%d", aqm, pol, year)
	rr := e.concentrationFactorCache.NewRequest(ctx, concAQMPolYear{aqm: aqm, pol: pol, year: year}, key)
	resultI, err := rr.Result()
	if err != nil {
		return nil, fmt.Errorf("eieio.concentrationFactors: %s, %v", key, err)
	}
	return resultI.(*mat.Dense), nil
}

// concentrationFactors returns spatially-explicit pollution concentrations per unit of economic
// production for each industry. In the result matrix, the rows represent
// air quality model grid cells and the columns represent industries.
func (e *SpatialEIO) concentrationFactorsWorker(ctx context.Context, request interface{}) (interface{}, error) {
	aqmPolyear := request.(concAQMPolYear)
	prod, err := e.domesticProductionSCC(aqmPolyear.year)
	if err != nil {
		return nil, err
	}
	var concFac *mat.Dense
	for i, refTemp := range e.SpatialRefs {
		if len(refTemp.SCCs) == 0 {
			return nil, fmt.Errorf("bea: industry %d; no SCCs", i)
		}
		ref := refTemp
		ref.EmisYear = int(aqmPolyear.year)
		ref.AQM = aqmPolyear.aqm
		concentrations, err := e.CSTConfig.ConcentrationSurrogate(ctx, &ref)
		if err != nil {
			return nil, err
		}
		var industryConc []float64
		switch aqmPolyear.pol {
		case PNH4:
			industryConc = concentrations.PNH4
		case PNO3:
			industryConc = concentrations.PNO3
		case PSO4:
			industryConc = concentrations.PSO4
		case SOA:
			industryConc = concentrations.SOA
		case PrimaryPM25:
			industryConc = concentrations.PrimaryPM25
		case TotalPM25:
			industryConc = concentrations.TotalPM25()
		default:
			return nil, fmt.Errorf("eieio.concentrations: invalid pollutant %v", aqmPolyear.pol)
		}
		if i == 0 {
			concFac = mat.NewDense(len(industryConc), len(e.SpatialRefs), nil)
		}
		for r, v := range industryConc {
			// The concentrations factor is the industry concentrations divided by the
			// industry economic production.
			concFac.Set(r, i, v/prod.At(i, 0))
		}
	}
	return concFac, nil
}
