package aim

import (
	"bitbucket.org/ctessum/sparse"
	"code.google.com/p/lvd.go/cdf"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sync"
)

const (
	dp    = 1.e-6   // m, particle diameter
	rhof  = 1.2466  // kg/m3, air density
	rhop  = 1000.   // kg/m3, density of droplet
	g     = 9.80665 // m/s2
	mu    = 1.5e-5  // kg/m/s
	kappa = 0.4     // von karmon's constant
	//		T            = 10. + 273.15 // K, atmospheric temperature
)

type MetData struct {
	nbins, Nx, Ny, Nz              int
	xFactor, yFactor, zFactor      int
	Ubins, Vbins, Wbins            *sparse.DenseArray // m/s
	Ufreq, Vfreq, Wfreq            *sparse.DenseArray // fraction
	orgPartitioning, SPartitioning *sparse.DenseArray // gaseous fraction
	NOPartitioning, NHPartitioning *sparse.DenseArray // gaseous fraction
	wdParticle, wdSO2, wdOtherGas  *sparse.DenseArray // wet deposition rate, 1/s
	layerHeights                   *sparse.DenseArray // heights at layer edges, m
	verticalDiffusivity            *sparse.DenseArray // vertical diffusivity, m2/s
	temperature                    *sparse.DenseArray // Average temperature, K
	windSpeed                      *sparse.DenseArray // RMS wind speed, m/s
	s1                             *sparse.DenseArray // stability parameter
	sClass                         *sparse.DenseArray // stability class: "0=Unstable; 1=Stable
	Dx, Dy                         float64            // meters
	Dz                             *sparse.DenseArray // meters, varies by grid cell
	Dt                             float64            // seconds
	vs                             float64            // Settling velocity, m/s
	wg                             sync.WaitGroup
	VOCoxidationRate               float64 // VOC oxidation rate constant
	xRandom                        float64 // random number set with newRand()
	yRandom                        float64 // random number set with newRand()
	zRandom                        float64 // random number set with newRand()

}

func InitMetData(filename string, zFactor, yFactor, xFactor int) *MetData {
	m := new(MetData)
	ff, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer ff.Close()
	f, err := cdf.Open(ff)
	if err != nil {
		panic(err)
	}
	m.nbins = f.Header.Lengths("Ubins")[0]
	dims := f.Header.Lengths("orgPartitioning")
	m.Nz = dims[0] * zFactor
	m.Ny = dims[1] * yFactor
	m.Nx = dims[2] * xFactor
	m.Dx, m.Dy = 12000./float64(xFactor), 12000./float64(yFactor)
	m.xFactor, m.yFactor, m.zFactor = xFactor, yFactor, zFactor
	m.VOCoxidationRate = f.Header.GetAttribute("", "VOCoxidationRate").([]float64)[0]
	m.wg.Add(19)
	go m.readNCF(filename, "Ubins")
	go m.readNCF(filename, "Vbins")
	go m.readNCF(filename, "Wbins")
	go m.readNCF(filename, "Ufreq")
	go m.readNCF(filename, "Vfreq")
	go m.readNCF(filename, "Wfreq")
	go m.readNCF(filename, "orgPartitioning")
	go m.readNCF(filename, "SPartitioning")
	go m.readNCF(filename, "NOPartitioning")
	go m.readNCF(filename, "NHPartitioning")
	go m.readNCF(filename, "wdParticle")
	go m.readNCF(filename, "wdSO2")
	go m.readNCF(filename, "wdOtherGas")
	go m.readNCF(filename, "layerHeights")
	go m.readNCF(filename, "temperature")
	go m.readNCF(filename, "windSpeed")
	go m.readNCF(filename, "S1")
	go m.readNCF(filename, "Sclass")
	go m.readNCF(filename, "verticalDiffusivity")
	m.wg.Wait()
	// calculate Dz (varies by layer)
	m.Dz = sparse.ZerosDense(m.Nz, m.Ny, m.Nx)
	for k := 0; k < m.Nz; k++ {
		for j := 0; j < m.Ny; j++ {
			for i := 0; i < m.Nx; i++ {
				m.Dz.Set(m.layerHeights.Get(k+1, j, i)-
					m.layerHeights.Get(k, j, i), k, j, i)
			}
		}
	}
	// Settling velocity, m/s
	m.vs = (rhop - rhof) * g * dp * dp / -18. / mu
	fmt.Printf("Settling velocity: %v s\n", m.vs)
	return m
}

// set random numbers for weighted random walk.
func (m *MetData) newRand() {
	m.xRandom = rand.Float64()
	m.yRandom = rand.Float64()
	m.zRandom = rand.Float64()
}

