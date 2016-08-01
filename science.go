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

import "github.com/ctessum/atmos/advect"

// Mixing returns a function that calculates vertical mixing based on Pleim (2007), which is
// combined local-nonlocal closure scheme, for
// boundary layer and based on Wilson (2004) for above the boundary layer.
// Also calculate horizontal mixing.
func Mixing() CellManipulator {
	return func(c *Cell, Δt float64) {
		for ii := range c.Cf {
			// Pleim (2007) Equation 10.
			for g := c.groundLevel.first; g != nil; g = g.next { // Upward convection
				c.Cf[ii] += c.M2u * g.Ci[ii] * Δt * g.info.coverFrac
			}
			for a := c.above.first; a != nil; a = a.next {
				// Convection balancing downward mixing
				c.Cf[ii] += (a.M2d*a.Ci[ii]*a.Dz/c.Dz - c.M2d*c.Ci[ii]) *
					Δt * a.info.coverFrac
				// Mixing with above
				c.Cf[ii] += 1. / c.Dz * (a.info.diff * (a.Ci[ii] - c.Ci[ii]) /
					a.info.centerDistance) * Δt * a.info.coverFrac
			}
			for b := c.below.first; b != nil; b = b.next { // Mixing with below
				c.Cf[ii] += 1. / c.Dz * (b.info.diff * (b.Ci[ii] - c.Ci[ii]) /
					b.info.centerDistance) * Δt * b.info.coverFrac
			}
			// Horizontal mixing
			for w := c.west.first; w != nil; w = w.next { // Mixing with West
				flux := 1. / c.Dx * (w.info.diff *
					(w.Ci[ii] - c.Ci[ii]) / w.info.centerDistance) * Δt * w.info.coverFrac
				c.Cf[ii] += flux * w.Dz / c.Dz
				if w.boundary { // keep track of mass that leaves the domain.
					w.Cf[ii] -= flux * c.Volume / w.Volume
				}
			}
			for e := c.east.first; e != nil; e = e.next { // Mixing with East
				flux := 1. / c.Dx * (e.info.diff *
					(e.Ci[ii] - c.Ci[ii]) / e.info.centerDistance) * Δt * e.info.coverFrac
				c.Cf[ii] += flux
				if e.boundary { // keep track of mass that leaves the domain.
					e.Cf[ii] -= flux * c.Volume / e.Volume
				}
			}
			for s := c.south.first; s != nil; s = s.next { // Mixing with South
				flux := 1. / c.Dy * (s.info.diff *
					(s.Ci[ii] - c.Ci[ii]) / s.info.centerDistance) * Δt * s.info.coverFrac
				c.Cf[ii] += flux * s.Dz / c.Dz
				if s.boundary { // keep track of mass that leaves the domain.
					s.Cf[ii] -= flux * c.Volume / s.Volume
				}
			}
			for n := c.north.first; n != nil; n = n.next { // Mixing with North
				flux := 1. / c.Dy * (n.info.diff *
					(n.Ci[ii] - c.Ci[ii]) / n.info.centerDistance) * Δt * n.info.coverFrac
				c.Cf[ii] += flux
				if n.boundary { // keep track of mass that leaves the domain.
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
			for w := c.west.first; w != nil; w = w.next {
				flux := advect.UpwindFlux(c.UAvg, w.Ci[ii], c.Ci[ii], c.Dx) *
					w.info.coverFrac * Δt
				// Multiply by Dz ratio to correct for differences in cell heights.
				c.Cf[ii] += flux * w.Dz / c.Dz
				if w.boundary { // keep track of mass that leaves the domain.
					w.Cf[ii] -= flux * c.Volume / w.Volume
				}
			}

			for e := c.east.first; e != nil; e = e.next {
				flux := advect.UpwindFlux(e.UAvg, c.Ci[ii], e.Ci[ii], c.Dx) *
					e.info.coverFrac * Δt
				c.Cf[ii] -= flux
				if e.boundary { // keep track of mass that leaves the domain.
					e.Cf[ii] += flux * c.Volume / e.Volume
				}
			}

			for s := c.south.first; s != nil; s = s.next {
				flux := advect.UpwindFlux(c.VAvg, s.Ci[ii], c.Ci[ii], c.Dy) *
					s.info.coverFrac * Δt
				// Multiply by Dz ratio to correct for differences in cell heights.
				c.Cf[ii] += flux * s.Dz / c.Dz
				if s.boundary { // keep track of mass that leaves the domain.
					s.Cf[ii] -= flux * c.Volume / s.Volume
				}
			}

			for n := c.north.first; n != nil; n = n.next {
				flux := advect.UpwindFlux(n.VAvg, c.Ci[ii], n.Ci[ii], c.Dy) *
					n.info.coverFrac * Δt
				c.Cf[ii] -= flux
				if n.boundary { // keep track of mass that leaves the domain.
					n.Cf[ii] += flux * c.Volume / n.Volume
				}
			}

			for b := c.below.first; b != nil; b = b.next {
				if c.Layer > 0 {
					flux := advect.UpwindFlux(c.WAvg, b.Ci[ii], c.Ci[ii], c.Dz) *
						b.info.coverFrac * Δt
					// Multiply by Dz ratio to correct for differences in cell heights.
					c.Cf[ii] += flux
				}
			}

			for a := c.above.first; a != nil; a = a.next {
				flux := advect.UpwindFlux(a.WAvg, c.Ci[ii], a.Ci[ii], c.Dz) *
					a.info.coverFrac * Δt
				c.Cf[ii] -= flux
				if a.boundary { // keep track of mass that leaves the domain.
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

			for w := c.west.first; w != nil; w = w.next { // Mixing with West
				flux := 1. / c.Dx * c.UDeviation *
					(w.Ci[ii] - c.Ci[ii]) * Δt * w.info.coverFrac
				// Multiply by Dz ratio to correct for differences in cell heights.
				c.Cf[ii] += flux * w.Dz / c.Dz
				if w.boundary {
					w.Cf[ii] -= flux * c.Volume / w.Volume
				}
			}
			for e := c.east.first; e != nil; e = e.next { // Mixing with East
				flux := 1. / c.Dx * (e.UDeviation *
					(e.Ci[ii] - c.Ci[ii])) * Δt * e.info.coverFrac
				c.Cf[ii] += flux
				if e.boundary {
					e.Cf[ii] -= flux * c.Volume / e.Volume
				}
			}
			for s := c.south.first; s != nil; s = s.next { // Mixing with South
				flux := 1. / c.Dy * (c.VDeviation *
					(s.Ci[ii] - c.Ci[ii])) * Δt * s.info.coverFrac
				c.Cf[ii] += flux * s.Dz / c.Dz
				if s.boundary {
					s.Cf[ii] -= flux * c.Volume / s.Volume
				}
			}
			for n := c.north.first; n != nil; n = n.next { // Mixing with North
				flux := 1. / c.Dy * (n.VDeviation *
					(n.Ci[ii] - c.Ci[ii])) * Δt * n.info.coverFrac
				c.Cf[ii] += flux
				if n.boundary {
					n.Cf[ii] -= flux * c.Volume / n.Volume
				}
			}
		}
	}
}

// Chemistry returns a function that calculates the secondary formation of PM2.5.
// It explicitly calculates formation of particulate sulfate
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
