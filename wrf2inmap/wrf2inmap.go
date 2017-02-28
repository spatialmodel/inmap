/*
Copyright © 2013 the InMAP authors.
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

// Package wrf2inmap processes WRFChem output files into InMAP input files.
package main

import (
	"fmt"
	"time"

	"bitbucket.org/ctessum/sparse"
)

// WRF variables currently used:
/* hc5,hc8,olt,oli,tol,xyl,csl,cvasoa1,cvasoa2,cvasoa3,cvasoa4,iso,api,sesq,lim,
cvbsoa1,cvbsoa2,cvbsoa3,cvbsoa4,asoa1i,asoa1j,asoa2i,asoa2j,asoa3i,asoa3j,asoa4i,
asoa4j,bsoa1i,bsoa1j,bsoa2i,bsoa2j,bsoa3i,bsoa3j,bsoa4i,bsoa4j,no,no2,no3ai,no3aj,
so2,sulf,so4ai,so4aj,nh3,nh4ai,nh4aj,PM2_5_DRY,U,V,W,PBLH,PH,PHB,HFX,UST,PBLH,T,
PB,P,ho,h2o2,LU_INDEX,QRAIN,CLDFRA,QCLOUD,ALT,SWDOWN,GLW */

const wrfFormat = "2006-01-02_15_04_05"

// WRFChem is an InMAP preprocessor for WRF-Chem output.
type WRFChem struct {
	aVOC, bVOC, aSOA, bSOA, nox, no, no2, pNO, sox, pS, nh3, pNH, totalPM25 map[string]float64

	start, end time.Time

	cfg *ConfigInfo

	recordDelta, fileDelta time.Duration
}

// NewWRFChem initializes a WRF-Chem preprocessor from the given
// configuration information.
func NewWRFChem(cfg *ConfigInfo) (*WRFChem, error) {
	w := WRFChem{
		// RACM VOC species and molecular weights (g/mol);
		// Only includes anthropogenic precursors to SOA from
		// anthropogenic (aSOA) and biogenic (bSOA) sources as
		// in Ahmadov et al. (2012)
		// Assume condensable vapor from SOA has molar mass of 70
		aVOC: map[string]float64{"hc5": 72, "hc8": 114,
			"olt": 42, "oli": 68, "tol": 92, "xyl": 106, "csl": 108,
			"cvasoa1": 70, "cvasoa2": 70, "cvasoa3": 70, "cvasoa4": 70},
		bVOC: map[string]float64{"iso": 68, "api": 136, "sesq": 84.2,
			"lim": 136, "cvbsoa1": 70, "cvbsoa2": 70,
			"cvbsoa3": 70, "cvbsoa4": 70},
		// VBS SOA species (anthropogenic only)
		aSOA: map[string]float64{"asoa1i": 1, "asoa1j": 1, "asoa2i": 1,
			"asoa2j": 1, "asoa3i": 1, "asoa3j": 1, "asoa4i": 1, "asoa4j": 1},
		// VBS SOA species (biogenic only)
		bSOA: map[string]float64{"bsoa1i": 1, "bsoa1j": 1, "bsoa2i": 1,
			"bsoa2j": 1, "bsoa3i": 1, "bsoa3j": 1, "bsoa4i": 1, "bsoa4j": 1},
		// NOx is RACM NOx species and molecular weights, multiplied by their
		// nitrogen fractions
		nox: map[string]float64{"no": 30 / 30 * mwN, "no2": 46 / 46 * mwN},
		// NO is the mass of N in NO
		no: map[string]float64{"no": 1.},
		// NO2 is the mass of N in  NO2
		no2: map[string]float64{"no2": 1.},
		// pNO is the MADE particulate NO species, nitrogen fraction
		pNO: map[string]float64{"no3ai": mwN / mwNO3, "no3aj": mwN / mwNO3},
		// SOx is the RACM SOx species and molecular weights
		sox: map[string]float64{"so2": 64 / 64 * mwS, "sulf": 98 / 98 * mwS},
		// pS is the MADE particulate Sulfur species; sulfur fraction
		pS: map[string]float64{"so4ai": mwS / mwSO4, "so4aj": mwS / mwSO4},
		// NH3 is ammonia
		nh3: map[string]float64{"nh3": 17.03056 * 17.03056 / mwN},
		// pNH is the MADE particulate ammonia species, nitrogen fraction
		pNH:       map[string]float64{"nh4ai": mwN / mwNH4, "nh4aj": mwN / mwNH4},
		totalPM25: map[string]float64{"PM2_5_DRY": 1.},

		cfg: cfg,
	}

	var err error
	w.start, err = time.Parse(inDateFormat, w.cfg.StartDate)
	if err != nil {
		return nil, fmt.Errorf("inmap: WRF-Chem preprocessor start time: %v", err)
	}
	w.end, err = time.Parse(inDateFormat, w.cfg.EndDate)
	if err != nil {
		return nil, fmt.Errorf("inmap: WRF-Chem preprocessor end time: %v", err)
	}
	w.end = w.end.AddDate(0, 0, 1) // add 1 day to the end

	w.recordDelta, err = time.ParseDuration("1h")
	if err != nil {
		return nil, fmt.Errorf("inmap: WRF-Chem preprocessor recordDelta: %v", err)
	}
	w.fileDelta, err = time.ParseDuration("24h")
	if err != nil {
		return nil, fmt.Errorf("inmap: WRF-Chem preprocessor fileDelta: %v", err)
	}

	return &w, nil
}

