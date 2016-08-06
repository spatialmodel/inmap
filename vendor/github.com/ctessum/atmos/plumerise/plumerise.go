package plumerise

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

const (
	g = 9.80665 // m/s2
)

// ASME takes emissions stack height(m), diameter (m), temperature (K),
// and exit velocity (m/s) and calculates the k index of the equivalent
// emissions height after accounting for plume rise.
// Additional required inputs are model layer heights (staggered grid; layerHeights [m]),
// temperature at each layer [K] (unstaggered grid),
// wind speed at each layer [m/s] (unstaggered grid),
// stability class (sClass [0 or 1], unstaggered grid),
// and stability parameter (s1 [unknown units], unstaggered grid).
// Uses the plume rise calculation: ASME (1973), as described in Sienfeld and Pandis,
// ``Atmospheric Chemistry and Physics - From Air Pollution to Climate Change
func ASME(stackHeight, stackDiam, stackTemp,
	stackVel float64, layerHeights, temperature, windSpeed,
	sClass, s1 []float64) (plumeLayer int, plumeHeight float64, err error) {

	stackLayer, err := findLayer(layerHeights, stackHeight)
	if err != nil {
		return
	}
	deltaH, err := calcDeltaH(stackLayer, temperature, windSpeed, sClass, s1,
		stackHeight, stackTemp, stackVel, stackDiam)
	if err != nil {
		return
	}

	plumeHeight = stackHeight + deltaH
	plumeLayer, err = findLayer(layerHeights, plumeHeight)
	return
}

// ASMEPrecomputed is the same as ASME except it takes
// precomputed (averaged) meteorological parameters:
// the inverse of the stability parameter (s1Inverse [1/unknown units],
// unstaggered grid), windSpeedMinusOnePointFour [(m/s)^(-1.4)] (unstaggered grid),
// windSpeedMinusThird [(m/s)^(-1/3)] (unstaggered grid),
// and windSpeedInverse [(m/s)^(-1)] (unstaggered grid),
// Uses the plume rise calculation: ASME (1973), as described in Sienfeld and Pandis,
// ``Atmospheric Chemistry and Physics - From Air Pollution to Climate Change
func ASMEPrecomputed(stackHeight, stackDiam, stackTemp,
	stackVel float64, layerHeights, temperature, windSpeed,
	sClass, s1, windSpeedMinusOnePointFour, windSpeedMinusThird,
	windSpeedInverse []float64) (plumeLayer int, plumeHeight float64, err error) {

	stackLayer, err := findLayer(layerHeights, stackHeight)
	if err != nil {
		return
	}
	deltaH, err := calcDeltaHPrecomputed(stackLayer, temperature, windSpeed, sClass,
		s1, stackHeight, stackTemp, stackVel, stackDiam,
		windSpeedMinusOnePointFour, windSpeedMinusThird, windSpeedInverse)
	if err != nil {
		return
	}

	plumeHeight = stackHeight + deltaH
	plumeLayer, err = findLayer(layerHeights, plumeHeight)
	return
}

// Find K level of stack or plume
func findLayer(layerHeights []float64, stackHeight float64) (int, error) {
	stackLayer := sort.SearchFloat64s(layerHeights, stackHeight)
	if stackLayer == len(layerHeights) {
		stackLayer -= 2
		return stackLayer, ErrAboveModelTop
	}
	if stackLayer != 0 {
		stackLayer--
	}
	return stackLayer, nil
}

