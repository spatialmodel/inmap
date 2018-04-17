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
	"math"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/proj"
	"github.com/ctessum/requestcache"
	"github.com/spatialmodel/epi"
)

func init() {
	gob.Register(popIncidence{})
}

// PopulationIncidence returns gridded population counts and underlying
// mortality incidence rates for the type specified by popType.
// Valid population types are specified by the CensusPopColumns attribute of the
// receiver. The returned value p is population counts and Io is underlying
// incidence rates. hr specifies the function used to calculate the hazard ratio.
func (c *CSTConfig) PopulationIncidence(ctx context.Context, year int, popType string, hr epi.HRer) (p, Io []float64, err error) {
	c.loadPopulationOnce.Do(func() {
		c.popRequestCache = loadCacheOnce(c.popIncidenceWorker, 1, 1, c.SpatialCache,
			requestcache.MarshalGob, requestcache.UnmarshalGob)
	})
	if _, ok := c.censusFile[year]; !ok {
		return c.interpolatePopulationIncidence(ctx, year, popType, hr)
	}
	r := c.popRequestCache.NewRequest(ctx, struct {
		year int
		hr   epi.HRer
	}{year: year, hr: hr}, fmt.Sprintf("populationIncidence_%d_%s", year, hr.Name()))
	resultI, err := r.Result()
	if err != nil {
		return nil, nil, err
	}
	result := resultI.(popIncidence)
	p, ok := result.P[popType]
	if !ok {
		return nil, nil, fmt.Errorf("slca: invalid population type %s", popType)
	}
	io, ok := result.Io[popType]
	if !ok {
		return nil, nil, fmt.Errorf("slca: invalid population type %s", popType)
	}
	return p, io, nil
}

type popIncidence struct {
	P, Io map[string][]float64
}

// popIncidenceWorker calculates the population and underlying mortality incidence rate.
// The population in each cell is calculated as an area-weighted average.
// The mortality rate in each cell is calculated as a population-weighted average. If
// multiple mortality rate polygons overlap or lie within a single population
// polygon, the mortality rate in each cell is equal to the population-weighted
// average of: the area-weighted average of mortality rates within each population polygon.
func (c *CSTConfig) popIncidenceWorker(ctx context.Context, yearHRI interface{}) (interface{}, error) {
	yearHR := yearHRI.(struct {
		year int
		hr   epi.HRer
	})
	pop, popIndices, mort, mortIndices, err := c.loadPopMort(yearHR.year)
	if err != nil {
		return nil, err
	}
	griddedPop, err := c.gridPopulation(pop, popIndices)
	if err != nil {
		return nil, err
	}
	mortIndex, err := c.regionalIncidence(ctx, pop, popIndices, mort, mortIndices, yearHR.year, yearHR.hr)
	if err != nil {
		return nil, err
	}
	griddedIo, err := c.griddedIncidence(mortIndex, pop, griddedPop, mortIndices, popIndices)
	if err != nil {
		return nil, err
	}
	return popIncidence{P: griddedPop, Io: griddedIo}, nil
}

func (c *CSTConfig) gridPopulation(pop *rtree.Rtree, popIndices map[string]int) (map[string][]float64, error) {
	cells, err := c.Geometry()
	if err != nil {
		return nil, err
	}

	o := make(map[string][]float64)

	ncpu := runtime.GOMAXPROCS(0)
	for popType, j := range popIndices {
		var wg sync.WaitGroup
		wg.Add(ncpu)
		oT := make([]float64, len(cells))
		for p := 0; p < ncpu; p++ {
			go func(p, j int, oT []float64) {
				for i := p; i < len(cells); i += ncpu {
					g := cells[i]
					var cellPop float64
					// First, intersect each grid cell with population polygons
					for _, pInterface := range pop.SearchIntersect(g.Bounds()) {
						p := pInterface.(*population)
						pIntersection := g.Intersection(p)
						pAreaIntersect := pIntersection.Area()
						if pAreaIntersect == 0 {
							continue
						}
						pArea := p.Area() // we want to conserve the total population
						if pArea == 0. {
							panic("divide by zero")
						}
						pAreaFrac := pAreaIntersect / pArea
						cellPop += p.PopData[j] * pAreaFrac
					}
					oT[i] = cellPop
				}
				wg.Done()
			}(p, j, oT)
		}
		wg.Wait()
		o[popType] = oT
	}
	return o, nil
}

