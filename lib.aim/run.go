package aim

import (
	"bitbucket.org/ctessum/sparse"
	"fmt"
	"math"
	"time"
)

// Chemical mass conversions
const (
	// grams per mole
	mwNOx = 46.0055
	mwN   = 14.0067
	mwNO3 = 62.00501
	mwNH3 = 17.03056
	mwNH4 = 18.03851
	mwS   = 32.0655
	mwSO2 = 64.0644
	mwSO4 = 96.0632
	// ratios
	NOxToN = mwN / mwNOx
	NtoNO3 = mwNO3 / mwN
	SOxToS = mwSO2 / mwS
	StoSO4 = mwS / mwSO4
	NH3ToN = mwN / mwNH3
	NtoNH4 = mwNH4 / mwN
)

//const nDaysCheckConvergence = 1.
const nDaysCheckConvergence = 0.01
const tolerance = 0.001
const secondsPerDay = 1. / 3600. / 24.

// These are the names of pollutants accepted as emissions (μg/s)
var EmisNames = []string{"VOC", "NOx", "NH3", "SOx", "PM2_5"}

// These are the names of pollutants within the model
var polNames = []string{"gOrg", "pOrg", // gaseous and particulate organic matter
	"PM2_5",      // PM2.5
	"gNH", "pNH", // gaseous and particulate N in ammonia
	"gS", "pS", // gaseous and particulate S in sulfur
	"gNO", "pNO"} // gaseous and particulate N in nitrate

// Indicies of individual pollutants in arrays.
const (
	igOrg, ipOrg, iPM2_5, igNH, ipNH, igS, ipS, igNO, ipNO = 0, 1, 2, 3, 4, 5, 6, 7, 8
)

// These are the names of pollutants output by the model (μg/m3)
var OutputNames = []string{"VOC", "SOA", "PrimaryPM2_5", "NH3", "pNH4",
	"SOx", "pSO4", "NOx", "pNO3", "TotalPM2_5"}

