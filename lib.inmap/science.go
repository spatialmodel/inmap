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

// Mixing calculates vertical mixing based on Pleim (2007), which is
// combined local-nonlocal closure scheme, for
// boundary layer and based on Wilson (2004) for above the boundary layer.
// Also calculate horizontal mixing.
func (c *Cell) Mixing(Δt float64) {
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
			c.Cf[ii] += 1. / c.Dx * (c.KxxWest[i] *
				(w.Ci[ii] - c.Ci[ii]) / c.DxMinusHalf[i]) * Δt * c.WestFrac[i]
		}
		for i, e := range c.East { // Mixing with East
			c.Cf[ii] += 1. / c.Dx * (c.KxxEast[i] *
				(e.Ci[ii] - c.Ci[ii]) / c.DxPlusHalf[i]) * Δt * c.EastFrac[i]
		}
		for i, s := range c.South { // Mixing with South
			c.Cf[ii] += 1. / c.Dy * (c.KyySouth[i] *
				(s.Ci[ii] - c.Ci[ii]) / c.DyMinusHalf[i]) * Δt * c.SouthFrac[i]
		}
		for i, n := range c.North { // Mixing with North
			c.Cf[ii] += 1. / c.Dy * (c.KyyNorth[i] *
				(n.Ci[ii] - c.Ci[ii]) / c.DyPlusHalf[i]) * Δt * c.NorthFrac[i]
		}
	}
}

// Calculates advective flux in West and East directions
// using upwind flux-form spatial approximation for δ(uq)/δx.
// Returns mass flux per unit area per unit time.
func (c *Cell) westEastFlux(ii int) float64 {
	var flux float64
	for i, w := range c.West {
		flux += (w.UPlusSpeed*w.Ci[ii] -
			c.Ci[ii]*c.UMinusSpeed) * c.WestFrac[i]
	}
	for i, e := range c.East {
		flux += (e.UMinusSpeed*e.Ci[ii] -
			c.Ci[ii]*c.UPlusSpeed) * c.EastFrac[i]
	}
	return flux
}

// Calculates advective flux in South and North directions
// using upwind flux-form spatial approximation for δ(uq)/δx.
// Returns mass flux per unit area per unit time.
func (c *Cell) southNorthFlux(ii int) float64 {
	var flux float64
	for i, s := range c.South {
		flux += (s.VPlusSpeed*s.Ci[ii] -
			c.Ci[ii]*c.VMinusSpeed) * c.SouthFrac[i]
	}
	for i, n := range c.North {
		flux += (n.VMinusSpeed*n.Ci[ii] -
			c.Ci[ii]*c.VPlusSpeed) * c.NorthFrac[i]
	}
	return flux
}

// Calculates advective flux in Below and Above directions
// using upwind flux-form spatial approximation for δ(uq)/δx.
// Returns mass flux per unit area per unit time.
func (c *Cell) belowAboveFlux(ii int) float64 {
	var flux float64
	if c.Layer != 0 { // Can't advect downwards from bottom cell
		for i, b := range c.Below {
			flux += (b.WPlusSpeed*b.Ci[ii] -
				c.Ci[ii]*c.WMinusSpeed) * c.BelowFrac[i]
		}
	}
	for i, a := range c.Above {
		flux += (a.WMinusSpeed*a.Ci[ii] -
			c.Ci[ii]*c.WPlusSpeed) * c.AboveFrac[i]
	}
	return flux
}

const advectionFactor = 2.

// UpwindAdvection calculates advection in the cell based
// on the upwind differences scheme.
func (c *Cell) UpwindAdvection(Δt float64) {
	for ii := range c.Cf {
		c.Cf[ii] += c.westEastFlux(ii) / c.Dx * Δt * advectionFactor
		c.Cf[ii] += c.southNorthFlux(ii) / c.Dy * Δt * advectionFactor
		c.Cf[ii] += c.belowAboveFlux(ii) / c.Dz * Δt * advectionFactor
	}
}

