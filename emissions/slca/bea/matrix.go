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
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"github.com/ctessum/requestcache"
	"github.com/tealeg/xlsx"
	"gonum.org/v1/gonum/mat"
)

// loadExcelFile loads an Microsoft Excel file from disk, utizilizing
// a cache to avoid loading the same file more than once.
func (e *EIO) loadExcelFile(fileName string) (*xlsx.File, error) {
	// Create a request cache to avoid loading files more than once.
	e.loadExcelCacheOnce.Do(func() {
		e.excelCache = requestcache.NewCache(func(ctx context.Context, req interface{}) (interface{}, error) {
			filename := req.(string)
			f, err := xlsx.OpenFile(filename)
			if err != nil {
				return nil, fmt.Errorf("bea: opening xlsx file: %v", err)
			}
			return f, nil
		}, runtime.GOMAXPROCS(-1), requestcache.Memory(1000))
	})
	// Get the file from the cache or generate it.
	r := e.excelCache.NewRequest(context.Background(), fileName, fileName)
	fI, err := r.Result()
	if err != nil {
		return nil, err
	}
	return fI.(*xlsx.File), nil
}

// matrixFromExcel creates a matrix from data in a Microsoft Excel file with the
// given fileName and sheet name within the file, based on the data starting
// at [startRow, startCol] (inclusive) and ending at [endRow, endCol] (exclusive).
func (e *EIO) matrixFromExcel(fileName, sheet string, startRow, endRow, startCol, endCol int) (*mat.Dense, error) {
	f, err := e.loadExcelFile(fileName)
	if err != nil {
		return nil, err
	}
	s, ok := f.Sheet[sheet]
	if !ok {
		return nil, fmt.Errorf("bea: reading matrix from Excel; no sheet %s", sheet)
	}

	o := mat.NewDense(endRow-startRow, endCol-startCol, nil)

	for j := startRow; j < endRow; j++ {
		for i := startCol; i < endCol; i++ {
			cellString := s.Cell(j, i).Value
			var v float64
			if !(cellString == "..." || cellString == "") { // v = 0 for these cell contents.
				v, err = strconv.ParseFloat(cellString, 64)
				if err != nil {
					return nil, fmt.Errorf("bea: reading matrix from Excel: %v", err)
				}
			}
			o.Set(j-startRow, i-startCol, v)
		}
	}
	return o, nil
}

// textColumnFromExcel returns a slice of strings extracted from the given column
// and row range in the given file and sheet.
func (e *EIO) textColumnFromExcel(fileName, sheet string, column, startRow, endRow int) ([]string, error) {
	f, err := e.loadExcelFile(fileName)
	if err != nil {
		return nil, err
	}
	s, ok := f.Sheet[sheet]
	if !ok {
		return nil, fmt.Errorf("bea: reading text column from Excel; no sheet %s", sheet)
	}

	o := make([]string, endRow-startRow)

	for j := startRow; j < endRow; j++ {
		o[j-startRow] = strings.TrimSpace(s.Cell(j, column).Value)
	}
	return o, nil
}

// textRowFromExcel returns a slice of strings extracted from the given column
// range and row in the given file and sheet.
func (e *EIO) textRowFromExcel(fileName, sheet string, row, startCol, endCol int) ([]string, error) {
	f, err := e.loadExcelFile(fileName)
	if err != nil {
		return nil, err
	}
	s, ok := f.Sheet[sheet]
	if !ok {
		return nil, fmt.Errorf("bea: reading text row from Excel; no sheet %s", sheet)
	}

	o := make([]string, endCol-startCol)

	for i := startCol; i < endCol; i++ {
		o[i-startCol] = strings.TrimSpace(s.Cell(row, i).Value)
	}
	return o, nil
}

// industriesSummary returns the industry codes and descriptions from the
// given summary detail-level Excel file,
// as well as a map in the format map[code]description.
func (e *EIO) industriesSummary(fileName string) (codes, descriptions []string, err error) {
	const startRow, endRow = 7, 78
	return e.industries(fileName, startRow, endRow)
}

// industriesDetail returns the industry codes and descriptions from the
// given detailed detail-level Excel file,
// as well as a map in the format map[code]description.
func (e *EIO) industriesDetail(fileName string) (codes, descriptions []string, err error) {
	const startRow, endRow = 5, 394
	return e.industries(fileName, startRow, endRow)
}

