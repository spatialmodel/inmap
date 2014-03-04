package aim

import (
	"fmt"
	"math"
	"runtime"
	"sync"
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

const tolerance = 0.005   // tolerance for convergence
const checkPeriod = 3600. // seconds, how often to check for convergence
const daysPerSecond = 1. / 3600. / 24.
const topLayerToCalc = 28 // The top layer to do calculations for

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
func (d *AIMdata) Run(emissions map[string][]float64) (
	outputConc map[string][][]float64) {

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

	oldSum := make([]float64, len(polNames))
	iteration := 0
	nDaysRun := 0.
	timeSinceLastCheck := 0.
	nprocs := runtime.GOMAXPROCS(0) // number of processors
	funcChan := make([]chan func(*AIMcell, *AIMdata), nprocs)
	var wg sync.WaitGroup

	for procNum := 0; procNum < nprocs; procNum++ {
		funcChan[procNum] = make(chan func(*AIMcell, *AIMdata), 1)
		// Start thread for concurrent computations
		go d.doScience(nprocs, procNum, funcChan[procNum], &wg)
	}

	// make list of science functions to run at each timestep
	scienceFuncs := []func(c *AIMcell, d *AIMdata){
		func(c *AIMcell, d *AIMdata) { c.addEmissionsFlux(d) },
		func(c *AIMcell, d *AIMdata) { c.UpwindAdvection(d.Dt) },
		func(c *AIMcell, d *AIMdata) {
			c.Mixing(d.Dt)
			c.Chemistry(d)
			c.DryDeposition(d)
			c.WetDeposition(d.Dt)
		}}

	for { // Run main calculation loop until pollutant concentrations stabilize

		// Send all of the science functions to the concurrent
		// processors for calculating
		wg.Add(len(scienceFuncs) * nprocs)
		for _, function := range scienceFuncs {
			for pp := 0; pp < nprocs; pp++ {
				funcChan[pp] <- function
			}
		}

		// do some things while waiting for the science to finish
		iteration++
		nDaysRun += d.Dt * daysPerSecond
		fmt.Printf("马上。。。Iteration %-4d  walltime=%6.3gh  Δwalltime=%4.2gs  "+
			"timestep=%2.0fs  day=%.3g\n",
			iteration, time.Since(startTime).Hours(),
			time.Since(timeStepTime).Seconds(), d.Dt, nDaysRun)
		timeStepTime = time.Now()
		timeSinceLastCheck += d.Dt

		// Occasionally, check to see if the pollutant concentrations have converged
		if timeSinceLastCheck >= checkPeriod {
			wg.Wait() // Wait for the science to finish, only when we need to check
			// for convergence.
			timeToQuit := true
			timeSinceLastCheck = 0.
			for ii, pol := range polNames {
				var sum float64
				for _, c := range d.Data {
					sum += c.Cf[ii]
				}
				if !checkConvergence(sum, oldSum[ii], pol) {
					timeToQuit = false
				}
				oldSum[ii] = sum
			}
			if timeToQuit {
				break // leave calculation loop because we're finished
			}
		}
	}
	// Prepare output data
	outputConc = make(map[string][][]float64)
	for _, pol := range OutputNames {
		outputConc[pol] = make([][]float64, d.nLayers)
		for k := 0; k < d.nLayers; k++ {
			if pol == "TotalPM2_5" {
				outputConc[pol][k] = make([]float64, len(d.Data))
				for _, subspecies := range []string{"PrimaryPM2_5", "SOA",
					"pNH4", "pSO4", "pNO3"} {
					for i, val := range outputConc[subspecies][k] {
						outputConc[pol][k][i] += val
					}
				}
			} else {
				outputConc[pol][k] = d.toArray(pol, k)
			}
		}
	}
	return
}

// Carry out the atmospheric chemistry and physics calculations
func (d *AIMdata) doScience(nprocs, procNum int,
	funcChan chan func(*AIMcell, *AIMdata), wg *sync.WaitGroup) {
	var c *AIMcell
	for f := range funcChan {
		for ii := procNum; ii < len(d.Data); ii += nprocs {
			c = d.Data[ii]
			c.lock.Lock() // Lock the cell to avoid race conditions
			if c.Layer <= topLayerToCalc {
				f(c, d) // run function
			}
			c.lock.Unlock() // Unlock the cell: we're done editing it
		}
		wg.Done()
	}
}

// Calculate emissions flux given emissions array in units of μg/s
// and a scale for molecular mass conversion.
func (d *AIMdata) addEmisFlux(arr []float64, scale float64, iPol int) {
	for row, val := range arr {
		fluxScale := 1. / d.Data[row].Dx / d.Data[row].Dy /
			d.Data[row].Dz // μg/s /m/m/m = μg/m3/s
		d.Data[row].emisFlux[iPol] = val * scale * fluxScale
	}
	return
}

func checkConvergence(newSum, oldSum float64, Var string) bool {
	bias := (newSum - oldSum) / oldSum
	fmt.Printf("%v: total mass difference = %3.2g%% from last check.\n",
		Var, bias*100)
	if math.Abs(bias) > tolerance || math.IsInf(bias, 0) {
		return false
	} else {
		return true
	}
}
