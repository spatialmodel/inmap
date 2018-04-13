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
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package aeputil

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/ctessum/requestcache"
	"github.com/tealeg/xlsx"
	"gonum.org/v1/gonum/mat"
)

// excelCache holds previously opened Microsoft Excel files
// to avoid reading the same file multiple times.
var excelCache *requestcache.Cache

var loadExcelCacheOnce sync.Once

// loadExcelFile loads an Microsoft Excel file from disk, utizilizing
// a cache to avoid loading the same file more than once.
func loadExcelFile(fileName string) (*xlsx.File, error) {
	// Create a request cache to avoid loading files more than once.
	loadExcelCacheOnce.Do(func() {
		excelCache = requestcache.NewCache(func(ctx context.Context, req interface{}) (interface{}, error) {
			filename := req.(string)
			f, err := xlsx.OpenFile(filename)
			if err != nil {
				return nil, fmt.Errorf("aeputil: opening xlsx file: %v", err)
			}
			return f, nil
		}, runtime.GOMAXPROCS(-1), requestcache.Memory(1000))
	})
	// Get the file from the cache or generate it.
	r := excelCache.NewRequest(context.Background(), fileName, fileName)
	fI, err := r.Result()
	if err != nil {
		return nil, err
	}
	return fI.(*xlsx.File), nil
}

// matrixFromExcel creates a matrix from data in a Microsoft Excel file with the
// given fileName and sheet name within the file, based on the data starting
// at [startRow, startCol] (inclusive) and ending at [endRow, endCol] (exclusive).
func matrixFromExcel(fileName, sheet string, startRow, endRow, startCol, endCol int) (*mat.Dense, error) {
	f, err := loadExcelFile(fileName)
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
func textColumnFromExcel(fileName, sheet string, column, startRow, endRow int) ([]string, error) {
	f, err := loadExcelFile(fileName)
	if err != nil {
		return nil, err
	}
	s, ok := f.Sheet[sheet]
	if !ok {
		return nil, fmt.Errorf("aeputil: reading text column from Excel; no sheet %s", sheet)
	}

	o := make([]string, endRow-startRow)

	for j := startRow; j < endRow; j++ {
		o[j-startRow] = strings.TrimSpace(s.Cell(j, column).Value)
	}
	return o, nil
}

// textRowFromExcel returns a slice of strings extracted from the given column
// range and row in the given file and sheet.
func textRowFromExcel(fileName, sheet string, row, startCol, endCol int) ([]string, error) {
	f, err := loadExcelFile(fileName)
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
