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

package inmap

import (
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ctessum/atmos/seinfeld"
	"github.com/ctessum/atmos/wesely1989"
	"github.com/ctessum/cdf"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/index/rtree"

	"github.com/ctessum/sparse"
)

const (
	geosFormat     = "20060102"
	geosChemFormat = "20060102.150000"
)

// GEOSChem is an InMAP preprocessor for GEOS-Chem output.
// Information regarding GEOS-Chem output variables is available from
// http://wiki.seas.harvard.edu/geos-chem/index.php/List_of_GEOS-FP_met_fields
// and
// http://wiki.seas.harvard.edu/geos-chem/index.php/Species_in_GEOS-Chem.
type GEOSChem struct {
	aVOC, bVOC, aSOA, bSOA, nox, no, no2, pNO, sox, pS, nh3, pNH, totalPM25 map[string]float64

	noChemHour bool

	start, end time.Time

	chemRecordDeltaInterval, chemFileDeltaInterval time.Duration

	recordDelta1h, recordDelta3h time.Duration
	fileDelta24h, fileDelta3h    time.Duration

	landUse *sparse.DenseArray

	nz int

	dx, dy float64

	xCenters, yCenters []float64

	geosA1     string
	geosA3Cld  string
	geosA3Dyn  string
	geosI3     string
	geosA3MstE string
	geosApBp   string
	geosChem   string

	// If dash is '-', GEOS-Chem chemical variable names are assumed to be in the
	// form 'IJ-AVG-S__xxx'. If dash is '_', they are assumed to be in the form
	// 'IJ_AVG_S_xxx'.
	dash string

	msgChan chan string
}

