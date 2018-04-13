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
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/spatialmodel/inmap/emissions/aep"
	"gonum.org/v1/gonum/mat"
)

// Scale applies scaling factors to the given emissions records.
func Scale(emis map[string][]aep.Record, f ScaleFunc) error {
	for _, recs := range emis {
		for _, rec := range recs {
			f2 := func(p aep.Pollutant) (float64, error) {
				return f(rec, p)
			}
			if err := rec.GetEmissions().Scale(f2); err != nil {
				return err
			}
		}
	}
	return nil
}

// ScaleFunc returns an emissions scaling factor for the given pollutant
// in the given record.
type ScaleFunc func(aep.Record, aep.Pollutant) (float64, error)

// ScaleNEIStateTrends provides an emissions scaling function to scale
// NEI emissions from baseYear to the specified scaleYear using EPA
// emissions summaries by year, state, SCC code, and pollutant available
// from https://www.epa.gov/sites/production/files/2016-12/state_tier1_90-16.xls.
// The "xls" file must be converted to an "xlsx" file before opening.
func ScaleNEIStateTrends(summaryFile string, sccDescriptions io.Reader, baseYear, scaleYear int) (ScaleFunc, error) {
	if baseYear == scaleYear {
		return func(rec aep.Record, pol aep.Pollutant) (float64, error) { return 1, nil }, nil
	}
	const (
		sheetName                         = "state_trends"
		dataRowStart, dataRowEnd          = 2, 5323
		dataColStart, dataColEnd          = 5, 27
		stateFIPSCol, sccTier1Col, polCol = 0, 2, 4
	)
	sccRef, err := sccToTier1(sccDescriptions)
	if err != nil {
		return nil, err
	}
	data, err := matrixFromExcel(summaryFile, sheetName, dataRowStart, dataRowEnd, dataColStart, dataColEnd)
	if err != nil {
		return nil, err
	}
	yearStrings, err := textRowFromExcel(summaryFile, sheetName, dataRowStart-1, dataColStart, dataColEnd)
	if err != nil {
		return nil, err
	}
	years, err := yearStringsToYears(yearStrings)
	if err != nil {
		return nil, err
	}
	var baseData mat.Vector
	for i, year := range years {
		if year == baseYear {
			baseData = data.ColView(i)
		}
	}
	if baseData == nil {
		return nil, fmt.Errorf("aeputil.ScaleNEIStateTrends: invalid base year %d", baseYear)
	}
	var yearData mat.Vector
	for i, year := range years {
		if year == scaleYear {
			yearData = data.ColView(i)
		}
	}
	if yearData == nil {
		return nil, fmt.Errorf("aeputil.ScaleNEIStateTrends: invalid scale year %d", scaleYear)
	}
	// Calculate scaling factors.
	scaleData := mat.NewVecDense(baseData.Len(), nil)
	for i := 0; i < baseData.Len(); i++ { // Set NaN and Inf to 0.
		b, y := baseData.At(i, 0), yearData.At(i, 0)
		v := y / b
		if math.IsInf(v, 0) || math.IsNaN(v) {
			scaleData.SetVec(i, 0)
		} else {
			scaleData.SetVec(i, v)
		}
	}

	stateFIPS, err := textColumnFromExcel(summaryFile, sheetName, stateFIPSCol, dataRowStart, dataRowEnd)
	if err != nil {
		return nil, err
	}
	sccTier1, err := textColumnFromExcel(summaryFile, sheetName, sccTier1Col, dataRowStart, dataRowEnd)
	if err != nil {
		return nil, err
	}
	pols, err := textColumnFromExcel(summaryFile, sheetName, polCol, dataRowStart, dataRowEnd)
	if err != nil {
		return nil, err
	}

	type item struct {
		stateFIPS, sccTier1, pol string
	}
	rows := make(map[item]int)
	for i := range stateFIPS {
		if len(sccTier1[i]) == 1 {
			sccTier1[i] = "0" + sccTier1[i]
		}
		rows[item{stateFIPS: stateFIPS[i], sccTier1: sccTier1[i], pol: pols[i]}] = i
	}

	return func(rec aep.Record, pol aep.Pollutant) (float64, error) {
		if rec.GetCountry() != aep.USA {
			return 1, nil // We only scale US emissions.
		}
		stateFIPS := rec.GetFIPS()[0:2]
		scc := rec.GetSCC()
		tier1, ok := sccRef[scc]
		if !ok {
			return math.NaN(), fmt.Errorf("aeputil.ScaleNEIStateTrends: no tier 1 code for SCC %s", scc)
		}
		if stateFIPS == "72" || stateFIPS == "78" {
			// No scaling for Puerto Rico or Virgin Islands.
			return 1, nil
		} else if stateFIPS == "88" || stateFIPS == "85" || stateFIPS == "98" {
			// It's not clear what these state codes represent.
			// Maybe shipping lanes? Replace them with California.
			stateFIPS = "06"
		}

		key := item{stateFIPS: stateFIPS, sccTier1: tier1, pol: scalingPol(pol)}
		row, ok := rows[key]
		if !ok {
			fmt.Printf("aeputil.ScaleNEIStateTrends: no scaling factor for key %+v\n", key)
			return 1, nil
		}
		scale := scaleData.At(row, 0)
		return scale, nil
	}, nil
}

