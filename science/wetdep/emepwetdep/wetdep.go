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

// Package emepwetdep provides a atmospheric wet deposition algorithm from
// the EMEP model.
package emepwetdep

import "github.com/spatialmodel/inmap"

// SO2 specifies array indicies that hold sulfur dioxide concentrations.
type SO2 []int

// OtherGas specifies array indicies that hold non-SO2 gas concentrations.
type OtherGas []int

// PM25 specifies array indicies that hold fine particulate matter concentrations.
type PM25 []int

// WetDeposition returns a function that calculates particle removal by wet deposition.
// The function arguments represent array indices of the chemical species.
// Each species can be associated with more than one array index.
func WetDeposition(indices func() (SO2, OtherGas, PM25)) inmap.CellManipulator {
	so2, otherGas, pm25 := indices()
	return func(c *inmap.Cell, Δt float64) {
		particleFrac := c.ParticleWetDep * Δt
		SO2Frac := c.SO2WetDep * Δt
		otherGasFrac := c.OtherGasWetDep * Δt
		for _, i := range so2 {
			c.Cf[i] -= c.Ci[i] * SO2Frac
		}
		for _, i := range otherGas {
			c.Cf[i] -= c.Ci[i] * otherGasFrac
		}
		for _, i := range pm25 {
			c.Cf[i] -= c.Ci[i] * particleFrac
		}
	}
}
