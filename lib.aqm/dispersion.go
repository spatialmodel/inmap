package aqm

import (
	"bitbucket.org/ctessum/aep/sparse"
	"math"
)

// DiffusiveFlux calculates diffusive fluxes given diffusivity (D; m2/s) and
// initial concentration (Co; arbitrary units) arrays, x, y, and z array
// indicies (i,j, and k, respectively) and x, y, and z grid
// resolutions (dx,dy,dz; units of meters). Returns diffusive flux
// (from Fick's first law)
// in units of (Co units per second).
func DiffusiveFlux(D, Co *sparse.DenseArray, i, j, k int,
	dx, dy, dz float64) (xdiff, ydiff, zdiff float64) {

	// deal with boundaries
	// assume inversion at top layer
	// no flux boundary at bottom layer
	// this could be avoided by using staggered grid.
	var klo, khi int
	if k == 0 {
		klo = 1
	} else {
		klo = k
	}
	if k == Co.Shape[2]-1 {
		khi = Co.Shape[2] - 2
	} else {
		khi = k
	}

	Di := D.Get(i, j, k)
	Diplus := D.Get(i+1, j, k)
	Diminus := D.Get(i-1, j, k)
	Coi := Co.Get(i, j, k)
	Coiplus := Co.Get(i+1, j, k)
	Coiminus := Co.Get(i-1, j, k)

	xdiff = 2*Di*Diplus/(Di+Diplus)*(Coiplus-Coi)/math.Pow(dx, 2) -
		2*Di*Diminus/(Di+Diminus)*(Coi-Coiminus)/math.Pow(dx, 2)

	Djplus := D.Get(i, j+1, k)
	Djminus := D.Get(i, j-1, k)
	Cojplus := Co.Get(i, j+1, k)
	Cojminus := Co.Get(i, j-1, k)

	ydiff = 2*Di*Djplus/(Di+Djplus)*(Cojplus-Coi)/math.Pow(dy, 2) -
		2*Di*Djminus/(Di+Djminus)*(Coi-Cojminus)/math.Pow(dy, 2)

	Dkplus := D.Get(i, j, khi+1)
	Dkminus := D.Get(i, j, klo-1)
	Cokplus := Co.Get(i, j, khi+1)
	Cokminus := Co.Get(i, j, klo-1)

	zdiff = 2*Di*Dkplus/(Di+Dkplus)*(Cokplus-Coi)/math.Pow(dz, 2) -
		2*Di*Dkminus/(Di+Dkminus)*(Coi-Cokminus)/math.Pow(dz, 2)

	return
}

// Advective flux is calcuated based on an initial concentration array (Co,
// arbitrary units), x, y, and z wind speed (U, V, and W, respectively; units
// of meters per second), x, y, and z array indicies (i,j, and k, respectively)
// and x, y, and z grid resolutions (dx,dy,dz; units of meters).
// Results are in units of (Co units per second).
func AdvectiveFlux(Co *sparse.DenseArray, U, V, W float64, i, j, k int,
	dx, dy, dz float64) (xadv, yadv, zadv float64) {

	// deal with boundaries
	// assume inversion at top layer
	// no flux boundary at bottom layer
	// this could be avoided by using staggered grid.
	var klo, khi int
	if k == 0 {
		klo = 1
	} else {
		klo = k
	}
	if k == Co.Shape[2]-1 {
		khi = Co.Shape[2] - 2
	} else {
		khi = k
	}
	Coi := Co.Get(i, j, k)
	Coiplus := Co.Get(i+1, j, k)
	Coiminus := Co.Get(i-1, j, k)
	Cojplus := Co.Get(i, j+1, k)
	Cojminus := Co.Get(i, j-1, k)
	Cokplus := Co.Get(i, j, khi+1)
	Cokminus := Co.Get(i, j, klo-1)

	if U >= 0. {
		xadv = U * (Coiminus - Coi) / dx
	} else {
		xadv = U * (Coi - Coiplus) / dx
	}
	if V >= 0. {
		yadv = V * (Cojminus - Coi) / dy
	} else {
		yadv = V * (Coi - Cojplus) / dy
	}
	if W >= 0. {
		zadv = W * (Cokminus - Coi) / dz
	} else {
		zadv = W * (Coi - Cokplus) / dz
	}
	return
}

// Reactive flux calculates the formation and destruction of pollutants
// based on an initial concentration values (VOCi, PM25i, NH3i, SOxi NOxi;
// arbitrary units), x, y, and z array indicies (i,j, and k, respectively)
// and x, y, and z grid resolutions (dx,dy,dz; units of meters) and time step
// (dt; units of seconds).
// The returned values are the resulting concentrations of each pollutant.
func ReactiveFlux(VOCi, PM25i, NH3i, SOxi, NOxi, dt float64) (
	VOCf, PM25f, NH3f, SOxf, NOxf float64) {

	VOCf = VOCi
	PM25f = PM25i
	NH3f = NH3i
	SOxf = SOxi
	NOxf = NOxi
	return
}