// yearStringsToYears converts header strings in the form of "emissionsXX"
// where "XX" is a 2-digit year to 4-digit year integers.
func yearStringsToYears(yearStrings []string) ([]int, error) {
	years := make([]int, len(yearStrings))
	for i, ys := range yearStrings {
		year, err := strconv.ParseInt(strings.TrimLeft(ys, "emissions"), 10, 64)
		if err != nil {
			return nil, err
		}
		// Convert 2-digit year to 4 digits.
		if year > 50 {
			year += 1900
		} else {
			year += 2000
		}
		years[i] = int(year)
	}
	return years, nil
}

// scalingPol returns the scaling factor pollutant name that
// corresponds to the input
func scalingPol(pol aep.Pollutant) string {
	switch pol.Name {
	case "PM2_5", "PM25-PRI", "OC", "EC", "PMFINE", "SO4", "NO3", "PTI", "PSO4", "PSI", "POC", "PNO3", "PNH4",
		"PMOTHR", "PMN", "PMG", "PM25", "PK", "PFE", "PEC", "PCL", "PCA", "DIESEL-PM25", "PAL":
		return "PM25"
	case "XYL", "UNR", "TOL", "TERP", "PAR", "OLE", "NVOL", "MEOH", "ISOP", "IOLE", "FORM",
		"ETOH", "ETHA", "ETH", "VOC_INV", "NMOG", "NHTOG", "ETHANOL", "CB05_XYL", "CB05_TOL",
		"CB05_PAR", "CB05_OLE", "CB05_MEOH", "CB05_ISOP", "CB05_IOLE", "CB05_FORM",
		"CB05_ETOH", "CB05_ETHA", "CB05_ETH", "CB05_BENZENE", "CB05_ALDX", "CB05_ALD2",
		"ALDX", "ALD2":
		return "VOC"
	case "NO2", "NO", "HONO":
		return "NOX"
	default:
		return pol.Name
	}
}

// sccToTier1 reads in a crosswalk between SCC codes and the tier 1
// codes in the EPA summaries based on the information from:
// https://ofmpub.epa.gov/sccsearch/. The format of the output is
// map[SCC]tier1.
func sccToTier1(f io.Reader) (map[string]string, error) {
	r := csv.NewReader(f)
	lines, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	out := make(map[string]string)
	for i, line := range lines {
		if i == 0 {
			continue // Skip header.
		}
		scc := line[0]
		if len(scc) == 8 {
			scc = "00" + scc
		}
		if len(scc) != 10 {
			return nil, fmt.Errorf("aeputil.sccToTier1: invalid SCC code %s", scc)
		}
		tier1 := line[16]
		if len(tier1) == 1 {
			tier1 = "0" + tier1
		}
		out[scc] = tier1
	}
	return out, nil
}
