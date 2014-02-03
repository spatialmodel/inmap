package aim

import (
	"fmt"
	"math"
)

const (
	dp    = 1.e-6   // m, particle diameter
	rhof  = 1.2466  // kg/m3, air density
	rhop  = 1000.   // kg/m3, density of droplet
	g     = 9.80665 // m/s2
	mu    = 1.5e-5  // kg/m/s
	kappa = 0.4     // von karmon's constant
)

// Calculates settling velocity [m/s] based on
// Stokes law, with no slip correction.
func (d *AIMdata) SettlingVelocity() {
	d.vs = (rhop - rhof) * g * dp * dp / -18. / mu
}

// Calculate vertical mixing based on Pleim (2007), which is
// combined local-nonlocal closure scheme, for
// boundary layer and Wilson (2004) for above the boundary layer.
// Also calculate horizontal mixing assuming that Kxx and Kyy
// are the same as Kzz.
func (c *AIMcell) Mixing(Δt float64) {
	a := c.Above
	b := c.Below
	g := c.GroundLevel
	for ii, _ := range c.Cf {
		// Pleim (2007) Equation 10.
		if c.k < f2i(c.kPblTop) { // Within boundary layer
			c.Cf[ii] += (g.M2u*g.Ci[ii] - c.M2d*c.Ci[ii] +
				a.M2d*a.Ci[ii]*a.Dz/c.Dz +
				1./c.Dz*(a.Kz*(a.Ci[ii]-c.Ci[ii])/c.dzPlusHalf+
					c.Kz*(b.Ci[ii]-c.Ci[ii])/c.dzMinusHalf)) * Δt
		} else { // Above boundary layer: no convective or horizontal mixing
			c.Cf[ii] += 1. / c.Dz * (a.Kz*(a.Ci[ii]-c.Ci[ii])/c.dzPlusHalf +
				c.Kz*(b.Ci[ii]-c.Ci[ii])/c.dzMinusHalf) * Δt * 3. //////////////////////////////////////////////////////////////////////////////////
		}
		// Horizontal mixing
		c.Cf[ii] += 1. / c.Dx * (c.East.KxxWest*(c.East.Ci[ii]-c.Ci[ii])/c.Dx +
			c.KxxWest*(c.West.Ci[ii]-c.Ci[ii])/c.Dx) * Δt //* 1000. ///////////////////////////////////////////////////////////////////////////////////
		c.Cf[ii] += 1. / c.Dy * (c.North.KyySouth*(c.North.Ci[ii]-c.Ci[ii])/c.Dy +
			c.KyySouth*(c.South.Ci[ii]-c.Ci[ii])/c.Dy) * Δt //* 1000. ///////////////////////////////////////////////////////////////////////////
	}
}

// Calculates advective flux in West and East directions
// using upwind flux-form spatial approximation for δ(uq)/δx.
func (c *AIMcell) westEastFlux(ii int) float64 {
	return c.West.uPlusSpeed*c.West.Ci[ii] - c.Ci[ii]*c.uMinusSpeed +
		c.East.uMinusSpeed*c.East.Ci[ii] - c.Ci[ii]*c.uPlusSpeed
}

// Calculates advective flux in South and North directions
// using upwind flux-form spatial approximation for δ(uq)/δx.
func (c *AIMcell) southNorthFlux(ii int) float64 {
	return c.South.vPlusSpeed*c.South.Ci[ii] - c.Ci[ii]*c.vMinusSpeed +
		c.North.vMinusSpeed*c.North.Ci[ii] - c.Ci[ii]*c.vPlusSpeed
}

// Calculates advective flux in Below and Above directions
// using upwind flux-form spatial approximation for δ(uq)/δx.
func (c *AIMcell) belowAboveFlux(ii int) float64 {
	return c.Below.wPlusSpeed*c.Below.Ci[ii] - c.Ci[ii]*c.wMinusSpeed +
		c.Above.wMinusSpeed*c.Above.Ci[ii] - c.Ci[ii]*c.wPlusSpeed
}