// Run air quality model. Emissions are assumed to be in units
// of μg/s, and must only include the pollutants listed in "EmisNames".
func (m *MetData) Run(emissions map[string]*sparse.DenseArray) (
	outputConc map[string]*sparse.DenseArray) {

	startTime := time.Now()
	timeStepTime := time.Now()

	// Emissions: all except PM2.5 go to gas phase
	emisFlux := make(map[string]*sparse.DenseArray)
	for pol, arr := range emissions {
		switch pol {
		case "VOC":
			emisFlux["gOrg"] = m.calcEmisFlux(arr, 1.)
		case "NOx":
			emisFlux["gNO"] = m.calcEmisFlux(arr, NOxToN)
		case "NH3":
			emisFlux["gNH"] = m.calcEmisFlux(arr, NH3ToN)
		case "SOx":
			emisFlux["gS"] = m.calcEmisFlux(arr, SOxToS)
		case "PM2_5":
			emisFlux["PM2_5"] = m.calcEmisFlux(arr, 1.)
		default:
			panic(fmt.Sprintf("Unknown emissions pollutant %v.", pol))
		}
	}

	// Initialize arrays
	// values at start of timestep
	m.initialConc = make([]*sparse.DenseArray, len(polNames))
	// values at end of timestep
	m.finalConc = make([]*sparse.DenseArray, len(polNames))
	// arrays for calculating convergence
	oldFinalConcSum := make([]float64, len(polNames))
	finalConcSum := make([]float64, len(polNames))
	for i, _ := range polNames {
		m.initialConc[i] = sparse.ZerosDense(m.Nz, m.Ny, m.Nx)
		m.finalConc[i] = sparse.ZerosDense(m.Nz, m.Ny, m.Nx)
	}

	iteration := 0
	nDaysRun := 0.
	nDaysSinceConvergenceCheck := 0.
	nIterationsSinceConvergenceCheck := 0
	for {
		iteration++
		nIterationsSinceConvergenceCheck++
		m.newRand()  // set random numbers for weighted random walk
		m.setTstep() // set timestep
		nDaysRun += m.Dt * secondsPerDay
		nDaysSinceConvergenceCheck += m.Dt * secondsPerDay
		fmt.Printf("马上。。。Iteration %v\twalltime=%.4gh\tΔwalltime=%.2gs\t"+
			"timestep=%.0fs\tday=%.3g\n",
			iteration, time.Since(startTime).Hours(),
			time.Since(timeStepTime).Seconds(), m.Dt, nDaysRun)
		timeStepTime = time.Now()

		// Add in emissions
		m.arrayLock.Lock()
		for i, pol := range polNames {
			if arr, ok := emisFlux[pol]; ok {
				m.initialConc[i].AddDense(arr.ScaleCopy(m.Dt))
			}
		}
		m.arrayLock.Unlock()

		type empty struct{}
		sem := make(chan empty, m.Nz) // semaphore pattern
		for i := 1; i < m.Nx-1; i += 1 {
			go func(i int) { // concurrent processing
				var xadv, yadv, zadv, zdiff float64
				c := new(Neighborhood)
				d := new(Neighborhood)
				tempconc := make([]float64, len(polNames)) // concentration holder
				for j := 1; j < m.Ny-1; j += 1 {
					for k := 0; k < m.Nz; k += 1 {
						Uminus := m.getBinX(m.Ufreq, m.Ubins, k, j, i)
						Uplus := m.getBinX(m.Ufreq, m.Ubins, k, j, i+1)
						Vminus := m.getBinY(m.Vfreq, m.Vbins, k, j, i)
						Vplus := m.getBinY(m.Vfreq, m.Vbins, k, j+1, i)
						Wminus := m.getBinZ(m.Wfreq, m.Wbins, k, j, i)
						Wplus := m.getBinZ(m.Wfreq, m.Wbins, k+1, j, i)
						FillKneighborhood(d, m.verticalDiffusivity, k, j, i)
						for q, Carr := range m.initialConc {
							FillNeighborhood(c, Carr, m.Dz, k, j, i)
							zdiff = m.DiffusiveFlux(c, d)
							//xadv, yadv, zadv = m.AdvectiveFluxRungeKutta(
							//xadv, yadv, zadv = m.AdvectiveFluxRungeKuttaJacobson(
							xadv, yadv, zadv = m.AdvectiveFluxUpwind(
								c, Uminus, Uplus, Vminus, Vplus, Wminus, Wplus)

							var gravSettling float64
							var VOCoxidation float64
							switch q {
							case iPM2_5, ipOrg, ipNH, ipNO, ipS:
								gravSettling = m.GravitationalSettling(c, k)
							case igOrg:
								VOCoxidation = m.VOCoxidationFlux(c)
							}

							tempconc[q] = Carr.Get(k, j, i) +
								(xadv + yadv + zadv + gravSettling + VOCoxidation +
									zdiff)
						}
						m.WetDeposition(tempconc, k, j, i)
						m.ChemicalPartitioning(tempconc, k, j, i)

						for q, val := range tempconc {
							m.finalConc[q].Set(val, k, j, i)
						}
					}
				}
				sem <- empty{}
			}(i)
		}
		for i := 1; i < m.Nx-1; i++ { // wait for routines to finish
			<-sem
		}
		for i, arr := range m.finalConc {
			finalConcSum[i] += arr.Sum()
		}
		if nDaysSinceConvergenceCheck > nDaysCheckConvergence {
			timeToQuit := true
			for q := 0; q < len(finalConcSum); q++ {
				finalConcSum[q] /= float64(nIterationsSinceConvergenceCheck)
				if !checkConvergence(finalConcSum[q], oldFinalConcSum[q], polNames[q]) {
					timeToQuit = false
				}
				oldFinalConcSum[q] = finalConcSum[q]
			}
			nDaysSinceConvergenceCheck = 0.
			nIterationsSinceConvergenceCheck = 0
			if timeToQuit {
				break
			}
		}
		m.arrayLock.Lock()
		for q, _ := range m.finalConc {
			m.initialConc[q] = m.finalConc[q].Copy()
			m.finalConc[q] = sparse.ZerosDense(m.Nz, m.Ny, m.Nx)
		}
		m.arrayLock.Unlock()
	}
	outputConc = make(map[string]*sparse.DenseArray)
	outputConc["VOC"] = m.finalConc[igOrg]                       // gOrg
	outputConc["SOA"] = m.finalConc[ipOrg]                       // pOrg
	outputConc["PrimaryPM2_5"] = m.finalConc[iPM2_5]             // PM2_5
	outputConc["NH3"] = m.finalConc[igNH].ScaleCopy(1. / NH3ToN) // gNH
	outputConc["pNH4"] = m.finalConc[ipNH].ScaleCopy(NtoNH4)     // pNH
	outputConc["SOx"] = m.finalConc[igS].ScaleCopy(1. / SOxToS)  // gS
	outputConc["pSO4"] = m.finalConc[ipS].ScaleCopy(StoSO4)      // pS
	outputConc["NOx"] = m.finalConc[igNO].ScaleCopy(1. / NOxToN) // gNO
	outputConc["pNO3"] = m.finalConc[ipNO].ScaleCopy(NtoNO3)     // pNO
	outputConc["TotalPM2_5"] = m.finalConc[iPM2_5].Copy()
	outputConc["TotalPM2_5"].AddDense(outputConc["SOA"])
	outputConc["TotalPM2_5"].AddDense(outputConc["pNH4"])
	outputConc["TotalPM2_5"].AddDense(outputConc["pSO4"])
	outputConc["TotalPM2_5"].AddDense(outputConc["pNO3"])

	return
}

// Calculate emissions flux given emissions array in units of μg/s
// and a scale for molecular mass conversion.
func (m *MetData) calcEmisFlux(arr *sparse.DenseArray, scale float64) (
	emisFlux *sparse.DenseArray) {
	emisFlux = sparse.ZerosDense(m.Nz, m.Ny, m.Nx)
	for k := 0; k < m.Nz; k++ {
		for j := 0; j < m.Ny; j++ {
			for i := 0; i < m.Nx; i++ {
				fluxScale := 1. / m.Dx / m.Dy /
					m.Dz.Get(k, j, i) // μg/s /m/m/m = μg/m3/s
				emisFlux.Set(arr.Get(k, j, i)*scale*fluxScale, k, j, i)
			}
		}
	}
	return
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

func checkConvergence(newSum, oldSum float64, name string) bool {
	bias := (newSum - oldSum) / oldSum
	fmt.Printf("%v: difference = %3.2g%%\n", name, bias*100)
	if math.Abs(bias) > tolerance || math.IsInf(bias, 0) {
		return false
	} else {
		return true
	}
}
