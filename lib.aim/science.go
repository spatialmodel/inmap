package aim

import (
	"fmt"
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

// The fourth-order flux-form spatial approximation for
// δ(uq)/δx. Equation 4b from Wicker and Skamarock (2002).
func f4(u, q, q1, qopposite1, q2 float64) float64 {
	return u / 12. * (7*(q+q1) - (qopposite1 + q2))
}

// The third order Runge-Kutta advection scheme with
// fourth-order spatial differencing. Equation 3
// from Wicker and Skamarock (2002).
// Fourth-order spatial differencing was chosen, even
// though Wicker and Skamarock recommend 5th order spatial
// differencing, because the equation is simpler and doesn't
// involve any cells more than 2 removed from the calculation
// cell.
func rk3_4(uplus, uminus, q, qplus, qminus, q2plus, q2minus, Δt, Δx float64) (
	Δqfinal float64) {
	fplus := f4(uplus, q, qplus, qminus, q2plus)
	fminus := f4(uminus, q, qminus, qplus, q2minus)
	qˣ := q - Δt/3./Δx*(fplus-fminus)

	fplus = f4(uplus, qˣ, qplus, qminus, q2plus)
	fminus = f4(uminus, qˣ, qminus, qplus, q2minus)
	qˣˣ := q - Δt/2./Δx*(fplus-fminus)

	fplus = f4(uplus, qˣˣ, qplus, qminus, q2plus)
	fminus = f4(uminus, qˣˣ, qminus, qplus, q2minus)
	Δqfinal = -Δt / Δx * (fplus - fminus)
	return
}

// Calculates advective flux given the concentrations of
// the cell in question and its neighbors (c), as
// well as the neighboring velocities on the Arakawa
// C grid (U₋, U₊, V₋, V₊, W₋, W₊; units of m/s).
// From Wicker and Skamarock (2002).
// Returned fluxes are in the same units as c.
//func (m *MetData) AdvectiveFluxRungeKutta(c *Neighborhood,
//	Uminus, Uplus, Vminus, Vplus, Wminus, Wplus float64) (
//	xadv, yadv, zadv float64) {
//	xadv = rk3_4(Uplus, Uminus, c.center, c.iplus, c.iminus,
//		c.i2plus, c.i2minus, m.Dt, m.Dx)
//	yadv = rk3_4(Vplus, Vminus, c.center, c.jplus, c.jminus,
//		c.j2plus, c.j2minus, m.Dt, m.Dy)
//	zadv = rk3_4(Wplus, Wminus, c.center, c.kplus, c.kminus,
//		c.k2plus, c.k2minus, m.Dt, c.Dz)
//	return
//}

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

	for ii, _ := range c.finalConc {
		if c.Uwest > 0. {
			c.finalConc[ii] += c.Uwest * c.WestNeighbor.initialConc[ii] /
				c.Dx * Δt
		} else {
			c.finalConc[ii] += c.Uwest * c.initialConc[ii] /
				c.Dx * Δt
		}
		if c.EastNeighbor.Uwest > 0. {
			c.finalConc[ii] -= c.EastNeighbor.Uwest * c.initialConc[ii] /
				c.Dx * Δt
		} else {
			c.finalConc[ii] -= c.EastNeighbor.Uwest *
				c.EastNeighbor.initialConc[ii] / c.Dx * Δt
		}

		if c.Vsouth > 0. {
			c.finalConc[ii] += c.Vsouth * c.SouthNeighbor.initialConc[ii] /
				c.Dy * Δt
		} else {
			c.finalConc[ii] += c.Vsouth * c.initialConc[ii] / c.Dy * Δt
		}
		if c.NorthNeighbor.Vsouth > 0. {
			c.finalConc[ii] -= c.NorthNeighbor.Vsouth * c.initialConc[ii] /
				c.Dy * Δt
		} else {
			c.finalConc[ii] -= c.NorthNeighbor.Vsouth *
				c.NorthNeighbor.initialConc[ii] / c.Dy * Δt
		}

		if c.Wbelow > 0. {
			c.finalConc[ii] += c.Wbelow * c.BelowNeighbor.initialConc[ii] /
				c.Dz * Δt
		} else {
			c.finalConc[ii] += c.Wbelow * c.initialConc[ii] / c.Dz * Δt
		}
		if c.AboveNeighbor.Wbelow > 0. {
			c.finalConc[ii] -= c.AboveNeighbor.Wbelow * c.initialConc[ii] /
				c.Dz * Δt
		} else {
			c.finalConc[ii] -= c.AboveNeighbor.Wbelow *
				c.AboveNeighbor.initialConc[ii] / c.Dz * Δt
		}
	}
}

