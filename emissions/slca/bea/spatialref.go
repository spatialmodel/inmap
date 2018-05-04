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

	"github.com/spatialmodel/inmap/emissions/slca"
	"gonum.org/v1/gonum/mat"
)

// loadSCCMap loads the mapping between IO industry sectors and SCC codes.
func (s *SpatialEIO) loadSCCMap(sccMapFile string) error {
	f, err := s.loadExcelFile(sccMapFile)
	if err != nil {
		return fmt.Errorf("bea: loading SCC map: %v", err)
	}
	sheet := f.Sheets[0]

	s.SCCs = make([]slca.SCC, len(sheet.Rows)-1)
	s.sccMap = make([][]int, len(s.SCCs))
	s.SpatialRefs = make([]slca.SpatialRef, len(s.SCCs))
	for i := 0; i < len(s.SCCs); i++ {
		r := sheet.Rows[i+1]
		s.SCCs[i] = slca.SCC(r.Cells[0].String())

		s.SpatialRefs[i] = slca.SpatialRef{
			SCCs:            []slca.SCC{s.SCCs[i]},
			EmisYear:        -9,
			Type:            slca.Stationary,
			NoNormalization: true,
		}

		for j := 2; j < len(r.Cells); j++ { // Skip first two columns.
			industry := r.Cells[j].String()
			ioRow, err := s.IndustryIndex(industry)
			if err != nil {
				return fmt.Errorf("bea: loading SCC map: %v", err)
			}
			s.sccMap[i] = append(s.sccMap[i], ioRow)
		}
	}
	return nil
}

// requirementsSCC returns a detailed requirements matrix
// that is mapped from the IO industries in the input matrix
// to individual SCC codes.
// The resulting SCC requirements are the sum of the requirements for
// all of the IO industries that are mapped to each SCC.
func (s *SpatialEIO) requirementsSCC(ioR *mat.Dense) (*mat.Dense, error) {
	_, cols := ioR.Dims()
	rows := len(s.SCCs)
	m := mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for _, ioRow := range s.sccMap[i] {
			for c := 0; c < cols; c++ {
				m.Set(i, c, m.At(i, c)+ioR.At(ioRow, c))
			}
		}
	}
	return m, nil
}

// SCCDescription returns the description of the SCC code at index
// i of the emitting sectors.
func (s *SpatialEIO) SCCDescription(i int) (string, error) {
	SCC := s.SCCs[i]
	desc, ok := s.sccDescriptions[string(SCC)]
	if !ok {
		return "", fmt.Errorf("missing description for SCC %s at index %d", SCC, i)
	}
	return desc, nil
}
