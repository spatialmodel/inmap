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
	"os"
	"strings"

	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/emissions/aep"
	"github.com/spatialmodel/inmap/emissions/aep/aeputil"
	"github.com/spatialmodel/inmap/sr"

	"github.com/ctessum/geom/proj"
	"github.com/ctessum/requestcache"
	"github.com/spatialmodel/epi"

	"bitbucket.org/ctessum/sparse"
)

func init() {
	gob.Register(sr.Concentrations{})
	gob.Register([]*inmap.EmisRecord{})
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

// ConcentrationSurrogate calculates the pollutant concentration impacts of
// spatialRef, accounting for the effects of elevated emissions plumes.
func (c *CSTConfig) ConcentrationSurrogate(ctx context.Context, spatialRef *SpatialRef) (*sr.Concentrations, error) {
	c.loadConcentrationOnce.Do(func() {
		c.concRequestCache = loadCacheOnce(c.inMAPSurrogate, 1, c.MaxCacheEntries, c.ConcentrationCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	r := c.concRequestCache.NewRequest(ctx, spatialRef, spatialRef.Key())
	result, err := r.Result()
	if err != nil {
		return nil, err
	}
	switch result.(type) {
	case *sr.Concentrations:
		return result.(*sr.Concentrations), nil
	case sr.Concentrations:
		r := result.(sr.Concentrations)
		return &r, nil
	default:
		panic(fmt.Errorf("result is invalid type: %#v", result))
	}
}

// inMAPSurrogate calculates the impact of each type of
// emissions from request.
func (c *CSTConfig) inMAPSurrogate(ctx context.Context, request interface{}) (interface{}, error) {
	if err := c.lazyLoadSR(); err != nil {
		return nil, err
	}
	r := request.(*SpatialRef)
	// Get the spatial surrogate.
	emis, err := c.spatialSurrogate(ctx, r)
	if err != nil {
		return nil, err
	}
	conc, err := c.sr.Concentrations(emis...)
	if err != nil {
		if _, ok := err.(sr.AboveTopErr); !ok {
			return nil, err
		}
	}
	return conc, nil
}

type sRHR struct {
	sr *SpatialRef
	hr epi.HRer
}

// HealthSurrogate calculates the health impact of the given spatial reference.
// HR specifies the function used to calculate the hazard ratio.
// Output format = map[popType][pol]values
func (c *CSTConfig) HealthSurrogate(ctx context.Context, spatialRef *SpatialRef, HR epi.HRer) (map[string]map[string]*sparse.DenseArray, error) {
	c.loadHealthOnce.Do(func() {
		c.healthRequestCache = loadCacheOnce(c.healthSurrogate, 1, c.MaxCacheEntries, c.HealthCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	r := c.healthRequestCache.NewRequest(ctx, sRHR{sr: spatialRef, hr: HR}, fmt.Sprintf("%s_%s", spatialRef.Key(), HR.Name()))
	result, err := r.Result()
	if err != nil {
		return nil, err
	}
	return result.(map[string]map[string]*sparse.DenseArray), nil
}

// healthSurrogate calculates the health impact of 1 kg/year of each type of
// emissions from request. Output format = map[popType][pol]values
func (c *CSTConfig) healthSurrogate(ctx context.Context, request interface{}) (interface{}, error) {
	if err := c.lazyLoadSR(); err != nil {
		return nil, err
	}

	health := make(map[string]map[string]*sparse.DenseArray)
	req := request.(sRHR)
	// The inmapSurrogate contains PM2.5 impacts of this SpatialRef.
	inmapSurrogate, err := c.ConcentrationSurrogate(ctx, req.sr)
	if err != nil {
		return nil, err
	}

	if len(c.CensusPopColumns) == 0 {
		return nil, fmt.Errorf("slca: CensusPopColumns configuration is not specified")
	}

	for _, popType := range c.CensusPopColumns {
		pop, _, err := c.PopulationIncidence(ctx, req.sr.EmisYear, popType, req.hr)
		if err != nil {
			return nil, err
		}
		cr, err := c.ConcentrationResponseAverage(ctx, req.sr.EmisYear, popType, req.hr)
		if err != nil {
			return nil, err
		}
		for pol, cc := range map[string][]float64{
			"pNO3":        inmapSurrogate.PNO3,
			"pNH4":        inmapSurrogate.PNH4,
			"PrimaryPM25": inmapSurrogate.PrimaryPM25,
			"pSO4":        inmapSurrogate.PSO4,
			"SOA":         inmapSurrogate.SOA,
			totalPM25:     inmapSurrogate.TotalPM25(),
		} {
			d := sparse.ZerosDense(len(cc))
			for i, ccc := range cc {
				p := pop[i]
				if p != 0 && ccc != 0 {
					d.Elements[i] = cr[i] * p * ccc
				}
			}
			if _, ok := health[popType]; !ok {
				health[popType] = make(map[string]*sparse.DenseArray)
			}
			health[popType][pol] = d
		}
	}
	return health, nil
}

// ConcentrationResponseAverage calculates the average concentration response
// for PM2.5 (deaths per year per ug/m3 per capita) for a non-linear concentration-
// response function. hr specifies the function used to calculate the hazard ratio.
func (c *CSTConfig) ConcentrationResponseAverage(ctx context.Context, year int, popType string, hr epi.HRer) ([]float64, error) {
	c.loadCROnce.Do(func() {
		c.crRequestCache = loadCacheOnce(c.concentrationResponseAverageWorker, 1, 1, c.HealthCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	r := c.crRequestCache.NewRequest(ctx, struct {
		year    int
		popType string
		hr      epi.HRer
	}{year: year, popType: popType, hr: hr}, fmt.Sprintf("concentrationResponse_%s_%d_%s", popType, year, hr.Name()))

	result, err := r.Result()
	if err != nil {
		return nil, err
	}
	return result.([]float64), nil
}

// concentrationResponseAverageWorker calculates the average concentration response
// for PM2.5 (deaths per year per ug/m3 per capita) for a non-linear concentration-
// response function.
func (c *CSTConfig) concentrationResponseAverageWorker(ctx context.Context, yearPopTypeI interface{}) (interface{}, error) {
	ypt := yearPopTypeI.(struct {
		year    int
		popType string
		hr      epi.HRer
	})
	concentrations, err := c.EvaluationConcentrations(ctx, ypt.year)
	if err != nil {
		return nil, err
	}
	conc := concentrations.TotalPM25()
	o := make([]float64, len(conc))
	_, io, err := c.PopulationIncidence(ctx, ypt.year, ypt.popType, ypt.hr)
	if err != nil {
		return nil, err
	}
	for i, z := range conc {
		const people = 1
		o[i] = epi.Outcome(people, z, epi.Io(z, ypt.hr, io[i]/100000), ypt.hr) / z
	}
	return o, nil
}

// EvaluationEmissions returns an array of emissions records calculated using
// the EvaluationInventoryConfig and AdditionalEmissionsShapefilesForEvaluation
// fields of the receiver, adjusting emissions to the specified year.
func (c *CSTConfig) EvaluationEmissions(ctx context.Context, year int) ([]*inmap.EmisRecord, error) {
	c.loadEvalEmisOnce.Do(func() {
		c.evalEmisRequestCache = loadCacheOnce(c.evaluationEmissions, 1, 1, c.SpatialCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	r := c.evalEmisRequestCache.NewRequest(ctx, year, fmt.Sprintf("evaluation_%d", year))
	result, err := r.Result()
	if err != nil {
		return nil, err
	}
	return result.([]*inmap.EmisRecord), nil
}

// EvaluationConcentrations returns an array of concentrations calculated using
// the EvaluationInventoryConfig and AdditionalEmissionsShapefilesForEvaluation
// fields of the receiver, adjusting emissions to the specified year.
func (c *CSTConfig) EvaluationConcentrations(ctx context.Context, year int) (*sr.Concentrations, error) {
	c.loadEvalConcOnce.Do(func() {
		c.evalConcRequestCache = loadCacheOnce(c.inMAPEval, 1, 1, c.ConcentrationCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	r := c.evalConcRequestCache.NewRequest(ctx, year, fmt.Sprintf("evaluation_%d", year))
	result, err := r.Result()
	if err != nil {
		return nil, err
	}
	switch result.(type) {
	case *sr.Concentrations:
		return result.(*sr.Concentrations), nil
	case sr.Concentrations:
		r := result.(sr.Concentrations)
		return &r, nil
	default:
		panic(fmt.Errorf("result is invalid type: %#v", result))
	}
}

// EvaluationHealth returns an array of health impacts calculated using
// the EvaluationInventoryConfig and AdditionalEmissionsShapefilesForEvaluation
// fields of the receiver, adjusting emissions to the specified year.
// HR specifies the function used to calculate the hazard ratio.
// Output format = map[popType][pol]values
func (c *CSTConfig) EvaluationHealth(ctx context.Context, year int, HR epi.HRer) (map[string]map[string]*sparse.DenseArray, error) {
	c.loadEvalHealthOnce.Do(func() {
		c.evalHealthRequestCache = loadCacheOnce(c.evaluationHealth, 1, 1, c.HealthCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	r := c.evalHealthRequestCache.NewRequest(ctx, struct {
		year int
		hr   epi.HRer
	}{year: year, hr: HR}, fmt.Sprintf("evaluation_%d_%s", year, HR.Name()))
	result, err := r.Result()
	if err != nil {
		return nil, err
	}
	return result.(map[string]map[string]*sparse.DenseArray), nil
}

// evaluationEmissions returns an array of emissions calculated using
// the EvaluationInventoryConfig and AdditionalEmissionsShapefilesForEvaluation
// fields of the receiver, adjusting emissions to the specified year.
func (c *CSTConfig) evaluationEmissions(ctx context.Context, yearI interface{}) (interface{}, error) {
	VOC, NOx, NH3, SOx, PM25, err := inmapPols(c.EvaluationInventoryConfig.PolsToKeep)
	if err != nil {
		return nil, err
	}

	if err = c.lazyLoadSR(); err != nil {
		return nil, err
	}
	year := yearI.(int)

	fmt.Println("Filtering out New York State commercial cooking emissions.")
	c.EvaluationInventoryConfig.FilterFunc = func(r aep.Record) bool {
		switch r.GetSCC() {
		case "2302002000", "2302002100", "2302002200", "2302003000", "2302003100", "2302003200":
			// Commercial meat cooking
			fips := r.GetFIPS()
			if fips[0:2] == "36" { // New York State
				return false
			}
		}
		return true
	}

	emis, _, err := c.EvaluationInventoryConfig.ReadEmissions()
	if err != nil {
		return nil, err
	}

	emis, err = c.groupBySCCAndApplyAdj(emis)
	if err != nil {
		return nil, err
	}

	// Scale emissions for the requested year.
	f, err := os.Open(c.SCCReference)
	if err != nil {
		return nil, fmt.Errorf("slca: opening SCCReference: %v", err)
	}
	if c.NEIBaseYear != 0 && year != 0 {
		emisScale, err := aeputil.ScaleNEIStateTrends(c.NEITrends, f, c.NEIBaseYear, year)
		if err != nil {
			return nil, fmt.Errorf("slca: Scaling NEI emissions: %v", err)
		}
		if err = aeputil.Scale(emis, emisScale); err != nil {
			return nil, err
		}
	}

	sp, err := c.SpatialConfig.SpatialProcessor()
	if err != nil {
		return nil, err
	}
	var aepRecs []aep.Record
	for _, e := range emis {
		aepRecs = append(aepRecs, e...)
	}
	spatialEmis, err := inmap.FromAEP(aepRecs, sp, 0, VOC, NOx, NH3, SOx, PM25)
	if err != nil {
		return nil, err
	}
	gridSR, err := proj.Parse(c.SpatialConfig.OutputSR)
	if err != nil {
		return nil, err
	}

	extraEmis, err := inmap.ReadEmissionShapefiles(gridSR, "tons/year", nil, c.AdditionalEmissionsShapefilesForEvaluation...)
	if err != nil {
		return nil, err
	}
	spatialEmis = append(spatialEmis, extraEmis.EmisRecords()...)
	return spatialEmis, nil
}

// inMAPEval returns an array of emissions calculated using
// the EvaluationInventoryConfig and AdditionalEmissionsShapefilesForEvaluation
// fields of the receiver, adjusting emissions to the specified year.
func (c *CSTConfig) inMAPEval(ctx context.Context, yearI interface{}) (interface{}, error) {
	if err := c.lazyLoadSR(); err != nil {
		return nil, err
	}

	spatialEmis, err := c.EvaluationEmissions(ctx, yearI.(int))
	if err != nil {
		return nil, err
	}

	conc, err := c.sr.Concentrations(spatialEmis...)
	if err != nil {
		if _, ok := err.(sr.AboveTopErr); !ok {
			return nil, err
		}
	}
	return conc, nil
}

// evaluationHealth returns helath impacts calculated using
// the EvaluationInventoryConfig and AdditionalEmissionsShapefilesForEvaluation
// fields of the receiver, adjusting emissions to the specified year.
// Output format = map[popType][pol]values
func (c *CSTConfig) evaluationHealth(ctx context.Context, yearHRI interface{}) (interface{}, error) {
	if err := c.lazyLoadSR(); err != nil {
		return nil, err
	}

	health := make(map[string]map[string]*sparse.DenseArray)
	yearHR := yearHRI.(struct {
		year int
		hr   epi.HRer
	})

	inmapSurrogate, err := c.EvaluationConcentrations(ctx, yearHR.year)
	if err != nil {
		return nil, err
	}

	if len(c.CensusPopColumns) == 0 {
		return nil, fmt.Errorf("slca: CensusPopColumns configuration is not specified")
	}

	for _, popType := range c.CensusPopColumns {
		pop, _, err := c.PopulationIncidence(ctx, yearHR.year, popType, yearHR.hr)
		if err != nil {
			return nil, err
		}
		cr, err := c.ConcentrationResponseAverage(ctx, yearHR.year, popType, yearHR.hr)
		if err != nil {
			return nil, err
		}
		for pol, cc := range map[string][]float64{
			"pNO3":        inmapSurrogate.PNO3,
			"pNH4":        inmapSurrogate.PNH4,
			"PrimaryPM25": inmapSurrogate.PrimaryPM25,
			"pSO4":        inmapSurrogate.PSO4,
			"SOA":         inmapSurrogate.SOA,
			totalPM25:     inmapSurrogate.TotalPM25(),
		} {
			d := sparse.ZerosDense(len(cc))
			for i, ccc := range cc {
				p := pop[i]
				if p != 0 && ccc != 0 {
					d.Elements[i] = cr[i] * p * ccc
				}
			}
			if _, ok := health[popType]; !ok {
				health[popType] = make(map[string]*sparse.DenseArray)
			}
			health[popType][pol] = d
		}
	}
	return health, nil
}

// inmapPols returns lists of the NEI pollutants
// matching each of the InMAP pollutants.
func inmapPols(pols aep.Speciation) (VOC, NOx, NH3, SOx, PM25 []aep.Pollutant, err error) {
	for p, pType := range pols {
		switch pType.SpecType {
		case aep.VOC, aep.VOCUngrouped:
			VOC = append(VOC, aep.Pollutant{Name: p})
		case aep.NOx:
			NOx = append(NOx, aep.Pollutant{Name: p})
		case aep.PM25:
			PM25 = append(PM25, aep.Pollutant{Name: p})
		default:
			var foundName bool
			for _, n := range pType.SpecNames.Names {
				switch n {
				case "NH3":
					NH3 = append(NH3, aep.Pollutant{Name: p})
					foundName = true
				case "SOx":
					SOx = append(SOx, aep.Pollutant{Name: p})
					foundName = true
				}
			}
			if !foundName {
				return nil, nil, nil, nil, nil, fmt.Errorf("invalid pollutant %s: %+v", p, pType)
			}
		}
	}
	return
}
