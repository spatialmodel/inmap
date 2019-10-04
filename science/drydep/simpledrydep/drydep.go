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

// Package simpledrydep provides a atmospheric dry deposition algorithm
// for a small number of chemical species.
package simpledrydep

import (
	"math"

	"github.com/spatialmodel/inmap"
)

// SOx specifies array indicies that hold sulfur oxide concentrations.
type SOx []int

// NH3 specifies array indicies that hold ammonia concentrations.
type NH3 []int

// NOx specifies array indicies that hold oxides of Nitrogen concentrations.
type NOx []int

// VOC specifies array indicies that hold volatile organic compound concentrations.
type VOC []int

// PM25 specifies array indicies that hold fine particulate matter concentrations.
type PM25 []int

// DryDeposition returns a function that calculates particle removal by dry deposition.
// The function arguments represent array indices of the chemical species.
// Each species can be associated with more than one array index.
func DryDeposition(indices func() (SOx, NH3, NOx, VOC, PM25)) inmap.CellManipulator {
	sox, nh3, nox, voc, pm25 := indices()
	return func(c *inmap.Cell, Δt float64) {
		if c.Layer == 0 {
			fac := 1. / c.Dz * Δt
			noxfac := math.Exp(-c.NOxDryDep * fac)
			so2fac := math.Exp(-c.SO2DryDep * fac)
			vocfac := math.Exp(-c.VOCDryDep * fac)
			nh3fac := math.Exp(-c.NH3DryDep * fac)
			pm25fac := math.Exp(-c.ParticleDryDep * fac)
			for _, i := range voc {
				c.Cf[i] -= c.Ci[i] - c.Ci[i]*vocfac
			}
			for _, i := range pm25 {
				c.Cf[i] -= c.Ci[i] - c.Ci[i]*pm25fac
			}
			for _, i := range nh3 {
				c.Cf[i] -= c.Ci[i] - c.Ci[i]*nh3fac
			}
			for _, i := range sox {
				c.Cf[i] -= c.Ci[i] - c.Ci[i]*so2fac
			}
			for _, i := range nox {
				c.Cf[i] -= c.Ci[i] - c.Ci[i]*noxfac
			}
		}
	}
}
