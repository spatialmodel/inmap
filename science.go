/*
Copyright (C) 2013-2014 Regents of the University of Minnesota.
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

import "github.com/ctessum/atmos/advect"

// Mixing returns a function that calculates vertical mixing based on Pleim (2007), which is
// combined local-nonlocal closure scheme, for
// boundary layer and based on Wilson (2004) for above the boundary layer.
// Also calculate horizontal mixing.
func Mixing() CellManipulator {
	return func(c *Cell, Δt float64) {
		for ii := range c.Cf {
			// Pleim (2007) Equation 10.
			for i, g := range c.GroundLevel { // Upward convection
				c.Cf[ii] += c.M2u * g.Ci[ii] * Δt * c.GroundLevelFrac[i]
			}
			for i, a := range c.Above {
				// Convection balancing downward mixing
				c.Cf[ii] += (a.M2d*a.Ci[ii]*a.Dz/c.Dz - c.M2d*c.Ci[ii]) *
					Δt * c.AboveFrac[i]
				// Mixing with above
				c.Cf[ii] += 1. / c.Dz * (c.KzzAbove[i] * (a.Ci[ii] - c.Ci[ii]) /
					c.DzPlusHalf[i]) * Δt * c.AboveFrac[i]
			}
			for i, b := range c.Below { // Mixing with below
				c.Cf[ii] += 1. / c.Dz * (c.KzzBelow[i] * (b.Ci[ii] - c.Ci[ii]) /
					c.DzMinusHalf[i]) * Δt * c.BelowFrac[i]
			}
			// Horizontal mixing
			for i, w := range c.West { // Mixing with West
				flux := 1. / c.Dx * (c.KxxWest[i] *
					(w.Ci[ii] - c.Ci[ii]) / c.DxMinusHalf[i]) * Δt * c.WestFrac[i]
				c.Cf[ii] += flux * w.Dz / c.Dz
				if w.Boundary { // keep track of mass that leaves the domain.
					w.Cf[ii] -= flux * c.Volume / w.Volume
				}
			}
			for i, e := range c.East { // Mixing with East
				flux := 1. / c.Dx * (c.KxxEast[i] *
					(e.Ci[ii] - c.Ci[ii]) / c.DxPlusHalf[i]) * Δt * c.EastFrac[i]
				c.Cf[ii] += flux
				if e.Boundary { // keep track of mass that leaves the domain.
					e.Cf[ii] -= flux * c.Volume / e.Volume
				}
			}
			for i, s := range c.South { // Mixing with South
				flux := 1. / c.Dy * (c.KyySouth[i] *
					(s.Ci[ii] - c.Ci[ii]) / c.DyMinusHalf[i]) * Δt * c.SouthFrac[i]
				c.Cf[ii] += flux * s.Dz / c.Dz
				if s.Boundary { // keep track of mass that leaves the domain.
					s.Cf[ii] -= flux * c.Volume / s.Volume
				}
			}
			for i, n := range c.North { // Mixing with North
				flux := 1. / c.Dy * (c.KyyNorth[i] *
					(n.Ci[ii] - c.Ci[ii]) / c.DyPlusHalf[i]) * Δt * c.NorthFrac[i]
				c.Cf[ii] += flux
				if n.Boundary { // keep track of mass that leaves the domain.
					n.Cf[ii] -= flux * c.Volume / n.Volume
				}
			}
		}
	}
}

// UpwindAdvection returns a function that calculates advection in the cell based
// on the upwind differences scheme.
func UpwindAdvection() CellManipulator {
	return func(c *Cell, Δt float64) {
		for ii := range c.Cf {
			for i, w := range c.West {
				flux := advect.UpwindFlux(c.UAvg, w.Ci[ii], c.Ci[ii], c.Dx) *
					c.WestFrac[i] * Δt
				// Multiply by Dz ratio to correct for differences in cell heights.
				c.Cf[ii] += flux * w.Dz / c.Dz
				if w.Boundary { // keep track of mass that leaves the domain.
					w.Cf[ii] -= flux * c.Volume / w.Volume
				}
			}

			for i, e := range c.East {
				flux := advect.UpwindFlux(e.UAvg, c.Ci[ii], e.Ci[ii], c.Dx) *
					c.EastFrac[i] * Δt
				c.Cf[ii] -= flux
				if e.Boundary { // keep track of mass that leaves the domain.
					e.Cf[ii] += flux * c.Volume / e.Volume
				}
			}

			for i, s := range c.South {
				flux := advect.UpwindFlux(c.VAvg, s.Ci[ii], c.Ci[ii], c.Dy) *
					c.SouthFrac[i] * Δt
				// Multiply by Dz ratio to correct for differences in cell heights.
				c.Cf[ii] += flux * s.Dz / c.Dz
				if s.Boundary { // keep track of mass that leaves the domain.
					s.Cf[ii] -= flux * c.Volume / s.Volume
				}
			}

			for i, n := range c.North {
				flux := advect.UpwindFlux(n.VAvg, c.Ci[ii], n.Ci[ii], c.Dy) *
					c.NorthFrac[i] * Δt
				c.Cf[ii] -= flux
				if n.Boundary { // keep track of mass that leaves the domain.
					n.Cf[ii] += flux * c.Volume / n.Volume
				}
			}

			for i, b := range c.Below {
				if c.Layer > 0 {
					flux := advect.UpwindFlux(c.WAvg, b.Ci[ii], c.Ci[ii], c.Dz) *
						c.BelowFrac[i] * Δt
					// Multiply by Dz ratio to correct for differences in cell heights.
					c.Cf[ii] += flux
				}
			}

			for i, a := range c.Above {
				flux := advect.UpwindFlux(a.WAvg, c.Ci[ii], a.Ci[ii], c.Dz) *
					c.AboveFrac[i] * Δt
				c.Cf[ii] -= flux
				if a.Boundary { // keep track of mass that leaves the domain.
					a.Cf[ii] += flux * c.Volume / a.Volume
				}
			}

		}
	}
}

// MeanderMixing returns a function that calculates changes in concentrations caused by meanders:
// adevection that is resolved by the underlying comprehensive chemical
// transport model but is not resolved by InMAP.
func MeanderMixing() CellManipulator {
	return func(c *Cell, Δt float64) {
		for ii := range c.Ci {

			for i, w := range c.West { // Mixing with West
				flux := 1. / c.Dx * c.UDeviation *
					(w.Ci[ii] - c.Ci[ii]) * Δt * c.WestFrac[i]
				// Multiply by Dz ratio to correct for differences in cell heights.
				c.Cf[ii] += flux * w.Dz / c.Dz
				if w.Boundary {
					w.Cf[ii] -= flux * c.Volume / w.Volume
				}
			}
			for i, e := range c.East { // Mixing with East
				flux := 1. / c.Dx * (e.UDeviation *
					(e.Ci[ii] - c.Ci[ii])) * Δt * c.EastFrac[i]
				c.Cf[ii] += flux
				if e.Boundary {
					e.Cf[ii] -= flux * c.Volume / e.Volume
				}
			}
			for i, s := range c.South { // Mixing with South
				flux := 1. / c.Dy * (c.VDeviation *
					(s.Ci[ii] - c.Ci[ii])) * Δt * c.SouthFrac[i]
				c.Cf[ii] += flux * s.Dz / c.Dz
				if s.Boundary {
					s.Cf[ii] -= flux * c.Volume / s.Volume
				}
			}
			for i, n := range c.North { // Mixing with North
				flux := 1. / c.Dy * (n.VDeviation *
					(n.Ci[ii] - c.Ci[ii])) * Δt * c.NorthFrac[i]
				c.Cf[ii] += flux
				if n.Boundary {
					n.Cf[ii] -= flux * c.Volume / n.Volume
				}
			}
		}
	}
}

// Chemistry returns a function that calculates the secondary formation of PM2.5.
// It explicitely calculates formation of particulate sulfate
// from gaseous and aqueous SO2.
// It partitions organic matter ("gOrg" and "pOrg"), the
// nitrogen in nitrate ("gNO and pNO"), and the nitrogen in ammonia ("gNH" and
// "pNH) between gaseous and particulate phase
// based on the spatially explicit partioning present in the baseline data.
func Chemistry() CellManipulator {
	return func(c *Cell, Δt float64) {
		// All SO4 forms particles, so sulfur particle formation is limited by the
		// SO2 -> SO4 reaction.
		ΔS := c.SO2oxidation * c.Cf[igS] * Δt
		c.Cf[ipS] += ΔS
		c.Cf[igS] -= ΔS
		// NH3 / pNH4 partitioning
		totalNH := c.Cf[igNH] + c.Cf[ipNH]
		c.Cf[ipNH] = totalNH * c.NHPartitioning
		c.Cf[igNH] = totalNH * (1 - c.NHPartitioning)

		// NOx / pN0 partitioning
		totalNO := c.Cf[igNO] + c.Cf[ipNO]
		c.Cf[ipNO] = totalNO * c.NOPartitioning
		c.Cf[igNO] = totalNO * (1 - c.NOPartitioning)

		// VOC/SOA partitioning
		totalOrg := c.Cf[igOrg] + c.Cf[ipOrg]
		c.Cf[ipOrg] = totalOrg * c.AOrgPartitioning
		c.Cf[igOrg] = totalOrg * (1 - c.AOrgPartitioning)
	}
}

// DryDeposition returns a function that calculates particle removal by dry deposition.
func DryDeposition() CellManipulator {
	return func(c *Cell, Δt float64) {
		if c.Layer == 0 {
			fac := 1. / c.Dz * Δt
			noxfac := c.NOxDryDep * fac
			so2fac := c.SO2DryDep * fac
			vocfac := c.VOCDryDep * fac
			nh3fac := c.NH3DryDep * fac
			pm25fac := c.ParticleDryDep * fac
			c.Cf[igOrg] -= c.Ci[igOrg] * vocfac
			c.Cf[ipOrg] -= c.Ci[ipOrg] * pm25fac
			c.Cf[iPM2_5] -= c.Ci[iPM2_5] * pm25fac
			c.Cf[igNH] -= c.Ci[igNH] * nh3fac
			c.Cf[ipNH] -= c.Ci[ipNH] * pm25fac
			c.Cf[igS] -= c.Ci[igS] * so2fac
			c.Cf[ipS] -= c.Ci[ipS] * pm25fac
			c.Cf[igNO] -= c.Ci[igNO] * noxfac
			c.Cf[ipNO] -= c.Ci[ipNO] * pm25fac
		}
	}
}

// WetDeposition returns a function that calculates particle removal by wet deposition.
func WetDeposition() CellManipulator {
	return func(c *Cell, Δt float64) {
		particleFrac := c.ParticleWetDep * Δt
		SO2Frac := c.SO2WetDep * Δt
		otherGasFrac := c.OtherGasWetDep * Δt
		c.Cf[igOrg] -= c.Ci[igOrg] * otherGasFrac
		c.Cf[ipOrg] -= c.Ci[ipOrg] * particleFrac
		c.Cf[iPM2_5] -= c.Ci[iPM2_5] * particleFrac
		c.Cf[igNH] -= c.Ci[igNH] * otherGasFrac
		c.Cf[ipNH] -= c.Ci[ipNH] * particleFrac
		c.Cf[igS] -= c.Ci[igS] * SO2Frac
		c.Cf[ipS] -= c.Ci[ipS] * particleFrac
		c.Cf[igNO] -= c.Ci[igNO] * otherGasFrac
		c.Cf[ipNO] -= c.Ci[ipNO] * particleFrac
	}
}

func max(vals ...float64) float64 {
	m := vals[0]
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	return m
}
func min(v1, v2 float64) float64 {
	if v1 < v2 {
		return v1
	}
	return v2
}
func amin(vals ...float64) float64 {
	m := vals[0]
	for _, v := range vals {
		if v < m {
			m = v
		}
	}
	return m
}
