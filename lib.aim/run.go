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
func (d *AIMdata) Run(emissions map[string]*sparse.DenseArray) (
	outputConc map[string]*sparse.DenseArray) {

	startTime := time.Now()
	timeStepTime := time.Now()

	// Emissions: all except PM2.5 go to gas phase
	for pol, arr := range emissions {
		switch pol {
		case "VOC":
			d.addEmisFlux(arr, 1., igOrg)
		case "NOx":
			d.addEmisFlux(arr, NOxToN, igNO)
		case "NH3":
			d.addEmisFlux(arr, NH3ToN, igNH)
		case "SOx":
			d.addEmisFlux(arr, SOxToS, igS)
		case "PM2_5":
			d.addEmisFlux(arr, 1., iPM2_5)
		default:
			panic(fmt.Sprintf("Unknown emissions pollutant %v.", pol))
		}
	}

	// Initialize arrays
	// arrays for calculating convergence
	oldFinalConcSum := make([]float64, len(polNames))
	finalConcSum := make([]float64, len(polNames))

	iteration := 0
	nDaysRun := 0.
	nDaysSinceConvergenceCheck := 0.
	nIterationsSinceConvergenceCheck := 0
	for {
		iteration++
		nIterationsSinceConvergenceCheck++
		d.SetupTimeStep() // prepare data for this time step
		nDaysRun += d.Dt * secondsPerDay
		nDaysSinceConvergenceCheck += d.Dt * secondsPerDay
		fmt.Printf("马上。。。Iteration %v\twalltime=%.4gh\tΔwalltime=%.2gs\t"+
			"timestep=%.0fs\tday=%.3g\n",
			iteration, time.Since(startTime).Hours(),
			time.Since(timeStepTime).Seconds(), d.Dt, nDaysRun)
		timeStepTime = time.Now()

		type empty struct{}
		sem := make(chan empty, m.Nz) // semaphore pattern
		for ii := 1; ii < len(d.Data); ii += 1 {
			go func(ii int) { // concurrent processing
				c := d.Data[ii]
				//zdiff = m.DiffusiveFlux(c, d)
				c.AdvectiveFluxUpwind(d.Dt)
				c.GravitationalSettling(d)
				c.VOCoxidationFlux(d)
				c.WetDeposition(d.Dt)
				c.ChemicalPartitioning(d.Dt)
				d.Data[ii] = c

				sem <- empty{}
			}(ii)
		}
		for ii := 1; ii < len(d.Data); ii += 1 { // wait for routines to finish
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
func (d *AIMdata) addEmisFlux(arr *sparse.DenseArray, scale float64, iPol int) {
	ii := 0
	for k := 0; k < d.Nz; k++ {
		for j := 0; j < d.Ny; j++ {
			for i := 0; i < d.Nx; i++ {
				fluxScale := 1. / d.Data[ii].Dx / d.Data[ii].Dy /
					d.Data[ii].Dz // μg/s /m/m/m = μg/m3/s
				d.Data[ii].emisFlux[iPol] = arr.Get(k, j, i) * scale * fluxScale
				ii++
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
