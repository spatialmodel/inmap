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

	"github.com/spatialmodel/inmap/emissions/slca"

	"github.com/ctessum/requestcache"
	"gonum.org/v1/gonum/mat"
)

// Emissions returns spatially-explicit emissions caused by the
// specified economic demand. industries
// specifies the industries emissions should be calculated for.
// If industries == nil, combined emissions for all industries are calculated.
func (e *SpatialEIO) Emissions(ctx context.Context, demand *mat.VecDense, industries *Mask, pol slca.Pollutant, year Year, loc Location) (*mat.VecDense, error) {
	// Calculate emission factors. matrix dimension: [# grid cells, # industries]
	ef, err := e.emissionFactors(ctx, pol, year)
	if err != nil {
		return nil, err
	}

	// Calculate economic activity. vector dimension: [# industries, 1]
	activity, err := e.EconomicImpacts(demand, year, loc)
	if err != nil {
		return nil, err
	}

	if industries != nil {
		// Set activity in industries we're not interested in to zero.
		industries.Mask(activity)
	}

	r, _ := ef.Dims()
	emis := mat.NewVecDense(r, nil)
	emis.MulVec(ef, activity)
	return emis, nil
}

// EmissionsMatrix returns spatially- and industry-explicit emissions caused by the
// specified economic demand. In the result matrix, the rows represent air quality
// model grid cells and the columns represent industries.
func (e *SpatialEIO) EmissionsMatrix(ctx context.Context, demand *mat.VecDense, pol slca.Pollutant, year Year, loc Location) (*mat.Dense, error) {
	ef, err := e.emissionFactors(ctx, pol, year) // rows = grid cells, cols = industries
	if err != nil {
		return nil, err
	}

	activity, err := e.EconomicImpacts(demand, year, loc) // rows = industries
	if err != nil {
		return nil, err
	}

	r, c := ef.Dims()
	emis := mat.NewDense(r, c, nil)
	emis.Apply(func(_, j int, v float64) float64 {
		// Multiply each emissions factor column by the corresponding activity row.
		return v * activity.At(j, 0)
	}, ef)
	return emis, nil
}

// emissionFactors returns spatially-explicit emissions per unit of economic
// production for each industry. In the result matrix, the rows represent
// air quality model grid cells and the columns represent industries.
func (e *SpatialEIO) emissionFactors(ctx context.Context, pol slca.Pollutant, year Year) (*mat.Dense, error) {
	e.loadEFOnce.Do(func() {
		if e.SpatialCache == "" {
			e.emissionFactorCache = requestcache.NewCache(e.emissionFactorsWorker, 1, requestcache.Deduplicate(),
				requestcache.Memory(1))
		} else {
			e.emissionFactorCache = requestcache.NewCache(e.emissionFactorsWorker, 1, requestcache.Deduplicate(),
				requestcache.Memory(1), requestcache.Disk(e.SpatialCache, matrixMarshal, matrixUnmarshal))
		}
	})
	rr := e.emissionFactorCache.NewRequest(ctx, polYear{pol: pol, year: year}, fmt.Sprintf("emissionFactors_%v_%d", pol, year))
	resultI, err := rr.Result()
	if err != nil {
		return nil, err
	}
	return resultI.(*mat.Dense), nil
}

// emissionFactors returns spatially-explicit emissions per unit of economic
// production for each industry. In the result matrix, the rows represent
// air quality model grid cells and the columns represent industries.
func (e *SpatialEIO) emissionFactorsWorker(ctx context.Context, request interface{}) (interface{}, error) {
	polyear := request.(polYear)
	prod, err := e.EIO.domesticProduction(polyear.year)
	if err != nil {
		return nil, err
	}
	spatialRefs, ok := e.SpatialRefs[polyear.year]
	if !ok {
		return nil, fmt.Errorf("bea: SpatialEIO missing SpatialRefs for year %d", polyear.year)
	}
	var emisFac *mat.Dense
	for i, ref := range spatialRefs {
		if len(ref.SCCs) == 0 {
			continue
		}
		industryEmis, err := e.CSTConfig.EmissionsSurrogate(ctx, polyear.pol, ref)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			emisFac = mat.NewDense(industryEmis.Shape[0], len(spatialRefs), nil)
		}
		for r, v := range industryEmis.Elements {
			// The emissions factor is the industry emissions divided by the
			// industry economic production.
			if p := prod.At(i, 0); p != 0 {
				emisFac.Set(r, i, v/prod.At(i, 0))
			}
		}
	}
	return emisFac, nil
}
