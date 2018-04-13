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
	"fmt"
	"log"
	"math"
	"testing"
)

const tolerance = 1.e-10 // tolerance for float comparison

func different(a, b float64) bool {
	if math.IsNaN(a) || math.IsNaN(b) || math.IsInf(a, 0) || math.IsInf(b, 0) {
		return true
	}
	if math.Abs((a-b)/b) > tolerance {
		return true
	}
	return false
}

func TestEIO(t *testing.T) {
	t.Parallel()

	e := loadSpatial(t).EIO

	fd2007, err := e.FinalDemand(All, nil, 2007, Total)
	if err != nil {
		t.Fatal(err)
	}
	// Impacts from total demand = total production.
	production2007, err := e.EconomicImpacts(fd2007, 2007, Total)
	if err != nil {
		t.Fatal(err)
	}
	// Oilseed commodity is same index as oilseed industry.
	iOilSeed, err := e.CommodityIndex("Oilseed farming")
	if err != nil {
		t.Fatal(err)
	}
	oilseedProduction2007 := production2007.At(iOilSeed, 0)
	// the want value is from IOUse_Before_Redefinitions_PRO_2007_Detail.xlsx,
	// "total industry output" row. The spreadsheet value is $21,425 million.
	// The calculated value is slightly different because we are setting negative
	// final demand to zero.
	const wantOilseedProduction2007 = 2.65690994625e+10
	if different(oilseedProduction2007, wantOilseedProduction2007) {
		t.Errorf("oilseed production 2007: have %g, want %g", oilseedProduction2007, wantOilseedProduction2007)
	}

	domestic, err := e.EconomicImpacts(fd2007, 2007, Domestic)
	if err != nil {
		t.Fatal(err)
	}
	imports, err := e.EconomicImpacts(fd2007, 2007, Imported)
	if err != nil {
		t.Fatal(err)
	}
	domesticOilseed := domestic.At(iOilSeed, 0)
	importedOilseed := imports.At(iOilSeed, 0)
	if different(domesticOilseed+importedOilseed, wantOilseedProduction2007) {
		t.Errorf("domestic+imported oilseed production 2007: have %g, want %g", domesticOilseed+importedOilseed, wantOilseedProduction2007)
	}
}

func BenchmarkLoadEIO(b *testing.B) {
	cfg := Config{
		Years:                       []Year{2007, 2011},
		DetailYear:                  2007,
		UseSummary:                  "data/IOUse_Before_Redefinitions_PRO_1997-2015_Summary.xlsx",
		UseDetail:                   "data/IOUse_Before_Redefinitions_PRO_2007_Detail.xlsx",
		ImportsSummary:              "data/ImportMatrices_Before_Redefinitions_SUM_1997-2016.xlsx",
		ImportsDetail:               "data/ImportMatrices_Before_Redefinitions_DET_2007.xlsx",
		TotalRequirementsSummary:    "data/IxC_TR_1997-2015_Summary.xlsx",
		TotalRequirementsDetail:     "data/IxC_TR_2007_Detail.xlsx",
		DomesticRequirementsSummary: "data/IxC_Domestic_1997-2015_Summary.xlsx",
		DomesticRequirementsDetail:  "data/IxC_Domestic_2007_Detail.xlsx",
	}

	_, err := New(&cfg)
	if err != nil {
		b.Fatal(err)
	}
}

// This example estimates the domestic and imported purchases of
// petroleum caused by the demand for $1 million worth of
// coal in the U.S. in year 2011.
func Example() {
	// Set up the configuration information for the simulation.
	cfg := Config{
		Years:                       []Year{2011},
		DetailYear:                  2007, // DetailYear is always 2007.
		UseSummary:                  "data/IOUse_Before_Redefinitions_PRO_1997-2015_Summary.xlsx",
		UseDetail:                   "data/IOUse_Before_Redefinitions_PRO_2007_Detail.xlsx",
		ImportsSummary:              "data/ImportMatrices_Before_Redefinitions_SUM_1997-2016.xlsx",
		ImportsDetail:               "data/ImportMatrices_Before_Redefinitions_DET_2007.xlsx",
		TotalRequirementsSummary:    "data/IxC_TR_1997-2015_Summary.xlsx",
		TotalRequirementsDetail:     "data/IxC_TR_2007_Detail.xlsx",
		DomesticRequirementsSummary: "data/IxC_Domestic_1997-2015_Summary.xlsx",
		DomesticRequirementsDetail:  "data/IxC_Domestic_2007_Detail.xlsx",
	}

	// Create a new model.
	e, err := New(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	// Specify $1 million of demand for coal.
	finalDemand, err := e.FinalDemandSingle("Coal mining", 1.0e6)
	if err != nil {
		log.Fatal(err)
	}

	// Calculate the impacts of our economic demand on the year 2011 economy.
	domestic, err := e.EconomicImpacts(finalDemand, 2011, Domestic)
	if err != nil {
		log.Fatal(err)
	}
	imports, err := e.EconomicImpacts(finalDemand, 2011, Imported)
	if err != nil {
		log.Fatal(err)
	}
	// We're interested in purchases of petroleum, so find the appropriate sector.
	petroleumSector, err := e.IndustryIndex("Petroleum refineries")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("domestic purchases of petroleum: $%0.0f\n", domestic.At(petroleumSector, 0))
	fmt.Printf("imported purchases of petroleum: $%0.0f\n", imports.At(petroleumSector, 0))
	// Output:
	// domestic purchases of petroleum: $49589
	// imported purchases of petroleum: $21037
}
