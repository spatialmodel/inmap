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

// Package bea implements an Economic Input-Output life cycle assessment
// model based on the US Bureau of Economic Analysis (BEA)
// Annual Input-Output Accounts Data from
// https://www.bea.gov/industry/io_annual.htm
package bea

import (
	"fmt"
	"sync"

	"github.com/ctessum/requestcache"
	"gonum.org/v1/gonum/mat"
)

// Config holds simulation configuration imformation.
type Config struct {
	// Years specifies the simulation years.
	Years []Year

	// DetailYear specifies the year for which detailed information is
	// available. With current default data, this should be 2007.
	DetailYear Year

	// UseSummary is the locations of the BEA use file
	// at the summary level of detail.
	UseSummary string

	// UseDetail is the location of the BEA use file
	// at the detailed level of detail.
	UseDetail string

	// ImportsSummary is the locations of the BEA import demand file
	// at the summary level of detail.
	ImportsSummary string

	// ImportsDetail is the location of the BEA import demand file
	// at the detailed level of detail.
	ImportsDetail string

	// TotalRequirementsSummary and DomesticRequirementsSummary are
	// the locations of the BEA total requirements and domestic requirements
	// files (Industry x Commodity) at the summary level of detail.
	TotalRequirementsSummary, DomesticRequirementsSummary string

	// TotalRequirementsDetail and DomesticRequirementsDetail are
	// the locations of the BEA total requirements and domestic requirements
	// files (Industry x Commodity) at the detailed level of detail.
	TotalRequirementsDetail, DomesticRequirementsDetail string
}

// Year specifies the year of the analysis.
type Year int

// EIO is a holder for EIO LCA data.
type EIO struct {
	// totalRequirements is the total requirements
	// (direct + indirect; domestic + import) per unit of final demand
	// for each year at the detailed sector level.
	totalRequirements map[Year]*mat.Dense

	// domesticRequirements is the total requirements
	// (direct + indirect; domestic) per unit of final demand
	// for each year at the detailed sector level.
	domesticRequirements map[Year]*mat.Dense

	// importRequirements is the total requirements
	// (direct + indirect; import) per unit of final demand
	// for each year at the detailed sector level.
	importRequirements map[Year]*mat.Dense

	// totalFinalDemand is total final economic demand of different
	// types for each year at the detailed sector level.
	totalFinalDemand map[Year]map[FinalDemand]*mat.VecDense

	// importFinalDemand is final economic demand of different
	// types for imports for each year at the detailed sector level.
	importFinalDemand map[Year]map[FinalDemand]*mat.VecDense

	// Industries and Commodities are the industry and commodity
	// sectors in the model
	Industries, Commodities []string

	// industryIndices and commodityIndices map the sector
	// names to array indices.
	industryIndices, commodityIndices map[string]int

	// excelCache holds previously opened Microsoft Excel files
	// to avoid reading the same file multiple times.
	excelCache         *requestcache.Cache
	loadExcelCacheOnce sync.Once
}