// NewGEOSChem initializes a GEOS-Chem preprocessor from the given
// configuration information.
//
// GEOSA1 is the location of the GEOS 1-hour time average files.
// [DATE] should be used as a wild card for the simulation date.
//
// GEOSA3Cld is the location of the GEOS 3-hour average cloud
// parameter files. [DATE] should be used as a wild card for
// the simulation date.
//
// GEOSA3Cld is the location of the GEOS 3-hour average dynamical
// parameter files. [DATE] should be used as a wild card for
// the simulation date.
//
// GEOSI3 is the location of the GEOS 3-hour instantaneous parameter
// files. [DATE] should be used as a wild card for
// the simulation date.
//
// GEOSA3MstE is the location of the GEOS 3-hour average moist parameters
// on level edges files. [DATE] should be used as a wild card for
// the simulation date.
//
// geosApBp is the location of the pressure level variable file.
// It is optional; if it is not specified the Ap and Bp information
// will be extracted from the geosChem file.
//
// GEOSChemOut is the location of GEOS-Chem output files.
// [DATE] should be used as a wild card for the simulation date.
//
// OlsonLandMap is the location of the GEOS-Chem Olson land use map file,
// which is described here:
// http://wiki.seas.harvard.edu/geos-chem/index.php/Olson_land_map
//
// startDate and endDate are the dates of the beginning and end of the
// simulation, respectively, in the format "YYYYMMDD".
//
// If dash is true, GEOS-Chem chemical variable names are assumed to be in the
// form 'IJ-AVG-S__xxx'. If dash is false, they are assumed to be in the form
// 'IJ_AVG_S_xxx'.
//
// If msgChan is not nil, status messages will be sent to it.
//
// chemRecordInterval is the time interval between different records in
// the GEOS-Chem output. It is specified by the user as a string
// (chemRecordStr), e.g. "3h" for 3 hours.
//
// chemFileInterval is the time interval of each file. It is specified
// as a string (chemFileStr), e.g. "3h" for 3 hours.
//
// If noChemHour is true, then the GEOS-Chem output files will be
// assumed to not contain a time dimension.
func NewGEOSChem(GEOSA1, GEOSA3Cld, GEOSA3Dyn, GEOSI3, GEOSA3MstE, GEOSApBp, GEOSChemOut, OlsonLandMap, startDate, endDate string, dash bool, chemRecordStr, chemFileStr string, noChemHour bool, msgChan chan string) (*GEOSChem, error) {
	var d string
	if dash {
		d = "-"
	} else {
		d = "_"
	}
	gc := GEOSChem{
		// These maps contain the GEOS-Chem variables that make
		// up the chemical species groups, as well as the
		// multiplication factors required to convert concentrations
		// to mass fractions [μg/kg dry air].

		// GEOS-Chem VOC species;
		// Only includes anthropogenic precursors to SOA from
		// anthropogenic (aSOA) and biogenic (bSOA) sources.
		// Additional information available from:
		// http://wiki.seas.harvard.edu/geos-chem/index.php/Species_in_GEOS-Chem.
		// We assume condensable vapor from SOA has molar mass of 70.
		aVOC: map[string]float64{
			"IJ" + d + "AVG" + d + "S__BENZ": ppbcToUgKg(78.11, 6),
			"IJ" + d + "AVG" + d + "S__TOLU": ppbcToUgKg(92.14, 7),
			"IJ" + d + "AVG" + d + "S__XYLE": ppbcToUgKg(106.16, 8),
			"IJ" + d + "AVG" + d + "S__NAP":  ppbcToUgKg(128.1705, 10),
			"IJ" + d + "AVG" + d + "S__POG1": ppbvToUgKg(12),
			"IJ" + d + "AVG" + d + "S__POG2": ppbvToUgKg(12),
		},
		bVOC: map[string]float64{
			"IJ" + d + "AVG" + d + "S__ISOP": ppbcToUgKg(68.12, 5),
			"IJ" + d + "AVG" + d + "S__LIMO": ppbvToUgKg(136.23),
			"IJ" + d + "AVG" + d + "S__MTPA": ppbvToUgKg(136.23),
			"IJ" + d + "AVG" + d + "S__MTPO": ppbvToUgKg(136.23),
		},
		// SOA species (anthropogenic only)
		aSOA: map[string]float64{
			"IJ" + d + "AVG" + d + "S__ASOA1": ppbvToUgKg(150),
			"IJ" + d + "AVG" + d + "S__ASOA2": ppbvToUgKg(150),
			"IJ" + d + "AVG" + d + "S__ASOA3": ppbvToUgKg(150),
			"IJ" + d + "AVG" + d + "S__ASOAN": ppbvToUgKg(150),
		},
		// SOA species (biogenic only)
		bSOA: map[string]float64{
			"IJ" + d + "AVG" + d + "S__TSOA0":  ppbvToUgKg(150),
			"IJ" + d + "AVG" + d + "S__TSOA1":  ppbvToUgKg(150),
			"IJ" + d + "AVG" + d + "S__TSOA2":  ppbvToUgKg(150),
			"IJ" + d + "AVG" + d + "S__TSOA3":  ppbvToUgKg(150),
			"IJ" + d + "AVG" + d + "S__SOAGX":  ppbvToUgKg(58),
			"IJ" + d + "AVG" + d + "S__SOAMG":  ppbvToUgKg(72),
			"IJ" + d + "AVG" + d + "S__SOAIE":  ppbvToUgKg(118),
			"IJ" + d + "AVG" + d + "S__SOAME":  ppbvToUgKg(102),
			"IJ" + d + "AVG" + d + "S__LVOCOA": ppbvToUgKg(154),
			"IJ" + d + "AVG" + d + "S__ISN1OA": ppbvToUgKg(226),
		},
		// NOx species. We are only interested in the mass
		// of Nitrogen, rather than the mass of the whole molecule, so
		// we use the molecular weight of Nitrogen.
		nox: map[string]float64{
			"IJ" + d + "AVG" + d + "S__NO":  ppbvToUgKg(mwN),
			"IJ" + d + "AVG" + d + "S__NO2": ppbvToUgKg(mwN),
		},
		// pNO is the Nitrogen fraction of the particulate
		// NO species.
		pNO: map[string]float64{
			"IJ" + d + "AVG" + d + "S__NIT":  ppbvToUgKg(mwN),
			"IJ" + d + "AVG" + d + "S__NITs": ppbvToUgKg(mwN),
		},
		// SOx species. We are only interested in the mass
		// of Sulfur, rather than the mass of the whole molecule, so
		// we use the molecular weight of Sulfur.
		sox: map[string]float64{
			"IJ" + d + "AVG" + d + "S__SO2": ppbvToUgKg(mwS),
		},
		// pS is the MADE particulate Sulfur species; sulfur fraction
		// sulfate (SO4) plus sulfate on the surface of sea ice (SO4s).
		pS: map[string]float64{
			"IJ" + d + "AVG" + d + "S__SO4":  ppbvToUgKg(mwS),
			"IJ" + d + "AVG" + d + "S__SO4s": ppbvToUgKg(mwS),
			"IJ" + d + "AVG" + d + "S__DMS":  ppbvToUgKg(mwS),
		},
		// NH3 is ammonia. We are only interested in the mass
		// of Nitrogen, rather than the mass of the whole molecule, so
		// we use the molecular weight of Nitrogen.
		nh3: map[string]float64{"IJ" + d + "AVG" + d + "S__NH3": ppbvToUgKg(mwN)},
		// pNH is the Nitrogen fraction of the particulate
		// ammonia species.
		pNH: map[string]float64{"IJ" + d + "AVG" + d + "S__NH4": ppbvToUgKg(mwN)},
		// totalPM25 is total mass of PM2.5.
		// It is calculated based on the formula at:
		// http://wiki.seas.harvard.edu/geos-chem/index.php/Particulate_matter_in_GEOS-Chem
		totalPM25: map[string]float64{
			"IJ" + d + "AVG" + d + "S__NH4":    ppbvToUgKg(18) * 1.33,
			"IJ" + d + "AVG" + d + "S__NIT":    ppbvToUgKg(62) * 1.33,
			"IJ" + d + "AVG" + d + "S__SO4":    ppbvToUgKg(96) * 1.33,
			"IJ" + d + "AVG" + d + "S__BCPI":   ppbvToUgKg(12),
			"IJ" + d + "AVG" + d + "S__BCPO":   ppbvToUgKg(12),
			"IJ" + d + "AVG" + d + "S__POA1":   ppbvToUgKg(12) * 1.4,
			"IJ" + d + "AVG" + d + "S__POA2":   ppbvToUgKg(12) * 1.4,
			"IJ" + d + "AVG" + d + "S__OPOA1":  ppbvToUgKg(12) * 2.1,
			"IJ" + d + "AVG" + d + "S__OPOA2":  ppbvToUgKg(12) * 2.1,
			"IJ" + d + "AVG" + d + "S__TSOA0":  ppbvToUgKg(150) * 1.16,
			"IJ" + d + "AVG" + d + "S__TSOA1":  ppbvToUgKg(150) * 1.16,
			"IJ" + d + "AVG" + d + "S__TSOA2":  ppbvToUgKg(150) * 1.16,
			"IJ" + d + "AVG" + d + "S__TSOA3":  ppbvToUgKg(150) * 1.16,
			"IJ" + d + "AVG" + d + "S__ASOAN":  ppbvToUgKg(150) * 1.16,
			"IJ" + d + "AVG" + d + "S__ASOA1":  ppbvToUgKg(150) * 1.16,
			"IJ" + d + "AVG" + d + "S__ASOA2":  ppbvToUgKg(150) * 1.16,
			"IJ" + d + "AVG" + d + "S__ASOA3":  ppbvToUgKg(150) * 1.16,
			"IJ" + d + "AVG" + d + "S__SOAGX":  ppbvToUgKg(58) * 1.16,
			"IJ" + d + "AVG" + d + "S__INDIOL": ppbvToUgKg(102) * 1.16,
			"IJ" + d + "AVG" + d + "S__SOAMG":  ppbvToUgKg(72) * 1.16,
			"IJ" + d + "AVG" + d + "S__SOAIE":  ppbvToUgKg(118) * 1.16,
			"IJ" + d + "AVG" + d + "S__SOAME":  ppbvToUgKg(102) * 1.16,
			"IJ" + d + "AVG" + d + "S__LVOCOA": ppbvToUgKg(154) * 1.16,
			"IJ" + d + "AVG" + d + "S__ISN1OA": ppbvToUgKg(226) * 1.16,
			"IJ" + d + "AVG" + d + "S__DST1":   ppbvToUgKg(29),
			"IJ" + d + "AVG" + d + "S__DST2":   ppbvToUgKg(29) * 0.38,
			"IJ" + d + "AVG" + d + "S__SALA":   ppbvToUgKg(31.4) * 1.86,
		},

		geosA1:     GEOSA1,
		geosA3Cld:  GEOSA3Cld,
		geosA3Dyn:  GEOSA3Dyn,
		geosI3:     GEOSI3,
		geosA3MstE: GEOSA3MstE,
		geosApBp:   GEOSApBp,
		geosChem:   GEOSChemOut,

		dash:       d,
		msgChan:    msgChan,
		noChemHour: noChemHour,
	}

	var err error
	gc.start, err = time.Parse(inDateFormat, startDate)
	if err != nil {
		return nil, fmt.Errorf("inmap: GEOS-Chem preprocessor start time: %v", err)
	}
	gc.end, err = time.Parse(inDateFormat, endDate)
	if err != nil {
		return nil, fmt.Errorf("inmap: GEOS-Chem preprocessor end time: %v", err)
	}

	if !gc.end.After(gc.start) {
		if err != nil {
			return nil, fmt.Errorf("inmap: GEOS-Chem preprocessor end time %v is not after start time %v", gc.end, gc.start)
		}
	}

	gc.chemRecordDeltaInterval, err = time.ParseDuration(chemRecordStr)
	if err != nil {
		return nil, fmt.Errorf("inmap: GEOS-Chem preprocessor recordDelta: %v", err)
	}
	gc.recordDelta1h, err = time.ParseDuration("1h")
	if err != nil {
		return nil, fmt.Errorf("inmap: GEOS-Chem preprocessor recordDelta: %v", err)
	}
	gc.recordDelta3h, err = time.ParseDuration("3h")
	if err != nil {
		return nil, fmt.Errorf("inmap: GEOS-Chem preprocessor recordDelta: %v", err)
	}
	gc.chemFileDeltaInterval, err = time.ParseDuration(chemFileStr)
	if err != nil {
		return nil, fmt.Errorf("inmap: GEOS-Chem preprocessor fileDelta: %v", err)
	}
	gc.fileDelta24h, err = time.ParseDuration("24h")
	if err != nil {
		return nil, fmt.Errorf("inmap: GEOS-Chem preprocessor fileDelta: %v", err)
	}
	gc.fileDelta3h, err = time.ParseDuration("3h")
	if err != nil {
		return nil, fmt.Errorf("inmap: GEOS-Chem preprocessor fileDelta: %v", err)
	}

	gc.nz, err = gc.Nz()
	if err != nil {
		return nil, err
	}
	gc.dx, err = gc.DX()
	if err != nil {
		return nil, err
	}
	gc.dy, err = gc.DY()
	if err != nil {
		return nil, err
	}
	gc.xCenters, err = gc.XCenters()
	if err != nil {
		return nil, err
	}
	gc.yCenters, err = gc.YCenters()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(OlsonLandMap)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	cfile, err := cdf.Open(file)
	if err != nil {
		return nil, fmt.Errorf("inmap: Olson land use file: %v", err)
	}
	gc.landUse, err = gc.largestLandUse(cfile)
	if err != nil {
		return nil, err
	}

	return &gc, nil
}

