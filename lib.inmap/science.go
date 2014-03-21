package inmap

const (
	dp    = 1.e-6   // m, particle diameter
	rhof  = 1.2466  // kg/m3, air density
	rhop  = 1000.   // kg/m3, density of droplet
	g     = 9.80665 // m/s2
	mu    = 1.5e-5  // kg/m/s
	kappa = 0.4     // von karmon's constant
)

// Calculate vertical mixing based on Pleim (2007), which is
// combined local-nonlocal closure scheme, for
// boundary layer and based on Wilson (2004) for above the boundary layer.
// Also calculate horizontal mixing.
func (c *Cell) Mixing(Δt float64) {
	for ii, _ := range c.Cf {
		// Pleim (2007) Equation 10.
		if c.Layer < f2i(c.PblTopLayer) { // Convective mixing
			for i, g := range c.GroundLevel { // Upward convection
				c.Cf[ii] += g.M2u * g.Ci[ii] * Δt * c.GroundLevelFrac[i]
			}
			for i, a := range c.Above { // Balancing downward mixing
				c.Cf[ii] += (a.M2d*a.Ci[ii]*a.Dz/c.Dz - c.M2d*c.Ci[ii]) *
					Δt * c.AboveFrac[i]
			}
		}
		for i, a := range c.Above { // Mixing with above
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

// Calculates advection in the cell based
// on the upwind differences scheme.
func (c *Cell) UpwindAdvection(Δt float64) {
	for ii, _ := range c.Cf {
		c.Cf[ii] += c.westEastFlux(ii) / c.Dx * Δt
		c.Cf[ii] += c.southNorthFlux(ii) / c.Dy * Δt
		c.Cf[ii] += c.belowAboveFlux(ii) / c.Dz * Δt
	}
}

// Partitions organic matter ("gOrg" and "pOrg"), the
// nitrogen in nitrate ("gNO and pNO"), the nitrogen in ammonia ("gNH" and
// "pNH) and sulfur ("gS" and "pS") between gaseous and particulate phase
// based on the spatially explicit partioning present in the baseline data.
func (c *Cell) ChemicalPartitioning() {

	// Gas/particle partitioning
	totalOrg := c.Cf[igOrg] + c.Cf[ipOrg]
	c.Cf[igOrg] = totalOrg * c.OrgPartitioning
	c.Cf[ipOrg] = totalOrg * (1 - c.OrgPartitioning)

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
func (c *Cell) Chemistry(d *InMAPdata) {
	//totalSparticle := (c.Cf[ipS] + c.Cbackground[ipS]) / mwS    // moles S
	//totalNHgas := (c.Cf[igNH] + c.Cbackground[igNH]) // μg N
	//totalNHparticle := (c.Cf[ipNH] + c.Cbackground[ipNH]) / mwN // moles N

	// Rate of SO2 conversion to SO4 (1/s); Muller Table 2
	//const kS = 0.000002083
	// Rate of NOx conversion to NO3 (1/s); Muller Table 2, multiplied by COBRA
	// seasonal coefficient of 0.25 for NH4NO3 formation.
	//const kNO = 0.000005556 * 0.25

	// All SO4 forms particles, so sulfur particle formation is limited by the
	// SO2 -> SO4 reaction.
	ΔS := c.SO2oxidation * c.Cf[igS] * d.Dt
	//ΔS := kS * c.Cf[igS] * d.Dt
	c.Cf[igS] -= ΔS
	c.Cf[ipS] += ΔS

	// VOC/SOA partitioning
	totalOrg := c.Cf[igOrg] + c.Cf[ipOrg]
	c.Cf[igOrg] = totalOrg * c.OrgPartitioning
	c.Cf[ipOrg] = totalOrg * (1 - c.OrgPartitioning)

	// NH3 / NH4 partitioning
	totalNH := c.Cf[igNH] + c.Cf[ipNH]
	c.Cf[igNH] = totalNH * c.NHPartitioning
	c.Cf[ipNH] = totalNH * (1 - c.NHPartitioning)

	// NOx / pN0 partitioning
	totalNO := c.Cf[igNO] + c.Cf[ipNO]
	c.Cf[igNO] = totalNO * c.NOPartitioning
	c.Cf[ipNO] = totalNO * (1 - c.NOPartitioning)

	// Step 2. NH4NO3 formation
	//if totalNHgas > 0. {
	//	ΔN := kNO * c.Cf[igNO] * d.Dt
	//	ΔNO := min(totalNHgas, ΔN)
	//	ΔNH := min(c.Cf[igNH], ΔNO)
	//	c.Cf[igNH] -= ΔNH
	//	c.Cf[ipNH] += ΔNH
	//	c.Cf[igNO] -= ΔNO
	//	c.Cf[ipNO] += ΔNO
	//}
}

// VOC oxidation flux
//func (c *Cell) VOCoxidationFlux(d *InMAPdata) {
//	c.Cf[igOrg] -= c.Ci[igOrg] * d.VOCoxidationRate * d.Dt
//}

// Calculates Dry deposition using deposition velocities from Muller and
// Mendelsohn (2006), Hauglustain et al. (1994), Phillips et al. (2004),
// and and the GOCART aerosol module in WRF/Chem.
func (c *Cell) DryDeposition(d *InMAPdata) {
	const (
		vNO2 = 0.01  // m/s; Muller and Mendelsohn Table 2
		vSO2 = 0.005 // m/s; Muller and Mendelsohn Table 2
		vVOC = 0.001 // m/s; Hauglustaine Table 2
		vNH3 = 0.01  // m/s; Phillips abstract
	)
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
	m := 0.
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
	} else {
		return v2
	}
}