// PBLH helps fulfill the Preprocessor interface.
func (w *WRFChem) PBLH() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "PBLH", w.start, w.end, w.recordDelta, w.fileDelta)
}

// PH helps fulfill the Preprocessor interface.
func (w *WRFChem) PH() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "PH", w.start, w.end, w.recordDelta, w.fileDelta)
}

// PHB helps fulfill the Preprocessor interface.
func (w *WRFChem) PHB() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "PHB", w.start, w.end, w.recordDelta, w.fileDelta)
}

// ALT helps fulfill the Preprocessor interface.
func (w *WRFChem) ALT() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "ALT", w.start, w.end, w.recordDelta, w.fileDelta)
}

// U helps fulfill the Preprocessor interface.
func (w *WRFChem) U() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "U", w.start, w.end, w.recordDelta, w.fileDelta)
}

// V helps fulfill the Preprocessor interface.
func (w *WRFChem) V() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "V", w.start, w.end, w.recordDelta, w.fileDelta)
}

// W helps fulfill the Preprocessor interface.
func (w *WRFChem) W() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "W", w.start, w.end, w.recordDelta, w.fileDelta)
}

// AVOC helps fulfill the Preprocessor interface.
func (w *WRFChem) AVOC() NextData {
	return nextDataWRFGasGroupNCF(w.cfg.Wrfout, wrfFormat, w.aVOC, "ALT", w.start, w.end, w.recordDelta, w.fileDelta)
}

// BVOC helps fulfill the Preprocessor interface.
func (w *WRFChem) BVOC() NextData {
	return nextDataWRFGasGroupNCF(w.cfg.Wrfout, wrfFormat, w.bVOC, "ALT", w.start, w.end, w.recordDelta, w.fileDelta)
}

// NOx helps fulfill the Preprocessor interface.
func (w *WRFChem) NOx() NextData {
	return nextDataWRFGasGroupNCF(w.cfg.Wrfout, wrfFormat, w.nox, "ALT", w.start, w.end, w.recordDelta, w.fileDelta)
}

// SOx helps fulfill the Preprocessor interface.
func (w *WRFChem) SOx() NextData {
	return nextDataWRFGasGroupNCF(w.cfg.Wrfout, wrfFormat, w.sox, "ALT", w.start, w.end, w.recordDelta, w.fileDelta)
}

// NH3 helps fulfill the Preprocessor interface.
func (w *WRFChem) NH3() NextData {
	return nextDataWRFGasGroupNCF(w.cfg.Wrfout, wrfFormat, w.nh3, "ALT", w.start, w.end, w.recordDelta, w.fileDelta)
}

// ASOA helps fulfill the Preprocessor interface.
func (w *WRFChem) ASOA() NextData {
	return nextDataWRFParticleGroupNCF(w.cfg.Wrfout, wrfFormat, w.aSOA, "ALT", w.start, w.end, w.recordDelta, w.fileDelta)
}

// BSOA helps fulfill the Preprocessor interface.
func (w *WRFChem) BSOA() NextData {
	return nextDataWRFParticleGroupNCF(w.cfg.Wrfout, wrfFormat, w.bSOA, "ALT", w.start, w.end, w.recordDelta, w.fileDelta)
}

// PNO helps fulfill the Preprocessor interface.
func (w *WRFChem) PNO() NextData {
	return nextDataWRFParticleGroupNCF(w.cfg.Wrfout, wrfFormat, w.pNO, "ALT", w.start, w.end, w.recordDelta, w.fileDelta)
}

// PS helps fulfill the Preprocessor interface.
func (w *WRFChem) PS() NextData {
	return nextDataWRFParticleGroupNCF(w.cfg.Wrfout, wrfFormat, w.pS, "ALT", w.start, w.end, w.recordDelta, w.fileDelta)
}