// choose a bin using a weighted random method and return the bin value
func (m *MetData) getBin(freqs, vals *sparse.DenseArray, k, j, i int,
	r float64) float64 {
	for b := 0; b < m.nbins; b++ {
		if r <= freqs.Get(b, k, j, i) {
			return vals.Get(b, k, j, i)
		}
	}
	panic(fmt.Sprintf("Could not choose a bin using seed %v "+
		"(max cumulative frequency=%v).", r, vals.Get(m.nbins-1, k, j, i)))
	return 0.
}

func (m *MetData) getBinX(freqs, vals *sparse.DenseArray, k, j, i int) float64 {
	return m.getBin(freqs, vals, k, j, i, m.xRandom)
}
func (m *MetData) getBinY(freqs, vals *sparse.DenseArray, k, j, i int) float64 {
	return m.getBin(freqs, vals, k, j, i, m.yRandom)
}
func (m *MetData) getBinZ(freqs, vals *sparse.DenseArray, k, j, i int) float64 {
	return m.getBin(freqs, vals, k, j, i, m.zRandom)
}

//  Set the time step using the Courant–Friedrichs–Lewy (CFL) condition.
func (m *MetData) setTstep() {
	//m.Dt = m.Dx * 3 / 1000.
	m.Dt = 30.
	//	const Cmax = 1.26 * 0.75
	//	val := 0.
	//	// don't worry about the edges of the staggered grids.
	//	for k := 0; k < m.Nz; k++ {
	//		for j := 0; j < m.Ny; j++ {
	//			for i := 0; i < m.Nx; i++ {
	//				uval := math.Abs(m.getBin(m.Ufreq, m.Ubins,k, j, i)) / m.Dx
	//				vval := math.Abs(m.getBin(m.Vfreq, m.Vbins,k, j, i)) / m.Dy
	//				wval := math.Abs(m.getBin(m.Wfreq, m.Wbins,k, j, i)) /
	//					m.Dz.Get(k, j, i)
	//				thisval := max(uval,vval,wval)
	//				if thisval > val {
	//					val = thisval
	//				}
	//			}
	//		}
	//	}
	//	m.Dt = Cmax / math.Pow(3., 0.5) / val // seconds
}

func min(v1, v2 float64) float64 {
	if v1 < v2 {
		return v1
	} else {
		return v2
	}
}

func (m *MetData) readNCF(filename, Var string) {
	ff, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	f, err := cdf.Open(ff)
	if err != nil {
		panic(err)
	}
	dims := f.Header.Lengths(Var)
	defer ff.Close()
	defer m.wg.Done()
	nread := 1
	for _, dim := range dims {
		nread *= dim
	}
	r := f.Reader(Var, nil, nil)
	buf := r.Zero(nread)
	_, err = r.Read(buf)
	if err != nil {
		panic(err)
	}
	dat := buf.([]float32)
	out := sparse.ZerosDense(dims...)
	for i, val := range dat {
		out.Elements[i] = float64(val)
	}
	switch Var {
	case "Ubins":
		m.Ubins = out
	case "Vbins":
		m.Vbins = out
	case "Wbins":
		m.Wbins = out
	case "Ufreq":
		m.Ufreq = out
	case "Vfreq":
		m.Vfreq = out
	case "Wfreq":
		m.Wfreq = out
	case "orgPartitioning":
		m.orgPartitioning = out
	case "SPartitioning":
		m.SPartitioning = out
	case "NOPartitioning":
		m.NOPartitioning = out
	case "NHPartitioning":
		m.NHPartitioning = out
	case "wdParticle":
		m.wdParticle = out
	case "wdSO2":
		m.wdSO2 = out
	case "wdOtherGas":
		m.wdOtherGas = out
	case "verticalDiffusivity":
		m.verticalDiffusivity = out
	case "layerHeights":
		m.layerHeights = out
	case "temperature":
		m.temperature = out
	case "windSpeed":
		m.windSpeed = out
	case "S1":
		m.s1 = out
	case "Sclass":
		m.sClass = out
	default:
		panic(fmt.Sprintf("Unknown variable %v.", Var))
	}
}

// Lower boundary is same as lowest grid cell value.
func getLowerBoundary(C *sparse.DenseArray, j, i int) float64 {
	return C.Get(0, j, i)
}

// All other boundaries = 0.
func getUpperBoundary(_, _ int) float64 { return 0. }
func getNorthBoundary(_, _ int) float64 { return 0. }
func getSouthBoundary(_, _ int) float64 { return 0. }
func getEastBoundary(_, _ int) float64  { return 0. }
func getWestBoundary(_, _ int) float64  { return 0. }