// calcDeltaH calculates plume rise (ASME, 1973).
func calcDeltaH(stackLayer int, temperature, windSpeed, sClass, s1 []float64,
	stackHeight, stackTemp, stackVel, stackDiam float64) (float64, error) {
	deltaH := 0. // Plume rise, (m).

	airTemp := temperature[stackLayer]
	windSpd := windSpeed[stackLayer]

	if (stackTemp-airTemp) < 50. &&
		stackVel > windSpd && stackVel > 10. {

		// Plume is dominated by momentum forces
		deltaH = stackDiam * math.Pow(stackVel, 1.4) / math.Pow(windSpd, 1.4)

	} else { // Plume is dominated by buoyancy forces

		// Bouyancy flux, m4/s3
		F := g * (stackTemp - airTemp) / stackTemp * stackVel *
			math.Pow(stackDiam/2, 2)

		if sClass[stackLayer] > 0.5 { // stable conditions

			deltaH = 29. * math.Pow(
				F/s1[stackLayer], 0.333333333) /
				math.Pow(windSpd, 0.333333333)

		} else { // unstable conditions

			deltaH = 7.4 * math.Pow(F*math.Pow(stackHeight, 2.),
				0.333333333) / windSpd
		}
	}
	if math.IsNaN(deltaH) {
		err := fmt.Errorf("plume height == NaN\n"+
			"deltaH: %v, stackDiam: %v,\n"+
			"stackVel: %v, windSpd: %v, stackTemp: %v,\n"+
			"airTemp: %v, stackHeight: %v\n",
			deltaH, stackDiam, stackVel,
			windSpd, stackTemp, airTemp, stackHeight)
		return deltaH, err
	}
	return deltaH, nil
}

// calcDeltaHPrecomputed calculates plume rise, the same as calcDeltaH,
// (ASME, 1973), except that it uses precomputed meteorological parameters.
func calcDeltaHPrecomputed(stackLayer int, temperature, windSpeed, sClass,
	s1 []float64,
	stackHeight, stackTemp, stackVel, stackDiam float64,
	windSpeedMinusOnePointFour, windSpeedMinusThird,
	windSpeedInverse []float64) (float64, error) {

	deltaH := 0. // Plume rise, (m).

	airTemp := temperature[stackLayer]
	windSpd := windSpeed[stackLayer]

	if (stackTemp-airTemp) < 50. &&
		stackVel > windSpd && stackVel > 10. {

		// Plume is dominated by momentum forces
		deltaH = stackDiam * math.Pow(stackVel, 1.4) *
			windSpeedMinusOnePointFour[stackLayer]

		if math.IsNaN(deltaH) {
			return deltaH, fmt.Errorf("plumerise: momentum-dominated deltaH is NaN. "+
				"stackDiam: %g, stackVel: %g, windSpeedMinusOnePointFour: %g",
				stackDiam, stackVel, windSpeedMinusOnePointFour[stackLayer])
		}

	} else { // Plume is dominated by buoyancy forces

		var tempDiff float64
		if stackTemp-airTemp == 0 {
			tempDiff = 0
		} else {
			tempDiff = 2 * (stackTemp - airTemp) / (stackTemp + airTemp)
		}

		// Bouyancy flux, m4/s3
		F := g * tempDiff * stackVel *
			math.Pow(stackDiam/2, 2)

		if sClass[stackLayer] > 0.5 && s1[stackLayer] != 0 && F > 0 { // stable conditions

			// Ideally, we would also use the inverse of S1,
			// but S1 is zero sometimes so that doesn't work well.
			deltaH = 29. * math.Pow(
				F/s1[stackLayer], 0.333333333) * windSpeedMinusThird[stackLayer]

			if math.IsNaN(deltaH) {
				return deltaH, fmt.Errorf("plumerise: stable bouyancy-dominated deltaH is NaN. "+
					"F: %g, s1: %g, windSpeedMinusThird: %g",
					F, s1[stackLayer], windSpeedMinusThird[stackLayer])
			}

		} else if F > 0. { // unstable conditions

			deltaH = 7.4 * math.Pow(F*math.Pow(stackHeight, 2.),
				0.333333333) * windSpeedInverse[stackLayer]

			if math.IsNaN(deltaH) {
				return deltaH, fmt.Errorf("plumerise: unstable bouyancy-dominated deltaH is NaN. "+
					"F: %g, stackHeight: %g, windSpeedInverse: %g",
					F, stackHeight, windSpeedInverse[stackLayer])
			}
		} else {
			// If F < 0, the unstable algorithm above will give an imaginary plume rise.
			deltaH = 0
		}
	}
	return deltaH, nil
}

// ErrAboveModelTop is returned when the plume is above the top
// model layer.
var ErrAboveModelTop = errors.New("plume rise > top of grid")
