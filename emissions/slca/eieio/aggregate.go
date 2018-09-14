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
	"fmt"

	"github.com/spatialmodel/inmap/emissions/slca"
	"gonum.org/v1/gonum/mat"
)

// Aggregator provides functionality for grouping industry and commodity sectors.
type Aggregator struct {
	aggregateNames, aggregateAbbrevs        []string
	aggregates                              map[string]string
	IndustryAggregates, CommodityAggregates []string
}

// Names returns the names of the aggregated groups.
func (a *Aggregator) Names() []string { return a.aggregateNames }

// Abbreviations returns the abbreviated names of the aggregated groups.
func (a *Aggregator) Abbreviations() []string { return a.aggregateAbbrevs }

// Abbreviation returns the abbreviation associated with the given
// name.
func (a *Aggregator) Abbreviation(name string) (string, error) {
	abbrev, ok := a.aggregates[name]
	if !ok {
		return "", fmt.Errorf("eieio.Aggregator: invalid name %s", name)
	}
	return abbrev, nil
}

// NewIOAggregator initializes a new Aggregator of Input-Output categories
// from the information in the provided file.
func (e *EIO) NewIOAggregator(fileName string) (*Aggregator, error) {
	const (
		industryCol           = 1
		industryAggregateCol  = 2
		commodityCol          = 5
		commodityAggregateCol = 6
		aggregateNameCol      = 8
		aggregateAbbrevCol    = 9
		startRow              = 1
		endRow                = 390
		aggregateEndRow       = 8
		sheet                 = "bea"
	)
	// Check industries.
	industries, err := e.textColumnFromExcel(fileName, sheet, industryCol, startRow, endRow)
	if err != nil {
		return nil, err
	}
	if len(industries) != len(e.industries) {
		return nil, fmt.Errorf("eieio.NewIOAggregator: incorrect number of industries: %d != %d", len(industries), len(e.industries))
	}
	for i, ind := range e.industries {
		if industries[i] != ind {
			return nil, fmt.Errorf("eieio.NewIOAggregator: industries don't match: %s != %s", industries[i], ind)
		}
	}
	// Check commodities.
	commodities, err := e.textColumnFromExcel(fileName, sheet, commodityCol, startRow, endRow)
	if err != nil {
		return nil, err
	}
	if len(commodities) != len(e.commodities) {
		return nil, fmt.Errorf("eieio.NewIOAggregator: incorrect number of commodities: %d != %d", len(commodities), len(e.commodities))
	}
	for i, com := range e.commodities {
		if commodities[i] != com {
			return nil, fmt.Errorf("eieio.NewIOAggregator: commodities don't match: %s != %s", commodities[i], com)
		}
	}

	a := new(Aggregator)

	a.aggregateNames, err = e.textColumnFromExcel(fileName, sheet, aggregateNameCol, startRow, aggregateEndRow)
	if err != nil {
		return nil, err
	}
	a.aggregateAbbrevs, err = e.textColumnFromExcel(fileName, sheet, aggregateAbbrevCol, startRow, aggregateEndRow)
	if err != nil {
		return nil, err
	}
	a.IndustryAggregates, err = e.textColumnFromExcel(fileName, sheet, industryAggregateCol, startRow, endRow)
	if err != nil {
		return nil, err
	}
	a.CommodityAggregates, err = e.textColumnFromExcel(fileName, sheet, commodityAggregateCol, startRow, endRow)
	if err != nil {
		return nil, err
	}

	a.aggregates = make(map[string]string)
	abbrevNames := make(map[string]struct{})
	for i, n := range a.aggregateNames {
		a.aggregates[n] = a.aggregateAbbrevs[i]
		abbrevNames[a.aggregateAbbrevs[i]] = struct{}{}
	}

	for _, abbrevs := range [][]string{a.IndustryAggregates, a.CommodityAggregates} {
		for _, abbrev := range abbrevs {
			if _, ok := abbrevNames[abbrev]; !ok {
				return nil, fmt.Errorf("eieio.NewIOAggregator: invalid aggregation category %s", abbrev)
			}
		}
	}
	return a, nil
}