// regionalIncidence calculates region-averaged underlying incidence rates.
func (c *CSTConfig) regionalIncidence(ctx context.Context, popIndex *rtree.Rtree, popIndices map[string]int,
	mort []*mortality, mortIndices map[string]int, year int, hr epi.HRer) (*rtree.Rtree, error) {
	ncpu := runtime.GOMAXPROCS(0)

	concTemp, err := c.EvaluationConcentrations(ctx, year)
	if err != nil {
		return nil, err
	}
	conc := concTemp.TotalPM25()
	for mortType, popType := range c.MortalityRateColumns {
		mi, ok := mortIndices[mortType]
		if !ok {
			panic(fmt.Errorf("missing mortality type %s", mortType))
		}
		pi, ok := popIndices[popType]
		if !ok {
			panic(fmt.Errorf("missing population type %s", popType))
		}

		var wg sync.WaitGroup
		wg.Add(ncpu)
		for p := 0; p < ncpu; p++ {
			go func(p, mi, pi int) {
				for i := p; i < len(mort); i += ncpu {
					m := mort[i]
					regionPopIsect := popIndex.SearchIntersect(m.Bounds())
					regionPop := make([]float64, len(regionPopIsect))
					regionConc := make([]float64, len(regionPopIsect))
					for i, pI := range regionPopIsect {
						pp := pI.(*population)
						pArea := pp.Area()
						isectFrac := pp.Intersection(m).Area() / pArea
						if pArea == 0 || isectFrac == 0 {
							continue
						}
						regionPop[i] = pp.PopData[pi] * isectFrac
						for _, gI := range c.gridIndex.SearchIntersect(pp.Bounds()) {
							g := gI.(gridIndex)
							regionConc[i] += conc[g.i] * g.Intersection(pp).Area() / pArea
						}
					}
					m.Io[mi] = epi.IoRegional(regionPop, regionConc, hr, m.MortData[mi])
				}
				wg.Done()
			}(p, mi, pi)
		}
		wg.Wait()
	}
	o := rtree.NewTree(25, 50)
	for _, m := range mort {
		o.Insert(m)
	}
	return o, nil
}

// griddedIncidence allocates baseline incidence rates to cells, weighting by
// population.
func (c *CSTConfig) griddedIncidence(mortIndex, popIndex *rtree.Rtree, griddedPop map[string][]float64,
	mortIndices, popIndices map[string]int) (map[string][]float64, error) {
	ncpu := runtime.GOMAXPROCS(0)

	cells, err := c.Geometry()
	if err != nil {
		return nil, err
	}

	o := make(map[string][]float64)

	for mortType, popType := range c.MortalityRateColumns {
		oT := make([]float64, len(cells))
		pi, ok := popIndices[popType]
		if !ok {
			panic(fmt.Errorf("missing population type %s", popType))
		}
		mi, ok := mortIndices[mortType]
		if !ok {
			panic(fmt.Errorf("missing mortality type %s", mortType))
		}

		var wg sync.WaitGroup
		wg.Add(ncpu)
		for p := 0; p < ncpu; p++ {
			go func(p, pi int, oT []float64) {
				for i := p; i < len(cells); i += ncpu {
					g := cells[i]
					if griddedPop[popType][i] == 0 {
						continue
					}
					var popTotal float64
					var mPop float64
					cellPop := popIndex.SearchIntersect(g.Bounds())
					for _, mI := range mortIndex.SearchIntersect(g.Bounds()) {
						m := mI.(*mortality)
						for _, pI := range cellPop {
							p := pI.(*population)
							popTemp := p.Intersection(g).Intersection(m).Area() / p.Area() * p.PopData[pi]
							mPop += m.Io[mi] * popTemp
							popTotal += popTemp
						}
					}
					if popTotal != 0 {
						oT[i] = mPop / popTotal
					}
				}
				wg.Done()
			}(p, pi, oT)
		}
		wg.Wait()
		o[popType] = oT
	}
	return o, nil
}

