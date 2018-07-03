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

	"bitbucket.org/ctessum/sparse"

	"github.com/spatialmodel/epi"
	eieiorpc "github.com/spatialmodel/inmap/emissions/slca/eieio/grpc/gogrpc"

	"gonum.org/v1/gonum/mat"
)

type healthRequest struct {
	demand     *mat.VecDense
	industries *Mask
	pol        Pollutant
	pop        string
	year       Year
	loc        Location
	hr         epi.HRer
}

func (er *healthRequest) Key() string {
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
	bAll := append(append(append(append(append(append(b, b2...), b3...), []byte(er.pop)...), b4...), b5...), []byte(er.hr.Name())...)
	bytes := sha256.Sum256(bAll)
	return fmt.Sprintf("health_%x", bytes[0:sha256.Size])
}

// Health returns spatially-explicit pollutant air quality-related health impacts caused by the
// specified economic demand.  industries
// specify the industries emissions should be calculated for.
// If industries == nil, combined emissions for all industries are calculated.
// pop must be one of the population types defined in the configuration file.
func (e *SpatialEIO) Health(ctx context.Context, request *eieiorpc.HealthInput) (*eieiorpc.Vector, error) {
	e.loadHealthOnce.Do(func() {
		var c string
		if e.EIEIOCache != "" {
			c = e.EIEIOCache + "/individual"
		}
		e.healthCache = loadCacheOnce(func(ctx context.Context, request interface{}) (interface{}, error) {
			r := request.(*healthRequest)
			return e.health(ctx, r.demand, r.industries, r.pol, r.pop, r.year, r.loc, r.hr) // Actually calculate the health impacts.
		}, 1, e.MemCacheSize, c, vectorMarshal, vectorUnmarshal)
	})
	hr, ok := e.hr[request.HR]
	if !ok {
		return nil, fmt.Errorf("eieio: hazard ratio function `%s` is not registered", request.HR)
	}
	req := &healthRequest{
		demand:     array2vec(request.Demand),
		industries: (*Mask)(array2vec(request.Industries)),
		pol:        Pollutant(request.Pollutant),
		pop:        request.Population,
		year:       Year(request.Year),
		loc:        Location(request.Location),
		hr:         hr,
	}
	rr := e.healthCache.NewRequest(ctx, req, req.Key())
	resultI, err := rr.Result()
	if err != nil {
		return nil, err
	}
	return vec2rpc(resultI.(*mat.VecDense)), nil
}

// health returns spatially-explicit pollutant air quality-related health impacts caused by the
// specified economic demand.  industries
// specify the industries emissions should be calculated for.
// If industries == nil, combined emissions for all industries are calculated.
// pop must be one of the population types defined in the configuration file.
func (e *SpatialEIO) health(ctx context.Context, demand *mat.VecDense, industries *Mask, pol Pollutant, pop string, year Year, loc Location, HR epi.HRer) (*mat.VecDense, error) {
	hf, err := e.healthFactors(ctx, pol, pop, year, HR)
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

	r, _ := hf.Dims()
	health := mat.NewVecDense(r, nil)
	health.MulVec(hf, activity)
	return health, nil
}

// HealthMatrix returns spatially- and industry-explicit air quality-related health impacts caused by the
// specified economic demand. In the result matrix, the rows represent air quality
// model grid cells and the columns represent industries.
func (e *SpatialEIO) HealthMatrix(ctx context.Context, request *eieiorpc.HealthMatrixInput) (*eieiorpc.Matrix, error) {
	hr, ok := e.hr[request.HR]
	if !ok {
		return nil, fmt.Errorf("eieio: hazard ratio function `%s` is not registered", request.HR)
	}
	hf, err := e.healthFactors(ctx, Pollutant(request.Pollutant), request.Population, Year(request.Year), hr) // rows = grid cells, cols = industries
	if err != nil {
		return nil, err
	}

	activity, err := e.economicImpactsSCC(array2vec(request.Demand), Year(request.Year), Location(request.Location)) // rows = industries
	if err != nil {
		return nil, err
	}

	r, c := hf.Dims()
	health := mat.NewDense(r, c, nil)
	health.Apply(func(_, j int, v float64) float64 {
		// Multiply each emissions factor column by the corresponding activity row.
		return v * activity.At(j, 0)
	}, hf)
	return mat2rpc(health), nil
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
	e.loadHealthFactorsOnce.Do(func() {
		e.healthFactorCache = loadCacheOnce(e.healthFactorsWorker, 1, 1, e.EIEIOCache, matrixMarshal, matrixUnmarshal)
	})
	key := fmt.Sprintf("healthFactors_%v_%v_%d_%s", pol, pop, year, HR.Name())
	rr := e.healthFactorCache.NewRequest(ctx, concPolPopYearHR{pol: pol, year: year, pop: pop, hr: HR}, key)
	resultI, err := rr.Result()
	if err != nil {
		return nil, fmt.Errorf("bea: healthFactors: %s: %v", key, err)
	}
	return resultI.(*mat.Dense), nil
}

// healthFactorsWorker returns spatially-explicit pollution concentrations per unit of economic
// production for each industry. In the result matrix, the rows represent
// air quality model grid cells and the columns represent industries.
func (e *SpatialEIO) healthFactorsWorker(ctx context.Context, request interface{}) (interface{}, error) {
	polyearHR := request.(concPolPopYearHR)
	prod, err := e.domesticProductionSCC(polyearHR.year)
	if err != nil {
		return nil, err
	}
	var healthFac *mat.Dense
	for i, refTemp := range e.SpatialRefs {
		if len(refTemp.SCCs) == 0 {
			return nil, fmt.Errorf("bea: industry %d; no SCCs", i)
		}
		ref := refTemp
		ref.EmisYear = int(polyearHR.year)
		health, err := e.CSTConfig.HealthSurrogate(ctx, &ref, polyearHR.hr.Name())
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
			return nil, fmt.Errorf("eieio.health: invalid pollutant %v", polyearHR.pol)
		}
		if _, ok := health[polyearHR.pop]; !ok {
			return nil, fmt.Errorf("eieio.health: invalid population %v", polyearHR.pop)
		}
		industryHealth, ok := health[polyearHR.pop][pol]
		if !ok {
			return nil, fmt.Errorf("eieio.health: invalid pollutant %v", pol)
		}
		if i == 0 {
			healthFac = mat.NewDense(industryHealth.Shape[0], len(e.SpatialRefs), nil)
		}
		for r, v := range industryHealth.Elements {
			// The health factor is the industry health impacts divided by the
			// industry economic production.
			healthFac.Set(r, i, v/prod.At(i, 0))
		}
	}
	return healthFac, nil
}