type Neighborhood struct {
	center, iplus, iminus, jplus, jminus, kplus, kminus float64
	//	i2plus, j2plus, k2plus, i2minus, j2minus, k2minus   float64
	Dz, Dzsquared float64 // Dz varies by grid cell
}

// For a given array index, get the value at the index,
// plus the values of all the neighbors. The input array
// needs to be 3d.
func FillNeighborhood(n *Neighborhood, A, Dz *sparse.DenseArray, k, j, i int) {
	nx := A.Shape[2]
	ny := A.Shape[1]
	nz := A.Shape[0]
	zStride := nx * ny
	x := A.Index1d(k, j, i)
	n.center = A.Elements[x]
	if i == nx-1 {
		n.iplus = getEastBoundary(k, j)
		//		n.i2plus = getEastBoundary(k, j)
		//	} else if i == nx-2 {
		//		n.iplus = A.Elements[x+1]
		//		n.i2plus = getEastBoundary(k, j)
	} else {
		n.iplus = A.Elements[x+1]
		//		n.i2plus = A.Elements[x+2]
	}
	if i == 0 {
		n.iminus = getWestBoundary(k, j)
		//		n.i2minus = getWestBoundary(k, j)
		//	} else if i == 1 {
		//		n.iminus = A.Elements[x-1]
		//		n.i2minus = getWestBoundary(k, j)
	} else {
		n.iminus = A.Elements[x-1]
		//		n.i2minus = A.Elements[x-2]
	}
	if j == ny-1 {
		n.jplus = getNorthBoundary(k, i)
		//		n.j2plus = getNorthBoundary(k, i)
		//	} else if j == ny-2 {
		//		n.jplus = A.Elements[x+nx]
		//		n.j2plus = getNorthBoundary(k, i)
	} else {
		n.jplus = A.Elements[x+nx]
		//		n.j2plus = A.Elements[x+2*nx]
	}
	if j == 0 {
		n.jminus = getSouthBoundary(k, i)
		//		n.j2minus = getSouthBoundary(k, i)
		//	} else if j == 1 {
		//		n.jminus = A.Elements[x-nx]
		//		n.j2minus = getSouthBoundary(k, i)
	} else {
		n.jminus = A.Elements[x-nx]
		//		n.j2minus = A.Elements[x-2*nx]
	}
	if k == nz-1 {
		n.kplus = getUpperBoundary(j, i)
		//		n.k2plus = getUpperBoundary(j, i)
		//	} else if k == nz-2 {
		//		n.kplus = A.Elements[x+zStride]
		//		n.k2plus = getUpperBoundary(j, i)
	} else {
		n.kplus = A.Elements[x+zStride]
		//		n.k2plus = A.Elements[x+2*zStride]
	}
	if k == 0 {
		n.kminus = getLowerBoundary(A, j, i)
		//		n.k2minus = getLowerBoundary(A, j, i)
		//	} else if k == 1 {
		//		n.kminus = A.Elements[x-zStride]
		//		n.k2minus = getLowerBoundary(A, j, i)
	} else {
		n.kminus = A.Elements[x-zStride]
		//		n.k2minus = A.Elements[x-2*zStride]
	}
	n.Dz = Dz.Get(k, j, i)
	n.Dzsquared = n.Dz * n.Dz
	return
}

// Same as GetNeighborhood, but only populates values for center and iplus
func FillIneighborhood(n *Neighborhood, A *sparse.DenseArray, k, j, i int) {
	nx := A.Shape[2]
	x := A.Index1d(k, j, i)
	n.center = A.Elements[x]
	if i == nx-1 {
		n.iplus = getEastBoundary(k, j)
	} else {
		n.iplus = A.Elements[x+1]
	}
	return
}

// Same as GetNeighborhood, but only populates values for center and jplus
func FillJneighborhood(n *Neighborhood, A *sparse.DenseArray, k, j, i int) {
	n = new(Neighborhood)
	x := A.Index1d(k, j, i)
	nx := A.Shape[2]
	ny := A.Shape[1]
	n.center = A.Elements[x]
	if j == ny-1 {
		n.jplus = getNorthBoundary(k, i)
	} else {
		n.jplus = A.Elements[x+nx]
	}
	return
}

// Same as GetNeighborhood, but only populates values for center and kplus
func FillKneighborhood(n *Neighborhood, A *sparse.DenseArray, k, j, i int) {
	x := A.Index1d(k, j, i)
	nx := A.Shape[2]
	ny := A.Shape[1]
	nz := A.Shape[0]
	zStride := nx * ny
	n.center = A.Elements[x]
	if k == nz-1 {
		n.kplus = getUpperBoundary(j, i)
	} else {
		n.kplus = A.Elements[x+zStride]
	}
	return
}