// loadPopMort loads the population and mortality rate data from the shapefiles
// specified in the receiver.
func (c *CSTConfig) loadPopMort(year int) (*rtree.Rtree, map[string]int, []*mortality, map[string]int, error) {
	gridSR, err := proj.Parse(c.SpatialConfig.OutputSR)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("slca: while parsing OutputSR: %v", err)
	}
	pop, popIndex, err := c.loadPopulation(year, gridSR)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("slca: while loading population: %v", err)
	}
	mort, mortIndex, err := c.loadMortality(year, gridSR)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("slca: while loading mortality rate: %v", err)
	}
	return pop, popIndex, mort, mortIndex, nil
}

type population struct {
	geom.Polygonal

	// PopData holds the number of people in each population category
	PopData []float64
}

type mortality struct {
	geom.Polygonal

	// MortData holds the mortality rate for each population category
	MortData []float64 // Deaths per 100,000 people per year

	// Io holds the underlying incidence rate for each population category
	Io []float64
}

// loadPopulation loads population information from a shapefile, converting it
// to spatial reference sr. The function outputs an index holding the population
// information.
func (c *CSTConfig) loadPopulation(year int, sr *proj.SR) (*rtree.Rtree, map[string]int, error) {
	var err error
	f, ok := c.censusFile[year]
	if !ok {
		return nil, nil, fmt.Errorf("slca: missing population data for year %d", year)
	}
	popshp, err := shp.NewDecoder(f)
	if err != nil {
		return nil, nil, err
	}
	popsr, err := popshp.SR()
	if err != nil {
		return nil, nil, err
	}
	trans, err := popsr.NewTransform(sr)
	if err != nil {
		return nil, nil, err
	}
	// Create a list of array indices for each population type.
	popIndices := make(map[string]int)
	for i, p := range c.CensusPopColumns {
		popIndices[p] = i
	}

	pop := rtree.NewTree(25, 50)
	for {
		g, fields, more := popshp.DecodeRowFields(c.CensusPopColumns...)
		if !more {
			break
		}
		p := new(population)
		p.PopData = make([]float64, len(c.CensusPopColumns))
		for i, pop := range c.CensusPopColumns {
			s, ok := fields[pop]
			if !ok {
				return nil, nil, fmt.Errorf("inmap: loading population shapefile: missing attribute column %s", pop)
			}
			p.PopData[i], err = s2f(s)
			if err != nil {
				return nil, nil, err
			}
			if math.IsNaN(p.PopData[i]) {
				return nil, nil, fmt.Errorf("inmap: loadPopulation: NaN population value")
			}
		}
		gg, err := g.Transform(trans)
		if err != nil {
			return nil, nil, err
		}
		switch gg.(type) {
		case geom.Polygonal:
			p.Polygonal = gg.(geom.Polygonal)
		default:
			return nil, nil, fmt.Errorf("inmap: loadPopulation: population shapes need to be polygons")
		}
		pop.Insert(p)
	}
	if err := popshp.Error(); err != nil {
		return nil, nil, err
	}
	popshp.Close()
	return pop, popIndices, nil
}

