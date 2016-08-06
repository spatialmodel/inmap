package seinfeld

import (
	"math"
)

const (
	Rkcal    = 1.9872041e-3 // [kcal K-1 mol-1] Universal gas constant
	R        = 0.08205      // [atm L mol-1 K-1] Universal gas constant
	ppbRatio = 1.e-9
)

// Seinfeld and Pandis equation 7.5. Calculates a
// temperature adjusted reaction rate
// given reaction rate at 298K (k298), the heat of dissolution
// divided by the gas constant (EperR [K]), and
// the temperature (T [K]).
// Also works for adjusting Henry's law coefficients
func TemperatureAdjustRate(k298, EperR, T float64) (k float64) {
	k = k298 * math.Exp(EperR*(1/298.-1/T))
	return
}

// Seinfeld and Pandis equation 7.7. Calculates the ratio
// of gaseous-phase mass concentration to aqueous-phase
// mass concentration per unit volume of air.
// Inputs are Henry's law coefficient
// (H [M atm-1]), temperature (T [K]), cloud/fog liquid
// water mixing ratio (wL [vol water/vol air]).
func GasLiquidDistributionFactor(H, T, wL float64) float64 {
	return H * R * T * wL
}

// Seinfeld and Pandis equation 7.84. Calculates the
// aqueous oxdation rate of SO2 [1/s] by H2O2 when given the H2O2
// mixing ratio (H2O2 [ppb]), the water droplet pH (pH [-log(M)]),
// temperature (T [K]), pressure (P [atm]), and cloud/fog
// liquid water mixing ratio (wL [vol water/vol air]).
func SulfurH2O2aqueousOxidationRate(H2O2, pH, T, P, wL float64) float64 {
	const K = 13.                  // [M-1] from text below eq. 7.84
	const Hso2at298 = 1.23         // [M atm-1] Henry's law coeff. from table 7.2
	const HperRso2 = -6.25 / Rkcal // [K] Heat of dissolution
	Hso2 := TemperatureAdjustRate(
		Hso2at298, HperRso2, T) * P * ppbRatio // [M ppb-1]
	const Hh2o2 = 1.e5                              // [M atm-1], H2O2 Henry's coeff.
	Ks1 := TemperatureAdjustRate(1.3e-2, -1960., T) // [M] Eq. 7.34
	k := TemperatureAdjustRate(7.5e7, 4430., T)     // [M-2 s-1] Table 7.A.7
	H := math.Pow(10, -pH)                          // [M] hydrogen ion conc.
	H2O2aq := H2O2 * P * ppbRatio * Hh2o2           // [M] H2O2 aq. conc.
	koverall := k * Hso2 * Ks1 * H2O2aq / (1 + K*H) // [M ppb-1 s-1] eq. 7.84
	return koverall * wL * R * T / P / ppbRatio     // [s-1]
}