// ppbcToUgKg returns a multiplier to convert a concentration in
// ppbc (parts per billion carbon) dry air to a mass fraction
// [micrograms per kilogram dry air]
// for a chemical species with the given molecular weight
// (mw, g/mol) and number of carbons (nc).
func ppbcToUgKg(mw, nc float64) float64 {
	return mw / nc / MWa
}

// ppbvToUgKg returns a multiplier to convert a concentration in
// ppbv dry air to a mass fraction [micrograms per kilogram dry air]
// for a chemical species with the given molecular weight in g/mol.
func ppbvToUgKg(mw float64) float64 {
	return mw / MWa
}

func (gc *GEOSChem) readA3Dyn(varName string) NextData {
	conv := geosLayerConvert(gc.nz)
	return conv(nextDataNCF(gc.geosA3Dyn, geosFormat, varName, gc.start, gc.end, gc.recordDelta3h, gc.fileDelta24h, readNCF, gc.msgChan))
}

func (gc *GEOSChem) readA3MstE(varName string) NextData {
	conv := geosLayerConvert(gc.nz)
	return conv(nextDataNCF(gc.geosA3MstE, geosFormat, varName, gc.start, gc.end, gc.recordDelta3h, gc.fileDelta24h, readNCF, gc.msgChan))
}

func (gc *GEOSChem) readA3Cld(varName string) NextData {
	conv := geosLayerConvert(gc.nz)
	return conv(nextDataNCF(gc.geosA3Cld, geosFormat, varName, gc.start, gc.end, gc.recordDelta3h, gc.fileDelta24h, readNCF, gc.msgChan))
}

func (gc *GEOSChem) readA1(varName string) NextData {
	// All variables in A1 are 2-d, so we don't need to perform a layer conversion.
	return nextDataNCF(gc.geosA1, geosFormat, varName, gc.start, gc.end, gc.recordDelta1h, gc.fileDelta24h, readNCF, gc.msgChan)
}

func (gc *GEOSChem) readI3(varName string) NextData {
	conv := geosLayerConvert(gc.nz)
	return conv(nextDataNCF(gc.geosI3, geosFormat, varName, gc.start, gc.end, gc.recordDelta3h, gc.fileDelta24h, readNCF, gc.msgChan))
}

func (gc *GEOSChem) readChem(varName string) NextData {
	if gc.noChemHour {
		return nextDataNCF(gc.geosChem, geosChemFormat, varName, gc.start, gc.end, gc.chemRecordDeltaInterval, gc.chemFileDeltaInterval, readNCFNoHour, gc.msgChan)
	}
	return nextDataNCF(gc.geosChem, geosChemFormat, varName, gc.start, gc.end, gc.chemRecordDeltaInterval, gc.chemFileDeltaInterval, readNCF, gc.msgChan)
}

func (gc *GEOSChem) readApBp(varName string) NextData {
	if gc.geosApBp != "" {
		return nextDataConstantNCF(strings.ToLower(varName), gc.geosApBp)
	}
	return nextDataNCF(gc.geosChem, geosChemFormat, varName, gc.start, gc.end, gc.recordDelta3h, gc.fileDelta3h, readNCFNoHour, gc.msgChan)
}

func (gc *GEOSChem) readChemGroupAlt(varGroup map[string]float64) NextData {
	if gc.noChemHour {
		return nextDataGroupAltNCF(gc.geosChem, geosChemFormat, varGroup, gc.ALT(), gc.start, gc.end, gc.chemRecordDeltaInterval, gc.chemFileDeltaInterval, readNCFNoHour, gc.msgChan)
	}
	return nextDataGroupAltNCF(gc.geosChem, geosChemFormat, varGroup, gc.ALT(), gc.start, gc.end, gc.chemRecordDeltaInterval, gc.chemFileDeltaInterval, readNCF, gc.msgChan)
}

var geosLayerConvert = func(nz int) func(NextData) NextData {
	const (
		geosLayers          = 72
		geosChemShortLayers = 47
	)
	if nz == geosLayers {
		// If GEOS-Chem is using the full 72 layers, we don't need to perform a
		// conversion.
		return func(d NextData) NextData { return d }
	} else if nz != geosChemShortLayers {
		panic(fmt.Errorf("inmap preprocessor: invalid number of GEOS layers (%d); should be 72 or 47", nz))
	}

	// GEOS always has 72 unstaggered layers, but sometimes GEOS-Chem only uses 47.
	// layerMap is a mapping between the 72 and 47 layer versions for
	// unstaggered variables.
	var layerMap = map[int]int{
		0: 0, 1: 1, 2: 2, 3: 3, 4: 4, 5: 5, 6: 6, 7: 7, 8: 8,
		9: 9, 10: 10, 11: 11, 12: 12, 13: 13, 14: 14, 15: 15,
		16: 16, 17: 17, 18: 18, 19: 19, 20: 20, 21: 21, 22: 22,
		23: 23, 24: 24, 25: 25, 26: 26, 27: 27, 28: 28, 29: 29,
		30: 30, 31: 31, 32: 32, 33: 33, 34: 34, 35: 35, 36: 36,
		37: 36, 38: 37, 39: 37, 40: 38, 41: 38, 42: 39, 43: 39,
		44: 40, 45: 40, 46: 40, 47: 40, 48: 41, 49: 41, 50: 41,
		51: 41, 52: 42, 53: 42, 54: 42, 55: 42, 56: 43, 57: 43,
		58: 43, 59: 43, 60: 44, 61: 44, 62: 44, 63: 44, 64: 45,
		65: 45, 66: 45, 67: 45, 68: 46, 69: 46, 70: 46, 71: 46,
	}
	// layerCount is the number of GEOS layers in each chemistry layer.
	layerCount := make(map[int]float64)
	for _, iChem := range layerMap {
		layerCount[iChem] += float64(1)
	}

	// staggeredLayerMap is the map between the 73 and 48 staggered layers in
	// GEOS and GEOS-Chem.
	staggeredLayerMap := make(map[int]int)
	for iG, iChem := range layerMap {
		if _, ok := staggeredLayerMap[iChem]; !ok {
			staggeredLayerMap[iChem] = iG
		} else if iG < staggeredLayerMap[iChem] {
			// We want to assign the lowest matching GEOS level to each GEOS-Chem
			// level.
			staggeredLayerMap[iChem] = iG
		}
	}
	staggeredLayerMap[47] = 72 // This is the model top edge.

	return func(in NextData) NextData {
		return func() (*sparse.DenseArray, error) {
			d, err := in()
			if err != nil {
				return nil, err
			}

			if len(d.Shape) < 3 {
				return d, nil // not a 3-d array.
			}
			if len(d.Shape) != 3 {
				panic(fmt.Errorf("inmap preprocessor: GEOS array is more than 3 dimensions (%d)", len(d.Shape)))
			}
			switch d.Shape[0] {
			case 72:
				return geosLayerConvertUnstaggered(d, layerMap, layerCount), nil
			case 73:
				return geosLayerConvertStaggered(d, staggeredLayerMap), nil
			case 1: // only one vertical layer (not 3-d)
				return d, nil
			default:
				panic(fmt.Errorf("inmap preprocessor: invalid vertical dimension %d", d.Shape[0]))
			}
		}
	}
}