// NewSCCAggregator initializes a new Aggregator of Source Classification Codes
// from the information in the provided file.
func (e *SpatialEIO) NewSCCAggregator(fileName string) (*Aggregator, error) {
	const (
		sccCol             = 6
		sccAggregateCol    = 7
		aggregateNameCol   = 3
		aggregateAbbrevCol = 4
		startRow           = 1
		endRow             = 5435
		aggregateEndRow    = 15
		sheet              = "scc"
	)
	// Check SCCs.
	sccs, err := e.textColumnFromExcel(fileName, sheet, sccCol, startRow, endRow)
	if err != nil {
		return nil, err
	}
	for i, scc := range sccs {
		if scc == "" {
			sccs = sccs[0:i]
			break
		}
	}
	if len(sccs) != len(e.SCCs) {
		return nil, fmt.Errorf("eieio.NewSCCAggregator: incorrect number of SCCs: %d != %d", len(sccs), len(e.SCCs))
	}
	for i, scc := range e.SCCs {
		if sccs[i] != string(scc) {
			return nil, fmt.Errorf("eieio.NewSCCAggregator: SCCs don't match: %s != %s", sccs[i], scc)
		}
	}

	a := new(Aggregator)

	a.aggregateNames, err = e.textColumnFromExcel(fileName, sheet, aggregateNameCol, startRow, aggregateEndRow)
	if err != nil {
		return nil, err
	}
	a.aggregateAbbrevs, err = e.textColumnFromExcel(fileName, sheet, aggregateAbbrevCol, startRow, aggregateEndRow)
	if err != nil {
		return nil, err
	}
	a.IndustryAggregates, err = e.textColumnFromExcel(fileName, sheet, sccAggregateCol, startRow, endRow)
	if err != nil {
		return nil, err
	}
	for i, v := range a.IndustryAggregates {
		if v == "" {
			a.IndustryAggregates = a.IndustryAggregates[0:i]
			break
		}
	}

	a.aggregates = make(map[string]string)
	abbrevNames := make(map[string]struct{})
	for i, n := range a.aggregateNames {
		a.aggregates[n] = a.aggregateAbbrevs[i]
		abbrevNames[a.aggregateAbbrevs[i]] = struct{}{}
	}

	for _, abbrev := range a.IndustryAggregates {
		if _, ok := abbrevNames[abbrev]; !ok {
			return nil, fmt.Errorf("eieio.NewSCCAggregator: invalid aggregation category '%s'", abbrev)
		}
	}
	return a, nil
}

// A Mask is a vector of ones and zeros.
type Mask mat.VecDense

// Mask multiplies v by the receiver, element-wise.
func (m *Mask) Mask(v *mat.VecDense) {
	v2 := (mat.VecDense)(*m)
	v.MulElemVec(&v2, v)
}

// IndustryMask returns a vector of ones in industry sectors that match
// the given aggregated group abbrevation and zeros elsewhere.
func (a *Aggregator) IndustryMask(abbrev string) *Mask { return mask(abbrev, a.IndustryAggregates) }

// CommodityMask returns a vector of ones in commodity sectors that match
// the given aggregated group abbrevation and zeros elsewhere.
func (a *Aggregator) CommodityMask(abbrev string) *Mask { return mask(abbrev, a.CommodityAggregates) }

func mask(abbrev string, aggs []string) *Mask {
	m := mat.NewVecDense(len(aggs), nil)
	for i, ag := range aggs {
		if ag == abbrev {
			m.SetVec(i, 1)
		}
	}
	mm := Mask(*m)
	return &mm
}

// SCCMask returns a mask to single out the given SCC code.
func (e *SpatialEIO) SCCMask(code slca.SCC) (*Mask, error) {
	i, ok := e.sccIndex[code]
	if !ok {
		return nil, fmt.Errorf("eieio: missing SCC code %s", code)
	}
	m := mat.NewVecDense(len(e.SCCs), nil)
	m.SetVec(i, 1)
	mm := Mask(*m)
	return &mm, nil
}

// IndustryMask returns a mask to single out the given industry.
func (e *EIO) IndustryMask(name string) (*Mask, error) {
	i, err := e.IndustryIndex(name)
	if err != nil {
		return nil, err
	}
	m := mat.NewVecDense(len(e.industries), nil)
	m.SetVec(i, 1)
	mm := Mask(*m)
	return &mm, nil
}

// CommodityMask returns a mask to single out the given commodity.
func (e *EIO) CommodityMask(name string) (*Mask, error) {
	i, err := e.CommodityIndex(name)
	if err != nil {
		return nil, err
	}
	m := mat.NewVecDense(len(e.commodities), nil)
	m.SetVec(i, 1)
	mm := Mask(*m)
	return &mm, nil
}
