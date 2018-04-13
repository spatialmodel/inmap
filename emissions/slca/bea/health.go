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

	"bitbucket.org/ctessum/sparse"

	"github.com/ctessum/requestcache"
	"github.com/spatialmodel/epi"
	"gonum.org/v1/gonum/mat"
)

// Health returns spatially-explicit pollutant air quality-related health impacts caused by the
// specified economic demand.  industries
// specify the industries emissions should be calculated for.
// If industries == nil, combined emissions for all industries are calculated.
// pop must be one of the population types defined in the configuration file.
func (e *SpatialEIO) Health(ctx context.Context, demand *mat.VecDense, industries *Mask, pol Pollutant, pop string, year Year, loc Location, HR epi.HRer) (*mat.VecDense, error) {
	hf, err := e.healthFactors(ctx, pol, pop, year, HR)
	if err != nil {
		return nil, err
	}

	activity, err := e.EconomicImpacts(demand, year, loc)
	if err != nil {
		return nil, err
	}

	if industries != nil {
		// Set activity in industries we're not interested in to zero.
		industries.Mask(activity)
	}

	r, _ := hf.Dims()
	health := mat.NewVecDense(r, nil)
	health.MulVec(hf, activity)
	return health, nil
}

// HealthMatrix returns spatially- and industry-explicit air quality-related health impacts caused by the
// specified economic demand. In the result matrix, the rows represent air quality
// model grid cells and the columns represent industries.
func (e *SpatialEIO) HealthMatrix(ctx context.Context, demand *mat.VecDense, pol Pollutant, pop string, year Year, loc Location, HR epi.HRer) (*mat.Dense, error) {
	hf, err := e.healthFactors(ctx, pol, pop, year, HR) // rows = grid cells, cols = industries
	if err != nil {
		return nil, err
	}

	activity, err := e.EconomicImpacts(demand, year, loc) // rows = industries
	if err != nil {
		return nil, err
	}

	r, c := hf.Dims()
	health := mat.NewDense(r, c, nil)
	health.Apply(func(_, j int, v float64) float64 {
		// Multiply each emissions factor column by the corresponding activity row.
		return v * activity.At(j, 0)
	}, hf)
	return health, nil
}

type concPolPopYearHR struct {
	pol  Pollutant
	year Year
	pop  string
	hr   epi.HRer
}

// healthFactors returns spatially-explicit air quality-related health impacts per unit of economic
// production for each industry. In the result matrix, the rows represent
// air quality model grid cells and the columns represent industries.
func (e *SpatialEIO) healthFactors(ctx context.Context, pol Pollutant, pop string, year Year, HR epi.HRer) (*mat.Dense, error) {
	e.loadHealthOnce.Do(func() {
		if e.SpatialCache == "" {
			e.healthFactorCache = requestcache.NewCache(e.healthFactorsWorker, 1, requestcache.Deduplicate(),
				requestcache.Memory(1))
		} else {
			e.healthFactorCache = requestcache.NewCache(e.healthFactorsWorker, 1, requestcache.Deduplicate(),
				requestcache.Memory(1), requestcache.Disk(e.SpatialCache, matrixMarshal, matrixUnmarshal))
		}
	})
	rr := e.healthFactorCache.NewRequest(ctx, concPolPopYearHR{pol: pol, year: year, pop: pop, hr: HR}, fmt.Sprintf("healthFactors_%v_%v_%d_%s", pol, pop, year, HR.Name()))
	resultI, err := rr.Result()
	if err != nil {
		return nil, err
	}
	return resultI.(*mat.Dense), nil
}

// healthFactorsWorker returns spatially-explicit pollution concentrations per unit of economic
// production for each industry. In the result matrix, the rows represent
// air quality model grid cells and the columns represent industries.
func (e *SpatialEIO) healthFactorsWorker(ctx context.Context, request interface{}) (interface{}, error) {
	polyearHR := request.(concPolPopYearHR)
	prod, err := e.EIO.domesticProduction(polyearHR.year)
	if err != nil {
		return nil, err
	}
	spatialRefs, ok := e.SpatialRefs[polyearHR.year]
	if !ok {
		return nil, fmt.Errorf("bea: SpatialEIO missing SpatialRefs for year %d", polyearHR.year)
	}
	var healthFac *mat.Dense
	for i, ref := range spatialRefs {
		if len(ref.SCCs) == 0 {
			continue
		}
		health, err := e.CSTConfig.HealthSurrogate(ctx, ref, polyearHR.hr)
		if err != nil {
			return nil, err
		}
		var industryHealth *sparse.DenseArray
		var pol string
		switch polyearHR.pol {
		case PNH4:
			pol = "pNH4"
		case PNO3:
			pol = "pNO3"
		case PSO4:
			pol = "pSO4"
		case SOA:
			pol = "SOA"
		case PrimaryPM25:
			pol = "PrimaryPM25"
		case TotalPM25:
			pol = "TotalPM2_5"
		default:
			return nil, fmt.Errorf("bea.health: invalid pollutant %v", polyearHR.pol)
		}
		if _, ok := health[polyearHR.pop]; !ok {
			return nil, fmt.Errorf("bea.health: invalid population %v", polyearHR.pop)
		}
		industryHealth, ok := health[polyearHR.pop][pol]
		if !ok {
			return nil, fmt.Errorf("bea.health: invalid pollutant %v", pol)
		}
		if i == 0 {
			healthFac = mat.NewDense(industryHealth.Shape[0], len(e.SpatialRefs[polyearHR.year]), nil)
		}
		for r, v := range industryHealth.Elements {
			// The health factor is the industry health impacts divided by the
			// industry economic production.
			healthFac.Set(r, i, v/prod.At(i, 0))
		}
	}
	return healthFac, nil
}
