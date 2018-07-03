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
	"crypto/sha256"
	"fmt"
	"strconv"

	"github.com/spatialmodel/inmap/emissions/slca"
	eieiorpc "github.com/spatialmodel/inmap/emissions/slca/eieio/grpc/gogrpc"

	"gonum.org/v1/gonum/mat"
)

type emissionsRequest struct {
	demand     *mat.VecDense
	industries *Mask
	pol        slca.Pollutant
	year       Year
	loc        Location
}

func (er *emissionsRequest) Key() string {
	b, err := er.demand.MarshalBinary()
	if err != nil {
		panic(err)
	}
	var b2 []byte
	if er.industries != nil {
		b2, err = (*mat.VecDense)(er.industries).MarshalBinary()
		if err != nil {
			panic(err)
		}
	}
	b3 := []byte(strconv.Itoa((int)(er.pol)))
	b4 := []byte(strconv.Itoa((int)(er.year)))
	b5 := []byte(strconv.Itoa((int)(er.loc)))
	bAll := append(append(append(append(b, b2...), b3...), b4...), b5...)
	bytes := sha256.Sum256(bAll)
	return fmt.Sprintf("emis_%x", bytes[0:sha256.Size])
}

// Emissions returns spatially-explicit emissions caused by the
// specified economic demand. industries
// specifies the industries emissions should be calculated for.
// If industries == nil, combined emissions for all industries are calculated.
func (e *SpatialEIO) Emissions(ctx context.Context, request *eieiorpc.EmissionsInput) (*eieiorpc.Vector, error) {
	e.loadEmissionsOnce.Do(func() {
		var c string
		if e.EIEIOCache != "" {
			c = e.EIEIOCache + "/individual"
		}
		e.emissionsCache = loadCacheOnce(func(ctx context.Context, request interface{}) (interface{}, error) {
			r := request.(*emissionsRequest)
			return e.emissions(ctx, r.demand, r.industries, r.pol, r.year, r.loc) // Actually calculate the emissions.
		}, 1, e.MemCacheSize, c, vectorMarshal, vectorUnmarshal)
	})
	req := &emissionsRequest{
		demand:     array2vec(request.Demand),
		industries: (*Mask)(array2vec(request.Industries)),
		pol:        slca.Pollutant(request.Emission),
		year:       Year(request.Year),
		loc:        Location(request.Location),
	}
	rr := e.emissionsCache.NewRequest(ctx, req, req.Key())
	resultI, err := rr.Result()
	if err != nil {
		return nil, err
	}
	return vec2rpc(resultI.(*mat.VecDense)), nil
}

// emissions returns spatially-explicit emissions caused by the
// specified economic demand. industries
// specifies the industries emissions should be calculated for.
// If industries == nil, combined emissions for all industries are calculated.
func (e *SpatialEIO) emissions(ctx context.Context, demand *mat.VecDense, industries *Mask, pol slca.Pollutant, year Year, loc Location) (*mat.VecDense, error) {
	// Calculate emission factors. matrix dimension: [# grid cells, # industries]
	ef, err := e.emissionFactors(ctx, pol, year)
	if err != nil {
		return nil, err
	}

	// Calculate economic activity. vector dimension: [# industries, 1]
	activity, err := e.economicImpactsSCC(demand, year, loc)
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
func (e *SpatialEIO) EmissionsMatrix(ctx context.Context, request *eieiorpc.EmissionsMatrixInput) (*eieiorpc.Matrix, error) {
	ef, err := e.emissionFactors(ctx, slca.Pollutant(request.Emission), Year(request.Year)) // rows = grid cells, cols = industries
	if err != nil {
		return nil, err
	}

	activity, err := e.economicImpactsSCC(array2vec(request.Demand), Year(request.Year), Location(request.Location)) // rows = industries
	if err != nil {
		return nil, err
	}

	r, c := ef.Dims()
	emis := mat.NewDense(r, c, nil)
	emis.Apply(func(_, j int, v float64) float64 {
		// Multiply each emissions factor column by the corresponding activity row.
		return v * activity.At(j, 0)
	}, ef)
	return mat2rpc(emis), nil
}

// emissionFactors returns spatially-explicit emissions per unit of economic
// production for each industry. In the result matrix, the rows represent
// air quality model grid cells and the columns represent industries.
func (e *SpatialEIO) emissionFactors(ctx context.Context, pol slca.Pollutant, year Year) (*mat.Dense, error) {
	e.loadEFOnce.Do(func() {
		e.emissionFactorCache = loadCacheOnce(e.emissionFactorsWorker, 1, 1, e.EIEIOCache,
			matrixMarshal, matrixUnmarshal)
	})
	key := fmt.Sprintf("emissionFactors_%v_%d", pol, year)
	rr := e.emissionFactorCache.NewRequest(ctx, polYear{pol: pol, year: year}, key)
	resultI, err := rr.Result()
	if err != nil {
		return nil, fmt.Errorf("eieio.emissionFactors: %s: %v", key, err)
	}
	return resultI.(*mat.Dense), nil
}

// emissionFactors returns spatially-explicit emissions per unit of economic
// production for each industry. In the result matrix, the rows represent
// air quality model grid cells and the columns represent industries.
func (e *SpatialEIO) emissionFactorsWorker(ctx context.Context, request interface{}) (interface{}, error) {
	polyear := request.(polYear)
	prod, err := e.domesticProductionSCC(polyear.year)
	if err != nil {
		return nil, err
	}
	var emisFac *mat.Dense
	for i, refTemp := range e.SpatialRefs {
		if len(refTemp.SCCs) == 0 {
			return nil, fmt.Errorf("bea: industry %d; no SCCs", i)
		}
		ref := refTemp
		ref.EmisYear = int(polyear.year)
		industryEmis, err := e.CSTConfig.EmissionsSurrogate(ctx, polyear.pol, &ref)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			emisFac = mat.NewDense(industryEmis.Shape[0], len(e.SpatialRefs), nil)
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