// geosLayerConvertUnstaggered sets the GEOS-Chem layer values
// to the average of the GEOS layer values that fall within them.
func geosLayerConvertUnstaggered(in *sparse.DenseArray, layerMap map[int]int, layerCount map[int]float64) *sparse.DenseArray {
	o := sparse.ZerosDense(47, in.Shape[1], in.Shape[2])
	for iG, iChem := range layerMap {
		count := layerCount[iChem]
		for j := 0; j < in.Shape[1]; j++ {
			for i := 0; i < in.Shape[2]; i++ {
				o.AddVal(in.Get(iG, j, i)/count, iChem, j, i)
			}
		}
	}
	return o
}

// geosLayerConvertStaggered sets the GEOS-Chem layer values
// to the GEOS layer values that have coinciding edges with them.
func geosLayerConvertStaggered(in *sparse.DenseArray, staggeredLayerMap map[int]int) *sparse.DenseArray {
	o := sparse.ZerosDense(48, in.Shape[1], in.Shape[2])
	for iChem, iG := range staggeredLayerMap {
		for j := 0; j < in.Shape[1]; j++ {
			for i := 0; i < in.Shape[2]; i++ {
				o.Set(in.Get(iG, j, i), iChem, j, i)
			}
		}
	}
	return o
}

// Nx helps fulfill the Preprocessor interface by returning
// the number of grid cells in the West-East direction.
func (gc *GEOSChem) Nx() (int, error) {
	f, ff, err := ncfFromTemplate(gc.geosA3Dyn, geosFormat, gc.start)
	if err != nil {
		return -1, err
	}
	defer f.Close()
	v := "RH"
	dims := ff.Header.Lengths(v)
	if len(dims) == 0 {
		return -1, fmt.Errorf("geos: missing variable %s", v)
	}
	return dims[3], nil
}

// Ny helps fulfill the Preprocessor interface by returning
// the number of grid cells in the South-North direction.
func (gc *GEOSChem) Ny() (int, error) {
	f, ff, err := ncfFromTemplate(gc.geosA3Dyn, geosFormat, gc.start)
	if err != nil {
		return -1, err
	}
	defer f.Close()
	v := "RH"
	dims := ff.Header.Lengths(v)
	if len(dims) == 0 {
		return -1, fmt.Errorf("geos: missing variable %s", v)
	}
	return dims[2], nil
}

// Nz helps fulfill the Preprocessor interface by returning
// the number of grid cells in the below-above direction.
func (gc *GEOSChem) Nz() (int, error) {
	// We get Nz from the GEOS-Chem output to make sure we're using the
	// GEOS-Chem number of layers rather than the GEOS number of layers.
	f, ff, err := ncfFromTemplate(gc.geosChem, geosChemFormat, gc.start)
	if err != nil {
		return -1, err
	}
	defer f.Close()
	v := "IJ" + gc.dash + "AVG" + gc.dash + "S__SO2"
	dims := ff.Header.Lengths(v)
	if len(dims) == 0 {
		return -1, fmt.Errorf("geoschem: missing variable %s", v)
	} else if len(dims) == 4 {
		dims = dims[1:4] // Sometimes GEOS-Chem files also have a time dimension.
	}
	return dims[0], nil
}

// Return the first set of values of a variable from a chemistry file.
func (gc *GEOSChem) chemFirstValues(v string) ([]float64, error) {
	f, ff, err := ncfFromTemplate(gc.geosChem, geosChemFormat, gc.start)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := readNCFNoHour(v, ff, 0)
	if err != nil {
		// If variable not in file, try all lowercase.
		data, err = readNCFNoHour(strings.ToLower(v), ff, 0)
		if err != nil {
			return nil, err
		}
	}
	return data.Elements, nil
}

// Return an attribute from a chemistry file.
func (gc *GEOSChem) chemAttribute(a string) (float64, error) {
	f, ff, err := ncfFromTemplate(gc.geosChem, geosChemFormat, gc.start)
	if err != nil {
		return math.NaN(), err
	}
	defer f.Close()
	attr := ff.Header.GetAttribute("", a)
	return float64(attr.([]float32)[0]), nil
}

// XCenters returns the x-coordinates of the grid points.
func (gc *GEOSChem) XCenters() ([]float64, error) { return gc.chemFirstValues("LON") }

// YCenters returns the y-coordinates of the grid points.
func (gc *GEOSChem) YCenters() ([]float64, error) { return gc.chemFirstValues("LAT") }

// DX returns the longitude grid spacing.
func (gc *GEOSChem) DX() (float64, error) { return gc.chemAttribute("Delta_Lon") }

// DY returns the latitude grid spacing.
func (gc *GEOSChem) DY() (float64, error) { return gc.chemAttribute("Delta_Lat") }

// PBLH helps fulfill the Preprocessor interface.
func (gc *GEOSChem) PBLH() NextData { return gc.readA1("PBLH") }

// Height returns a functions that calculates layer heights at each
// time step using the hyposometric equation.
func (gc *GEOSChem) Height() NextData {
	TFunc := gc.T() // Temperature [K]
	PFunc := gc.P() // Pressure [Pa]
	return func() (*sparse.DenseArray, error) {
		T, err := TFunc()
		if err != nil {
			return nil, err
		}
		P, err := PFunc()
		if err != nil {
			return nil, err
		}
		layerHeights := sparse.ZerosDense(T.Shape[0]+1, T.Shape[1], T.Shape[2])

		for k := 1; k < T.Shape[0]+1; k++ { // The height of layer zero is zero.
			for j := 0; j < T.Shape[1]; j++ {
				for i := 0; i < T.Shape[2]; i++ {
					p := P.Get(k, j, i)                       // Pressure [Pa]
					pBelow := P.Get(k-1, j, i)                // Pressure [Pa]
					t := T.Get(k-1, j, i)                     // tL in units on K
					h := -1 * math.Log(p/pBelow) * rr * t / g // in meters
					layerHeights.Set(h+layerHeights.Get(k-1, j, i), k, j, i)
				}
			}
		}
		return layerHeights, nil
	}
}

