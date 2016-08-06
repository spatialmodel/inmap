package wesely1989

import (
	//"fmt"
	"math"
)

type SeasonCategory int

const (
	Midsummer    SeasonCategory = iota // 0: Midsummer with lush vegetation
	Autumn                             // 1: Autumn with unharvested cropland
	LateAutumn                         // 2: Late autumn after frost, no snow
	Winter                             // 3: Winter, snow on ground and subfreezing
	Transitional                       // 4: Transitional spring with partially green short annuals
)

type LandUseCategory int

const (
	Urban        LandUseCategory = iota // 0: Urban land
	Agricultural                        // 1: Agricultural land
	Range                               // 2: Range land
	Deciduous                           // 3: Deciduous forest
	Coniferous                          // 4: Coniferous forest
	MixedForest                         // 5: Mixed forest including wetland
	Water                               // 6: Water, both salt and fresh
	Barren                              // 7: Barren land, mostly desert
	Wetland                             // 8: Nonforested wetland
	RangeAg                             // 9: Mixed agricultural and range land
	RockyShrubs                         // 10: Rocky open areas with low-growing shrubs
)

/*
Calculates surface resistance to dry depostion [s m-1] based on Wesely (1989)
equation 2 when given information on the chemical of interest (gasData),
solar irradiation (G [W m-2]), the surface air temperature (Ts [°C]),
the slope of the local terrain (Θ [radians]),
the season index (iSeason), the land use index (iLandUse), whether there is
currently rain or dew, and whether the chemical of interest is either SO2
(isSO2) or O3 (isO3).

From Wesely (1989) regarding rain and dew inputs:
	"A direct computation of the surface wetness would be most desirable, e.g.
	by estimating the amount of free surface water accumulated and then
	evaporated. Alternatively, surface relative humidity might be a useful
	index. After dewfall and rainfall events are completed, surface wetness
	often disappears as a result of evaporation after approximately 2
	hours of good atmospheric mixing, the period of time recommended earlier
	(Sheih et al., 1986)".
*/
func SurfaceResistance(gasData *GasData, G, Ts, Θ float64,
	iSeason SeasonCategory, iLandUse LandUseCategory,
	rain, dew, isSO2, isO3 bool) (r_c float64) {
	rs := r_s(G, Ts, int(iSeason), int(iLandUse), rain || dew)
	rmx := r_mx(gasData.Hstar, gasData.Fo)
	rsmx := r_smx(rs, gasData.Dh2oPerDx, rmx)
	rdc := r_dc(G, Θ)
	rlux := r_lux(gasData.Hstar, gasData.Fo, int(iSeason), int(iLandUse),
		rain, dew, isSO2, isO3)
	var rclx, rgsx float64
	switch {
	case isSO2:
		rclx = r_clS[int(iSeason)][int(iLandUse)]
		rgsx = r_gsS[int(iSeason)][int(iLandUse)]
	case isO3:
		rclx = r_clO[int(iSeason)][int(iLandUse)]
		rgsx = r_gsO[int(iSeason)][int(iLandUse)]
	default:
		rclx = r_clx(gasData.Hstar, gasData.Fo, int(iSeason), int(iLandUse))
		rgsx = r_gsx(gasData.Hstar, gasData.Fo, int(iSeason), int(iLandUse))
	}
	rac := r_ac[int(iSeason)][int(iLandUse)]

	// Correction for cold temperatures from page 4 column 1.
	if Ts < 0. {
		correction := 1000. * math.Exp(-Ts-4) // [s m-1]
		rlux += correction
		rclx += correction
		rgsx += correction
	}

	r_c = 1 / (1/(rsmx) + 1/rlux + 1/(rdc+rclx) + 1/(rac+rgsx))
	//fmt.Printf("\trs=%.0f rmx=%.0f rsmx=%.0f rlux=%.0f rdc=%.0f "+
	//	"rclx=%.0f\n\trac=%.0f rgsx=%.0f\n", rs, rmx, rsmx,
	//	rlux, rdc, rclx, rac, rgsx)
	r_c = max(r_c, 10.) // From "Results and conclusions" section
	// to avoid extremely high deposition velocities over
	// extremely rough surfaces.
	r_c = min(r_c, 9999.)
	return
}

func max(a, b float64) float64 {
	if a > b {
		return a
	} else {
		return b
	}
}
func min(a, b float64) float64 {
	if a < b {
		return a
	} else {
		return b
	}
}

// Calculate bulk canopy stomatal resistance [s m-1] based on Wesely (1989)
// equation 3 when given the solar irradiation (G [W m-2]), the
// surface air temperature (Ts [°C]), the season index (iSeason),
// the land use index (iLandUse), and whether there is currently rain or dew.
func r_s(G, Ts float64, iSeason, iLandUse int, rainOrDew bool) (rs float64) {
	if Ts >= 39.9 || Ts <= 0.1 {
		rs = inf
	} else {
		rs = r_i[iSeason][iLandUse] * (1 + math.Pow(200.*1./(G+0.1), 2.)) *
			(400. * 1. / (Ts * (40. - Ts)))
	}
	// Adjust for dew and rain (from "Effects of dew and rain" section).
	if rainOrDew {
		rs *= 3.
	}
	return
}

