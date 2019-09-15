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

package inmap

import (
	"fmt"
	"math"
	"time"

	"github.com/ctessum/atmos/seinfeld"
	"github.com/ctessum/atmos/wesely1989"

	"github.com/ctessum/sparse"
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

	wrfOut string

	recordDelta, fileDelta time.Duration

	msgChan chan string
}

// NewWRFChem initializes a WRF-Chem preprocessor from the given
// configuration information.
// WRFOut is the location of WRF-Chem output files.
// [DATE] should be used as a wild card for the simulation date.
// startDate and endDate are the dates of the beginning and end of the
// simulation, respectively, in the format "YYYYMMDD".
// If msgChan is not nil, status messages will be sent to it.
func NewWRFChem(WRFOut, startDate, endDate string, msgChan chan string) (*WRFChem, error) {
	w := WRFChem{
		// These maps contain the WRF-Chem variables that make
		// up the chemical species groups, as well as the
		// multiplication factors required to convert concentrations
		// to mass fractions [μg/kg dry air].

		// RACM VOC species and molecular weights (g/mol);
		// Only includes anthropogenic precursors to SOA from
		// anthropogenic (aSOA) and biogenic (bSOA) sources as
		// in Ahmadov et al. (2012)
		// Assume condensable vapor from SOA has molar mass of 70
		aVOC: map[string]float64{
			"hc5": ppmvToUgKg(72), "hc8": ppmvToUgKg(114),
			"olt": ppmvToUgKg(42), "oli": ppmvToUgKg(68), "tol": ppmvToUgKg(92),
			"xyl": ppmvToUgKg(106), "csl": ppmvToUgKg(108),
			"cvasoa1": ppmvToUgKg(70), "cvasoa2": ppmvToUgKg(70),
			"cvasoa3": ppmvToUgKg(70), "cvasoa4": ppmvToUgKg(70),
		},
		bVOC: map[string]float64{
			"iso": ppmvToUgKg(68), "api": ppmvToUgKg(136), "sesq": ppmvToUgKg(84.2),
			"lim": ppmvToUgKg(136), "cvbsoa1": ppmvToUgKg(70), "cvbsoa2": ppmvToUgKg(70),
			"cvbsoa3": ppmvToUgKg(70), "cvbsoa4": ppmvToUgKg(70),
		},
		// VBS SOA species (anthropogenic only) [μg/kg dry air].
		aSOA: map[string]float64{"asoa1i": 1, "asoa1j": 1, "asoa2i": 1,
			"asoa2j": 1, "asoa3i": 1, "asoa3j": 1, "asoa4i": 1, "asoa4j": 1},
		// VBS SOA species (biogenic only) [μg/kg dry air].
		bSOA: map[string]float64{"bsoa1i": 1, "bsoa1j": 1, "bsoa2i": 1,
			"bsoa2j": 1, "bsoa3i": 1, "bsoa3j": 1, "bsoa4i": 1, "bsoa4j": 1},
		// NOx is RACM NOx species. We are only interested in the mass
		// of Nitrogen, rather than the mass of the whole molecule, so
		// we use the molecular weight of Nitrogen.
		nox: map[string]float64{"no": ppmvToUgKg(mwN), "no2": ppmvToUgKg(mwN)},
		// pNO is the Nitrogen fraction of MADE particulate
		// NO species [μg/kg dry air].
		pNO: map[string]float64{"no3ai": mwN / mwNO3, "no3aj": mwN / mwNO3},
		// SOx is the RACM SOx species. We are only interested in the mass
		// of Sulfur, rather than the mass of the whole molecule, so
		// we use the molecular weight of Sulfur.
		sox: map[string]float64{"so2": ppmvToUgKg(mwS), "sulf": ppmvToUgKg(mwS)},
		// pS is the Sulfur fraction of the MADE particulate
		// Sulfur species [μg/kg dry air].
		pS: map[string]float64{"so4ai": mwS / mwSO4, "so4aj": mwS / mwSO4},
		// NH3 is ammonia. We are only interested in the mass
		// of Nitrogen, rather than the mass of the whole molecule, so
		// we use the molecular weight of Nitrogen.
		nh3: map[string]float64{"nh3": ppmvToUgKg(mwN)},
		// pNH is the Nitrogen fraction of the MADE particulate
		// ammonia species [μg/kg dry air].
		pNH: map[string]float64{"nh4ai": mwN / mwNH4, "nh4aj": mwN / mwNH4},
		// totalPM25 is total mass of PM2.5  [μg/m3].
		totalPM25: map[string]float64{"PM2_5_DRY": 1.},

		wrfOut:  WRFOut,
		msgChan: msgChan,
	}

	var err error
	w.start, err = time.Parse(inDateFormat, startDate)
	if err != nil {
		return nil, fmt.Errorf("inmap: WRF-Chem preprocessor start time: %v", err)
	}
	w.end, err = time.Parse(inDateFormat, endDate)
	if err != nil {
		return nil, fmt.Errorf("inmap: WRF-Chem preprocessor end time: %v", err)
	}

	if !w.end.After(w.start) {
		if err != nil {
			return nil, fmt.Errorf("inmap: WRF-Chem preprocessor end time %v is not after start time %v", w.end, w.start)
		}
	}

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

// ppmvToUgKg returns a multiplier to convert a concentration in
// ppmv dry air to a mass fraction [micrograms per kilogram dry air]
// for a chemical species with the given molecular weight in g/mol.
func ppmvToUgKg(mw float64) float64 {
	return mw * 1000.0 / MWa
}

func (w *WRFChem) read(varName string) NextData {
	return nextDataNCF(w.wrfOut, wrfFormat, varName, w.start, w.end, w.recordDelta, w.fileDelta, readNCF, w.msgChan)
}

func (w *WRFChem) readGroupAlt(varGroup map[string]float64) NextData {
	return nextDataGroupAltNCF(w.wrfOut, wrfFormat, varGroup, w.ALT(), w.start, w.end, w.recordDelta, w.fileDelta, readNCF, w.msgChan)
}

func (w *WRFChem) readGroup(varGroup map[string]float64) NextData {
	return nextDataGroupNCF(w.wrfOut, wrfFormat, varGroup, w.start, w.end, w.recordDelta, w.fileDelta, readNCF, w.msgChan)
}

// Nx helps fulfill the Preprocessor interface by returning
// the number of grid cells in the West-East direction.
func (w *WRFChem) Nx() (int, error) {
	f, ff, err := ncfFromTemplate(w.wrfOut, wrfFormat, w.start)
	if err != nil {
		return -1, fmt.Errorf("nx: %v", err)
	}
	defer f.Close()
	return ff.Header.Lengths("ALT")[3], nil
}

// Ny helps fulfill the Preprocessor interface by returning
// the number of grid cells in the South-North direction.
func (w *WRFChem) Ny() (int, error) {
	f, ff, err := ncfFromTemplate(w.wrfOut, wrfFormat, w.start)
	if err != nil {
		return -1, fmt.Errorf("ny: %v", err)
	}
	defer f.Close()
	return ff.Header.Lengths("ALT")[2], nil
}

// Nz helps fulfill the Preprocessor interface by returning
// the number of grid cells in the below-above direction.
func (w *WRFChem) Nz() (int, error) {
	f, ff, err := ncfFromTemplate(w.wrfOut, wrfFormat, w.start)
	if err != nil {
		return -1, fmt.Errorf("nz: %v", err)
	}
	defer f.Close()
	return ff.Header.Lengths("ALT")[1], nil
}

// PBLH helps fulfill the Preprocessor interface by returning
// planetary boundary layer height [m].
func (w *WRFChem) PBLH() NextData { return w.read("PBLH") }

// Height helps fulfill the Preprocessor interface by returning
// layer heights above ground level calculated based on geopotential height.
// For more information, refer to
// http://www.openwfm.org/wiki/How_to_interpret_WRF_variables.
func (w *WRFChem) Height() NextData {
	// ph is perturbation geopotential height [m2/s].
	phFunc := w.read("PH")
	// phb is baseline geopotential height [m2/s].
	phbFunc := w.read("PHB")
	return func() (*sparse.DenseArray, error) {
		ph, err := phFunc()
		if err != nil {
			return nil, err
		}
		phb, err := phbFunc()
		if err != nil {
			return nil, err
		}
		return geopotentialToHeight(ph, phb), nil
	}
}

func geopotentialToHeight(ph, phb *sparse.DenseArray) *sparse.DenseArray {
	layerHeights := sparse.ZerosDense(ph.Shape...)
	for k := 0; k < ph.Shape[0]; k++ {
		for j := 0; j < ph.Shape[1]; j++ {
			for i := 0; i < ph.Shape[2]; i++ {
				h := (ph.Get(k, j, i) + phb.Get(k, j, i) -
					ph.Get(0, j, i) - phb.Get(0, j, i)) / g // m
				layerHeights.Set(h, k, j, i)
			}
		}
	}
	return layerHeights
}

// ALT helps fulfill the Preprocessor interface by returning
// inverse air density [m3/kg].
func (w *WRFChem) ALT() NextData { return w.read("ALT") }

// U helps fulfill the Preprocessor interface by returning
// West-East wind speed [m/s].
func (w *WRFChem) U() NextData { return w.read("U") }

// V helps fulfill the Preprocessor interface by returning
// South-North wind speed [m/s].
func (w *WRFChem) V() NextData { return w.read("V") }

// W helps fulfill the Preprocessor interface by returning
// below-above wind speed [m/s].
func (w *WRFChem) W() NextData { return w.read("W") }

// AVOC helps fulfill the Preprocessor interface.
func (w *WRFChem) AVOC() NextData { return w.readGroupAlt(w.aVOC) }

// BVOC helps fulfill the Preprocessor interface.
func (w *WRFChem) BVOC() NextData { return w.readGroupAlt(w.bVOC) }

// NOx helps fulfill the Preprocessor interface.
func (w *WRFChem) NOx() NextData { return w.readGroupAlt(w.nox) }

// SOx helps fulfill the Preprocessor interface.
func (w *WRFChem) SOx() NextData { return w.readGroupAlt(w.sox) }

// NH3 helps fulfill the Preprocessor interface.
func (w *WRFChem) NH3() NextData { return w.readGroupAlt(w.nh3) }

// ASOA helps fulfill the Preprocessor interface.
func (w *WRFChem) ASOA() NextData { return w.readGroupAlt(w.aSOA) }

// BSOA helps fulfill the Preprocessor interface.
func (w *WRFChem) BSOA() NextData { return w.readGroupAlt(w.bSOA) }

// PNO helps fulfill the Preprocessor interface.
func (w *WRFChem) PNO() NextData { return w.readGroupAlt(w.pNO) }

// PS helps fulfill the Preprocessor interface.
func (w *WRFChem) PS() NextData { return w.readGroupAlt(w.pS) }

// PNH helps fulfill the Preprocessor interface.
func (w *WRFChem) PNH() NextData { return w.readGroupAlt(w.pNH) }

// TotalPM25 helps fulfill the Preprocessor interface.
func (w *WRFChem) TotalPM25() NextData { return w.readGroup(w.totalPM25) }

// SurfaceHeatFlux helps fulfill the Preprocessor interface
// by returning heat flux at the surface [W/m2].
func (w *WRFChem) SurfaceHeatFlux() NextData { return w.read("HFX") }

// UStar helps fulfill the Preprocessor interface
// by returning friction velocity [m/s].
func (w *WRFChem) UStar() NextData { return w.read("UST") }

// T helps fulfill the Preprocessor interface by
// returning temperature [K].
func (w *WRFChem) T() NextData {
	thetaFunc := w.read("T") // perturbation potential temperature [K]
	pFunc := w.P()           // Pressure [Pa]
	return wrfTemperatureConvert(thetaFunc, pFunc)
}

func wrfTemperatureConvert(thetaFunc, pFunc NextData) NextData {
	return func() (*sparse.DenseArray, error) {
		thetaPerturb, err := thetaFunc() // perturbation potential temperature [K]
		if err != nil {
			return nil, err
		}
		p, err := pFunc() // Pressure [Pa]
		if err != nil {
			return nil, err
		}

		T := sparse.ZerosDense(thetaPerturb.Shape...)
		for i, tp := range thetaPerturb.Elements {
			T.Elements[i] = thetaPerturbToTemperature(tp, p.Elements[i])
		}
		return T, nil
	}
}

// thetaPerturbToTemperature converts perburbation potential temperature
// to ambient temperature for the given pressure (p [Pa]).
func thetaPerturbToTemperature(thetaPerturb, p float64) float64 {
	const (
		po    = 101300. // Pa, reference pressure
		kappa = 0.2854  // related to von karman's constant
	)
	pressureCorrection := math.Pow(p/po, kappa)
	// potential temperature, K
	θ := thetaPerturb + 300.
	// Ambient temperature, K
	return θ * pressureCorrection
}

// P helps fulfill the Preprocessor interface
// by returning pressure [Pa].
func (w *WRFChem) P() NextData {
	pbFunc := w.read("PB") // baseline pressure [Pa]
	pFunc := w.read("P")   // perturbation pressure [Pa]
	return wrfPressureConvert(pFunc, pbFunc)
}

func wrfPressureConvert(pFunc, pbFunc NextData) NextData {
	return func() (*sparse.DenseArray, error) {
		pb, err := pbFunc() // baseline pressure [Pa]
		if err != nil {
			return nil, err
		}
		p, err := pFunc() // perturbation pressure [Pa]
		if err != nil {
			return nil, err
		}
		P := pb.Copy()
		P.AddDense(p)
		return P, nil
	}
}

// HO helps fulfill the Preprocessor interface
// by returning hydroxyl radical concentration [ppmv].
func (w *WRFChem) HO() NextData { return w.read("ho") }

// H2O2 helps fulfill the Preprocessor interface
// by returning hydrogen peroxide concentration [ppmv].
func (w *WRFChem) H2O2() NextData { return w.read("h2o2") }

// SeinfeldLandUse helps fulfill the Preprocessor interface
// by returning land use categories as
// specified in github.com/ctessum/atmos/seinfeld.
func (w *WRFChem) SeinfeldLandUse() NextData {
	luFunc := w.read("LU_INDEX") // USGS land use index
	return wrfSeinfeldLandUse(luFunc)
}

func wrfSeinfeldLandUse(luFunc NextData) NextData {
	return func() (*sparse.DenseArray, error) {
		lu, err := luFunc() // USGS land use index
		if err != nil {
			return nil, err
		}
		o := sparse.ZerosDense(lu.Shape...)
		for j := 0; j < lu.Shape[0]; j++ {
			for i := 0; i < lu.Shape[1]; i++ {
				o.Set(float64(USGSseinfeld[f2i(lu.Get(j, i))]), j, i)
			}
		}
		return o, nil
	}
}

// USGSseinfeld lookup table to go from USGS land classes to land classes for
// particle dry deposition.
var USGSseinfeld = []seinfeld.LandUseCategory{
	seinfeld.Desert,    //'Urban and Built-Up Land'
	seinfeld.Grass,     //'Dryland Cropland and Pasture'
	seinfeld.Grass,     //'Irrigated Cropland and Pasture'
	seinfeld.Grass,     //'Mixed Dryland/Irrigated Cropland and Pasture'
	seinfeld.Grass,     //'Cropland/Grassland Mosaic'
	seinfeld.Grass,     //'Cropland/Woodland Mosaic'
	seinfeld.Grass,     //'Grassland'
	seinfeld.Shrubs,    //'Shrubland'
	seinfeld.Shrubs,    //'Mixed Shrubland/Grassland'
	seinfeld.Grass,     //'Savanna'
	seinfeld.Deciduous, //'Deciduous Broadleaf Forest'
	seinfeld.Evergreen, //'Deciduous Needleleaf Forest'
	seinfeld.Deciduous, //'Evergreen Broadleaf Forest'
	seinfeld.Evergreen, //'Evergreen Needleleaf Forest'
	seinfeld.Deciduous, //'Mixed Forest'
	seinfeld.Desert,    //'Water Bodies'
	seinfeld.Grass,     //'Herbaceous Wetland'
	seinfeld.Deciduous, //'Wooded Wetland'
	seinfeld.Desert,    //'Barren or Sparsely Vegetated'
	seinfeld.Shrubs,    //'Herbaceous Tundra'
	seinfeld.Deciduous, //'Wooded Tundra'
	seinfeld.Shrubs,    //'Mixed Tundra'
	seinfeld.Desert,    //'Bare Ground Tundra'
	seinfeld.Desert,    //'Snow or Ice'
	seinfeld.Desert,    //'Playa'
	seinfeld.Desert,    //'Lava'
	seinfeld.Desert,    //'White Sand'
}

// WeselyLandUse helps fulfill the Preprocessor interface
// by returning land use categories as
// specified in github.com/ctessum/atmos/wesely1989.
func (w *WRFChem) WeselyLandUse() NextData {
	luFunc := w.read("LU_INDEX") // USGS land use index
	return wrfWeselyLandUse(luFunc)
}

func wrfWeselyLandUse(luFunc NextData) NextData {
	return func() (*sparse.DenseArray, error) {
		lu, err := luFunc() // USGS land use index
		if err != nil {
			return nil, err
		}
		o := sparse.ZerosDense(lu.Shape...)
		for j := 0; j < lu.Shape[0]; j++ {
			for i := 0; i < lu.Shape[1]; i++ {
				o.Set(float64(USGSwesely[f2i(lu.Get(j, i))]), j, i)
			}
		}
		return o, nil
	}
}

// USGSwesely lookup table to go from USGS land classes to land classes for
// gas dry deposition.
var USGSwesely = []wesely1989.LandUseCategory{
	wesely1989.Urban,        //'Urban and Built-Up Land'
	wesely1989.RangeAg,      //'Dryland Cropland and Pasture'
	wesely1989.RangeAg,      //'Irrigated Cropland and Pasture'
	wesely1989.RangeAg,      //'Mixed Dryland/Irrigated Cropland and Pasture'
	wesely1989.RangeAg,      //'Cropland/Grassland Mosaic'
	wesely1989.Agricultural, //'Cropland/Woodland Mosaic'
	wesely1989.Range,        //'Grassland'
	wesely1989.RockyShrubs,  //'Shrubland'
	wesely1989.RangeAg,      //'Mixed Shrubland/Grassland'
	wesely1989.Range,        //'Savanna'
	wesely1989.Deciduous,    //'Deciduous Broadleaf Forest'
	wesely1989.Coniferous,   //'Deciduous Needleleaf Forest'
	wesely1989.Deciduous,    //'Evergreen Broadleaf Forest'
	wesely1989.Coniferous,   //'Evergreen Needleleaf Forest'
	wesely1989.MixedForest,  //'Mixed Forest'
	wesely1989.Water,        //'Water Bodies'
	wesely1989.Wetland,      //'Herbaceous Wetland'
	wesely1989.Wetland,      //'Wooded Wetland'
	wesely1989.Barren,       //'Barren or Sparsely Vegetated'
	wesely1989.RockyShrubs,  //'Herbaceous Tundra'
	wesely1989.MixedForest,  //'Wooded Tundra'
	wesely1989.RockyShrubs,  //'Mixed Tundra'
	wesely1989.Barren,       //'Bare Ground Tundra'
	wesely1989.Barren,       //'Snow or Ice'
	wesely1989.Barren,       //'Playa'
	wesely1989.Barren,       //'Lava'
	wesely1989.Barren,       //'White Sand'
}

// Z0 helps fulfill the Preprocessor interface by
// returning roughness length.
func (w *WRFChem) Z0() NextData {
	LUIndexFunc := w.read("LU_INDEX") //USGS land use index
	return wrfZ0(LUIndexFunc)
}

func wrfZ0(LUIndexFunc NextData) NextData {
	return func() (*sparse.DenseArray, error) {
		luIndex, err := LUIndexFunc()
		if err != nil {
			return nil, err
		}
		zo := sparse.ZerosDense(luIndex.Shape...)
		for i, lu := range luIndex.Elements {
			zo.Elements[i] = USGSz0[f2i(lu)] // roughness length [m]
		}
		return zo, nil
	}
}

// USGSz0 holds Roughness lengths for USGS land classes ([m]), from WRF file
// VEGPARM.TBL.
var USGSz0 = []float64{.50, .1, .06, .1, 0.095, .20, .11,
	.03, .035, .15, .50, .50, .50, .50, .35, 0.0001, .20, .40,
	.01, .10, .30, .15, .075, 0.001, .01, .15, .01}

// QRain helps fulfill the Preprocessor interface by
// returning rain mass fraction.
func (w *WRFChem) QRain() NextData { return w.read("QRAIN") }

// CloudFrac helps fulfill the Preprocessor interface
// by returning the fraction of each grid cell filled
// with clouds [volume/volume].
func (w *WRFChem) CloudFrac() NextData { return w.read("CLDFRA") }

// QCloud helps fulfill the Preprocessor interface by returning
// the mass fraction of cloud water in each grid cell [mass/mass].
func (w *WRFChem) QCloud() NextData { return w.read("QCLOUD") }

// RadiationDown helps fulfill the Preprocessor interface by returning
// total downwelling radiation at ground level [W/m2].
func (w *WRFChem) RadiationDown() NextData {
	swDownFunc := w.read("SWDOWN") // downwelling short wave radiation at ground level [W/m2]
	glwFunc := w.read("GLW")       // downwelling long wave radiation at ground level [W/m2]
	return wrfRadiationDown(swDownFunc, glwFunc)
}

func wrfRadiationDown(swDownFunc, glwFunc NextData) NextData {
	return func() (*sparse.DenseArray, error) {
		swDown, err := swDownFunc() // downwelling short wave radiation at ground level [W/m2]
		if err != nil {
			return nil, err
		}
		glw, err := glwFunc() // downwelling long wave radiation at ground level [W/m2]
		if err != nil {
			return nil, err
		}
		rad := swDown.Copy()
		rad.AddDense(glw)
		return rad, nil
	}
}

// SWDown helps fulfill the Preprocessor interface by returning
// downwelling short wave radiation at ground level [W/m2].
func (w *WRFChem) SWDown() NextData { return w.read("SWDOWN") }

// GLW helps fulfill the Preprocessor interface by returning
// downwelling long wave radiation at ground level [W/m2].
func (w *WRFChem) GLW() NextData { return w.read("GLW") }
