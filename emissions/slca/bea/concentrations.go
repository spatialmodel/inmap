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
	"context"
	"fmt"
	"strings"

	"github.com/ctessum/requestcache"
	"gonum.org/v1/gonum/mat"
)

// Concentrations returns spatially-explicit pollutant concentrations caused by the
// specified economic demand. industries
// specifies the industries emissions should be calculated for.
// If industries == nil, combined emissions for all industries are calculated.
func (e *SpatialEIO) Concentrations(ctx context.Context, demand *mat.VecDense, industries *Mask, pol Pollutant, year Year, loc Location) (*mat.VecDense, error) {
	cf, err := e.concentrationFactors(ctx, pol, year)
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
// model grid cells and the columns represent industries.
func (e *SpatialEIO) ConcentrationMatrix(ctx context.Context, demand *mat.VecDense, pol Pollutant, year Year, loc Location) (*mat.Dense, error) {
	cf, err := e.concentrationFactors(ctx, pol, year) // rows = grid cells, cols = industries
	if err != nil {
		return nil, err
	}

	activity, err := e.economicImpactsSCC(demand, year, loc) // rows = industries
	if err != nil {
		return nil, err
	}

	r, c := cf.Dims()
	conc := mat.NewDense(r, c, nil)
	conc.Apply(func(_, j int, v float64) float64 {
		// Multiply each emissions factor column by the corresponding activity row.
		return v * activity.At(j, 0)
	}, cf)
	return conc, nil
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

type concPolYear struct {
	pol  Pollutant
	year Year
}

// loadCacheOnce inititalizes a request cache.
func loadCacheOnce(f requestcache.ProcessFunc, workers, memCacheSize int, cacheLoc string, marshal func(interface{}) ([]byte, error), unmarshal func([]byte) (interface{}, error)) *requestcache.Cache {
	if cacheLoc == "" {
		return requestcache.NewCache(f, workers, requestcache.Deduplicate(),
			requestcache.Memory(memCacheSize))
	} else if strings.HasPrefix(cacheLoc, "http") {
		return requestcache.NewCache(f, workers, requestcache.Deduplicate(),
			requestcache.Memory(memCacheSize), requestcache.HTTP(cacheLoc, unmarshal))
	}
	return requestcache.NewCache(f, workers, requestcache.Deduplicate(),
		requestcache.Memory(memCacheSize), requestcache.Disk(cacheLoc, marshal, unmarshal))
}

// concentrationFactors returns spatially-explicit pollutant concentrations per unit of economic
// production for each industry. In the result matrix, the rows represent
// air quality model grid cells and the columns represent industries.
func (e *SpatialEIO) concentrationFactors(ctx context.Context, pol Pollutant, year Year) (*mat.Dense, error) {
	e.loadConcOnce.Do(func() {
		e.concentrationFactorCache = loadCacheOnce(e.concentrationFactorsWorker, 1, 1, e.SpatialCache, matrixMarshal, matrixUnmarshal)
	})
	rr := e.concentrationFactorCache.NewRequest(ctx, concPolYear{pol: pol, year: year}, fmt.Sprintf("concentrationFactors_%v_%d", pol, year))
	resultI, err := rr.Result()
	if err != nil {
		return nil, err
	}
	return resultI.(*mat.Dense), nil
}

// concentrationFactors returns spatially-explicit pollution concentrations per unit of economic
// production for each industry. In the result matrix, the rows represent
// air quality model grid cells and the columns represent industries.
func (e *SpatialEIO) concentrationFactorsWorker(ctx context.Context, request interface{}) (interface{}, error) {
	polyear := request.(concPolYear)
	prod, err := e.domesticProductionSCC(polyear.year)
	if err != nil {
		return nil, err
	}
	var concFac *mat.Dense
	for i, refTemp := range e.SpatialRefs {
		if len(refTemp.SCCs) == 0 {
			return nil, fmt.Errorf("bea: industry %d; no SCCs", i)
		}
		ref := refTemp
		ref.EmisYear = int(polyear.year)
		concentrations, err := e.CSTConfig.ConcentrationSurrogate(ctx, &ref)
		if err != nil {
			return nil, err
		}
		var industryConc []float64
		switch polyear.pol {
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
			return nil, fmt.Errorf("bea.concentrations: invalid pollutant %v", polyear.pol)
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