// ALT helps fulfill the Preprocessor interface, returning
// inverse air density [m3/kg].
func (gc *GEOSChem) ALT() NextData {
	densityFunc1 := gc.readChem("TIME" + gc.dash + "SER__AIRDEN") // Air density in molec/cm3.
	densityFunc2 := gc.readChem("BXHGHT_S__AIRNUMDE")             // Alternate: Dry air density in molec/cm3.
	return func() (*sparse.DenseArray, error) {
		density, err := densityFunc1()
		if err != nil {
			if err == io.EOF {
				return nil, err
			}
			holdErr := err
			density, err = densityFunc2()
			if err != nil {
				if err == io.EOF {
					return nil, err
				}
				return nil, fmt.Errorf("%s; %s", holdErr, err)
			}
		}
		alt := sparse.ZerosDense(density.Shape...)
		for i, val := range density.Elements {
			alt.Elements[i] = 1 / (val * (MWa / avNum) * 1000.)
		}
		return alt, nil
	}
}

// U helps fulfill the Preprocessor interface.
func (gc *GEOSChem) U() NextData { return stagger(gc.readA3Dyn("U"), 2) } // (unstaggered)

// V helps fulfill the Preprocessor interface.
func (gc *GEOSChem) V() NextData { return stagger(gc.readA3Dyn("V"), 1) } // (unstaggered)

// W helps fulfill the Preprocessor interface.
func (gc *GEOSChem) W() NextData {
	omegaFunc := gc.readA3Dyn("OMEGA") // Vertical pressure velocity [Pa/s] (unstaggered).
	PFunc := gc.P()
	TFunc := gc.T()
	return func() (*sparse.DenseArray, error) {
		omega, err := omegaFunc()
		if err != nil {
			return nil, err
		}
		P, err := PFunc()
		if err != nil {
			return nil, err
		}
		T, err := TFunc()
		if err != nil {
			return nil, err
		}
		w := sparse.ZerosDense(omega.Shape...)
		for k := 0; k < omega.Shape[0]; k++ {
			for j := 0; j < omega.Shape[1]; j++ {
				for i := 0; i < omega.Shape[2]; i++ {
					dz := -1 * math.Log(P.Get(k+1, j, i)/P.Get(k, j, i)) * rr * T.Get(k, j, i) / g // in meters
					wVal := omega.Get(k, j, i) / (P.Get(k+1, j, i) - P.Get(k, j, i)) / dz
					w.Set(wVal, k, j, i)
				}
			}
		}
		return staggerWorker(w, 0), nil
	}
}

// AVOC helps fulfill the Preprocessor interface.
func (gc *GEOSChem) AVOC() NextData { return gc.readChemGroupAlt(gc.aVOC) }

// BVOC helps fulfill the Preprocessor interface.
func (gc *GEOSChem) BVOC() NextData { return gc.readChemGroupAlt(gc.bVOC) }

// NOx helps fulfill the Preprocessor interface.
func (gc *GEOSChem) NOx() NextData { return gc.readChemGroupAlt(gc.nox) }

// SOx helps fulfill the Preprocessor interface.
func (gc *GEOSChem) SOx() NextData { return gc.readChemGroupAlt(gc.sox) }

// NH3 helps fulfill the Preprocessor interface.
func (gc *GEOSChem) NH3() NextData { return gc.readChemGroupAlt(gc.nh3) }

// ASOA helps fulfill the Preprocessor interface.
func (gc *GEOSChem) ASOA() NextData { return gc.readChemGroupAlt(gc.aSOA) }

// BSOA helps fulfill the Preprocessor interface.
func (gc *GEOSChem) BSOA() NextData { return gc.readChemGroupAlt(gc.bSOA) }

// PNO helps fulfill the Preprocessor interface.
func (gc *GEOSChem) PNO() NextData { return gc.readChemGroupAlt(gc.pNO) }

// PS helps fulfill the Preprocessor interface.
func (gc *GEOSChem) PS() NextData { return gc.readChemGroupAlt(gc.pS) }

// PNH helps fulfill the Preprocessor interface.
func (gc *GEOSChem) PNH() NextData { return gc.readChemGroupAlt(gc.pNH) }

// TotalPM25 helps fulfill the Preprocessor interface.
func (gc *GEOSChem) TotalPM25() NextData { return gc.readChemGroupAlt(gc.totalPM25) }

// SurfaceHeatFlux helps fulfill the Preprocessor interface by returning
// sensible heat flux from turbulence [W/m2].
func (gc *GEOSChem) SurfaceHeatFlux() NextData { return gc.readA1("HFLUX") }

// UStar helps fulfill the Preprocessor interface by returning
// friction velocity [m/s].
func (gc *GEOSChem) UStar() NextData { return gc.readA1("USTAR") }

// T helps fulfill the Preprocessor interface by returning temperature [K].
func (gc *GEOSChem) T() NextData { return gc.readI3("T") }

// P helps fulfill the Preprocessor interface by returning pressure [Pa].
func (gc *GEOSChem) P() NextData {
	PSFunc := gc.readI3("PS")   // Surface pressure [hPa]
	apFunc := gc.readApBp("Ap") // Hybrid-grid A parameter [hPa]
	bpFunc := gc.readApBp("Bp") // Hypbrid-grid b parameter [-]
	return func() (*sparse.DenseArray, error) {
		PS, err := PSFunc()
		if err != nil {
			return nil, err
		}
		ap, err := apFunc()
		if err != nil {
			return nil, err
		}
		bp, err := bpFunc()
		if err != nil {
			return nil, err
		}
		p := sparse.ZerosDense(ap.Shape[0], PS.Shape[0], PS.Shape[1])
		for k := 0; k < ap.Shape[0]; k++ {
			for j := 0; j < PS.Shape[0]; j++ {
				for i := 0; i < PS.Shape[1]; i++ {
					const hPa2Pa = 100.0                                      // Convert hPa to Pa.
					p.Set((PS.Get(j, i)*bp.Get(k)+ap.Get(k))*hPa2Pa, k, j, i) // Pressure [Pa]
				}
			}
		}
		return p, nil
	}
}