const ammoniaFactor = 4.

// Chemistry calculates the secondary formation of PM2.5.
// Explicitely calculates formation of particulate sulfate
// from gaseous and aqueous SO2.
// Partitions organic matter ("gOrg" and "pOrg"), the
// nitrogen in nitrate ("gNO and pNO"), and the nitrogen in ammonia ("gNH" and
// "pNH) between gaseous and particulate phase
// based on the spatially explicit partioning present in the baseline data.
func (c *Cell) Chemistry(d *InMAPdata) {

	// All SO4 forms particles, so sulfur particle formation is limited by the
	// SO2 -> SO4 reaction.
	ΔS := c.SO2oxidation * c.Cf[igS] * d.Dt
	c.Cf[ipS] += ΔS
	c.Cf[igS] -= ΔS

	// NH3 / NH4 partitioning
	// Assume that NH4 formation (but not evaporation)
	// is limited by SO4 formation.
	totalNH := c.Cf[igNH] + c.Cf[ipNH]
	// Caclulate difference from equilibrium particulate NH conc.
	eqNHpDistance := totalNH*c.NHPartitioning - c.Cf[ipNH]
	if eqNHpDistance > 0. { // particles will form
		ΔNH := min(max(ammoniaFactor*ΔS*mwN/mwS, 0.), eqNHpDistance)
		c.Cf[ipNH] += ΔNH
		c.Cf[igNH] -= ΔNH
	} else {
		c.Cf[ipNH] += eqNHpDistance
		c.Cf[igNH] -= eqNHpDistance
	}

	// NOx / pN0 partitioning
	totalNO := c.Cf[igNO] + c.Cf[ipNO]
	c.Cf[ipNO] = totalNO * c.NOPartitioning
	c.Cf[igNO] = totalNO * (1 - c.NOPartitioning)

	// VOC/SOA partitioning
	totalOrg := c.Cf[igOrg] + c.Cf[ipOrg]
	c.Cf[ipOrg] = totalOrg * c.AOrgPartitioning
	c.Cf[igOrg] = totalOrg * (1 - c.AOrgPartitioning)
}

// DryDeposition calculates particle removal by dry deposition
func (c *Cell) DryDeposition(d *InMAPdata) {
	if c.Layer == 0 {
		fac := 1. / c.Dz * d.Dt
		noxfac := 1 - c.NOxDryDep*fac
		so2fac := 1 - c.SO2DryDep*fac
		vocfac := 1 - c.VOCDryDep*fac
		nh3fac := 1 - c.NH3DryDep*fac
		pm25fac := 1 - c.ParticleDryDep*fac
		c.Cf[igOrg] *= vocfac
		c.Cf[ipOrg] *= pm25fac
		c.Cf[iPM2_5] *= pm25fac
		c.Cf[igNH] *= nh3fac
		c.Cf[ipNH] *= pm25fac
		c.Cf[igS] *= so2fac
		c.Cf[ipS] *= pm25fac
		c.Cf[igNO] *= noxfac
		c.Cf[ipNO] *= pm25fac
	}
}

// WetDeposition calculates particle removal by wet deposition
func (c *Cell) WetDeposition(Δt float64) {
	particleFrac := 1. - c.ParticleWetDep*Δt
	SO2Frac := 1. - c.SO2WetDep*Δt
	otherGasFrac := 1 - c.OtherGasWetDep*Δt
	c.Cf[igOrg] *= otherGasFrac
	c.Cf[ipOrg] *= particleFrac
	c.Cf[iPM2_5] *= particleFrac
	c.Cf[igNH] *= otherGasFrac
	c.Cf[ipNH] *= particleFrac
	c.Cf[igS] *= SO2Frac
	c.Cf[ipS] *= particleFrac
	c.Cf[igNO] *= otherGasFrac
	c.Cf[ipNO] *= particleFrac
}

// convert float to int (rounding)
func f2i(f float64) int {
	return int(f + 0.5)
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
