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
			c.Cf[ii] += 1. / c.Dz * (a.Kz*10.*(a.Ci[ii]-c.Ci[ii])/c.dzPlusHalf +
				c.Kz*10.*(b.Ci[ii]-c.Ci[ii])/c.dzMinusHalf) * Δt // Multiplied by 10 !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
		}
	}
}

var verticalMixing = func(c *AIMcell, d *AIMdata) {
	c.VerticalMixing(d.Dt)
}

// The second through sixth-order flux-form spatial approximations for
// δ(uq)/δx. From equation 4 from Wicker and Skamarock (2002),
// except for second order flux which is from WRF subroutine
// "ADVECT SCALAR".
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

// Calculates advective flux in the cell based
// on a third order Runge-Kutta scheme
// from Wicker and Skamarock (2002) Equation 3a.
func (c *AIMcell) RK3advectionPass1(d *AIMdata) {
	var fluxMinus, fluxPlus float64
	for ii, _ := range c.Cf {
		// i direction
		fluxMinus = upwindFlux(c.West.Ci[ii], c.Ci[ii], c.Uwest)
		fluxPlus = upwindFlux(c.Ci[ii], c.East.Ci[ii], c.East.Uwest)
		c.Cˣ[ii] = c.Ci[ii] - d.Dt/3./c.Dx*(fluxPlus-fluxMinus)
		// j direction
		fluxMinus = upwindFlux(c.South.Ci[ii], c.Ci[ii], c.Vsouth)
		fluxPlus = upwindFlux(c.Ci[ii], c.North.Ci[ii], c.North.Vsouth)
		c.Cˣ[ii] -= d.Dt / 3. / c.Dy * (fluxPlus - fluxMinus)
		// k direction
		fluxMinus = upwindFlux(c.Below.Ci[ii], c.Ci[ii], c.Wbelow)
		fluxPlus = upwindFlux(c.Ci[ii], c.Above.Ci[ii], c.Above.Wbelow)
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
		fluxMinus = upwindFlux(c.West.Cˣ[ii], c.Cˣ[ii], c.Uwest)
		fluxPlus = upwindFlux(c.Cˣ[ii], c.East.Cˣ[ii], c.East.Uwest)
		c.Cˣˣ[ii] = c.Cf[ii] - d.Dt/2./c.Dx*(fluxPlus-fluxMinus)
		// j direction
		fluxMinus = upwindFlux(c.South.Cˣ[ii], c.Cˣ[ii], c.Vsouth)
		fluxPlus = upwindFlux(c.Cˣ[ii], c.North.Cˣ[ii], c.North.Vsouth)
		c.Cˣˣ[ii] -= d.Dt / 2. / c.Dy * (fluxPlus - fluxMinus)
		// k direction
		fluxMinus = upwindFlux(c.Below.Cˣ[ii], c.Cˣ[ii], c.Wbelow)
		fluxPlus = upwindFlux(c.Cˣ[ii], c.Above.Cˣ[ii], c.Above.Wbelow)
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
		fluxMinus = upwindFlux(c.West.Cˣˣ[ii], c.Cˣˣ[ii], c.Uwest)
		fluxPlus = upwindFlux(c.Cˣˣ[ii], c.East.Cˣˣ[ii], c.East.Uwest)
		c.Cf[ii] -= d.Dt / c.Dx * (fluxPlus - fluxMinus)
		// j direction
		fluxMinus = upwindFlux(c.South.Cˣˣ[ii], c.Cˣˣ[ii], c.Vsouth)
		fluxPlus = upwindFlux(c.Cˣˣ[ii], c.North.Cˣˣ[ii], c.North.Vsouth)
		c.Cf[ii] -= d.Dt / c.Dy * (fluxPlus - fluxMinus)
		// k direction
		fluxMinus = upwindFlux(c.Below.Cˣˣ[ii], c.Cˣˣ[ii], c.Wbelow)
		fluxPlus = upwindFlux(c.Cˣˣ[ii], c.Above.Cˣˣ[ii], c.Above.Wbelow)
		c.Cf[ii] -= d.Dt / c.Dz * (fluxPlus - fluxMinus)
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
	particleFrac := 1. - c.wdParticle*Δt*10. // Multiplied by 10 ///////////////////////////////////////////////////////////////////////////////////
	SO2Frac := 1. - c.wdSO2*Δt
	otherGasFrac := 1 - c.wdOtherGas*Δt*10.
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
// chemical mechanisms from the COBRA model (COBRA user manual
// appendix A). Some artistic liberties have been taken.
// VOC/SOA partitioning is performed using the method above.
func (c *AIMcell) COBRAchemistry() {
	totalSgas := c.Cf[igS] + c.Cbackground[igS]
	totalNOgas := c.Cf[igNO] + c.Cbackground[igNO]
	totalNHgas := c.Cf[igNH] + c.Cbackground[igNH]

	if totalSgas > 0. && totalNHgas > 0. {
		// Step 1: Calcuate mole ratio of NH4 to SO4.
		R := (totalNHgas / mwN) / (totalSgas / mwS)
		if R < 1. { // 1a. A portion of gS converts to pS, all gNH converts to pNH
			sTransfer := min(totalNHgas/mwN*mwS, c.Cf[igS]) // μg Sulfur
			c.Cf[ipS] += sTransfer
			c.Cf[igS] -= sTransfer
			c.Cf[ipNH] += c.Cf[igNH]
			c.Cf[igNH] = 0.
		} else if R < 2. { // 1b. All gS converts to pS, all gNH converts to pNH.
			c.Cf[ipNH] += c.Cf[igNH]
			c.Cf[igNH] = 0.
			c.Cf[ipS] += c.Cf[igS]
			c.Cf[igS] = 0.
		} else { // 1c. All gS converts to pS, some  gNH converts to pNH.
			c.Cf[ipS] += c.Cf[igS]
			c.Cf[igS] = 0.
			nhTransfer := min(c.Cf[igNH], 2.*totalSgas/mwS*mwN) // μg Nitrogen
			c.Cf[ipNH] += nhTransfer
			c.Cf[igNH] -= nhTransfer
		}
		// Step 2. NH4NO3 formation
		if totalNOgas > 0. && c.Cf[ipNO] < 0.25*c.Cf[igNO] {
			transfer := min(totalNHgas, 0.25*c.Cf[igNO])
			c.Cf[igNH] -= transfer
			c.Cf[ipNH] += transfer
			c.Cf[igNO] -= transfer
			c.Cf[ipNO] += transfer
		}
	}

	// VOC/SOA partitioning
	totalOrg := c.Cf[igOrg] + c.Cf[ipOrg]
	c.Cf[igOrg] = totalOrg * c.orgPartitioning
	c.Cf[ipOrg] = totalOrg * (1 - c.orgPartitioning)
}

var cobraChemistry = func(c *AIMcell, d *AIMdata) {
	c.COBRAchemistry()
}

// VOC oxidation flux
func (c *AIMcell) VOCoxidationFlux(d *AIMdata) {
	c.Cf[igOrg] -= c.Ci[igOrg] * d.VOCoxidationRate * d.Dt
}

var vOCoxidationFlux = func(c *AIMcell, d *AIMdata) {
	c.VOCoxidationFlux(d)
}

var gravSettlingPols = []int{iPM2_5, ipOrg, ipNH, ipNO, ipS}

func (c *AIMcell) GravitationalSettling(d *AIMdata) {
	for _, iPol := range gravSettlingPols {
		if c.k == 0 {
			c.Cf[iPol] -= d.vs * c.Ci[iPol] / c.Dz * d.Dt
		} else {
			c.Cf[iPol] -= d.vs * (c.Ci[iPol] -
				c.Above.Ci[iPol]) / c.Dz * d.Dt
		}
	}
}

var gravitationalSettling = func(c *AIMcell, d *AIMdata) {
	c.GravitationalSettling(d)
}

// convert float to int (rounding)
func f2i(f float64) int {
	return int(f + 0.5)
}
