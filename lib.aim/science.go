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

func min(v1, v2 float64) float64 {
	if v1 < v2 {
		return v1
	} else {
		return v2
	}
}

func (d *AIMdata) SettlingVelocity() {
	// Settling velocity, m/s
	d.vs = (rhop - rhof) * g * dp * dp / -18. / mu
	fmt.Printf("Settling velocity: %v s\n", d.vs)
}

// Calculate vertical mixing based on Pleim (2007) for
// boundary layer and Wilson (2004) for above the boundary layer.
func (c *AIMcell) VerticalMixing(Δt float64) {
	a := c.Above
	b := c.Below
	g := c.GroundLevel
	for ii, _ := range c.Cf {
		// Pleim (2007) Equation 10.
		if c.k < f2i(c.kPblTop) {
			c.Cf[ii] += (g.M2u*g.Ci[ii] - c.M2d*c.Ci[ii] +
				a.M2d*a.Ci[ii]*a.Dz/c.Dz +
				1./c.Dz*(a.Kz*(a.Ci[ii]-c.Ci[ii])/c.dzPlusHalf+
					c.Kz*(b.Ci[ii]-c.Ci[ii])/c.dzMinusHalf)) * Δt
		} else {
			c.Cf[ii] += 1. / c.Dz * (a.Kz*(a.Ci[ii]-c.Ci[ii])/c.dzPlusHalf +
				c.Kz*(b.Ci[ii]-c.Ci[ii])/c.dzMinusHalf) * Δt
		}
	}
}

var verticalMixing = func(c *AIMcell, d *AIMdata) {
	c.VerticalMixing(d.Dt)
}

// The second through sixth-order flux-form spatial approximations for
// δ(uq)/δx. From equation 4 from Wicker and Skamarock (2002),
// except for second order flux which is from WRF subroutine
// "advect_scalar".
func rkFlux2(q_im1, q_i, ua float64) float64 {
	return 0.5 * ua * (q_i + q_im1)
}
func rkFlux4(q_im2, q_im1, q_i, q_ip1, ua float64) float64 {
	return ua * (7.*(q_i+q_im1) - (q_ip1 + q_im2)) / 12.0
}
func rkFlux3(q_im2, q_im1, q_i, q_ip1, ua float64) float64 {
	return rkFlux4(q_im2, q_im1, q_i, q_ip1, ua) +
		math.Abs(ua)*((q_ip1-q_im2)-3.*(q_i-q_im1))/12.0
}
func rkFlux6(q_im3, q_im2, q_im1, q_i, q_ip1, q_ip2, ua float64) float64 {
	return ua * (37.*(q_i+q_im1) - 8.*(q_ip1+q_im2) + (q_ip2 + q_im3)) / 60.0
}
func rkFlux5(q_im3, q_im2, q_im1, q_i, q_ip1, q_ip2, ua float64) float64 {
	return rkFlux6(q_im3, q_im2, q_im1, q_i, q_ip1, q_ip2, ua) -
		math.Abs(ua)*((q_ip2-q_im3)-5.*(q_ip1-q_im2)+10.*(q_i-q_im1))/60.0
}

// Upwind flux-form spatial approximation for δ(uq)/δx.
func upwindFlux(q_im1, q_i, ua float64) float64 {
	if ua > 0. {
		return ua * q_im1
	} else {
		return ua * q_i
	}
}

// Get advective flux in West and East directions
func (c *AIMcell) westEastFlux(ii int) (fluxWest, fluxEast float64) {
	//	if c.nextToEdge { // Use second order flux for cells by boundary
	fluxWest = upwindFlux(c.West.Ci[ii], c.Ci[ii], c.Uwest)
	fluxEast = upwindFlux(c.Ci[ii], c.East.Ci[ii], c.East.Uwest)
	//	} else if c.twoFromEdge {  // Use third order flux for cells 2 from boundary
	//		fluxWest = rkFlux3(c.West.West.Ci[ii], c.West.Ci[ii], c.Ci[ii],
	//			c.East.Ci[ii], c.Uwest)
	//		fluxEast = rkFlux3(c.West.Ci[ii], c.Ci[ii], c.East.Ci[ii],
	//			c.East.East.Ci[ii], c.East.Uwest)
	//	} else { // Use fifth order flux everywhere else
	//		fluxWest = rkFlux5(c.West.West.West.Ci[ii], c.West.West.Ci[ii],
	//			c.West.Ci[ii], c.Ci[ii], c.East.Ci[ii], c.East.East.Ci[ii],
	//			c.Uwest)
	//		fluxEast = rkFlux5(c.West.West.Ci[ii], c.West.Ci[ii], c.Ci[ii],
	//			c.East.Ci[ii], c.East.East.Ci[ii], c.East.East.East.Ci[ii],
	//			c.East.Uwest)
	//	}
	return
}

