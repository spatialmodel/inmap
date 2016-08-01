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

// IntakeFraction calculates intake fraction from InMAP results.
// The input value is average breathing rate [m³/day].
// The returned value is a map structure of intake fractions by
// pollutant and population type (map[pollutant][population]iF).
// This function will only give the correct results if run
// after InMAP finishes calculating.
func (d *InMAP) IntakeFraction(
	breathingRate float64) map[string]map[string]float64 {

	Qb := breathingRate / (24 * 60 * 60) // [m³/s]

	iF := make(map[string]map[string]float64)

	for l, ie := range emisLabels {
		iF[l] = make(map[string]float64)
		ic := gasParticleMap[ie]
		for p, i := range d.popIndices {
			erate := 0. // emissions rate [μg/s]
			irate := 0. // inhalation rate [μg/s]
			for c := d.cells.first; c != nil; c = c.next {
				if c.EmisFlux != nil {
					erate += c.EmisFlux[ie] * c.Volume
				}
				if c.Layer == 0 { // We only care about ground level concentrations
					irate += c.Cf[ic] * Qb * c.PopData[i]
				}
			}
			// Intake fraction is the rate of intake divided by
			// the rate of emission
			iF[l][p] = irate / erate
		}
	}
	return iF
}