func (n *Neighborhood) belowThreshold(calcMin float64) bool {
	if n.center < calcMin && n.iplus < calcMin && n.iminus < calcMin &&
		n.jplus < calcMin && n.jminus < calcMin && n.kplus < calcMin &&
		n.kminus < calcMin {
		return true
	} else {
		return false
	}
}

// DiffusiveFlux calculates diffusive fluxes given diffusivity (D; m2/s) and
// initial concentration (Co; arbitrary units) arrays, x, y, and z array
// indicies (i,j, and k, respectively) and x, y, and z grid
// resolutions (dx,dy,dz; units of meters). Returns diffusive flux
// (from Fick's first law)
// in units of (Co units).
func (m *MetData) DiffusiveFlux(c, d *Neighborhood) (
	zdiff float64) {

	zdiff = (d.kplus*(c.kplus-c.center)/c.Dzsquared +
		d.center*(c.kminus-c.center)/c.Dzsquared) * m.Dt
	return
}

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
// Returned fluxes are in the same units as c
func (m *MetData) AdvectiveFluxRungeKutta(c *Neighborhood,
	Uminus, Uplus, Vminus, Vplus, Wminus, Wplus float64) (
	xadv, yadv, zadv float64) {
	//	xadv = rk3_4(Uplus, Uminus, c.center, c.iplus, c.iminus,
	//		c.i2plus, c.i2minus, m.Dt, m.Dx)
	//	yadv = rk3_4(Vplus, Vminus, c.center, c.jplus, c.jminus,
	//		c.j2plus, c.j2minus, m.Dt, m.Dy)
	//	zadv = rk3_4(Wplus, Wminus, c.center, c.kplus, c.kminus,
	//		c.k2plus, c.k2minus, m.Dt, c.Dz)
	return
}

// Advective flux is calcuated based on an initial concentration array (Co,
// arbitrary units), x, y, and z wind speed (U, V, and W, respectively; units
// of meters per second), x, y, and z array indicies (i,j, and k, respectively)
// and x, y, and z grid resolutions (dx,dy,dz; units of meters).
// Results are in units of (Co units).
func (m *MetData) AdvectiveFluxUpwind(c *Neighborhood,
	U, Unext, V, Vnext, W, Wnext float64) (
	xadv, yadv, zadv float64) {

	if U > 0. {
		xadv += U * c.iminus / m.Dx * m.Dt
	} else {
		xadv += U * c.center / m.Dx * m.Dt
	}
	if Unext > 0. {
		xadv -= Unext * c.center / m.Dx * m.Dt
	} else {
		xadv -= Unext * c.iplus / m.Dx * m.Dt
	}

	if V > 0. {
		yadv += V * c.jminus / m.Dy * m.Dt
	} else {
		yadv += V * c.center / m.Dy * m.Dt
	}
	if Vnext > 0. {
		yadv -= Vnext * c.center / m.Dy * m.Dt
	} else {
		yadv -= Vnext * c.jplus / m.Dy * m.Dt
	}

	if W > 0. {
		zadv += W * c.kminus / c.Dz * m.Dt
	} else {
		zadv += W * c.center / c.Dz * m.Dt
	}
	if Wnext > 0. {
		zadv -= Wnext * c.center / c.Dz * m.Dt
	} else {
		zadv -= Wnext * c.kplus / c.Dz * m.Dt
	}
	return
}

func (m *MetData) WetDeposition(conc []float64, k, j, i int) {
	particleFrac := 1. - m.wdParticle.Get(k, j, i)*m.Dt
	SO2Frac := 1. - m.wdSO2.Get(k, j, i)*m.Dt
	otherGasFrac := 1 - m.wdOtherGas.Get(k, j, i)*m.Dt
	conc[igOrg] *= otherGasFrac  // gOrg
	conc[ipOrg] *= particleFrac  // pOrg
	conc[iPM2_5] *= particleFrac // PM2_5
	conc[igNH] *= otherGasFrac   // gNH
	conc[ipNH] *= particleFrac   // pNH
	conc[igS] *= SO2Frac         // gS
	conc[ipS] *= particleFrac    // pS
	conc[igNO] *= otherGasFrac   // gNO
	conc[ipNO] *= particleFrac   // pNO
}