func (c *AIMcell) WetDeposition(Δt float64) {
	particleFrac := 1. - c.wdParticle*Δt
	SO2Frac := 1. - c.wdSO2*Δt
	otherGasFrac := 1 - c.wdOtherGas*Δt
	c.finalConc[igOrg] *= otherGasFrac  // gOrg
	c.finalConc[ipOrg] *= particleFrac  // pOrg
	c.finalConc[iPM2_5] *= particleFrac // PM2_5
	c.finalConc[igNH] *= otherGasFrac   // gNH
	c.finalConc[ipNH] *= particleFrac   // pNH
	c.finalConc[igS] *= SO2Frac         // gS
	c.finalConc[ipS] *= particleFrac    // pS
	c.finalConc[igNO] *= otherGasFrac   // gNO
	c.finalConc[ipNO] *= particleFrac   // pNO
}

// Reactive flux partitions organic matter ("gOrg" and "pOrg"), the
// nitrogen in nitrate ("gNO and pNO"), the nitrogen in ammonia ("gNH" and
// "pNH) and sulfur ("gS" and "pS") between gaseous and particulate phase
// based on the spatially explicit partioning present in the baseline data.
// Inputs are an array of initial concentrations ("conc") and grid index
// ("k", "j", and "i").
func (c *AIMcell) ChemicalPartitioning() {

	// Gas/particle partitioning
	totalOrg := c.finalConc[igOrg] + c.finalConc[ipOrg]
	c.finalConc[igOrg] = totalOrg * c.orgPartitioning
	c.finalConc[ipOrg] = totalOrg * (1 - c.orgPartitioning)

	totalS := c.finalConc[igS] + c.finalConc[ipS]
	c.finalConc[igS] = totalS * c.SPartitioning
	c.finalConc[ipS] = totalS * (1 - c.SPartitioning)

	totalNO := c.finalConc[igNO] + c.finalConc[ipNO]
	c.finalConc[igNO] = totalNO * c.NOPartitioning
	c.finalConc[ipNO] = totalNO * (1 - c.NOPartitioning)

	totalNH := c.finalConc[igNH] + c.finalConc[ipNH]
	c.finalConc[igNH] = totalNH * c.NHPartitioning
	c.finalConc[ipNH] = totalNH * (1 - c.NHPartitioning)
}

// VOC oxidation flux
func (c *AIMcell) VOCoxidationFlux(d *AIMdata) {
	c.finalConc[igOrg] -= c.initialConc[igOrg] * d.VOCoxidationRate * d.Dt
}

var gravSettlingPols = []int{iPM2_5, ipOrg, ipNH, ipNO, ipS}

func (c *AIMcell) GravitationalSettling(d *AIMdata) {
	for _, iPol := range gravSettlingPols {
		if c.k == 0 {
			c.finalConc[iPol] -= d.vs * c.initialConc[iPol] / c.Dz * d.Dt
		} else {
			c.finalConc[iPol] -= d.vs * (c.initialConc[iPol] -
				c.AboveNeighbor.initialConc[iPol]) / c.Dz * d.Dt
		}
	}
}