// Calculates advection in the cell based
// on a third order Runge-Kutta scheme
// from Wicker and Skamarock (2002) Equation 3a.
func (c *AIMcell) RK3advectionPass1(d *AIMdata) {
	var flux float64
	for ii, _ := range c.Cf {
		// i direction
		flux = c.westEastFlux(ii)
		c.Cˣ[ii] = c.Ci[ii] + d.Dt/3./c.Dx*flux
		// j direction
		flux = c.southNorthFlux(ii)
		c.Cˣ[ii] += d.Dt / 3. / c.Dy * flux
		// k direction
		flux = c.belowAboveFlux(ii)
		c.Cˣ[ii] += d.Dt / 3. / c.Dz * flux
	}
	return
}

// Calculates advection in the cell based
// on a third order Runge-Kutta scheme
// from Wicker and Skamarock (2002) Equation 3b.
func (c *AIMcell) RK3advectionPass2(d *AIMdata) {
	var flux float64
	for ii, _ := range c.Cf {
		// i direction
		flux = c.westEastFlux(ii)
		c.Cˣˣ[ii] = c.Cf[ii] + d.Dt/2./c.Dx*flux
		// j direction
		flux = c.southNorthFlux(ii)
		c.Cˣˣ[ii] += d.Dt / 2. / c.Dy * flux
		// k direction
		flux = c.belowAboveFlux(ii)
		c.Cˣˣ[ii] += d.Dt / 2. / c.Dz * flux
	}
	return
}

// Calculates advection flux in the cell based
// on a third order Runge-Kutta scheme
// from Wicker and Skamarock (2002) Equation 3c.
func (c *AIMcell) RK3advectionPass3(d *AIMdata) {
	var flux float64
	for ii, _ := range c.Cf {
		// i direction
		flux = c.westEastFlux(ii)
		c.Cf[ii] += d.Dt / c.Dx * flux
		// j direction
		flux = c.southNorthFlux(ii)
		c.Cf[ii] += d.Dt / c.Dy * flux
		// k direction
		flux = c.belowAboveFlux(ii)
		c.Cf[ii] += d.Dt / c.Dz * flux

		if math.IsNaN(c.Cf[ii]) {
			panic(fmt.Sprintf("Found a NaN value. Pol: %v, k=%v, j=%v, i=%v",
				polNames[ii], c.k, c.j, c.i))
		}
	}
	return
}

// Partitions organic matter ("gOrg" and "pOrg"), the
// nitrogen in nitrate ("gNO and pNO"), the nitrogen in ammonia ("gNH" and
// "pNH) and sulfur ("gS" and "pS") between gaseous and particulate phase
// based on the spatially explicit partioning present in the baseline data.
func (c *AIMcell) ChemicalPartitioning() {

	// Gas/particle partitioning
	totalOrg := c.Cf[igOrg] + c.Cf[ipOrg]
	c.Cf[igOrg] = totalOrg * c.orgPartitioning
	c.Cf[ipOrg] = totalOrg * (1 - c.orgPartitioning)

	totalS := c.Cf[igS] + c.Cf[ipS]
	c.Cf[igS] = totalS * c.SPartitioning
	c.Cf[ipS] = totalS * (1 - c.SPartitioning)

	totalNO := c.Cf[igNO] + c.Cf[ipNO]
	c.Cf[igNO] = totalNO * c.NOPartitioning
	c.Cf[ipNO] = totalNO * (1 - c.NOPartitioning)

	totalNH := c.Cf[igNH] + c.Cf[ipNH]
	c.Cf[igNH] = totalNH * c.NHPartitioning
	c.Cf[ipNH] = totalNH * (1 - c.NHPartitioning)
}

