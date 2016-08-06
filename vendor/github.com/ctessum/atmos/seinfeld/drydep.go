package seinfeld

import (
	"fmt"
	"math"

	"github.com/ctessum/atmos/wesely1989"
)

const (
	g     = 9.81          // gravitational acceleration [m/s2]
	kappa = 0.4           // von Karman constant
	k     = 1.3806488e-23 // Boltzmann constant [m2 kg s-2 K-1]
)

// Function Ra calculates aerodynamic resistance to
// dry deposition where z is the top of the surface layer
// [m], zo is the roughness length [m], ustar is
// friction velocity [m/s], and L is Monin-Obukhov
// length [m], based on Seinfeld and Pandis (2006)
// equation 19.13.
func ra(z, zo, ustar, L float64) float64 {
	zeta := z / L
	zeta0 := zo / L
	if L > 0. { // stable
		return 1. / (kappa * ustar) * (math.Log(z/zo) + 4.7*(zeta-zeta0))
	} else if L == 0. { // neutral
		return 1. / (kappa * ustar) * math.Log(z/zo)
	} else { // unstable
		eta0 := math.Pow(1.-15.*zeta0, 0.25)
		etar := math.Pow(1.-15.*zeta, 0.25)
		return 1. / (kappa * ustar) * (math.Log(z/zo) +
			math.Log((eta0*eta0+1)*(eta0+1)*(eta0+1)/
				((etar*etar+1)*(etar+1)*(etar+1))) +
			2*(math.Atan(etar)-math.Atan(eta0)))
	}
}

// Function mu calculates the dynamic viscosity of
// air [kg m-1 s-1] where T is temperature [K].
func mu(T float64) float64 {
	return 1.8e-5 * math.Pow(T/298., 0.85)
}

// Function mfp calculates the mean free path of air [m] where
// T is temperature [K] P is pressure [Pa], and
// Mu is dynamic viscosity [kg/(m s)].
// From Seinfeld and Pandis (2006) equation 9.6
func mfp(T, P, Mu float64) float64 {
	const M = 28.97e-3  // [kg/mol] molecular weight of air
	const R = 8.3144621 //  [J K-1 mol-1] Gas constant
	return 2 * Mu / P / math.Sqrt(8*M/(math.Pi*R*T))
}

// Function cc calculates the Cunnningham slip correction factor
// where Dp is particle diameter [m], T is temperature [K], and
// P is pressure [Pa].
// From Seinfeld and Pandis (2006) equation 9.34.
func cc(Dp, T, P, Mu float64) float64 {
	lambda := mfp(T, P, Mu)
	return 1 + 2*lambda/Dp*
		(1.257+0.4*math.Exp(-1.1*Dp/(2*lambda)))
}

// Function vs calculates the terminal setting velocity of a
// particle where Dp is particle diameter [m], ρP is particle
// density [kg/m3], Cc is the Cunningham slip correction factor,
// and Mu is air dynamic viscosity [kg/(s m)]. From equation
// 9.42 in Seinfeld and Pandis (2006).
func vs(Dp, ρP, Cc, Mu float64) float64 {
	if Dp > 20.e-6 {
		panic(fmt.Sprintf("Particle diameter (%g m) is greater "+
			"than 20um; Stokes settling no longer applies.", Dp))
	}
	return Dp * Dp * ρP * g * Cc / (Mu * 18.)
}

// Function dParticle calculates the brownian diffusivity of a
// particle [m2/s] using the Stokes-Einstein-Sutherland relation
// (Seinfeld and Pandis eq. 9.73) where T is air temperature [K],
// P is pressure [Pa],
// Dp is particle diameter [m], and mu is air dynamic viscosity [kg/(s m)].
func dParticle(T, P, Dp, Cc, mu float64) float64 {
	return k * T * Cc / (3 * math.Pi * mu * Dp)
}

// Function dH2O calculates molecular diffusivity of water vapor
// in air [m2/s] where T is temperature [K]
//  using a regression fit to data in Bolz and Tuve (1976)
// found here: http://www.cambridge.org/us/engineering/author/nellisandklein/downloads/examples/EXAMPLE_9.2-1.pdf
func dH2O(T float64) float64 {
	return -2.775e-6 + 4.479e-8*T + 1.656e-10*T*T
}

// Function sc computes the dimensionless Schmidt number,
// where mu is dynamic viscosity of air [kg/(s m)], rho is air density [kg/m3],
// and D is the molecular diffusivity of the gas species
// of interest [m2/s].
func sc(mu, rho, D float64) float64 {
	return mu / rho / D
}

// Function stSmooth computes the dimensionless Stokes number
// for dry deposition of particles on smooth surfaces or
// surfaces with bluff roughness elements, where vs is
// settling velocity [m/s], ustar is friction velocity
// [m/s], mu is dynamic viscosity of air [kg/(s m)], and rho
// is air density [kg/m3], based on Seinfeld and Pandis (2006)
// equation 19.23.
func stSmooth(vs, ustar, mu, rho float64) float64 {
	return vs * ustar * ustar / g / mu * rho
}

// Function stVeg computes the dimensionless Stokes number
// for dry deposition of particles on vegetated surfaces, where vs is
// settling velocity [m/s], ustar is friction velocity
// [m/s], and A is the characteristic collector radius [m],
// based on Seinfeld and Pandis (2006)
// equation 19.23.
func stVeg(vs, ustar, A float64) float64 {
	return vs * ustar / g / A
}

// Function RbGas calculates the quasi-laminar
// sublayer resistance to dry deposition for a gas species [s/m],
// where Sc is the dimensionless Schmidt number and ustar is the
// friction velocity [m/s]. From Seinfeld and Pandis (2006) equation
// 19.17.
func rbGas(Sc, ustar float64) float64 {
	return 5 * math.Pow(Sc, 0.6666666666666666667) / ustar
}

