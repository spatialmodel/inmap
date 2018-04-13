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

	"gonum.org/v1/gonum/mat"
)

// totalRequirementsSummary reads a total requirements matrix at
// the summary level of detail from an Excel file.
func (e *EIO) totalRequirementsSummary(fileName string, year Year) (*mat.Dense, error) {
	const (
		startRow, endRow = 7, 78
		startCol, endCol = 2, 75
	)
	return e.matrixFromExcel(fileName, fmt.Sprintf("%d", year), startRow, endRow, startCol, endCol)
}

// totalRequirementsDetail reads a total requirements matrix at
// the detailed level of detail from an Excel file.
func (e *EIO) totalRequirementsDetail(fileName string, year Year) (*mat.Dense, error) {
	const (
		startRow, endRow = 5, 394
		startCol, endCol = 2, 391
	)
	return e.matrixFromExcel(fileName, fmt.Sprintf("%d", year), startRow, endRow, startCol, endCol)
}

// totalRequirementsAdjusted returns a detailed total requirements matrix
// that is adjusted from detailYear matrix in detailFileName
//  to the desired year based on the
// summary-level matrices in summaryFileName.
func (e *EIO) totalRequirementsAdjusted(detailFileName, summaryFileName string, year, detailYear Year) (*mat.Dense, error) {
	detail, err := e.totalRequirementsDetail(detailFileName, detailYear)
	if err != nil {
		return nil, err
	}
	if year == detailYear {
		return detail, nil
	}
	summaryYear, err := e.totalRequirementsSummary(summaryFileName, year)
	if err != nil {
		return nil, err
	}
	summaryDetailYear, err := e.totalRequirementsSummary(summaryFileName, detailYear)
	if err != nil {
		return nil, err
	}
	// Calculate the ratio of the requested year to the detail year.
	ratio := new(mat.Dense)
	ratio.Apply(func(i int, j int, v float64) float64 {
		detail := summaryDetailYear.At(i, j)
		if detail != 0 {
			return v / detail
		}
		return 0
	}, summaryYear)

	summaryIndustries, _, err := e.industriesSummary(summaryFileName)
	if err != nil {
		return nil, err
	}
	summaryCommodities, _, err := e.commoditiesSummary(summaryFileName)
	if err != nil {
		return nil, err
	}
	detailIndustries, _, err := e.industriesDetail(detailFileName)
	if err != nil {
		return nil, err
	}
	detailCommodities, _, err := e.commoditiesDetail(detailFileName)
	if err != nil {
		return nil, err
	}
	codeCrosswalk, err := e.codeCrosswalk(summaryFileName)
	if err != nil {
		return nil, err
	}
	// Expand the summary-level ratio to match the dimensions of the detail-level
	// matrix.
	expandedRatio := expandMatrix(ratio, summaryIndustries, summaryCommodities, detailIndustries, detailCommodities, codeCrosswalk)
	// Multiply the detail-level matrix by the ratio
	detail.MulElem(detail, expandedRatio)
	return detail, nil
}