// HO helps fulfill the Preprocessor interface by returning hydroxyl
// radical concentration [ppmv].
func (gc *GEOSChem) HO() NextData {
	HOFunc1 := gc.readChem("TIME" + gc.dash + "SER__OH") // OH density (molec / cm3)
	f := gc.readChem("CHEM_L_S__OH")                     // Alternate OH density (molec / cm3)
	HOFunc2 := func() (*sparse.DenseArray, error) {
		data, err := f()
		if err != nil {
			return nil, err
		}
		if data.Shape[0] != 59 { // Sometimes this variable has 59 layers instead of 72. TODO: Why?
			return data, nil
		}
		out := sparse.ZerosDense(72, data.Shape[1], data.Shape[2])
		for k := 0; k < data.Shape[0]; k++ {
			for j := 0; j < data.Shape[1]; j++ {
				for i := 0; i < data.Shape[2]; i++ {
					out.Set(data.Get(k, j, i), k, j, i)
				}
			}
		}
		return out, nil
	}
	altFunc := gc.ALT()
	return func() (*sparse.DenseArray, error) {
		HO, err := HOFunc1()
		if err != nil {
			if err == io.EOF {
				return nil, err
			}
			errHold := err
			HO, err = HOFunc2()
			if err != nil {
				return nil, fmt.Errorf("%s; %s", errHold, err)
			}
		}
		alt, err := altFunc()
		if err != nil {
			return nil, err
		}
		const cm3perm3 = 100. * 100. * 100.
		const gPerKg = 1000.0
		const airFactor = MWa / avNum * cm3perm3 / gPerKg // kg/molec.* cm3/m3
		ho := sparse.ZerosDense(HO.Shape...)
		for i, hoV := range HO.Elements {
			// molec HO / cm3 * m3 / kg air * kg air/molec. air* cm3/m3 * ppm
			ho.Elements[i] = hoV * alt.Elements[i] * airFactor * 1.0e6
		}
		return ho, nil
	}
}

// H2O2 helps fulfill the Preprocessor interface by returning
// hydrogen peroxide concentration [ppmv].
func (gc *GEOSChem) H2O2() NextData {
	H2O2Func := gc.readChem("IJ" + gc.dash + "AVG" + gc.dash + "S__H2O2") // H2O2 concentration [ppbv].
	return func() (*sparse.DenseArray, error) {
		H2O2, err := H2O2Func()
		if err != nil {
			return nil, err
		}
		return H2O2.ScaleCopy(1.0e-3), nil
	}
}

// Z0 helps fulfill the Preprocessor interface by returning
// momentum roughness length [m].
func (gc *GEOSChem) Z0() NextData { return gc.readA1("Z0M") }

// SeinfeldLandUse helps fulfill the Preprocessor interface by
// returning land use categories as
// specified in github.com/ctessum/atmos/seinfeld.
func (gc *GEOSChem) SeinfeldLandUse() NextData {
	// TODO (CT): Account for the fact that a single grid cell can have multiple land uses.
	snowFunc := gc.readA1("FRSNO") // Fraction land covered by snow
	return geosChemSeinfeldLandUse(snowFunc, gc.landUse)
}

func geosChemSeinfeldLandUse(snowFunc NextData, landUse *sparse.DenseArray) NextData {
	return func() (*sparse.DenseArray, error) {
		snowFrac, err := snowFunc() // Fraction land covered by snow
		if err != nil {
			return nil, err
		}
		o := sparse.ZerosDense(snowFrac.Shape...)
		for j := 0; j < snowFrac.Shape[0]; j++ {
			for i := 0; i < snowFrac.Shape[1]; i++ {
				snowV := snowFrac.Get(j, i)
				if snowV > 0.5 { // We assume that snow and desert have similar deposition properties.
					o.Set(float64(seinfeld.Desert), j, i)
				}
				o.Set(float64(geosChemSeinfeld[f2i(landUse.Get(j, i))]), j, i)
			}
		}
		return o, nil
	}
}

// geosChemSeinfeld provides a mapping between GEOSChem use categories
// described in http://wiki.seas.harvard.edu/geos-chem/index.php/Olson_land_map
// and the land use categories as
// specified in github.com/ctessum/atmos/seinfeld.
var geosChemSeinfeld = []seinfeld.LandUseCategory{
	seinfeld.Desert,    // 0 Water
	seinfeld.Deciduous, // 1 Urban
	seinfeld.Grass,     // 2 Shrub
	seinfeld.Desert,    // 3 ---
	seinfeld.Desert,    // 4 ---
	seinfeld.Desert,    // 5 ---
	seinfeld.Evergreen, // 6 Trop. evergreen
	seinfeld.Desert,    // 7 ---
	seinfeld.Desert,    // 8 Desert
	seinfeld.Desert,    // 9 ---
	seinfeld.Desert,    // 10 ---
	seinfeld.Desert,    // 11 ---
	seinfeld.Desert,    // 12 ---
	seinfeld.Desert,    // 13 ---
	seinfeld.Desert,    // 14 ---
	seinfeld.Desert,    // 15 ---
	seinfeld.Shrubs,    // 16 Scrub
	seinfeld.Desert,    // 17 Ice
	seinfeld.Desert,    // 18 ---
	seinfeld.Desert,    // 19 ---
	seinfeld.Evergreen, // 20 Conifer
	seinfeld.Evergreen, // 21 Conifer
	seinfeld.Evergreen, // 22 Conifer
	seinfeld.Evergreen, // 23 Conifer/Deciduous
	seinfeld.Deciduous, // 24 Deciduous/Conifer
	seinfeld.Deciduous, // 25 Deciduous
	seinfeld.Deciduous, // 26 Deciduous
	seinfeld.Evergreen, // 27 Conifer
	seinfeld.Deciduous, // 28 Dwarf forest
	seinfeld.Deciduous, // 29 Trop. broadleaf
	seinfeld.Deciduous, // 30 Agricultural
	seinfeld.Deciduous, // 31 Agricultural
	seinfeld.Deciduous, // 32 Dec. woodland
	seinfeld.Deciduous, // 33 Trop. rainforest
	seinfeld.Desert,    // 34 ---
	seinfeld.Desert,    // 35 ---
	seinfeld.Grass,     // 36 Rice paddies
	seinfeld.Shrubs,    // 37 agric
	seinfeld.Shrubs,    // 38 agric
	seinfeld.Shrubs,    // 39 agric.
	seinfeld.Shrubs,    // 40 shrub/grass
	seinfeld.Shrubs,    // 41 shrub/grass
	seinfeld.Shrubs,    // 42 shrub/grass
	seinfeld.Shrubs,    // 43 shrub/grass
	seinfeld.Shrubs,    // 44 shrub/grass
	seinfeld.Shrubs,    // 45 wetland
	seinfeld.Shrubs,    // 46 scrub
	seinfeld.Shrubs,    // 47 scrub
	seinfeld.Shrubs,    // 48 scrub
	seinfeld.Shrubs,    // 49 scrub
	seinfeld.Desert,    // 50 Desert
	seinfeld.Desert,    // 51 Desert
	seinfeld.Desert,    // 52 Steppe
	seinfeld.Desert,    // 53 Tundra
	seinfeld.Deciduous, // 54 rainforest
	seinfeld.Deciduous, // 55 mixed wood/open
	seinfeld.Deciduous, // 56 mixed wood/open
	seinfeld.Deciduous, // 57 mixed wood/open
	seinfeld.Deciduous, // 58 mixed wood/open
	seinfeld.Deciduous, // 59 mixed wood/open
	seinfeld.Evergreen, // 60 conifers
	seinfeld.Evergreen, // 61 conifers
	seinfeld.Evergreen, // 62 conifers
	seinfeld.Evergreen, // 63 Wooded tundra
	seinfeld.Grass,     // 64 Moor
	seinfeld.Desert,    // 65 coastal
	seinfeld.Desert,    // 66 coastal
	seinfeld.Desert,    // 67 coastal
	seinfeld.Desert,    // 68 coastal
	seinfeld.Desert,    // 69 desert
	seinfeld.Desert,    // 70 ice
	seinfeld.Desert,    // 71 salt flats
	seinfeld.Grass,     // 72 wetland
	seinfeld.Desert,    // 73 water
}