// Get advective flux in South and North directions
func (c *AIMcell) southNorthFlux(ii int) (fluxSouth, fluxNorth float64) {
	//	if c.nextToEdge {// Use second order flux for cells by boundary
	fluxSouth = upwindFlux(c.South.Ci[ii], c.Ci[ii], c.Vsouth)
	fluxNorth = upwindFlux(c.Ci[ii], c.North.Ci[ii], c.North.Vsouth)
	//	} else if c.twoFromEdge { // Use third order flux for cells 2 from boundary
	//		fluxSouth = rkFlux3(c.South.South.Ci[ii], c.South.Ci[ii], c.Ci[ii],
	//			c.North.Ci[ii], c.Vsouth)
	//		fluxNorth = rkFlux3(c.South.Ci[ii], c.Ci[ii], c.North.Ci[ii],
	//			c.North.North.Ci[ii], c.North.Vsouth)
	//	} else { // Use fifth order flux everywhere else
	//		fluxSouth = rkFlux5(c.South.South.South.Ci[ii], c.South.South.Ci[ii],
	//			c.South.Ci[ii], c.Ci[ii], c.North.Ci[ii], c.North.North.Ci[ii],
	//			c.Vsouth)
	//		fluxNorth = rkFlux5(c.South.South.Ci[ii], c.South.Ci[ii], c.Ci[ii],
	//			c.North.Ci[ii], c.North.North.Ci[ii], c.North.North.North.Ci[ii],
	//			c.North.Vsouth)
	//	}
	return
}

// Get advective flux in Below and Above directions
func (c *AIMcell) belowAboveFlux(ii int) (fluxBelow, fluxAbove float64) {
	//	if c.nextToEdge { // Use second order flux for cells by boundary
	fluxBelow = upwindFlux(c.Below.Ci[ii], c.Ci[ii], c.Wbelow)
	fluxAbove = upwindFlux(c.Ci[ii], c.Above.Ci[ii], c.Above.Wbelow)
	//	} else if c.twoFromEdge { // Use third order flux for cells 2 from boundary
	//		fluxBelow = rkFlux3(c.Below.Below.Ci[ii], c.Below.Ci[ii], c.Ci[ii],
	//			c.Above.Ci[ii], c.Wbelow)
	//		fluxAbove = rkFlux3(c.Below.Ci[ii], c.Ci[ii], c.Above.Ci[ii],
	//			c.Above.Above.Ci[ii], c.Above.Wbelow)
	//	} else { // Use fifth order flux everywhere else
	//		fluxBelow = rkFlux5(c.Below.Below.Below.Ci[ii], c.Below.Below.Ci[ii],
	//			c.Below.Ci[ii], c.Ci[ii], c.Above.Ci[ii], c.Above.Above.Ci[ii],
	//			c.Wbelow)
	//		fluxAbove = rkFlux5(c.Below.Below.Ci[ii], c.Below.Ci[ii], c.Ci[ii],
	//			c.Above.Ci[ii], c.Above.Above.Ci[ii], c.Above.Above.Above.Ci[ii],
	//			c.Above.Wbelow)
	//	}
	return
}

// Calculates advective flux in the cell based
// on a third order Runge-Kutta scheme
// from Wicker and Skamarock (2002) Equation 3a.
func (c *AIMcell) RK3advectionPass1(d *AIMdata) {
	var fluxMinus, fluxPlus float64
	for ii, _ := range c.Cf {
		// i direction
		fluxMinus, fluxPlus = c.westEastFlux(ii)
		c.Cˣ[ii] = c.Ci[ii] - d.Dt/3./c.Dx*(fluxPlus-fluxMinus)
		// j direction
		fluxMinus, fluxPlus = c.southNorthFlux(ii)
		c.Cˣ[ii] -= d.Dt / 3. / c.Dy * (fluxPlus - fluxMinus)
		// k direction
		fluxMinus, fluxPlus = c.belowAboveFlux(ii)
		c.Cˣ[ii] -= d.Dt / 3. / c.Dz * (fluxPlus - fluxMinus)
	}
	return
}

// Calculates advective flux in the cell based
// on a third order Runge-Kutta scheme
// from Wicker and Skamarock (2002) Equation 3b.
func (c *AIMcell) RK3advectionPass2(d *AIMdata) {
	var fluxMinus, fluxPlus float64
	for ii, _ := range c.Cf {
		// i direction
		fluxMinus, fluxPlus = c.westEastFlux(ii)
		c.Cˣˣ[ii] = c.Cf[ii] - d.Dt/2./c.Dx*(fluxPlus-fluxMinus)
		// j direction
		fluxMinus, fluxPlus = c.southNorthFlux(ii)
		c.Cˣˣ[ii] -= d.Dt / 2. / c.Dy * (fluxPlus - fluxMinus)
		// k direction
		fluxMinus, fluxPlus = c.belowAboveFlux(ii)
		c.Cˣˣ[ii] -= d.Dt / 2. / c.Dz * (fluxPlus - fluxMinus)
	}
	return
}

