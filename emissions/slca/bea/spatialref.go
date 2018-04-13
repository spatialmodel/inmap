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
	"sort"

	"github.com/spatialmodel/inmap/emissions/slca"
)

// NEISpatialRefs returns spatial references corresponding to the detailed EIO
// sectors based on information in the US National Emissions Inventory,
// where filename represents a spreadsheet containing information
// regarding which SCC codes correspond to which economic industry.
//
// In cases where an SCC corresponds to more than one industry, emissions
// are allocated among industries according to the economic production
// in each industry.
func NEISpatialRefs(filename string, year Year, eio *EIO) ([]*slca.SpatialRef, error) {
	f, err := eio.loadExcelFile(filename)
	if err != nil {
		return nil, err
	}
	s, ok := f.Sheet["Sheet1"]
	if !ok {
		return nil, fmt.Errorf("bea.NEISpatialRefs: excel file sheet 'Sheet1' is missing")
	}
	if len(s.Rows)-1 != len(eio.Industries) {
		return nil, fmt.Errorf("bea.NEISpatialRefs: invalid number of sectors in file: %d != %d", len(s.Rows)-1, len(eio.Industries))
	}

	production, err := eio.domesticProduction(year)
	if err != nil {
		return nil, err
	}
	spatialRefs := make([]*slca.SpatialRef, len(eio.Industries))
	sccFractions := make(map[slca.SCC]map[int]float64)

	for i := 1; i < len(s.Rows); i++ {
		r := s.Rows[i]
		if industry := r.Cells[1].Value; industry != eio.Industries[i-1] {
			return nil, fmt.Errorf("bea.NEISpatialRefs: invalid industry order: %s != %s", industry, eio.Industries[i-1])
		}
		spatialRef := &slca.SpatialRef{
			SCCs:            make([]slca.SCC, 0, len(r.Cells)-3),
			EmisYear:        int(year),
			Type:            slca.Stationary,
			NoNormalization: true,
		}
		for j := 3; j < len(r.Cells); j++ {
			scc := slca.SCC(r.Cells[j].Value)
			if scc == "" {
				continue
			}
			if len(scc) == 9 {
				scc += "0"
			}
			if len(scc) == 8 {
				scc = "00" + scc
			}
			if len(scc) != 10 {
				return nil, fmt.Errorf("bea.NEISpatialRefs: invalid SCC code '%s'", scc)
			}
			spatialRef.SCCs = append(spatialRef.SCCs, scc)

			if _, ok := sccFractions[scc]; !ok {
				sccFractions[scc] = make(map[int]float64)
			}
			sccFractions[scc][i-1] = production.At(i-1, 0)
		}
		spatialRefs[i-1] = spatialRef
	}

	// Make sure we loop through the SCCs in the same order
	// every time to avoid rounding differences in the fractions.
	sccs := make([]string, 0, len(sccFractions))
	for scc := range sccFractions {
		sccs = append(sccs, string(scc))
	}
	sort.Strings(sccs)

	// Normalize the SCC fractions.
	for _, scc := range sccs {
		sectors := sccFractions[slca.SCC(scc)]

		// Make sure we loop through the sectors in the same order
		// every time to avoid rounding differnces in the fractions.
		iSectors := make([]int, 0, len(sectors))
		for i := range sectors {
			iSectors = append(iSectors, i)
		}
		sort.Ints(iSectors)

		var total float64
		for _, i := range iSectors {
			total += sectors[i]
		}
		for _, i := range iSectors {
			sccFractions[slca.SCC(scc)][i] /= total
		}
	}

	for i, sr := range spatialRefs {
		sr.SCCFractions = make([]float64, len(sr.SCCs))
		for j, scc := range sr.SCCs {
			var ok bool
			sr.SCCFractions[j], ok = sccFractions[scc][i]
			if !ok {
				panic("missing SCC fraction")
			}
		}
	}
	return spatialRefs, nil
}
