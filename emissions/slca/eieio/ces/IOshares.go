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

// Package ces translates Consumer Expenditure Survey (CES) demographic data
// to EIO categories.
package ces

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gonum/floats"
	"github.com/spatialmodel/inmap/emissions/slca/eieio/eieiorpc"

	"github.com/tealeg/xlsx"
)

// CES holds the fractions of personal expenditures that are incurred by
// non-hispanic white people by year and 389-sector EIO category.
type CES struct {
	// StartYear and EndYear are the beginning and ending
	// years for data availability, respectively.
	StartYear, EndYear int

	whiteFractions  map[int]map[string]float64
	blackFractions  map[int]map[string]float64
	latinoFractions map[int]map[string]float64

	eio eieiorpc.EIEIOrpcServer
}

// txtToSlice converts a line-delimited list of strings in
// a text file into a slice.
func txtToSlice(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// matchSharesToSectors takes two maps:
// 	1. Map of each IO sector to one or more CE sectors that it consists of
// 	2. Map of each CE sector to the a) spending share and b) aggregate spending
// 		associated with the desired demographic group
//
// It returns a new map is created that matches each IO sector to a slice of one
// or more sets of spending shares and aggregate spending amounts from the
// relevant CE sectors.
func matchSharesToSectors(m map[string][]string, m2 map[string][]float64) (m3 map[string][]float64) {
	m3 = make(map[string][]float64)
	for key, vals := range m {
		for _, mVal := range vals {
			for _, m2Val := range m2[mVal] {
				m3[key] = append(m3[key], m2Val)
			}
		}
	}
	return m3
}

// weightedAvgShares takes the map created by matchSharesToSectors()
// and performs a weighted average (based on aggregate spending) of all the
// shares that are associated with each IO code.
func weightedAvgShares(m map[string][]float64) (m2 map[string]float64) {
	m2 = make(map[string]float64)
	for key, vals := range m {
		var aggregateSum float64
		for i, val := range vals {
			if (i+1)%2 != 0 {
				aggregateSum = aggregateSum + val
			}
		}
		var numerator float64
		for i := 0; i < len(vals)/2; i++ {
			numerator = numerator + vals[i*2]*vals[i*2+1]
		}
		if aggregateSum != 0 {
			m2[key] = numerator / aggregateSum
		} else {
			m2[key] = 0
		}
	}
	return m2
}

// dataToXlsxFile populates the output tables with spending shares
func dataToXlsxFile(d map[string]float64, xlsxFile *xlsx.File, sheet string) {
	var cell *xlsx.Cell
	for i, row := range xlsxFile.Sheet[sheet].Rows {
		// Skips top row which contains column headings
		if i == 0 {
			continue
		}
		cell = row.AddCell()
		cell.SetFloat(d[row.Cells[0].Value])
	}
}

// NewCES loads data into a new CES object.
func NewCES(eio eieiorpc.EIEIOrpcServer, dataDir string) (*CES, error) {
	dataDir = os.ExpandEnv(dataDir)
	// Create map of IO categories to CE categories
	ioCEMap := make(map[string][]string)
	ioCEXLSXPath := filepath.Join(dataDir, "IO-CEcrosswalk.xlsx")
	ioCEXLSX, err := xlsx.OpenFile(ioCEXLSXPath)
	if err != nil {
		return nil, err
	}

	ces := CES{
		StartYear:       2003,
		EndYear:         2015,
		whiteFractions:  make(map[int]map[string]float64),
		blackFractions:  make(map[int]map[string]float64),
		latinoFractions: make(map[int]map[string]float64),
		eio:             eio,
	}

	for _, sheet := range ioCEXLSX.Sheets {
		for i, row := range sheet.Rows {
			// Skip column headers
			if i == 0 {
				continue
			}
			key := row.Cells[0].Value // The key is the IO commodity
			for j := 1; j < len(row.Cells); j++ {
				if row.Cells[j].Value != "" {
					ioCEMap[key] = append(ioCEMap[key], row.Cells[j].Value)
				}
			}
		}
	}

	ceKeys, err := txtToSlice(filepath.Join(dataDir, "CEkeys.txt"))
	if err != nil {
		return nil, err
	}

	// Flags specific string values to be replaced when looping through data
	// a Value is too small to display.
	// b Data are likely to have large sampling errors.
	// c No data reported.
	r := strings.NewReplacer("a/", "", "b/", "", "c/", "", " ", "")
	s2f := func(s string) (float64, error) {
		s2 := r.Replace(s)
		if s2 == "" {
			return 0, nil
		}
		return strconv.ParseFloat(s2, 64)
	}

	// Loop through each year of available data
	// - data contain necessary metrics starting in 2003
	for year := ces.StartYear; year <= ces.EndYear; year++ {

		nonHispWhite := make(map[string][]float64)
		latino := make(map[string][]float64)
		black := make(map[string][]float64)

		// Open raw CE data files
		inputFileName := filepath.Join(dataDir, fmt.Sprintf("hispanic%d.xlsx", year))
		var inputFile *xlsx.File
		inputFile, err = xlsx.OpenFile(inputFileName)
		if err != nil {
			return nil, err
		}
		sheet := inputFile.Sheets[0]
		for _, row := range sheet.Rows {

			// Skip blank rows
			if len(row.Cells) == 0 {
				continue
			}

			// The key is the CES category.
			key := strings.Trim(row.Cells[0].Value, " ")

			// For each CE category that we are interested in, find the
			// corresponding row in the raw CE data files and pull spending
			// share and aggregate spending numbers.
			for _, line := range ceKeys {
				match, err := regexp.MatchString("^"+line+"$", key)
				if err != nil {
					return nil, err
				}
				if match {
					aggregate, err := s2f(row.Cells[1].Value)
					if err != nil {
						return nil, err
					}

					shareNonHispWhite, err := s2f(row.Cells[4].Value)
					if err != nil {
						return nil, err
					}
					shareLatino, err := s2f(row.Cells[2].Value)
					if err != nil {
						return nil, err
					}
					shareBlack, err := s2f(row.Cells[5].Value)
					if err != nil {
						return nil, err
					}
					nonHispWhite[key] = append(nonHispWhite[key], aggregate*shareNonHispWhite/100)
					nonHispWhite[key] = append(nonHispWhite[key], shareNonHispWhite/100)
					latino[key] = append(latino[key], aggregate*shareLatino/100)
					latino[key] = append(latino[key], shareLatino/100)
					black[key] = append(black[key], aggregate*shareBlack/100)
					black[key] = append(black[key], shareBlack/100)

				}
			}
		}
		nonHispWhiteIO := matchSharesToSectors(ioCEMap, nonHispWhite)
		nonHispWhiteFinal := weightedAvgShares(nonHispWhiteIO)
		latinoIO := matchSharesToSectors(ioCEMap, latino)
		latinoFinal := weightedAvgShares(latinoIO)
		blackIO := matchSharesToSectors(ioCEMap, black)
		blackFinal := weightedAvgShares(blackIO)

		ces.whiteFractions[year] = nonHispWhiteFinal
		ces.latinoFractions[year] = latinoFinal
		ces.blackFractions[year] = blackFinal
	}
	ces.normalize()
	return &ces, nil
}

func (ces *CES) normalize() {
	for year := range ces.whiteFractions {
		for sector := range ces.whiteFractions[year] {
			whiteV := ces.whiteFractions[year][sector]
			latinoV := ces.latinoFractions[year][sector]
			blackV := ces.blackFractions[year][sector]
			total := whiteV + latinoV + blackV
			ces.whiteFractions[year][sector] = whiteV / total
			ces.latinoFractions[year][sector] = latinoV / total
			ces.blackFractions[year][sector] = blackV / total
		}
	}
}

// ErrMissingSector happens when a IO sector is requested which there is
// no data for.
type ErrMissingSector struct {
	sector string
	year   int
}

func (e ErrMissingSector) Error() string {
	return fmt.Sprintf("ces: missing IO sector '%s'; year %d", e.sector, e.year)
}

// whiteOtherFrac returns the fraction of total consumption incurred by
// non-hispanic white people and other races in the given year and IO sector.
func (c *CES) whiteOtherFrac(year int, IOSector string) (float64, error) {
	if year > c.EndYear || year < c.StartYear {
		return math.NaN(), fmt.Errorf("ces: year %d is outside of allowed range %d--%d", year, c.StartYear, c.EndYear)
	}
	v, ok := c.whiteFractions[year][IOSector]
	if !ok {
		return math.NaN(), ErrMissingSector{sector: IOSector, year: year}
	}
	return v, nil
}

// blackFrac returns the fraction of total consumption incurred by
// black people in the given year and IO sector.
func (c *CES) blackFrac(year int, IOSector string) (float64, error) {
	if year > c.EndYear || year < c.StartYear {
		return math.NaN(), fmt.Errorf("ces: year %d is outside of allowed range %d--%d", year, c.StartYear, c.EndYear)
	}
	v, ok := c.blackFractions[year][IOSector]
	if !ok {
		return math.NaN(), ErrMissingSector{sector: IOSector, year: year}
	}
	return v, nil
}

// latinoFrac returns the fraction of total consumption incurred by
// latino people in the given year and IO sector.
func (c *CES) latinoFrac(year int, IOSector string) (float64, error) {
	if year > c.EndYear || year < c.StartYear {
		return math.NaN(), fmt.Errorf("ces: year %d is outside of allowed range %d--%d", year, c.StartYear, c.EndYear)
	}
	v, ok := c.latinoFractions[year][IOSector]
	if !ok {
		return math.NaN(), ErrMissingSector{sector: IOSector, year: year}
	}
	return v, nil
}

// DemographicConsumption returns domestic personal consumption final demand
// plus private final demand for the specified demograph.
// Personal consumption and private residential expenditures are directly adjusted
// using the frac function.
// Other private expenditures are adjusted by the scalar:
//		adj = sum(frac(personal + private_residential)) / sum(personal + private_residential)
// Acceptable demographs:
//		Black: People self-identifying as black or African-American.
//		Hispanic: People self-identifying as Hispanic or Latino.
//		WhiteOther: People self identifying as white or other races besides black, and not Hispanic.
//		All: The total population.
func (c *CES) DemographicConsumption(ctx context.Context, in *eieiorpc.DemographicConsumptionInput) (*eieiorpc.Vector, error) {
	var frac func(int, string) (float64, error)
	switch in.Demograph {
	case eieiorpc.Demograph_Black:
		frac = c.blackFrac
	case eieiorpc.Demograph_Hispanic:
		frac = c.latinoFrac
	case eieiorpc.Demograph_WhiteOther:
		frac = c.whiteOtherFrac
	case eieiorpc.Demograph_All:
		frac = func(int, string) (float64, error) { return 1, nil }
	default:
		return nil, fmt.Errorf("invalid demograph: %s", in.Demograph)
	}
	return c.adjustDemand(ctx, in.EndUseMask, in.Year, frac)
}

// adjustDemand returns domestic personal consumption final demand plus private final demand
// after adjusting it using the frac function.
// Personal consumption and private residential expenditures are directly adjusted
// using the frac function.
// Other private expenditures are adjusted by the scalar:
//		adj = sum(frac(personal + private_residential)) / sum(personal + private_residential)
func (c *CES) adjustDemand(ctx context.Context, commodities *eieiorpc.Mask, year int32, frac func(year int, commodity string) (float64, error)) (*eieiorpc.Vector, error) {
	// First, get the adjusted personal consumption.
	pc, err := c.eio.FinalDemand(ctx, &eieiorpc.FinalDemandInput{
		FinalDemandType: eieiorpc.FinalDemandType_PersonalConsumption,
		Year:            year,
		Location:        eieiorpc.Location_Domestic,
	})
	if err != nil {
		return nil, err
	}

	// Then, get the private residential expenditures.
	pcRes, err := c.eio.FinalDemand(ctx, &eieiorpc.FinalDemandInput{
		FinalDemandType: eieiorpc.FinalDemandType_PrivateResidential,
		Year:            year,
		Location:        eieiorpc.Location_Domestic,
	})
	if err != nil {
		return nil, err
	}

	// Now, add the two together
	floats.Add(pc.Data, pcRes.Data)

	// Next, adjust the personal consumption by the provided fractions.
	demand := &eieiorpc.Vector{
		Data: make([]float64, len(pc.Data)),
	}
	commodityList, err := c.eio.Commodities(ctx, nil)
	if err != nil {
		return nil, err
	}
	for i, sector := range commodityList.List {
		v := pc.Data[i]
		if v == 0 {
			continue
		}
		f, err := frac(int(year), sector)
		if err != nil {
			return nil, err
		}
		demand.Data[i] = v * f
	}

	// Now we create an adjustment factor and use it to adjust the
	// rest of the private expenditures.
	adj := floats.Sum(demand.Data) / floats.Sum(pc.Data)
	for _, dt := range []eieiorpc.FinalDemandType{
		eieiorpc.FinalDemandType_PrivateStructures,
		eieiorpc.FinalDemandType_PrivateEquipment,
		eieiorpc.FinalDemandType_PrivateIP,
		eieiorpc.FinalDemandType_InventoryChange} {

		d, err := c.eio.FinalDemand(ctx, &eieiorpc.FinalDemandInput{
			FinalDemandType: dt,
			Year:            year,
			Location:        eieiorpc.Location_Domestic,
		})
		if err != nil {
			return nil, err
		}
		floats.AddScaled(demand.Data, adj, d.Data)
	}
	if commodities != nil {
		floats.Mul(demand.Data, commodities.Data) // Apply the mask.
	}
	return demand, nil
}
