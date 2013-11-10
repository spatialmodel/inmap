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

// DiffusiveFlux calculates diffusive fluxes given diffusivity (D; m2/s) and
// initial concentration (Co; arbitrary units) arrays, x, y, and z array
// indicies (i,j, and k, respectively) and x, y, and z grid
// resolutions (dx,dy,dz; units of meters). Returns diffusive flux
// (from Fick's first law)
// in units of (Co units).
//func (m *MetData) DiffusiveFlux(c, d *Neighborhood) (
//	zdiff float64) {
//
//	zdiff = (d.kplus*(c.kplus-c.center)/c.Dzsquared +
//		d.center*(c.kminus-c.center)/c.Dzsquared) * m.Dt
//	return
//}

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

// Fourth order Runge-Kutta scheme for calculating advection.
// From Jacobson (2005) equations 6.53-6.55.
func rkJacobson(uplus, uminus, q, qplus, qminus, Δt, Δx float64) (
	Δqfinal float64) {

	rk := func(uplus, uminus, qplus, qminus, Δx float64) float64 {
		return -(uplus*qplus - uminus*qminus) / 2 / Δx
	}
	k1 := rk(uplus, uminus, qplus, qminus, Δx) * Δt
	qEst1plus := qplus + k1/2
	qEst1minus := qminus + k1/2
	k2 := rk(uplus, uminus, qEst1plus, qEst1minus, Δx) * Δt
	qEst2plus := qplus + k2/2
	qEst2minus := qminus + k2/2
	k3 := rk(uplus, uminus, qEst2plus, qEst2minus, Δx) * Δt
	qEst3plus := qplus + k3/2
	qEst3minus := qminus + k3/2
	k4 := rk(uplus, uminus, qEst3plus, qEst3minus, Δx) * Δt
	Δqfinal = k1/6. + k2/3. + k3/3. + k4/6.
	return
}

// Calculates advective flux given the concentrations of
// the cell in question and its neighbors (c), as
// well as the neighboring velocities on the Arakawa
// C grid (U₋, U₊, V₋, V₊, W₋, W₊; units of m/s).
// From Jacobson (2005).
// Returned fluxes are in the same units as c.
//func (m *MetData) AdvectiveFluxRungeKuttaJacobson(c *Neighborhood,
//	Uminus, Uplus, Vminus, Vplus, Wminus, Wplus float64) (
//	xadv, yadv, zadv float64) {
//	xadv = rkJacobson(Uplus, Uminus, c.center, c.iplus, c.iminus, m.Dt, m.Dx)
//	yadv = rkJacobson(Vplus, Vminus, c.center, c.jplus, c.jminus, m.Dt, m.Dy)
//	zadv = rkJacobson(Wplus, Wminus, c.center, c.kplus, c.kminus, m.Dt, c.Dz)
//
//	return
//}

// Advective flux is calcuated based on an initial concentration array (Co,
// arbitrary units), x, y, and z wind speed (U, V, and W, respectively; units
// of meters per second), x, y, and z array indicies (i,j, and k, respectively)
// and x, y, and z grid resolutions (dx,dy,dz; units of meters).
// Results are in units of (Co units).
func (c *AIMcell) AdvectiveFluxUpwind(Δt float64) {

	for ii, _ := range c.Cf {
		if c.Uwest > 0. {
			c.Cf[ii] += c.Uwest * c.West.Ci[ii] /
				c.Dx * Δt
		} else {
			c.Cf[ii] += c.Uwest * c.Ci[ii] /
				c.Dx * Δt
		}
		if c.East.Uwest > 0. {
			c.Cf[ii] -= c.East.Uwest * c.Ci[ii] /
				c.Dx * Δt
		} else {
			c.Cf[ii] -= c.East.Uwest *
				c.East.Ci[ii] / c.Dx * Δt
		}

		if c.Vsouth > 0. {
			c.Cf[ii] += c.Vsouth * c.South.Ci[ii] /
				c.Dy * Δt
		} else {
			c.Cf[ii] += c.Vsouth * c.Ci[ii] / c.Dy * Δt
		}
		if c.North.Vsouth > 0. {
			c.Cf[ii] -= c.North.Vsouth * c.Ci[ii] /
				c.Dy * Δt
		} else {
			c.Cf[ii] -= c.North.Vsouth *
				c.North.Ci[ii] / c.Dy * Δt
		}

		if c.Wbelow > 0. {
			c.Cf[ii] += c.Wbelow * c.Below.Ci[ii] /
				c.Dz * Δt
		} else {
			c.Cf[ii] += c.Wbelow * c.Ci[ii] / c.Dz * Δt
		}
		if c.Above.Wbelow > 0. {
			c.Cf[ii] -= c.Above.Wbelow * c.Ci[ii] /
				c.Dz * Δt
		} else {
			c.Cf[ii] -= c.Above.Wbelow *
				c.Above.Ci[ii] / c.Dz * Δt
		}
	}
}
func advectiveFluxUpwind(c *AIMcell, d *AIMdata) {
	c.AdvectiveFluxUpwind(d.Dt)
}

func (c *AIMcell) WetDeposition(Δt float64) {
	particleFrac := 1. - c.wdParticle*Δt*10.
	SO2Frac := 1. - c.wdSO2*Δt*10.
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

// Reactive flux partitions organic matter ("gOrg" and "pOrg"), the
// nitrogen in nitrate ("gNO and pNO"), the nitrogen in ammonia ("gNH" and
// "pNH) and sulfur ("gS" and "pS") between gaseous and particulate phase
// based on the spatially explicit partioning present in the baseline data.
// Inputs are an array of initial concentrations ("conc") and grid index
// ("k", "j", and "i").
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