// PNH helps fulfill the Preprocessor interface.
func (w *WRFChem) PNH() NextData {
	return nextDataWRFParticleGroupNCF(w.cfg.Wrfout, wrfFormat, w.pNH, "ALT", w.start, w.end, w.recordDelta, w.fileDelta)
}

// TotalPM25 helps fulfill the Preprocessor interface.
func (w *WRFChem) TotalPM25() NextData {
	return nextDataWRFParticleGroupNCF(w.cfg.Wrfout, wrfFormat, w.totalPM25, "ALT", w.start, w.end, w.recordDelta, w.fileDelta)
}

// SurfaceHeatFlux helps fulfill the Preprocessor interface.
func (w *WRFChem) SurfaceHeatFlux() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "HFX", w.start, w.end, w.recordDelta, w.fileDelta)
}

// UStar helps fulfill the Preprocessor interface.
func (w *WRFChem) UStar() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "UST", w.start, w.end, w.recordDelta, w.fileDelta)
}

// T helps fulfill the Preprocessor interface.
func (w *WRFChem) T() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "T", w.start, w.end, w.recordDelta, w.fileDelta)
}

// PB helps fulfill the Preprocessor interface.
func (w *WRFChem) PB() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "PB", w.start, w.end, w.recordDelta, w.fileDelta)
}

// P helps fulfill the Preprocessor interface.
func (w *WRFChem) P() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "P", w.start, w.end, w.recordDelta, w.fileDelta)
}

// HO helps fulfill the Preprocessor interface.
func (w *WRFChem) HO() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "ho", w.start, w.end, w.recordDelta, w.fileDelta)
}

// H2O2 helps fulfill the Preprocessor interface.
func (w *WRFChem) H2O2() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "h2o2", w.start, w.end, w.recordDelta, w.fileDelta)
}

// LUIndex helps fulfill the Preprocessor interface.
func (w *WRFChem) LUIndex() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "LU_INDEX", w.start, w.end, w.recordDelta, w.fileDelta)
}

// QRain helps fulfill the Preprocessor interface.
func (w *WRFChem) QRain() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "QRAIN", w.start, w.end, w.recordDelta, w.fileDelta)
}

// CloudFrac helps fulfill the Preprocessor interface.
func (w *WRFChem) CloudFrac() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "CLDFRA", w.start, w.end, w.recordDelta, w.fileDelta)
}

// QCloud helps fulfill the Preprocessor interface.
func (w *WRFChem) QCloud() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "QCLOUD", w.start, w.end, w.recordDelta, w.fileDelta)
}

// SWDown helps fulfill the Preprocessor interface.
func (w *WRFChem) SWDown() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "SWDOWN", w.start, w.end, w.recordDelta, w.fileDelta)
}

// GLW helps fulfill the Preprocessor interface.
func (w *WRFChem) GLW() NextData {
	return nextDataNCF(w.cfg.Wrfout, wrfFormat, "GLW", w.start, w.end, w.recordDelta, w.fileDelta)
}

func nextDataWRFGasGroupNCF(fileTemplate string, dateFormat string, varNames map[string]float64, altVar string, start, end time.Time, recordDelta, fileDelta time.Duration) NextData {
	f := nextDataWRFParticleGroupNCF(fileTemplate, dateFormat, varNames, altVar, start, end, recordDelta, fileDelta)
	return func() (*sparse.DenseArray, error) {
		data, err := f()
		if err != nil {
			return nil, err
		}
		data.Scale(1000.0 / MWa)
		return data, nil
	}
}

func nextDataWRFParticleGroupNCF(fileTemplate string, dateFormat string, varNames map[string]float64, altVar string, start, end time.Time, recordDelta, fileDelta time.Duration) NextData {
	altFunc := nextDataNCF(fileTemplate, dateFormat, altVar, start, end, recordDelta, fileDelta)
	dataFuncs := make(map[string]NextData)
	for v := range varNames {
		dataFuncs[v] = nextDataNCF(fileTemplate, dateFormat, v, start, end, recordDelta, fileDelta)
	}
	return func() (*sparse.DenseArray, error) {
		alt, err := altFunc()
		if err != nil {
			return nil, err
		}
		var out *sparse.DenseArray
		firstData := true
		for varName, f := range dataFuncs {
			data, err := f()
			if err != nil {
				return nil, err
			}
			if firstData {
				out = sparse.ZerosDense(data.Shape...)
				firstData = false
			}
			factor := varNames[varName]
			for i, val := range data.Elements {
				// convert ppm to μg/m3
				out.Elements[i] += val * factor / alt.Elements[i]
			}
		}
		return out, nil
	}
}