// Calculates advective flux in the cell based
// on a third order Runge-Kutta scheme
// from Wicker and Skamarock (2002) Equation 3c.
func (c *AIMcell) RK3advectionPass3(d *AIMdata) {
	var fluxMinus, fluxPlus float64
	for ii, _ := range c.Cf {
		// i direction
		fluxMinus, fluxPlus = c.westEastFlux(ii)
		c.Cf[ii] -= d.Dt / c.Dx * (fluxPlus - fluxMinus)
		// j direction
		fluxMinus, fluxPlus = c.southNorthFlux(ii)
		c.Cf[ii] -= d.Dt / c.Dy * (fluxPlus - fluxMinus)
		// k direction
		fluxMinus, fluxPlus = c.belowAboveFlux(ii)
		c.Cf[ii] -= d.Dt / c.Dz * (fluxPlus - fluxMinus)
		if math.IsNaN(c.Cf[ii]) {
			fmt.Println(c.Uwest, c.Vsouth, c.Wbelow)
			panic("x")
		}
	}
	return
}

var rk3AdvectionStep1 = func(c *AIMcell, d *AIMdata) {
	c.RK3advectionPass1(d)
}

var rk3AdvectionStep2 = func(c *AIMcell, d *AIMdata) {
	c.RK3advectionPass2(d)
}

var rk3AdvectionStep3 = func(c *AIMcell, d *AIMdata) {
	c.RK3advectionPass3(d)
}

func (c *AIMcell) WetDeposition(Δt float64) {
	particleFrac := 1. - c.wdParticle*Δt
	SO2Frac := 1. - c.wdSO2*Δt
	otherGasFrac := 1 - c.wdOtherGas*Δt
	c.Cf[igOrg] *= otherGasFrac  // gOrg
	c.Cf[ipOrg] *= particleFrac  // pOrg
	c.Cf[iPM2_5] *= particleFrac // PM2_5
	c.Cf[igNH] *= otherGasFrac   // gNH
	c.Cf[ipNH] *= particleFrac   // pNH
	c.Cf[igS] *= SO2Frac         // gS
	c.Cf[ipS] *= particleFrac    // pS
	c.Cf[igNO] *= otherGasFrac   // gNO
	c.Cf[ipNO] *= particleFrac   // pNO
}

var wetDeposition = func(c *AIMcell, d *AIMdata) {
	c.WetDeposition(d.Dt)
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

var chemicalPartitioning = func(c *AIMcell, d *AIMdata) {
	c.ChemicalPartitioning()
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
	//const kS = 0.000002083 * 1000.
	// Rate of NOx conversion to NO3 (1/s); Muller Table 2, multiplied by COBRA
	// seasonal coefficient of 0.25 for NH4NO3 formation.
	const kNO = 0.000005556 * 0.25

	// All SO4 forms particles, so sulfur particle formation is limited by the
	// SO2 -> SO4 reaction.
	ΔS := c.SO2oxidation * c.Cf[igS] * d.Dt
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

var cobraChemistry = func(c *AIMcell, d *AIMdata) {
	c.COBRAchemistry(d)
}

// VOC oxidation flux
func (c *AIMcell) VOCoxidationFlux(d *AIMdata) {
	c.Cf[igOrg] -= c.Ci[igOrg] * d.VOCoxidationRate * d.Dt
}

var vOCoxidationFlux = func(c *AIMcell, d *AIMdata) {
	c.VOCoxidationFlux(d)
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
		//		vPM25 = 0.001 // m/s; Seinfeld and Pandis Fig 19.2, adjusted from water to land
	)
	if c.k == 0 {
		fac := 1. / c.Dz * d.Dt
		no2fac := 1 - vNO2*fac
		so2fac := 1 - vSO2*fac
		vocfac := 1 - vVOC*fac
		nh3fac := 1 - vNH3*fac
		pm25fac := 1 - c.particleDryDep*fac
		c.Cf[igOrg] *= vocfac
		c.Cf[ipOrg] *= pm25fac
		c.Cf[iPM2_5] *= pm25fac
		c.Cf[igNH] *= nh3fac
		c.Cf[ipNH] *= pm25fac
		c.Cf[igS] *= so2fac
		c.Cf[ipS] *= pm25fac
		c.Cf[igNO] *= no2fac
		c.Cf[ipNO] *= pm25fac
	}
}

var dryDeposition = func(c *AIMcell, d *AIMdata) {
	c.DryDeposition(d)
}

// convert float to int (rounding)
func f2i(f float64) int {
	return int(f + 0.5)
}