// Reactive flux partitions organic matter ("gOrg" and "pOrg"), the
// nitrogen in nitrate ("gNO and pNO"), the nitrogen in ammonia ("gNH" and
// "pNH) and sulfur ("gS" and "pS") between gaseous and particulate phase
// based on the spatially explicit partioning present in the baseline data.
// Inputs are an array of initial concentrations ("conc") and grid index
// ("k", "j", and "i").
func (m *MetData) ChemicalPartitioning(conc []float64, k, j, i int) {

	// Gas/particle partitioning
	totalOrg := conc[igOrg] + conc[ipOrg] // gOrg + pOrg
	gasFrac := m.orgPartitioning.Get(k, j, i)
	conc[igOrg] = totalOrg * gasFrac       // gOrg
	conc[ipOrg] = totalOrg * (1 - gasFrac) // pOrg

	totalS := conc[igS] + conc[ipS] // gS + pS
	gasFrac = m.SPartitioning.Get(k, j, i)
	conc[igS] = totalS * gasFrac       // gS
	conc[ipS] = totalS * (1 - gasFrac) // pS

	totalNO := conc[igNO] + conc[ipNO] // gNO + pNO
	gasFrac = m.NOPartitioning.Get(k, j, i)
	conc[igNO] = totalNO * gasFrac       // gNO
	conc[ipNO] = totalNO * (1 - gasFrac) // pNO

	totalNH := conc[igNH] + conc[ipNH] // gNH + pNH
	gasFrac = m.NHPartitioning.Get(k, j, i)
	conc[igNH] = totalNH * gasFrac       // gNH
	conc[ipNH] = totalNH * (1 - gasFrac) // pNH
}

// VOC oxidation flux
func (m *MetData) VOCoxidationFlux(c *Neighborhood) float64 {
	return -c.center * m.VOCoxidationRate * m.Dt
}

func (m *MetData) GravitationalSettling(c *Neighborhood, k int) float64 {
	if k == 0 {
		return m.vs * c.center / c.Dz * m.Dt
	} else {
		return m.vs * (c.center - c.kplus) / c.Dz * m.Dt
	}
}

// CalcPlumeRise takes emissions stack height(m), diameter (m), temperature (K),
// and exit velocity (m/s) and calculates the k index of the equivalent
// emissions height after accounting for plume rise at grid index (y=j,x=i).
// Uses the plume rise calculation: ASME (1973), as described in Sienfeld and Pandis,
// ``Atmospheric Chemistry and Physics - From Air Pollution to Climate Change
func (m *MetData) CalcPlumeRise(stackHeight, stackDiam, stackTemp,
	stackVel float64, j, i int) (kPlume int) {
	// Find K level of stack
	kStak := 0
	for m.layerHeights.Get(kStak+1, j, i) < stackHeight {
		if kStak > m.Nz {
			msg := "stack height > top of grid"
			panic(msg)
		}
		kStak++
	}
	deltaH := 0. // Plume rise, (m).
	var calcType string

	airTemp := m.temperature.Get(kStak, j, i)
	windSpd := m.windSpeed.Get(kStak, j, i)

	if (stackTemp-airTemp) < 50. &&
		stackVel > windSpd && stackVel > 10. {
		// Plume is dominated by momentum forces
		calcType = "Momentum"

		deltaH = stackDiam * math.Pow(stackVel, 1.4) / math.Pow(windSpd, 1.4)

	} else { // Plume is dominated by buoyancy forces

		// Bouyancy flux, m4/s3
		F := g * (stackTemp - airTemp) / stackTemp * stackVel *
			math.Pow(stackDiam/2, 2)

		if m.sClass.Get(kStak, j, i) > 0.5 { // stable conditions
			calcType = "Stable"

			deltaH = 29. * math.Pow(
				F/m.s1.Get(kStak, j, i), 0.333333333) /
				math.Pow(windSpd, 0.333333333)

		} else { // unstable conditions
			calcType = "Unstable"

			deltaH = 7.4 * math.Pow(F*math.Pow(stackHeight, 2.),
				0.333333333) / windSpd

		}
	}
	if math.IsNaN(deltaH) {
		msg := "plume height == NaN\n" +
			fmt.Sprintf("calcType: %v, deltaH: %v, stackDiam: %v,\n",
				calcType, deltaH, stackDiam) +
			fmt.Sprintf("stackVel: %v, windSpd: %v, stackTemp: %v,\n",
				stackVel, windSpd, stackTemp) +
			fmt.Sprintf("airTemp: %v, stackHeight: %v\n", airTemp, stackHeight)
		panic(msg)
	}

	plumeHeight := stackHeight + deltaH

	// Find K level of plume
	for kPlume = 0; m.layerHeights.Get(kPlume+1, j, i) < plumeHeight; kPlume++ {
		if kPlume > m.Nz {
			break
		}
	}
	return
}