// industries returns the industry codes and descriptions from the
// given Excel file,
// as well as a map in the format map[code]description.
func (e *EIO) industries(fileName string, startRow, endRow int) (codes, descriptions []string, err error) {
	const codeCol, descCol = 0, 1
	codes, err = e.textColumnFromExcel(fileName, "2007", codeCol, startRow, endRow)
	if err != nil {
		return nil, nil, err
	}
	descriptions, err = e.textColumnFromExcel(fileName, "2007", descCol, startRow, endRow)
	if err != nil {
		return nil, nil, err
	}
	return codes, descriptions, nil
}

// commoditiesSummary returns the industry codes and descriptions from the
// given summary detail-level Excel file,
// as well as a map in the format map[code]description.
func (e *EIO) commoditiesSummary(fileName string) (codes, descriptions []string, err error) {
	const codeRow, descRow = 5, 6
	const startCol, endCol = 2, 75
	return e.commodities(fileName, codeRow, descRow, startCol, endCol)
}

// commoditiesDetail returns the industry codes and descriptions from the
// given detailed detail-level Excel file,
// as well as a map in the format map[code]description.
func (e *EIO) commoditiesDetail(fileName string) (codes, descriptions []string, err error) {
	const codeRow, descRow = 4, 3
	const startCol, endCol = 2, 391
	return e.commodities(fileName, codeRow, descRow, startCol, endCol)
}

// commodities returns the commodity codes and descriptions from the
// given Excel file,
// as well as a map in the format map[code]description.
func (e *EIO) commodities(fileName string, codeRow, descRow, startCol, endCol int) (codes, descriptions []string, err error) {
	codes, err = e.textRowFromExcel(fileName, "2007", codeRow, startCol, endCol)
	if err != nil {
		return nil, nil, err
	}
	descriptions, err = e.textRowFromExcel(fileName, "2007", descRow, startCol, endCol)
	if err != nil {
		return nil, nil, err
	}
	return codes, descriptions, nil
}

// codeCrosswalk reads in the crosswalk between the
// summary and detailed levels of sector detail.
// The return value is in the format map[summary code][]{detail codes}
func (e *EIO) codeCrosswalk(fileName string) (map[string][]string, error) {
	const (
		summaryCol, detailCol = 1, 2
		startRow, endRow      = 6, 649
	)
	summaryCodes, err := e.textColumnFromExcel(fileName, "NAICS codes", summaryCol, startRow, endRow)
	if err != nil {
		return nil, err
	}
	detailCodes, err := e.textColumnFromExcel(fileName, "NAICS codes", detailCol, startRow, endRow)
	if err != nil {
		return nil, err
	}
	o := make(map[string][]string)
	var currentSummaryCode string
	for i, s := range summaryCodes {
		if s != "" {
			currentSummaryCode = s
		}
		if d := detailCodes[i]; s == "" && d != "" {
			o[currentSummaryCode] = append(o[currentSummaryCode], d)
		}
	}
	return o, nil
}

// expandMatrix expands matrix m with the given rowCodes and colCodes to
// a matrix with the given expandedRowCodes and expandedColCodes,
// based on the given codeCrosswalk of the format map[code][]{expanded codes}.
func expandMatrix(m *mat.Dense, rowCodes, colCodes, expandedRowCodes, expandedColCodes []string, codeCrosswalk map[string][]string) *mat.Dense {
	o := mat.NewDense(len(expandedRowCodes), len(expandedColCodes), nil)

	// Create index lookups.
	rowIndices := indexLookup(expandedRowCodes)
	colIndices := indexLookup(expandedColCodes)

	var rowsFilled int

	for oldJ, rc := range rowCodes {
		newRs, ok := codeCrosswalk[rc]
		if !ok {
			panic(fmt.Errorf("crosswalk missing code %s", rc))
		}
		for _, newR := range newRs {
			j, ok := rowIndices[newR]
			if !ok {
				continue
			}
			rowsFilled++
			var colsFilled int
			for oldI, cc := range colCodes {
				newCs, ok := codeCrosswalk[cc]
				if !ok {
					panic(fmt.Errorf("crosswalk missing code %s", cc))
				}
				for _, newC := range newCs {
					i, ok := colIndices[newC]
					if !ok {
						continue
					}
					colsFilled++
					o.Set(j, i, m.At(oldJ, oldI))
				}
			}
			if colsFilled != len(expandedColCodes) {
				panic(fmt.Errorf("filled wrong number of columns: %d!=%d", colsFilled, len(expandedColCodes)))
			}
		}
	}
	if rowsFilled != len(expandedRowCodes) {
		panic(fmt.Errorf("filled wrong number of Rows: %d!=%d", rowsFilled, len(expandedRowCodes)))
	}
	return o
}

// indexLookup returns a map of the index number for each item in a.
func indexLookup(a []string) map[string]int {
	o := make(map[string]int)
	for i, s := range a {
		o[s] = i
	}
	return o
}