// Function RbParticle calculates the quasi-laminar
// sublayer resistance to dry deposition for a particles [s/m],
// where Sc is the dimensionless Schmidt number, ustar is the
// friction velocity [m/s], St is the dimensionless Stokes
// number, Dp is particle diameter [m], and iSeason and iLandUse
// are season and land use indexes, respectively.
// From Seinfeld and Pandis (2006) equation 19.27.
func rbParticle(Sc, ustar, St, Dp float64, iSeason SeasonalCategory,
	iLandUse LandUseCategory) float64 {

	// Values for the alpha parameter,
	// where the indexes are land use categories
	// (given in Seinfeld and Pandis Table 19.2)
	var alpha = []float64{1., 0.8, 1.2, 50.0, 1.3}

	// Values for the gamma parameter
	// where the indexes are land use categories.
	// (given in Seinfeld and Pandis Table 19.2)
	var gamma = []float64{.56, .56, .54, .54, .54}

	R1 := math.Exp(-math.Pow(St, 0.5))
	c1 := math.Pow(Sc, -gamma[int(iLandUse)])
	c2 := St / (alpha[int(iLandUse)] + St)
	c3 := Dp / a[int(iSeason)][int(iLandUse)]
	return 1. / (3. * ustar * (c1 + c2*c2 + 0.5*c3*c3) * R1)
}

// Values for the characteristic radii of collectors [m]
// where the columns are land use categories
// and the rows are seasonal categories.
// (given in Seinfeld and Pandis Table 19.2)
var a = [][]float64{
	{2.e-6, 5.e-6, 2.e-6, math.Inf(1), 10.e-6},
	{2.e-6, 5.e-6, 2.e-6, math.Inf(1), 10.e-6},
	{2.e-6, 10.e-6, 5.e-6, math.Inf(1), 10.e-6},
	{2.e-6, 10.e-6, 5.e-6, math.Inf(1), 10.e-6}}

type LandUseCategory int

const (
	Evergreen LandUseCategory = iota //	0. Evergreen-needleleaf trees
	Deciduous                        //	1. Deciduous broadleaf trees
	Grass                            //	2. Grass
	Desert                           //	3. Desert
	Shrubs                           //	4. Shrubs and interrupted woodlands
)

type SeasonalCategory int

const (
	Midsummer    SeasonalCategory = iota //	0. Midsummer with lush vegetation
	Autumn                               //	1. Autumn with cropland not harvested
	LateAutumn                           //	2. Late autumn after frost, no snow
	Winter                               //	3. Winter, snow on ground
	Transitional                         //	4. transitional
)

// Function DryDepGas calculates dry deposition velocity [m/s] for a gas species,
// where z is the height of the surface layer [m], zo is roughness
// length [m], ustar is friction velocity [m/s], L is Monin-Obukhov
// length [m], T is surface air temperature [K], rhoA is air density [kg/m3]
// gd is data about the gas species for surface resistance calculations, G is solar
// irradiation [W m-2], Θ is the slope of the local terrain [radians],
// iSeason and iLandUse are indexes for the season and land use,
// dew and rain indicate whether there is dew or rain on the ground,
// and isSO2 and isO3 indicate whether the gas species of interest
// is either SO2 or O3, respectively. Based on Seinfeld and Pandis (2006)
// equation 19.2.
func DryDepGas(z, zo, ustar, L, T, rhoA, G, Θ float64,
	gd *wesely1989.GasData,
	iSeason wesely1989.SeasonCategory,
	iLandUse wesely1989.LandUseCategory,
	rain, dew, isSO2, isO3 bool) float64 {
	Ra := ra(z, zo, ustar, L)
	Mu := mu(T)
	Dg := dH2O(T) / gd.Dh2oPerDx // Diffusivity of gas of interest [m2/s]
	Sc := sc(Mu, rhoA, Dg)
	Rb := rbGas(Sc, ustar)
	Rc := wesely1989.SurfaceResistance(gd, G, T, Θ,
		iSeason, iLandUse, rain, dew, isSO2, isO3)
	return 1. / (Ra + Rb + Rc)
}

// Function DryDepParticle calculates particle dry deposition
// velocity [m/s]
// where z is the height of the surface layer [m], zo is roughness
// length [m], ustar is friction velocity [m/s], L is Monin-Obukhov
// length [m], Dp is particle diameter [m],
// T is surface air temperature [K], P is pressure [Pa],
// ρParticle is particle density [kg/m3], ρAir is air density [kg/m3],
// and iSeason and iLandUse are indexes for the season and land use.
// Based on Seinfeld and Pandis (2006) equation 19.7.
func DryDepParticle(z, zo, ustar, L, Dp, T, P, ρParticle, ρAir float64, iSeason SeasonalCategory,
	iLandUse LandUseCategory) float64 {
	Ra := ra(z, zo, ustar, L)
	Mu := mu(T)
	Cc := cc(Dp, T, P, Mu)
	Vs := vs(Dp, ρParticle, Cc, Mu)
	var St float64
	switch iLandUse {
	case Desert:
		St = stSmooth(Vs, ustar, Mu, ρAir)
	default:
		St = stVeg(Vs, ustar, a[int(iSeason)][int(iLandUse)])
	}
	D := dParticle(T, P, Dp, Cc, Mu)
	Sc := sc(Mu, ρAir, D)
	Rb := rbParticle(Sc, ustar, St, Dp, iSeason, iLandUse)
	return 1./(Ra+Rb+Ra*Rb*Vs) + Vs
}