// New initializes a new EIO object based on the given
// configuration.
func New(cfg *Config) (*EIO, error) {
	eio := new(EIO)
	eio.totalRequirements = make(map[Year]*mat.Dense)
	eio.domesticRequirements = make(map[Year]*mat.Dense)
	eio.importRequirements = make(map[Year]*mat.Dense)
	eio.totalFinalDemand = make(map[Year]map[FinalDemand]*mat.VecDense)
	eio.importFinalDemand = make(map[Year]map[FinalDemand]*mat.VecDense)
	var err error

	type matYearErr struct {
		m *mat.Dense
		y Year
		e error
	}
	type demMatYearErr struct {
		dm map[FinalDemand]*mat.VecDense
		y  Year
		e  error
	}
	type stringsErr struct {
		s []string
		e error
	}
	totalReqChan := make(chan *matYearErr)
	domesticReqChan := make(chan *matYearErr)
	totalFinalDemChan := make(chan *demMatYearErr)
	importFinalDemChan := make(chan *demMatYearErr)
	industriesChan := make(chan *stringsErr)
	commoditiesChan := make(chan *stringsErr)

	for _, year := range cfg.Years {
		go func(year Year) {
			m, e := eio.totalRequirementsAdjusted(cfg.TotalRequirementsDetail, cfg.TotalRequirementsSummary, year, cfg.DetailYear)
			totalReqChan <- &matYearErr{m: m, y: year, e: e}
		}(year)

		go func(year Year) {
			m, e := eio.totalRequirementsAdjusted(cfg.DomesticRequirementsDetail, cfg.DomesticRequirementsSummary, year, cfg.DetailYear)
			domesticReqChan <- &matYearErr{m: m, y: year, e: e}
		}(year)

		go func(year Year) {
			imports := false
			dm, e := eio.loadFinalDemand(cfg.UseDetail, cfg.UseSummary, year, cfg.DetailYear, imports)
			totalFinalDemChan <- &demMatYearErr{dm: dm, y: year, e: e}
		}(year)

		go func(year Year) {
			imports := true
			dm, e := eio.loadFinalDemand(cfg.ImportsDetail, cfg.ImportsSummary, year, cfg.DetailYear, imports)
			importFinalDemChan <- &demMatYearErr{dm: dm, y: year, e: e}
		}(year)
	}

	go func() {
		_, s, e := eio.industriesDetail(cfg.TotalRequirementsDetail)
		industriesChan <- &stringsErr{s: s, e: e}
	}()
	go func() {
		_, s, e := eio.commoditiesDetail(cfg.TotalRequirementsDetail)
		commoditiesChan <- &stringsErr{s: s, e: e}
	}()

	for _ = range cfg.Years {
		tr := <-totalReqChan
		eio.totalRequirements[tr.y] = tr.m
		if tr.e != nil {
			return nil, tr.e
		}

		dr := <-domesticReqChan
		eio.domesticRequirements[dr.y] = dr.m
		if dr.e != nil {
			return nil, dr.e
		}

		tfd := <-totalFinalDemChan
		eio.totalFinalDemand[tfd.y] = tfd.dm
		if tfd.e != nil {
			return nil, tfd.e
		}

		ifd := <-importFinalDemChan
		eio.importFinalDemand[ifd.y] = ifd.dm
		if ifd.e != nil {
			return nil, ifd.e
		}
	}

	for _, year := range cfg.Years { // Calculate import requirements.
		imports := new(mat.Dense)
		imports.Sub(eio.totalRequirements[year], eio.domesticRequirements[year])
		eio.importRequirements[year] = imports
	}

	i := <-industriesChan
	eio.Industries = i.s
	if i.e != nil {
		return nil, i.e
	}
	c := <-commoditiesChan
	eio.Commodities = c.s
	if c.e != nil {
		return nil, c.e
	}
	eio.industryIndices = indexLookup(eio.Industries)
	eio.commodityIndices = indexLookup(eio.Commodities)

	return eio, err
}

//go:generate stringer -type=Location

// Location specifies where impacts occur, or where demanded commidities are from.
type Location int

const (
	// Domestic specifies impacts that occur locally or demand for domestic commodities.
	Domestic Location = iota
	// Imported specifies impacts that occur internationally or demand for imported commodities.
	Imported
	// Total is the combination of Domestic and Imported impacts or demand.
	Total
)

// EconomicImpacts returns the economic
// impacts of the given economic demand in the given year.
// The units of the output are the same as the units of demand.
// Location specifies whether return domestic, import, or total impacts.
func (e *EIO) EconomicImpacts(demand *mat.VecDense, year Year, loc Location) (*mat.VecDense, error) {
	var req *mat.Dense
	var ok bool
	switch loc {
	case Domestic:
		req, ok = e.domesticRequirements[year]
	case Imported:
		req, ok = e.importRequirements[year]
	case Total:
		req, ok = e.totalRequirements[year]
	default:
		return nil, fmt.Errorf("bea: invalid Location %v", loc)
	}
	if !ok {
		return nil, fmt.Errorf("invalid year %d", year)
	}
	o := new(mat.VecDense)
	o.MulVec(req, demand)
	return o, nil
}

