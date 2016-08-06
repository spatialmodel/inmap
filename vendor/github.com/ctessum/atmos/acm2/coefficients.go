package acm2

import (
	"math"
)

const (
	κ  = 0.41    // Von Kármán constant
	Cp = 1006.   // m2/s2-K; specific heat of air
	g  = 9.80665 // m/s2
)

// Calculates the Monin-Obukhov length [m] based on Pleim (2007) equation 14
// when given the surface heat flux (surfaceHeatFlux [W m-2]), air density
// (ρ [kg m-3]), the average temperature of the boundary layer (To [K]),
// and friction velocity (ustar [m/s]).
func ObukhovLen(surfaceHeatFlux, ρ, To, ustar float64) float64 {
	// Potential temperature flux = surfaceHeatFlux / Cp /  ρ
	// θf (K m / s) = hfx (W / m2) / Cp (J / kg-K) * alt (m3 / kg)
	θf := surfaceHeatFlux / Cp / ρ
	// L=Monin-Obukhov length, Pleim (2007) equation 14.
	return To * math.Pow(ustar, 3) / g / κ / θf
}

// Calculates the convective mixing fraction [-] based on
// Pleim (2007) equation 19 when given the
// Monin-Obukhov length (L [m]) and the boundary layer
// height (h [m]).
func ConvectiveFraction(L, h float64) (fconv float64) {
	fconv = max(0., 1/(1+math.Pow(κ, -2./3.)/.72*
		math.Pow(-h/L, -1./3.))) // Pleim 2007, Eq. 19
	return
}

// Calculate the upward convective mixing rate [1/s] from
// Pleim (2007) equations 9 and 11b when given the height of the top of the first
// model layer (z1plushalf [m]), the height of the top of the second model
// layer (z2plushalf [m]), the boundary layer height (h [m]), the
// Monin-Obukhov length (L [m]), and the friction velocity (ustar [m/s]).
func M2u(z1plushalf, z2plushalf, h, L, ustar, fconv float64) float64 {
	kh := calculateKh(z1plushalf, h, L, ustar)
	Δz1plushalf := z2plushalf / 2.
	return fconv * kh / Δz1plushalf / max(1., h-z1plushalf)
}

// Calculate the downward convective mixing rate [1/s] from
// Pleim (2007) equation 4 when given the upward convective mixing
// rate (M2u [1/s], the height of the bottom of the current model
// layer (z [m]), the thickness of the current model layer (Δz [m])
func M2d(M2u, z, Δz, h float64) float64 {
	return M2u * (h - z) / Δz
}

// Calculate local mixing coefficient [m2/s] based on Pleim (2007)
// equation 11b when given height at the bottom of
// the current model layer (z [m]), boundary layer height (h [m]),
// Monin-Obukhov length (L [m]), and friction velocity (ustar [m/s]),
// and convective mixing fraction (fconv [-]).
func Kzz(z, h, L, ustar, fconv float64) float64 {
	km := CalculateKm(z, h, L, ustar)
	return km * (1 - fconv)
}

// Calculate heat diffusivity
func calculateKh(z, h, L, ustar float64) (kh float64) {
	var zs, ϕ_h float64
	if L < 0. { // Unstable conditions
		// Pleim 2007, equation 12.5
		zs = min(z, 0.1*h)
		// Pleim Eq. 13
		ϕ_h = math.Pow(1.-16.*zs/L, -0.5)
	} else { // Stable conditions
		zs = z
		// Dyer, 1974 (Concluding Remarks)
		ϕ_h = 1. + 5.*zs/L
	}
	// Pleim Eq. 12; units = m2/s
	kh = κ * ustar / ϕ_h * z * math.Pow(1-z/h, 2)
	return
}

// Calculate mass diffusivity [m2/s] within the boundary
// layer when given
// height (z [m]), boundary layer height (h [m]),
// Monin-Obukhov length (L [m]), and friction velocity
// (ustar [m/s]).
func CalculateKm(z, h, L, ustar float64) (km float64) {
	var zs, ϕ_m float64
	if L < 0. { // Unstable conditions
		// Pleim 2007, equation 12.5
		zs = min(z, 0.1*h)
		// Pleim Eq. 13
		ϕ_m = math.Pow(1.-16.*zs/L, -0.25)
	} else { // Stable conditions
		zs = z
		// Dyer, 1974 (Concluding Remarks)
		ϕ_m = 1. + 5.*zs/L
	}
	// Pleim Eq. 12; units = m2/s
	km = κ * ustar / ϕ_m * z * math.Pow(1-z/h, 2)
	return
}

func max(vals ...float64) float64 {
	maxval := vals[0]
	for _, val := range vals {
		if val > maxval {
			maxval = val
		}
	}
	return maxval
}

func min(vals ...float64) float64 {
	minval := vals[0]
	for _, val := range vals {
		if val < minval {
			minval = val
		}
	}
	return minval
}