// WeselyLandUse helps fulfill the Preprocessor interface by
// returning land use categories as
// specified in github.com/ctessum/atmos/wesely1989.
func (gc *GEOSChem) WeselyLandUse() NextData {
	// TODO (CT): Account for the fact that a single grid cell can have multiple land uses.
	snowFunc := gc.readA1("FRSNO") // Fraction land covered by snow
	return geosChemSeinfeldLandUse(snowFunc, gc.landUse)
}

func geosChemWeselyLandUse(snowFunc NextData, landUse *sparse.DenseArray) NextData {
	return func() (*sparse.DenseArray, error) {
		snowFrac, err := snowFunc() // Fraction land covered by snow
		if err != nil {
			return nil, err
		}
		o := sparse.ZerosDense(snowFrac.Shape...)
		for j := 0; j < snowFrac.Shape[0]; j++ {
			for i := 0; i < snowFrac.Shape[1]; i++ {
				snowV := snowFrac.Get(j, i)
				if snowV > 0.5 { // We assume that snow and Barren have similar deposition properties.
					o.Set(float64(wesely1989.Barren), j, i)
				}
				o.Set(float64(geosChemWesely[f2i(landUse.Get(j, i))]), j, i)
			}
		}
		return o, nil
	}
}

// geosChemWesely provides a mapping between GEOSChem use categories
// described in http://wiki.seas.harvard.edu/geos-chem/index.php/Olson_land_map
// and the land use categories as
// specified in github.com/ctessum/atmos/wesely1989.
var geosChemWesely = []wesely1989.LandUseCategory{
	wesely1989.Water,        // 0 Water
	wesely1989.Urban,        // 1 Urban
	wesely1989.RockyShrubs,  // 2 Shrub
	wesely1989.Barren,       // 3 ---
	wesely1989.Barren,       // 4 ---
	wesely1989.Barren,       // 5 ---
	wesely1989.Coniferous,   // 6 Trop. evergreen
	wesely1989.Barren,       // 7 ---
	wesely1989.Barren,       // 8 Desert
	wesely1989.Barren,       // 9 ---
	wesely1989.Barren,       // 10 ---
	wesely1989.Barren,       // 11 ---
	wesely1989.Barren,       // 12 ---
	wesely1989.Barren,       // 13 ---
	wesely1989.Barren,       // 14 ---
	wesely1989.Barren,       // 15 ---
	wesely1989.RockyShrubs,  // 16 Scrub
	wesely1989.Barren,       // 17 Ice
	wesely1989.Barren,       // 18 ---
	wesely1989.Barren,       // 19 ---
	wesely1989.Coniferous,   // 20 Conifer
	wesely1989.Coniferous,   // 21 Conifer
	wesely1989.Coniferous,   // 22 Conifer
	wesely1989.MixedForest,  // 23 Conifer/Deciduous
	wesely1989.MixedForest,  // 24 Deciduous/Conifer
	wesely1989.Deciduous,    // 25 Deciduous
	wesely1989.Deciduous,    // 26 Deciduous
	wesely1989.Coniferous,   // 27 Conifer
	wesely1989.MixedForest,  // 28 Dwarf forest
	wesely1989.Deciduous,    // 29 Trop. broadleaf
	wesely1989.Agricultural, // 30 Agricultural
	wesely1989.Agricultural, // 31 Agricultural
	wesely1989.Deciduous,    // 32 Dec. woodland
	wesely1989.Deciduous,    // 33 Trop. rainforest
	wesely1989.Barren,       // 34 ---
	wesely1989.Barren,       // 35 ---
	wesely1989.Agricultural, // 36 Rice paddies
	wesely1989.Agricultural, // 37 agric
	wesely1989.Agricultural, // 38 agric
	wesely1989.Agricultural, // 39 agric.
	wesely1989.Range,        // 40 shrub/grass
	wesely1989.Range,        // 41 shrub/grass
	wesely1989.Range,        // 42 shrub/grass
	wesely1989.Range,        // 43 shrub/grass
	wesely1989.Range,        // 44 shrub/grass
	wesely1989.Wetland,      // 45 wetland
	wesely1989.RockyShrubs,  // 46 scrub
	wesely1989.RockyShrubs,  // 47 scrub
	wesely1989.RockyShrubs,  // 48 scrub
	wesely1989.RockyShrubs,  // 49 scrub
	wesely1989.Barren,       // 50 Desert
	wesely1989.Barren,       // 51 Desert
	wesely1989.RockyShrubs,  // 52 Steppe
	wesely1989.Range,        // 53 Tundra
	wesely1989.Deciduous,    // 54 rainforest
	wesely1989.MixedForest,  // 55 mixed wood/open
	wesely1989.MixedForest,  // 56 mixed wood/open
	wesely1989.MixedForest,  // 57 mixed wood/open
	wesely1989.MixedForest,  // 58 mixed wood/open
	wesely1989.MixedForest,  // 59 mixed wood/open
	wesely1989.Coniferous,   // 60 conifers
	wesely1989.Coniferous,   // 61 conifers
	wesely1989.Coniferous,   // 62 conifers
	wesely1989.MixedForest,  // 63 Wooded tundra
	wesely1989.Wetland,      // 64 Moor
	wesely1989.Wetland,      // 65 coastal
	wesely1989.Wetland,      // 66 coastal
	wesely1989.Wetland,      // 67 coastal
	wesely1989.Wetland,      // 68 coastal
	wesely1989.Barren,       // 69 desert
	wesely1989.Barren,       // 70 ice
	wesely1989.Barren,       // 71 salt flats
	wesely1989.Wetland,      // 72 wetland
	wesely1989.Water,        // 73 water
}