// economicImpactsSCC returns the quasi-economic
// impacts of the given economic demand in the given year.
// These do not represent real economic values, only the relationships between
// sectors, so they should only be used for emissions, air quality or health modeling.
// Location specifies whether return domestic, import, or total impacts.
func (e *SpatialEIO) economicImpactsSCC(demand *mat.VecDense, year Year, loc Location) (*mat.VecDense, error) {
	var req *mat.Dense
	var ok bool
	switch loc {
	case Domestic:
		req, ok = e.domesticRequirementsSCC[year]
	case Imported:
		req, ok = e.importRequirementsSCC[year]
	case Total:
		req, ok = e.totalRequirementsSCC[year]
	default:
		return nil, fmt.Errorf("bea: invalid Location %v", loc)
	}
	if !ok {
		return nil, fmt.Errorf("invalid year %d", year)
	}
	o := new(mat.VecDense)
	o.MulVec(req, demand)
	return o, nil
}

// FinalDemand returns a final demand vector for the given year
// and demand type. commodities
// specifies which commodities the demand should be calculated for.
// loc specifies the demand location (Domestic, Imported, or Total)
// If commodities == nil, demand for all commodities is included.
func (e *EIO) FinalDemand(demandType FinalDemand, commodities *Mask, year Year, loc Location) (*mat.VecDense, error) {
	td, ok := e.totalFinalDemand[year]
	if !ok {
		return nil, fmt.Errorf("bea: invalid total demand year %d", year)
	}
	id, ok := e.importFinalDemand[year]
	if !ok {
		return nil, fmt.Errorf("bea: invalid import demand year %d", year)
	}

	tvTemp, ok := td[demandType]
	if !ok {
		return nil, fmt.Errorf("bea: invalid total demand type %s", demandType)
	}
	ivTemp, ok := id[demandType]
	if !ok {
		return nil, fmt.Errorf("bea: invalid import demand type %s", demandType)
	}

	r, _ := tvTemp.Dims()
	v := mat.NewVecDense(r, nil)
	switch loc {
	case Domestic:
		v.SubVec(tvTemp, ivTemp)
	case Imported:
		v.CloneVec(ivTemp)
	case Total:
		v.CloneVec(tvTemp)
	default:
		panic(fmt.Errorf("invalid final demand location %s", loc.String()))
	}
	if commodities != nil {
		// Set activity in industries we're not interested in to zero.
		commodities.Mask(v)
	}
	return v, nil
}

// FinalDemandSingle returns a final demand vector with the given amount
// specified for the given commodity sector, and zeros for all other
// sectors.
func (e *EIO) FinalDemandSingle(commodity string, amount float64) (*mat.VecDense, error) {
	v := mat.NewVecDense(len(e.Commodities), nil)
	i, err := e.CommodityIndex(commodity)
	if err != nil {
		return nil, err
	}
	v.SetVec(i, amount)
	return v, nil
}

// SectorError is returned when an invalid sector is requested.
type SectorError struct {
	name string
}

func (err SectorError) Error() string {
	return fmt.Sprintf("invalid sector name `%v`", err.name)
}

// IndustryIndex returns the index number of the specified sector.
func (e *EIO) IndustryIndex(name string) (int, error) {
	if i, ok := e.industryIndices[name]; ok {
		return i, nil
	}
	return -1, SectorError{name}
}

// CommodityIndex returns the index number of the specified sector.
func (e *EIO) CommodityIndex(name string) (int, error) {
	if i, ok := e.commodityIndices[name]; ok {
		return i, nil
	}
	return -1, SectorError{name}
}
