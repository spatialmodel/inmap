package inmap

// IntakeFraction calculates intake fraction from InMAP results.
// The input value is average breathing rate [m³/day].
// The returned value is a map structure of intake fractions by
// pollutant and population type (map[pollutant][population]iF).
// This function will only give the correct results if run
// after InMAP finishes calculating.
func (d *InMAPdata) IntakeFraction(
	breathingRate float64) map[string]map[string]float64 {

	Qb := breathingRate / (24 * 60 * 60) // [m³/s]

	iF := make(map[string]map[string]float64)

	for l, ie := range emisLabels {
		iF[l] = make(map[string]float64)
		ic := gasParticleMap[ie]
		for p := range popNames {
			erate := 0. // emissions rate [μg/s]
			irate := 0. // inhalation rate [μg/s]
			for _, c := range d.Data {
				erate += c.emisFlux[ie] * c.Volume
				if c.Layer == 0 { // We only care about ground level concentrations
					irate += c.Cf[ic] * Qb * c.PopData[p]
				}
			}
			// Intake fraction is the rate of intake divided by
			// the rate of emission
			iF[l][p] = irate / erate
		}
	}
	return iF
}