// Calculate combined minimum stomatal and mesophyll resistance [s m-1] based
// on Wesely (1989) equation 4 when given stomatal resistance (r_s [s m-1]),
// ratio of water to chemical-of-interest diffusivities (Dh2oPerDx [-]),
// and mesophyll resistance (r_mx [s m-1]).
func r_smx(r_s, Dh2oPerDx, r_mx float64) float64 {
	return r_s*Dh2oPerDx + r_mx
}

// Calculate the resistance from the effects of mixing forced by buoyant
// convection when sunlight heats the ground or lower canopy and by
// penetration of wind into canopies on the sides of hills [s m-1] when
// given the solar irradiation (G [W m-2]) and the slope of the local
// terrain (Θ [radians]). From Wesely (1989) equation 5.
func r_dc(G, Θ float64) float64 {
	return 100 * (1. + 1000./(G+10.)) / (1. + 1000.*Θ)
}

// Calculate mesophyll resistance [s m-1] based on Wesely (1989) equation 6
// when given the effective Henry's law coefficient (Hstar [M atm-1]) and
// the reactivity factor (fo [-]), both available in Wesely (1989) table 2.
func r_mx(Hstar, fo float64) float64 {
	return 1. / (Hstar/3000. + 100.*fo)
}

// Calculate the resistance of the outer surfaces in the upper canopy
// (leaf cuticular resistance in healthy vegetation)
// based on Wesely (1989) equations 7 and 10-14
// when given the effective Henry's law coefficient (Hstar [M atm-1]),
// the reactivity factor (fo [-]) (both available in Wesely (1989) table 2),
// the season index (iSeason), the land use index (iLandUse), whether
// there is currently rain or dew, and whether the chemical of interest
// is either SO2 or O3.
func r_lux(Hstar, fo float64, iSeason, iLandUse int,
	rain, dew, isSO2, isO3 bool) float64 {
	var rlux float64
	if dew && iSeason != 3 { // Dew doesn't have any effect in the winter
		if isSO2 {
			if iLandUse == 0 {
				rlux = 50. // equation 13 and a half
			} else {
				rlux = 100. // equation 10.
			}
		} else if isO3 {
			// equation 11
			rlux = 1. / (1./3000. + 1./(3*r_lu[iSeason][iLandUse]))
		} else {
			rluO := 1. / (1./3000. + 1./(3*r_lu[iSeason][iLandUse])) // equation 11
			rlux = 1. / (1./(3*r_lu[iSeason][iLandUse]/(1.e-5*Hstar+fo)) + 1.e-7*Hstar +
				fo/rluO) // equation 14, modified to match Walmsley eq. 5g
		}
	} else if rain && iSeason != 3 {
		if isSO2 {
			if iLandUse == 0 {
				rlux = 50 // equation 13 and a half
			} else {
				// equation 12
				rlux = 1. / (1./5000. + 1./(3*r_lu[iSeason][iLandUse]))
			}
		} else if isO3 {
			// equation 13
			rlux = 1. / (1./1000. + 1./(3*r_lu[iSeason][iLandUse]))
		} else {
			rluO := 1. / (1./1000. + 1./(3*r_lu[iSeason][iLandUse])) // equation 13
			rlux = 1. / (1./(3*r_lu[iSeason][iLandUse]/(1.e-5*Hstar+fo)) + 1.e-7*Hstar +
				fo/rluO) // equation 14, modified to match Walmsley eq. 5g
		}
	} else {
		rlux = r_lu[iSeason][iLandUse] / (1.e-5*Hstar + fo)
	}
	return rlux
}

// Calculate the resistance of the exposed surfaces in the lower portions
// of structures (canopies, buildings) above the ground based on
// Wesely (1989) equation 8
// when given the effective Henry's law coefficient (Hstar [M atm-1]),
// the reactivity factor (fo [-]) (both available in Wesely (1989) table 2),
// the season index (iSeason), and the land use index (iLandUse).
func r_clx(Hstar, fo float64, iSeason, iLandUse int) float64 {
	return 1. / (Hstar/(1.e5*r_clS[iSeason][iLandUse]) +
		fo/r_clO[iSeason][iLandUse])
}

// Calculate the resistance to uptake at the 'ground' surface based on
// Wesely (1989) equation 9
// when given the effective Henry's law coefficient (Hstar [M atm-1]),
// the reactivity factor (fo [-]) (both available in Wesely (1989) table 2),
// the season index (iSeason), and the land use index (iLandUse).
func r_gsx(Hstar, fo float64, iSeason, iLandUse int) float64 {
	return 1. / (Hstar/(1.e5*r_gsS[iSeason][iLandUse]) +
		fo/r_gsO[iSeason][iLandUse])
}
