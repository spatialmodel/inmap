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
	"net/url"
	"os"
	"strings"

	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/emissions/aep"
	"github.com/spatialmodel/inmap/emissions/aep/aeputil"
	"github.com/spatialmodel/inmap/emissions/slca/eieio/eieiorpc"
	"github.com/spatialmodel/inmap/internal/hash"
	"github.com/spatialmodel/inmap/sr"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/requestcache"
	"github.com/spatialmodel/inmap/epi"

	"github.com/ctessum/sparse"
)

func init() {
	gob.Register(sr.Concentrations{})
	gob.Register([]*inmap.EmisRecord{})
	gob.Register(geom.Polygon{})
	gob.Register(geom.MultiPolygon{})
	gob.Register(geom.Point{})
	gob.Register(geom.MultiPoint{})
	gob.Register(geom.LineString{})
	gob.Register(geom.MultiLineString{})
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

// ConcentrationSurrogate calculates the pollutant concentration impacts of
// spatialRef, accounting for the effects of elevated emissions plumes.
func (c *CSTConfig) ConcentrationSurrogate(ctx context.Context, spatialRef *SpatialRef) (*sr.Concentrations, error) {
	c.loadConcentrationOnce.Do(func() {
		c.concRequestCache = loadCacheOnce(c.inMAPSurrogate, 1, c.MaxCacheEntries, c.ConcentrationCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	r := c.concRequestCache.NewRequest(ctx, spatialRef, "concsrg_"+hash.Hash(spatialRef))
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
	r := request.(*SpatialRef)
	_, srReader, _, err := c.srSetup(r.AQM)
	if err != nil {
		return nil, err
	}
	// Get the spatial surrogate.
	emis, err := c.spatialSurrogate(ctx, r)
	if err != nil {
		return nil, err
	}
	conc, err := srReader.Concentrations(emis...)
	if err != nil {
		if _, ok := err.(sr.AboveTopErr); !ok {
			return nil, err
		}
	}
	return conc, nil
}

type sRHR struct {
	sr *SpatialRef
	hr string
}

// HealthSurrogate calculates the health impact of the given spatial reference.
// HR specifies the function used to calculate the hazard ratio.
// Output format = map[popType][pol]values
func (c *CSTConfig) HealthSurrogate(ctx context.Context, spatialRef *SpatialRef, HR string) (map[string]map[string]*sparse.DenseArray, error) {
	c.loadHealthOnce.Do(func() {
		c.healthRequestCache = loadCacheOnce(c.healthSurrogate, 1, c.MaxCacheEntries, c.HealthCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	r := c.healthRequestCache.NewRequest(ctx, sRHR{sr: spatialRef, hr: HR}, fmt.Sprintf("%s_%s", hash.Hash(spatialRef), HR))
	result, err := r.Result()
	if err != nil {
		return nil, err
	}
	return result.(map[string]map[string]*sparse.DenseArray), nil
}

// healthSurrogate calculates the health impact of 1 kg/year of each type of
// emissions from request. Output format = map[popType][pol]values
func (c *CSTConfig) healthSurrogate(ctx context.Context, request interface{}) (interface{}, error) {
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
		pop, err := c.PopulationIncidence(ctx, &eieiorpc.PopulationIncidenceInput{
			Year: int32(req.sr.EmisYear), Population: popType, HR: req.hr, AQM: req.sr.AQM})
		if err != nil {
			return nil, err
		}
		cr, err := c.ConcentrationResponseAverage(ctx, &eieiorpc.ConcentrationResponseAverageInput{
			Year:       int32(req.sr.EmisYear),
			Population: popType,
			HR:         req.hr,
			AQM:        req.sr.AQM,
		})
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
				p := pop.Population[i]
				if p != 0 && ccc != 0 {
					d.Elements[i] = cr.Data[i] * p * ccc
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
func (c *CSTConfig) ConcentrationResponseAverage(ctx context.Context, request *eieiorpc.ConcentrationResponseAverageInput) (*eieiorpc.Vector, error) {
	c.loadCROnce.Do(func() {
		c.crRequestCache = loadCacheOnce(c.concentrationResponseAverageWorker, 1, 1, c.HealthCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	r := c.crRequestCache.NewRequest(ctx, struct {
		year    int
		popType string
		hr      string
		aqm     string
	}{year: int(request.Year), popType: request.Population, hr: request.HR, aqm: request.AQM},
		fmt.Sprintf("concentrationResponse_%s_%d_%s_%s", request.Population, request.Year, request.HR, request.AQM))

	result, err := r.Result()
	if err != nil {
		return nil, err
	}
	return &eieiorpc.Vector{Data: result.([]float64)}, nil
}

// concentrationResponseAverageWorker calculates the average concentration response
// for PM2.5 (deaths per year per ug/m3 per capita) for a non-linear concentration-
// response function.
func (c *CSTConfig) concentrationResponseAverageWorker(ctx context.Context, yearPopTypeAQMI interface{}) (interface{}, error) {
	yptaqm := yearPopTypeAQMI.(struct {
		year    int
		popType string
		hr      string
		aqm     string
	})
	HR, ok := c.hr[yptaqm.hr]
	if !ok {
		return nil, fmt.Errorf("slca.CSTConfig: hazard ratio `%s` has not been registered", yptaqm.hr)
	}
	// TODO: Refactor this duplicate code.
	c.loadEvalConcOnce.Do(func() {
		c.evalConcRequestCache = loadCacheOnce(c.inMAPEval, 1, 1, c.ConcentrationCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	r := c.evalConcRequestCache.NewRequest(ctx, aqmYear{aqm: yptaqm.aqm, year: yptaqm.year},
		fmt.Sprintf("evaluation_%s_%d", yptaqm.aqm, yptaqm.year))
	result, err := r.Result()
	if err != nil {
		return nil, err
	}
	var concentrations *sr.Concentrations
	switch result.(type) {
	case *sr.Concentrations:
		concentrations = result.(*sr.Concentrations)
	case sr.Concentrations:
		r := result.(sr.Concentrations)
		concentrations = &r
	default:
		panic(fmt.Errorf("result is invalid type: %#v", result))
	}
	conc := concentrations.TotalPM25()
	o := make([]float64, len(conc))
	io, err := c.PopulationIncidence(ctx, &eieiorpc.PopulationIncidenceInput{
		Year: int32(yptaqm.year), Population: yptaqm.popType, HR: yptaqm.hr, AQM: yptaqm.aqm})
	if err != nil {
		return nil, err
	}
	for i, z := range conc {
		const people = 1
		o[i] = epi.Outcome(people, z, epi.Io(z, HR, io.Incidence[i]/100000), HR) / z
	}
	return o, nil
}

// EvaluationEmissions returns an array of emissions records calculated using
// the EvaluationInventoryConfig and AdditionalEmissionsShapefilesForEvaluation
// fields of the receiver, adjusting emissions to the specified year and
// gridding them to match the specified air quality model (aqm).
func (c *CSTConfig) EvaluationEmissions(ctx context.Context, aqm string, year int) ([]*inmap.EmisRecord, error) {
	c.loadEvalEmisOnce.Do(func() {
		c.evalEmisRequestCache = loadCacheOnce(c.evaluationEmissions, 1, 1, c.SpatialCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	r := c.evalEmisRequestCache.NewRequest(ctx, aqmYear{aqm: aqm, year: year}, fmt.Sprintf("evaluation_%s_%d", aqm, year))
	result, err := r.Result()
	if err != nil {
		return nil, err
	}
	return result.([]*inmap.EmisRecord), nil
}

// EvaluationConcentrations returns an array of concentrations calculated using
// the EvaluationInventoryConfig and AdditionalEmissionsShapefilesForEvaluation
// fields of the receiver, adjusting emissions to the specified year.
func (c *CSTConfig) EvaluationConcentrations(ctx context.Context, request *eieiorpc.EvaluationConcentrationsInput) (*eieiorpc.Vector, error) {
	c.loadEvalConcOnce.Do(func() {
		c.evalConcRequestCache = loadCacheOnce(c.inMAPEval, 1, 1, c.ConcentrationCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	r := c.evalConcRequestCache.NewRequest(ctx, aqmYear{aqm: request.AQM, year: int(request.Year)},
		fmt.Sprintf("evaluationConc_%s_%d", request.AQM, request.Year))
	result, err := r.Result()
	if err != nil {
		return nil, err
	}
	var o *sr.Concentrations
	switch result.(type) {
	case *sr.Concentrations:
		o = result.(*sr.Concentrations)
	case sr.Concentrations:
		r := result.(sr.Concentrations)
		o = &r
	default:
		panic(fmt.Errorf("result is invalid type: %#v", result))
	}
	switch request.Pollutant {
	case eieiorpc.Pollutant_SOA:
		return &eieiorpc.Vector{Data: o.SOA}, nil
	case eieiorpc.Pollutant_PNH4:
		return &eieiorpc.Vector{Data: o.PNH4}, nil
	case eieiorpc.Pollutant_PNO3:
		return &eieiorpc.Vector{Data: o.PNO3}, nil
	case eieiorpc.Pollutant_PSO4:
		return &eieiorpc.Vector{Data: o.PSO4}, nil
	case eieiorpc.Pollutant_TotalPM25:
		return &eieiorpc.Vector{Data: o.TotalPM25()}, nil
	case eieiorpc.Pollutant_PrimaryPM25:
		return &eieiorpc.Vector{Data: o.PrimaryPM25}, nil
	default:
		return nil, fmt.Errorf("slca.CSTConfig.EvaluationConcentration: invalid pollutant `%s`", request.Pollutant)
	}
}

// EvaluationHealth returns an array of health impacts calculated using
// the EvaluationInventoryConfig and AdditionalEmissionsShapefilesForEvaluation
// fields of the receiver, adjusting emissions to the specified year.
// HR specifies the function used to calculate the hazard ratio.
// Output format = map[popType][pol]values
func (c *CSTConfig) EvaluationHealth(ctx context.Context, request *eieiorpc.EvaluationHealthInput) (*eieiorpc.Vector, error) {
	c.loadEvalHealthOnce.Do(func() {
		c.evalHealthRequestCache = loadCacheOnce(c.evaluationHealth, 1, 1, c.HealthCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	r := c.evalHealthRequestCache.NewRequest(ctx, struct {
		aqm  string
		year int
		hr   string
	}{year: int(request.Year), hr: request.HR, aqm: request.AQM}, fmt.Sprintf("evaluationHealth_%s_%d_%s", request.AQM, request.Year, request.HR))
	result, err := r.Result()
	if err != nil {
		return nil, err
	}
	r2, ok := result.(map[string]map[string]*sparse.DenseArray)[request.Population]
	if !ok {
		return nil, fmt.Errorf("slca.CSTConfig: invalid population type `%s`", request.Population)
	}
	var pol string
	switch request.Pollutant {
	case eieiorpc.Pollutant_PNO3:
		pol = "pNO3"
	case eieiorpc.Pollutant_PNH4:
		pol = "pNH4"
	case eieiorpc.Pollutant_PrimaryPM25:
		pol = "PrimaryPM25"
	case eieiorpc.Pollutant_PSO4:
		pol = "pSO4"
	case eieiorpc.Pollutant_SOA:
		pol = "SOA"
	case eieiorpc.Pollutant_TotalPM25:
		pol = totalPM25
	default:
		return nil, fmt.Errorf("slca.CSTConfig: invalid pollutant type `%s`", pol)
	}
	r3, ok := r2[pol]
	if !ok {
		return nil, fmt.Errorf("slca.CSTConfig: invalid pollutant type `%s`", pol)
	}
	return &eieiorpc.Vector{Data: r3.Elements}, nil
}

type aqmYear struct {
	aqm  string
	year int
}

// evaluationEmissions returns an array of emissions calculated using
// the EvaluationInventoryConfig and AdditionalEmissionsShapefilesForEvaluation
// fields of the receiver, adjusting emissions to the specified year.
func (c *CSTConfig) evaluationEmissions(ctx context.Context, aqmYearI interface{}) (interface{}, error) {
	VOC, NOx, NH3, SOx, PM25, err := inmapPols(c.EvaluationInventoryConfig.PolsToKeep)
	if err != nil {
		return nil, err
	}
	req := aqmYearI.(aqmYear)

	fmt.Println("Filtering out New York State commercial cooking emissions, dog waste emissions, and human perspiration.")
	c.EvaluationInventoryConfig.FilterFunc = func(r aep.Record) bool {
		switch r.GetSCC() {
		case "2302002000", "2302002100", "2302002200", "2302003000", "2302003100", "2302003200":
			// Commercial meat cooking
			fips := r.GetFIPS()
			if fips[0:2] == "36" { // New York State
				return false
			}
		case "2806015000", "2810010000", "2610000500": // Dog waste, human perspiration, and construction open burning
			return false
		}
		return true
	}

	emis, _, err := c.EvaluationInventoryConfig.ReadEmissions()
	if err != nil {
		return nil, err
	}

	// Scale emissions for the requested year.
	f, err := os.Open(c.SCCReference)
	if err != nil {
		return nil, fmt.Errorf("slca: opening SCCReference: %v", err)
	}
	if c.NEIBaseYear != 0 && req.year != 0 {
		emisScale, err := aeputil.ScaleNEIStateTrends(c.NEITrends, f, c.NEIBaseYear, req.year)
		if err != nil {
			return nil, fmt.Errorf("slca: Scaling NEI emissions: %v", err)
		}
		if err = aeputil.Scale(emis, emisScale); err != nil {
			return nil, err
		}
	}

	spatialConfig, _, _, err := c.srSetup(req.aqm)
	if err != nil {
		return nil, err
	}

	sp, err := spatialConfig.SpatialProcessor()
	if err != nil {
		return nil, err
	}

	emisGridded, err := c.groupBySCCAndApplyAdj(emis, req.aqm)
	if err != nil {
		return nil, err
	}

	var aepRecs []aep.RecordGridded
	for _, e := range emisGridded {
		aepRecs = append(aepRecs, e...)
	}
	spatialEmis, err := inmap.FromAEP(aepRecs, sp.Grids, 0, VOC, NOx, NH3, SOx, PM25)
	if err != nil {
		return nil, err
	}
	gridSR, err := proj.Parse(spatialConfig.OutputSR)
	if err != nil {
		return nil, err
	}

	extraEmis, err := inmap.ReadEmissionShapefiles(gridSR, "tons/year", nil, nil, c.AdditionalEmissionsShapefilesForEvaluation...)
	if err != nil {
		return nil, err
	}
	spatialEmis = append(spatialEmis, extraEmis.EmisRecords()...)
	return spatialEmis, nil
}

// inMAPEval returns an array of emissions calculated using
// the EvaluationInventoryConfig and AdditionalEmissionsShapefilesForEvaluation
// fields of the receiver, adjusting emissions to the specified year.
func (c *CSTConfig) inMAPEval(ctx context.Context, aqmYearI interface{}) (interface{}, error) {
	req := aqmYearI.(aqmYear)

	spatialEmis, err := c.EvaluationEmissions(ctx, req.aqm, req.year)
	if err != nil {
		return nil, err
	}

	_, srMatrix, _, err := c.srSetup(req.aqm)
	if err != nil {
		return nil, err
	}

	conc, err := srMatrix.Concentrations(spatialEmis...)
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
func (c *CSTConfig) evaluationHealth(ctx context.Context, aqmYearHRI interface{}) (interface{}, error) {
	health := make(map[string]map[string]*sparse.DenseArray)
	aqmYearHR := aqmYearHRI.(struct {
		aqm  string
		year int
		hr   string
	})

	// TODO: Refactor this duplicate code.
	c.loadEvalConcOnce.Do(func() {
		c.evalConcRequestCache = loadCacheOnce(c.inMAPEval, 1, 1, c.ConcentrationCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	r := c.evalConcRequestCache.NewRequest(ctx, aqmYear{aqm: aqmYearHR.aqm, year: aqmYearHR.year},
		fmt.Sprintf("evaluationConc_%s_%d", aqmYearHR.aqm, aqmYearHR.year))
	result, err := r.Result()
	if err != nil {
		return nil, err
	}
	var inmapSurrogate *sr.Concentrations
	switch result.(type) {
	case *sr.Concentrations:
		inmapSurrogate = result.(*sr.Concentrations)
	case sr.Concentrations:
		r := result.(sr.Concentrations)
		inmapSurrogate = &r
	default:
		panic(fmt.Errorf("result is invalid type: %#v", result))
	}

	if len(c.CensusPopColumns) == 0 {
		return nil, fmt.Errorf("slca: CensusPopColumns configuration is not specified")
	}

	for _, popType := range c.CensusPopColumns {
		pop, err := c.PopulationIncidence(ctx, &eieiorpc.PopulationIncidenceInput{
			AQM:        aqmYearHR.aqm,
			Year:       int32(aqmYearHR.year),
			Population: popType,
			HR:         aqmYearHR.hr},
		)
		if err != nil {
			return nil, err
		}
		cr, err := c.ConcentrationResponseAverage(ctx, &eieiorpc.ConcentrationResponseAverageInput{
			AQM:        aqmYearHR.aqm,
			Year:       int32(aqmYearHR.year),
			Population: popType,
			HR:         aqmYearHR.hr,
		})
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
				p := pop.Population[i]
				if p != 0 && ccc != 0 {
					d.Elements[i] = cr.Data[i] * p * ccc
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