// Calculates the secondary formation of PM2.5 based on the
// chemical mechanisms from the COBRA model and APEEP models
// (COBRA user manual appendix A; Muller and Mendelsohn 2006).
// Changes have been made to adapt the equations from gaussian
// plume model form to gridded model form.
// VOC/SOA partitioning is performed using the method above.
func (c *AIMcell) COBRAchemistry(d *AIMdata) {
	//totalSparticle := (c.Cf[ipS] + c.Cbackground[ipS]) / mwS    // moles S
	totalNHgas := (c.Cf[igNH] + c.Cbackground[igNH]) // μg N
	//totalNHparticle := (c.Cf[ipNH] + c.Cbackground[ipNH]) / mwN // moles N

	// Rate of SO2 conversion to SO4 (1/s); Muller Table 2
	//const kS = 0.000002083
	// Rate of NOx conversion to NO3 (1/s); Muller Table 2, multiplied by COBRA
	// seasonal coefficient of 0.25 for NH4NO3 formation.
	const kNO = 0.000005556 * 0.5 ///////////////////////////////////////////////////////////////////////////////////////////////////

	// All SO4 forms particles, so sulfur particle formation is limited by the
	// SO2 -> SO4 reaction.
	ΔS := c.SO2oxidation * c.Cf[igS] * d.Dt * 100. ////////////////////////////////////////////////////////////
	//ΔS := kS * c.Cf[igS] * d.Dt
	c.Cf[igS] -= ΔS
	c.Cf[ipS] += ΔS

	//if totalSparticle > 0. && totalNHparticle > 0. {
	// COBRA step 1: Calcuate mole ratio of NH4 to SO4.
	//	R := totalNHparticle / totalSparticle
	//	if R < 2. { // 1a and 1b: all gNH converts to pNH
	//		c.Cf[ipNH] += c.Cf[igNH]
	//		c.Cf[igNH] = 0.
	//	} else { // 1c. Some  gNH converts to pNH.
	//		nhTransfer := min(c.Cf[igNH], 2.*totalSparticle*mwN) // μg Nitrogen
	//		c.Cf[ipNH] += nhTransfer
	//		c.Cf[igNH] -= nhTransfer
	//	}

	// VOC/SOA partitioning
	totalOrg := c.Cf[igOrg] + c.Cf[ipOrg]
	c.Cf[igOrg] = totalOrg * c.orgPartitioning
	c.Cf[ipOrg] = totalOrg * (1 - c.orgPartitioning)

	// NH3 / NH4 partitioning
	totalNH := c.Cf[igNH] + c.Cf[ipNH]
	c.Cf[igNH] = totalNH * c.NHPartitioning
	c.Cf[ipNH] = totalNH * (1 - c.NHPartitioning)

	// Step 2. NH4NO3 formation
	if totalNHgas > 0. {
		ΔN := kNO * c.Cf[igNO] * d.Dt
		ΔNO := min(totalNHgas, ΔN)
		ΔNH := min(c.Cf[igNH], ΔNO)
		c.Cf[igNH] -= ΔNH
		c.Cf[ipNH] += ΔNH
		c.Cf[igNO] -= ΔNO
		c.Cf[ipNO] += ΔNO
	}
	//}

}

// VOC oxidation flux
func (c *AIMcell) VOCoxidationFlux(d *AIMdata) {
	c.Cf[igOrg] -= c.Ci[igOrg] * d.VOCoxidationRate * d.Dt
}

// Caluclates Dry deposition using deposition velocities from Muller and
// Mendelsohn (2006), Hauglustain et al. (1994), Phillips et al. (2004),
// and and the GOCART aerosol module in WRF/Chem.
func (c *AIMcell) DryDeposition(d *AIMdata) {
	const (
		vNO2 = 0.01  // m/s; Muller and Mendelsohn Table 2
		vSO2 = 0.005 // m/s; Muller and Mendelsohn Table 2
		vVOC = 0.001 // m/s; Hauglustaine Table 2
		vNH3 = 0.01  // m/s; Phillips abstract
	)
	if c.k == 0 {
		fac := 1. / c.Dz * d.Dt
		noxfac := 1 - c.NOxDryDep*fac
		so2fac := 1 - c.SO2DryDep*fac
		vocfac := 1 - c.VOCDryDep*fac
		nh3fac := 1 - c.NH3DryDep*fac
		pm25fac := 1 - c.particleDryDep*fac
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

func (c *AIMcell) WetDeposition(Δt float64) {
	particleFrac := 1. - c.particleWetDep*Δt
	SO2Frac := 1. - c.SO2WetDep*Δt
	otherGasFrac := 1 - c.otherGasWetDep*Δt
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

func min(v1, v2 float64) float64 {
	if v1 < v2 {
		return v1
	} else {
		return v2
	}
}
