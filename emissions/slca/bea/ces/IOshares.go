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
	"fmt"
	"go/build"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/spatialmodel/inmap/emissions/slca/bea"

	"gonum.org/v1/gonum/mat"

	"github.com/tealeg/xlsx"
)

// Filedir specifies the directory of the CES data files.
var Filedir string

func init() {
	cespkg, err := build.Import("bitbucket.org/ctessum/slca/bea/ces", "", build.FindOnly)
	if err != nil {
		panic(err)
	}
	Filedir = filepath.Join(cespkg.SrcRoot, cespkg.ImportPath)
}

// CES holds the fractions of personal expenditures that are incurred by
// non-hispanic white people by year and 389-sector EIO category.
type CES struct {
	// StartYear and EndYear are the beginning and ending
	// years for data availability, respectively.
	StartYear, EndYear int

	whiteFractions  map[int]map[string]float64
	blackFractions  map[int]map[string]float64
	latinoFractions map[int]map[string]float64
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
func NewCES() (*CES, error) {
	// Create map of IO categories to CE categories
	ioCEMap := make(map[string][]string)
	ioCEXLSXPath := filepath.Join(Filedir, "IO-CEcrosswalk.xlsx")
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
	}

	for _, sheet := range ioCEXLSX.Sheets {
		for i, row := range sheet.Rows {
			// Skip column headers
			if i == 0 {
				continue
			}
			key := row.Cells[0].Value
			ioCEMap[key] = append(ioCEMap[key], row.Cells[1].Value)
		}
	}

	ceKeys, err := txtToSlice(filepath.Join(Filedir, "CEkeys.txt"))
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
		inputFileName := filepath.Join(Filedir, fmt.Sprintf("hispanic%d.xlsx", year))
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
	return &ces, nil
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

var ignoreSectors = map[string]struct{}{
	"Noncomparable imports":                                                                  struct{}{},
	"Other nonmetallic mineral mining and quarrying":                                         struct{}{},
	"Iron and steel mills and ferroalloy manufacturing":                                      struct{}{},
	"Nonferrous metal (except copper and aluminum) rolling, drawing, extruding and alloying": struct{}{},
	"Nonferrous metal foundries":                                                             struct{}{},
	"Crown and closure manufacturing and metal stamping":                                     struct{}{},
	"Plate work and fabricated structural product manufacturing":                             struct{}{},
	"Metal can, box, and other metal container (light gauge) manufacturing":                  struct{}{},
	"Hardware manufacturing":                                                                 struct{}{},
	"Spring and wire product manufacturing":                                                  struct{}{},
	"Office machinery manufacturing":                                                         struct{}{},
	"Metal cutting and forming machine tool manufacturing":                                   struct{}{},
	"Other engine equipment manufacturing":                                                   struct{}{},
	"Grantmaking, giving, and social advocacy organizations":                                 struct{}{},
	"Civic, social, professional, and similar organizations":                                 struct{}{},
	"Private households":                                                                     struct{}{},
	"Other state and local government enterprises":                                           struct{}{},
	"Individual and family services":                                                         struct{}{},
	"Other support services":                                                                 struct{}{},
	"Veterinary services":                                                                    struct{}{},
	"Employment services":                                                                    struct{}{},
	"Business support services":                                                              struct{}{},
	"Travel arrangement and reservation services":                                            struct{}{},
	"Investigation and security services":                                                    struct{}{},
	"Commercial and industrial machinery and equipment rental and leasing":                   struct{}{},
	"Legal services": struct{}{},
	"Accounting, tax preparation, bookkeeping, and payroll services":  struct{}{},
	"Specialized design services":                                     struct{}{},
	"Scientific research and development services":                    struct{}{},
	"Advertising, public relations, and related services":             struct{}{},
	"Funds, trusts, and other financial vehicles":                     struct{}{},
	"Other real estate":                                               struct{}{},
	"Securities and commodity contracts intermediation and brokerage": struct{}{},
	"Other financial investment activities":                           struct{}{},
	"Couriers and messengers":                                         struct{}{},
	"Warehousing and storage":                                         struct{}{},
	"Wholesale trade":                                                 struct{}{},
	"Industrial gas manufacturing":                                    struct{}{},
	"Support activities for agriculture and forestry":                 struct{}{},
	"Religious organizations":                                         struct{}{},
}

// WhiteOtherDemand returns the domestic personal consumption final demand by white non-Latino
// people and people of other races besides Black and Latino.
func (c *CES) WhiteOtherDemand(eio *bea.EIO, commodities *bea.Mask, year bea.Year) (*mat.VecDense, error) {
	demand, err := eio.FinalDemand(bea.PersonalConsumption, commodities, year, bea.Domestic)
	if err != nil {
		return nil, err
	}
	for i, sector := range eio.Commodities {
		v := demand.At(i, 0)
		if v == 0 {
			continue
		}
		if _, ok := ignoreSectors[sector]; ok {
			demand.SetVec(i, 0)
			continue
		}
		f, err := c.whiteOtherFrac(int(year), sector)
		if err != nil {
			fmt.Println(err)
			f = 0
			//return nil, err
		}
		demand.SetVec(i, v*f)
	}
	return demand, nil
}

// BlackDemand returns the domestic personal consumption final demand by
// Black people.
func (c *CES) BlackDemand(eio *bea.EIO, commodities *bea.Mask, year bea.Year) (*mat.VecDense, error) {
	demand, err := eio.FinalDemand(bea.PersonalConsumption, commodities, year, bea.Domestic)
	if err != nil {
		return nil, err
	}
	for i, sector := range eio.Commodities {
		v := demand.At(i, 0)
		if v == 0 {
			continue
		}
		if _, ok := ignoreSectors[sector]; ok {
			demand.SetVec(i, 0)
			continue
		}
		f, err := c.blackFrac(int(year), sector)
		if err != nil {
			fmt.Println(err)
			f = 0
			//return nil, err
		}
		demand.SetVec(i, v*f)
	}
	return demand, nil
}

// LatinoDemand returns the domestic personal consumption final demand by
// Latino people.
func (c *CES) LatinoDemand(eio *bea.EIO, commodities *bea.Mask, year bea.Year) (*mat.VecDense, error) {
	demand, err := eio.FinalDemand(bea.PersonalConsumption, commodities, year, bea.Domestic)
	if err != nil {
		return nil, err
	}
	for i, sector := range eio.Commodities {
		v := demand.At(i, 0)
		if v == 0 {
			continue
		}
		if _, ok := ignoreSectors[sector]; ok {
			demand.SetVec(i, 0)
			continue
		}
		f, err := c.latinoFrac(int(year), sector)
		if err != nil {
			fmt.Println(err)
			f = 0
			//return nil, err
		}
		demand.SetVec(i, v*f)
	}
	return demand, nil
}