func (c *CSTConfig) loadMortality(year int, sr *proj.SR) ([]*mortality, map[string]int, error) {
	f, ok := c.mortalityRateFile[year]
	if !ok {
		return nil, nil, fmt.Errorf("slca: missing mortality rate data for year %d", year)
	}
	mortshp, err := shp.NewDecoder(f)
	if err != nil {
		return nil, nil, err
	}

	mortshpSR, err := mortshp.SR()
	if err != nil {
		return nil, nil, err
	}
	trans, err := mortshpSR.NewTransform(sr)
	if err != nil {
		return nil, nil, err
	}

	// Create a list of array indices for each mortality rate.
	mortIndices := make(map[string]int)
	// Extract mortality rate column names from map of population to mortality rates
	mortRateColumns := make([]string, len(c.MortalityRateColumns))
	i := 0
	for m := range c.MortalityRateColumns {
		mortRateColumns[i] = m
		i++
	}
	sort.Strings(mortRateColumns)
	for i, m := range mortRateColumns {
		mortIndices[m] = i
	}
	var mortRates []*mortality
	for {
		g, fields, more := mortshp.DecodeRowFields(mortRateColumns...)
		if !more {
			break
		}
		m := new(mortality)
		m.MortData = make([]float64, len(mortRateColumns))
		m.Io = make([]float64, len(m.MortData))
		for i, mort := range mortRateColumns {
			s, ok := fields[mort]
			if !ok {
				return nil, nil, fmt.Errorf("slca: loading mortality rate shapefile: missing attribute column %s", mort)
			}
			m.MortData[i], err = s2f(s)
			if err != nil {
				return nil, nil, err
			}
			if math.IsNaN(m.MortData[i]) {
				panic("NaN mortality rate!")
			}
		}
		gg, err := g.Transform(trans)
		if err != nil {
			return nil, nil, err
		}
		switch gg.(type) {
		case geom.Polygonal:
			m.Polygonal = gg.(geom.Polygonal)
		default:
			return nil, nil, fmt.Errorf("slca: loadMortality: mortality rate shapes need to be polygons")
		}
		mortRates = append(mortRates, m)
	}
	if err := mortshp.Error(); err != nil {
		return nil, nil, err
	}
	mortshp.Close()
	return mortRates, mortIndices, nil
}

// interpolatePopulationIncidence returns population and baseline incidence rates for
// years without population data, interpolated from years with population data.
// For years which there exists population data for years both before and after
// the year of interest, interpolation is used, otherwise results are assumed
// to be constant from the endpoint year.
func (c *CSTConfig) interpolatePopulationIncidence(ctx context.Context, year int, popType string, hr epi.HRer) (p, Io []float64, err error) {
	yearBefore := math.MinInt32
	yearAfter := math.MaxInt32
	var beforeOK, afterOK bool
	for y := range c.censusFile {
		// Find the closest before and after years with data, if any exist.
		if y < year {
			beforeOK = true
			if y > yearBefore {
				yearBefore = y
			}
		}
		if y > year {
			afterOK = true
			if y < yearAfter {
				yearAfter = y
			}
		}
	}

	if !beforeOK && !afterOK {
		return nil, nil, fmt.Errorf("slca: no population data has been specified")
	} else if beforeOK && !afterOK {
		return c.PopulationIncidence(ctx, yearBefore, popType, hr)
	} else if afterOK && !beforeOK {
		return c.PopulationIncidence(ctx, yearAfter, popType, hr)
	}

	popBefore, ioBefore, err := c.PopulationIncidence(ctx, yearBefore, popType, hr)
	if err != nil {
		return nil, nil, err
	}
	popAfter, ioAfter, err := c.PopulationIncidence(ctx, yearAfter, popType, hr)
	if err != nil {
		return nil, nil, err
	}
	frac := float64(year-yearBefore) / float64(yearAfter-yearBefore)
	pop := make([]float64, len(popBefore))
	io := make([]float64, len(ioBefore))
	for i := range pop {
		pop[i] = popBefore[i]*(1-frac) + popAfter[i]*frac
	}
	for i := range io {
		io[i] = ioBefore[i]*(1-frac) + ioAfter[i]*frac
	}
	return pop, io, nil
}

func s2f(s string) (float64, error) {
	if strings.Contains(s, "*") { // Null value
		return 0., nil
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err
}
