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

	"gonum.org/v1/gonum/mat"

	"github.com/spatialmodel/inmap/emissions/slca/eieio/eieiorpc"
)

// FinalDemand specifies the available types of final demand.
type FinalDemand string

// These constants specify the available types of final demand.
// The provided codes correspond to the codes at the summary level of
// detail. For the detailed level, "00" should be added to the end of
// each code.
const (
	// This group of demand types is directly available in the spreadsheet.
	PersonalConsumption   FinalDemand = "F010"
	PrivateStructures                 = "F02S"
	PrivateEquipment                  = "F02E"
	PrivateIP                         = "F02N"
	PrivateResidential                = "F02R"
	InventoryChange                   = "F030"
	Export                            = "F040"
	DefenseConsumption                = "F06C"
	DefenseStructures                 = "F06S"
	DefenseEquipment                  = "F06E"
	DefenseIP                         = "F06N"
	NondefenseConsumption             = "F07C"
	NondefenseStructures              = "F07S"
	NondefenseEquipment               = "F07E"
	NondefenseIP                      = "F07N"
	LocalConsumption                  = "F10C"
	LocalStructures                   = "F10S"
	LocalEquipment                    = "F10E"
	LocalIP                           = "F10N"

	// This group of demand types consists of aggregates of the
	// above types.
	AllDemand = FinalDemand(All) // All is a combination of all categories above.
	NonExport = "NonExport"      // NonExport is (All - Export)
)

func convertFinalDemand(in eieiorpc.FinalDemandType) FinalDemand {
	switch in {
	case eieiorpc.FinalDemandType_PersonalConsumption:
		return PersonalConsumption
	case eieiorpc.FinalDemandType_PrivateStructures:
		return PrivateStructures
	case eieiorpc.FinalDemandType_PrivateEquipment:
		return PrivateEquipment
	case eieiorpc.FinalDemandType_PrivateIP:
		return PrivateIP
	case eieiorpc.FinalDemandType_PrivateResidential:
		return PrivateResidential
	case eieiorpc.FinalDemandType_InventoryChange:
		return InventoryChange
	case eieiorpc.FinalDemandType_Export:
		return Export
	case eieiorpc.FinalDemandType_DefenseConsumption:
		return DefenseConsumption
	case eieiorpc.FinalDemandType_DefenseStructures:
		return DefenseStructures
	case eieiorpc.FinalDemandType_DefenseEquipment:
		return DefenseEquipment
	case eieiorpc.FinalDemandType_DefenseIP:
		return DefenseIP
	case eieiorpc.FinalDemandType_NondefenseConsumption:
		return NondefenseConsumption
	case eieiorpc.FinalDemandType_NondefenseStructures:
		return NondefenseStructures
	case eieiorpc.FinalDemandType_NondefenseEquipment:
		return NondefenseEquipment
	case eieiorpc.FinalDemandType_NondefenseIP:
		return NondefenseIP
	case eieiorpc.FinalDemandType_LocalConsumption:
		return LocalConsumption
	case eieiorpc.FinalDemandType_LocalStructures:
		return LocalStructures
	case eieiorpc.FinalDemandType_LocalEquipment:
		return LocalEquipment
	case eieiorpc.FinalDemandType_LocalIP:
		return LocalIP
	case eieiorpc.FinalDemandType_AllDemand:
		return AllDemand
	case eieiorpc.FinalDemandType_NonExport:
		return NonExport
	default:
		panic(fmt.Errorf("invalid demand type %v", in))
	}
}