// QRain helps fulfill the Preprocessor interface by returning
// rain mass fraction based on the GEOS precipitation rate [kg m-2 s-1]
// and the assumption (from the EMEP model wet deposition algorithm)
// that raindrops are falling at 5 m/s.
func (gc *GEOSChem) QRain() NextData {
	PFLCUFunc := gc.readA3MstE("PFLCU")     // 3d flux of liquid convective precipitation [kg m-2 s-1]
	PFLLSanFunc := gc.readA3MstE("PFLLSAN") // 3d flux of liquid non-convective precipitation [kg m-2 s-1]
	altFunc := gc.ALT()                     // m3 kg-1
	return func() (*sparse.DenseArray, error) {
		pflcu, err := PFLCUFunc()
		if err != nil {
			return nil, err
		}
		pfllSan, err := PFLLSanFunc()
		if err != nil {
			return nil, err
		}
		alt, err := altFunc()
		if err != nil {
			return nil, err
		}

		const Vdr = 5.0 // droplet velocity [m/s]
		qRain := sparse.ZerosDense(alt.Shape...)
		// pflcu and pfllSan are staggered but qRain is not, so
		// we average the rain flux values (pflcu and pfllSan) at the
		// bottom and top of each grid cell.
		for k := 0; k < qRain.Shape[0]; k++ {
			for j := 0; j < qRain.Shape[1]; j++ {
				for i := 0; i < qRain.Shape[2]; i++ {
					pflcuV := (pflcu.Get(k, j, i) + pflcu.Get(k+1, j, i)) / 2
					pfllSanV := (pfllSan.Get(k, j, i) + pfllSan.Get(k+1, j, i)) / 2
					// From EMEP algorithm: P = QRAIN * Vdr * ρgas => QRAIN = P / Vdr / ρgas
					// [kg m-2 s-1] / [m s-1] * [m3 kg-1]
					q := (pflcuV + pfllSanV) / Vdr * alt.Get(k, j, i)
					qRain.Set(q, k, j, i)
				}
			}
		}
		return qRain, nil
	}
}

// CloudFrac helps fulfill the Preprocessor interface by returning
// the cloud volume fraction.
func (gc *GEOSChem) CloudFrac() NextData { return gc.readA3Cld("CLOUD") }

// QCloud helps fulfill the Preprocessor interface by returning
// the cloud mass fraction.
func (gc *GEOSChem) QCloud() NextData { return gc.readA3Cld("QL") }

// RadiationDown helps fulfill the Preprocessor interface by
// returning downwelling radiation [W m-2].
func (gc *GEOSChem) RadiationDown() NextData {
	ParDFFunc := gc.readA1("PARDF") // Surface downwelling PAR diffusive flux [W m-2]
	ParDRFunc := gc.readA1("PARDR") // Surface downwelling PAR direct flux [W m-2]
	return func() (*sparse.DenseArray, error) {
		ParDF, err := ParDFFunc()
		if err != nil {
			return nil, err
		}
		ParDR, err := ParDRFunc()
		if err != nil {
			return nil, err
		}
		out := ParDF.Copy()
		out.AddDense(ParDR)
		return out, nil
	}
}

// olsonLandMap holds information about an Olson land map.
type olsonLandMap struct {
	data   *rtree.Rtree
	dx, dy float64
	xo, yo float64
	nx, ny int
}

type olsonGridCell struct {
	geom.Polygon
	category int
}

// readOlsonLandMap reads data from an Olson land map file described here:
// http://wiki.seas.harvard.edu/geos-chem/index.php/Olson_land_map.
// It may be downlaodable from:
// ftp://ftp.as.harvard.edu/gcgrid/geos-chem/data/ExtData/CHEM_INPUTS/Olson_Land_Map_201203/
// The file must be converted from netcdf version 4 to version 3 before
// use by this function.
// This can be done using the command:
// nccopy -k classic Olson_2001_Land_Map.025x025.generic.nc Olson_2001_Land_Map.025x025.generic.nc
func readOlsonLandMap(file *cdf.File) (*olsonLandMap, error) {
	dxStr := file.Header.GetAttribute("", "delta_lon").(string)
	dyStr := file.Header.GetAttribute("", "delta_lat").(string)
	dx, err := strconv.ParseFloat(dxStr, 64)
	if err != nil {
		return nil, fmt.Errorf("inmap: parsing Olson land map dx: %v", err)
	}
	dy, err := strconv.ParseFloat(dyStr, 64)
	if err != nil {
		return nil, fmt.Errorf("inmap: parsing Olson land map dy: %v", err)
	}

	dims := file.Header.Lengths("OLSON")
	ny := dims[1]
	nx := dims[2]

	r := file.Reader("OLSON", nil, nil)
	buf := r.Zero(-1).([]int32)
	o := &olsonLandMap{
		data: rtree.NewTree(25, 50),
		xo:   -180,
		yo:   -90,
		dx:   dx,
		dy:   dy,
		nx:   nx,
		ny:   ny,
	}

	if _, err := r.Read(buf); err != nil {
		return nil, fmt.Errorf("inmap: reading Olson land map: %v", err)
	}

	for iy := 0; iy < o.ny; iy++ {
		for ix := 0; ix < o.nx; ix++ {
			x0 := o.xo + o.dx*float64(ix)
			x1 := o.xo + o.dx*float64(ix+1)
			y0 := o.yo + o.dy*float64(iy)
			y1 := o.yo + o.dy*float64(iy+1)
			c := olsonGridCell{
				Polygon: geom.Polygon{{
					{X: x0, Y: y0},
					{X: x1, Y: y0},
					{X: x1, Y: y1},
					{X: x0, Y: y1},
				}},
				category: int(buf[iy*nx+ix]),
			}
			o.data.Insert(c)
		}
	}
	return o, nil
}

// fractions returns the fraction of land use types within the given polygon.
func (o *olsonLandMap) fractions(p geom.Polygon) map[int]float64 {
	out := make(map[int]float64)
	for _, cI := range o.data.SearchIntersect(p.Bounds()) {
		c := cI.(olsonGridCell)
		isect := p.Intersection(c)
		if isect != nil {
			out[c.category] += isect.Area()
		}
	}
	a := p.Area()
	for cat := range out {
		out[cat] /= a
	}
	return out
}

// largestLandUse returns the land use index with the largest area
// in each grid cell when given a Olson land map file.
func (gc *GEOSChem) largestLandUse(olsonLandMapFile *cdf.File) (*sparse.DenseArray, error) {
	o, err := readOlsonLandMap(olsonLandMapFile)
	if err != nil {
		return nil, err
	}

	out := sparse.ZerosDense(len(gc.yCenters), len(gc.xCenters))
	for j, y := range gc.yCenters {
		dy := gc.dy
		if j == 0 {
			dy = ((gc.yCenters[j+1] - y) - gc.dy/2) * 2
		}
		if j == len(gc.yCenters)-1 {
			dy = ((y - gc.yCenters[j-1]) - gc.dy/2) * 2
		}
		for i, x := range gc.xCenters {
			dx := gc.dx
			if i == 0 {
				dx = ((gc.xCenters[i+1] - x) - gc.dx/2) * 2
			}
			if i == len(gc.xCenters)-1 {
				dx = ((x - gc.xCenters[i-1]) - gc.dx/2) * 2
			}
			x0 := x - dx/2
			x1 := x + dx/2
			y0 := y - dy/2
			y1 := y + dy/2
			p := geom.Polygon{{
				{X: x0, Y: y0},
				{X: x1, Y: y0},
				{X: x1, Y: y1},
				{X: x0, Y: y1},
				{X: x0, Y: y0},
			}}
			fractions := o.fractions(p)
			if len(fractions) == 0 {
				return nil, fmt.Errorf("no land use information available for polygon %+v", p)
			}
			maxCat := math.MinInt32
			maxVal := math.Inf(-1)
			for c, v := range fractions {
				if v > maxVal {
					maxVal = v
					maxCat = c
				}
			}
			out.Set(float64(maxCat), j, i)
		}
	}
	return out, nil
}