// loadFinalDemand reads in the available types of final demand from the
// given Excel file, setting all negative numbers to zero. This is done
// because negative cash flows do not have physical significance in the
// way that positive cash flows do. If year != detailYear, the detailed
// demand for detailYear is adjusted to year using the summary demand.
// If imports is true, import final demand will be returned rather than
// total final demand.
func (e *EIO) loadFinalDemand(detailFileName, summaryFileName string, year, detailYear Year, imports bool) (map[FinalDemand]*mat.VecDense, error) {
	const detailCodeRow, summaryCodeRow = 5, 5
	const detailStartCol, detailEndCol = 392, 412
	var summaryStartCol, summaryEndCol = 76, 96
	const detailStartRow, detailEndRow = 6, 395
	var summaryStartRow, summaryEndRow = 7, 80
	if imports {
		summaryStartCol, summaryEndCol = 74, 94
		summaryStartRow, summaryEndRow = 6, 79
	}

	detailSheet := fmt.Sprintf("%d", detailYear)
	summarySheet := fmt.Sprintf("%d", year)
	detailCodes, err := e.textRowFromExcel(detailFileName, detailSheet, detailCodeRow, detailStartCol, detailEndCol)
	if err != nil {
		return nil, fmt.Errorf("bea: reading detailed final demand types (imports==%v): %v", imports, err)
	}
	summaryCodes, err := e.textRowFromExcel(summaryFileName, summarySheet, summaryCodeRow, summaryStartCol, summaryEndCol)
	if err != nil {
		return nil, fmt.Errorf("bea: reading summary final demand types (imports==%v): %v", imports, err)
	}
	detailIndex := indexLookup(detailCodes)
	summaryIndex := indexLookup(summaryCodes)

	detailRowCodes, err := e.textColumnFromExcel(detailFileName, detailSheet, 0, detailStartRow, detailEndRow)
	if err != nil {
		return nil, fmt.Errorf("bea: reading detailed final demand rows: %v", err)
	}
	summaryRowCodes, err := e.textColumnFromExcel(summaryFileName, summarySheet, 0, summaryStartRow, summaryEndRow)
	if err != nil {
		return nil, fmt.Errorf("bea: reading summary final demand rows: %v", err)
	}
	codeCrosswalk, err := e.codeCrosswalk(summaryFileName)
	if err != nil {
		return nil, fmt.Errorf("bea: reading code crosswalk: %v", err)
	}

	demands := []FinalDemand{
		PersonalConsumption, PrivateStructures,
		PrivateEquipment, PrivateIP, PrivateResidential,
		InventoryChange, Export, DefenseConsumption, DefenseStructures,
		DefenseEquipment, DefenseIP, NondefenseConsumption, NondefenseStructures,
		NondefenseEquipment, NondefenseIP, LocalConsumption, LocalStructures,
		LocalEquipment, LocalIP,
	}

	o := make(map[FinalDemand]*mat.VecDense)

	for _, demand := range demands {
		detailCol, ok := detailIndex[string(demand)+"00"]
		if !ok {
			return nil, fmt.Errorf("bea: reading detailed final demand data: missing code %s", demand+"00")
		}
		detailCol += detailStartCol
		detailDemand, err := e.matrixFromExcel(detailFileName, detailSheet, detailStartRow, detailEndRow, detailCol, detailCol+1)
		if err != nil {
			return nil, fmt.Errorf("bea: reading detailed final demand data: %v", err)
		}
		detailDemandVec := detailDemand.ColView(0).(*mat.VecDense)
		const demandMultiplier = 1.0e6 // Demand in the spreadsheet is in millions of dollars
		detailDemandVec.ScaleVec(demandMultiplier, detailDemandVec)
		if year == detailYear {
			// We have the right year, so we don't need to do any adjustment.
			o[demand] = detailDemandVec
			continue
		}
		// Adjust the detailed data using summary data.
		summaryCol, ok := summaryIndex[string(demand)]
		if !ok {
			return nil, fmt.Errorf("bea: reading summary final demand: missing code %s", demand)
		}
		summaryCol += summaryStartCol
		summaryDemandYear, err := e.matrixFromExcel(summaryFileName, summarySheet, summaryStartRow, summaryEndRow, summaryCol, summaryCol+1)
		if err != nil {
			return nil, fmt.Errorf("bea: reading summary final demand year %d (imports==%v): %v", year, imports, err)
		}
		summaryDemandDetailYear, err := e.matrixFromExcel(summaryFileName, detailSheet, summaryStartRow, summaryEndRow, summaryCol, summaryCol+1)
		if err != nil {
			return nil, fmt.Errorf("bea: reading summary final demand year %d: %v", detailYear, err)
		}
		ratio := new(mat.Dense)
		ratio.Apply(func(i int, j int, v float64) float64 {
			detail := summaryDemandDetailYear.At(i, j)
			if detail != 0 {
				return v / detail
			}
			return 0
		}, summaryDemandYear)

		// Expand the vetor, using dummy codes for the columns.
		ratioExpanded := expandMatrix(ratio, summaryRowCodes, []string{"211"}, detailRowCodes, []string{"211000"}, codeCrosswalk)
		detailDemandVec.MulElemVec(detailDemandVec, ratioExpanded.ColView(0).(*mat.VecDense))
		o[demand] = detailDemandVec
	}

	// Set negative demand to zero.
	for _, v := range o {
		for i := 0; i < v.Len(); i++ {
			if (!imports && v.At(i, 0) < 0) || (imports && v.At(i, 0) > 0) {
				v.SetVec(i, 0)
			}
		}
	}

	// Add in aggregated demand groups.
	aggregatedDemands := []FinalDemand{All, NonExport}
	EndUseGroups := [][]FinalDemand{
		[]FinalDemand{
			PersonalConsumption, PrivateStructures,
			PrivateEquipment, PrivateIP, PrivateResidential,
			InventoryChange, Export, DefenseConsumption, DefenseStructures,
			DefenseEquipment, DefenseIP, NondefenseConsumption, NondefenseStructures,
			NondefenseEquipment, NondefenseIP, LocalConsumption, LocalStructures,
			LocalEquipment, LocalIP,
		},
		[]FinalDemand{
			PersonalConsumption, PrivateStructures,
			PrivateEquipment, PrivateIP, PrivateResidential,
			InventoryChange, DefenseConsumption, DefenseStructures,
			DefenseEquipment, DefenseIP, NondefenseConsumption, NondefenseStructures,
			NondefenseEquipment, NondefenseIP, LocalConsumption, LocalStructures,
			LocalEquipment, LocalIP,
		},
	}
	for i, d := range aggregatedDemands {
		r, c := o[PersonalConsumption].Dims()
		v := mat.NewDense(r, c, nil)
		for _, dd := range EndUseGroups[i] {
			v.Add(v, o[dd])
		}
		o[d] = v.ColView(0).(*mat.VecDense)
	}

	return o, nil
}
