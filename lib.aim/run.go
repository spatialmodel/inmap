package aim

import (
	"bitbucket.org/ctessum/sparse"
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
		func(c *AIMcell, d *AIMdata) { c.RK3advectionPass1(d) },
		func(c *AIMcell, d *AIMdata) { c.RK3advectionPass2(d) },
		func(c *AIMcell, d *AIMdata) { c.RK3advectionPass3(d) },
		func(c *AIMcell, d *AIMdata) {
			c.Mixing(d.Dt)
			c.VOCoxidationFlux(d)
			c.COBRAchemistry(d)
			c.DryDeposition(d)
			c.WetDeposition(d.Dt)
		}}

	d.setTstepCFL(nprocs) // Set time step
	//d.setTstepRuleOfThumb() // Set time step

	for { // Run main calculation loop until pollutant concentrations stabilize

		// Send all of the science functions to the concurrent
		// processors for calculating
		d.arrayLock.Lock() // Lock the cell array to avoid race conditions
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

		d.arrayLock.Unlock() // Unlock the cell array: we're done editing it

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
	outputConc = make(map[string]*sparse.DenseArray)
	for _, pol := range OutputNames {
		if pol == "TotalPM2_5" {
			outputConc[pol] = sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
			for _, subspecies := range []string{"PrimaryPM2_5", "SOA",
				"pNH4", "pSO4", "pNO3"} {
				outputConc[pol].AddDense(outputConc[subspecies])
			}
		} else {
			outputConc[pol] = d.ToArray(pol)
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
			if c.k <= topLayerToCalc {
				f(c, d) // run function
			}
		}
		wg.Done()
	}
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

// Convert the concentration data into a regular array
func (d *AIMdata) ToArray(pol string) *sparse.DenseArray {
	o := sparse.ZerosDense(d.Nz, d.Ny, d.Nx)
	d.arrayLock.RLock()
	switch pol {
	case "VOC":
		for i, c := range d.Data {
			o.Elements[i] = c.Cf[igOrg]
		}
	case "SOA":
		for i, c := range d.Data {
			o.Elements[i] = c.Cf[ipOrg]
		}
	case "PrimaryPM2_5":
		for i, c := range d.Data {
			o.Elements[i] = c.Cf[iPM2_5]
		}
	case "NH3":
		for i, c := range d.Data {
			o.Elements[i] = c.Cf[igNH] / NH3ToN
		}
	case "pNH4":
		for i, c := range d.Data {
			o.Elements[i] = c.Cf[ipNH] * NtoNH4
		}
	case "SOx":
		for i, c := range d.Data {
			o.Elements[i] = c.Cf[igS] / SOxToS
		}
	case "pSO4":
		for i, c := range d.Data {
			o.Elements[i] = c.Cf[ipS] * StoSO4
		}
	case "NOx":
		for i, c := range d.Data {
			o.Elements[i] = c.Cf[igNO] / NOxToN
		}
	case "pNO3":
		for i, c := range d.Data {
			o.Elements[i] = c.Cf[ipNO] * NtoNO3
		}
	case "VOCemissions":
		for i, c := range d.Data {
			o.Elements[i] = c.emisFlux[igOrg]
		}
	case "NOxemissions":
		for i, c := range d.Data {
			o.Elements[i] = c.emisFlux[igNO]
		}
	case "NH3emissions":
		for i, c := range d.Data {
			o.Elements[i] = c.emisFlux[igNH]
		}
	case "SOxemissions":
		for i, c := range d.Data {
			o.Elements[i] = c.emisFlux[igS]
		}
	case "PM2_5emissions":
		for i, c := range d.Data {
			o.Elements[i] = c.emisFlux[iPM2_5]
		}
	case "uPlusSpeed":
		for i, c := range d.Data {
			o.Elements[i] = c.uPlusSpeed
		}
	case "uMinusSpeed":
		for i, c := range d.Data {
			o.Elements[i] = c.uMinusSpeed
		}
	case "vPlusSpeed":
		for i, c := range d.Data {
			o.Elements[i] = c.vPlusSpeed
		}
	case "vMinusSpeed":
		for i, c := range d.Data {
			o.Elements[i] = c.vMinusSpeed
		}
	case "wPlusSpeed":
		for i, c := range d.Data {
			o.Elements[i] = c.wPlusSpeed
		}
	case "wMinusSpeed":
		for i, c := range d.Data {
			o.Elements[i] = c.wMinusSpeed
		}
	case "Organicpartitioning":
		for i, c := range d.Data {
			o.Elements[i] = c.orgPartitioning
		}
	case "Sulfurpartitioning":
		for i, c := range d.Data {
			o.Elements[i] = c.SPartitioning
		}
	case "Nitratepartitioning":
		for i, c := range d.Data {
			o.Elements[i] = c.NOPartitioning
		}
	case "Ammoniapartitioning":
		for i, c := range d.Data {
			o.Elements[i] = c.NHPartitioning
		}
	case "Particlewetdeposition":
		for i, c := range d.Data {
			o.Elements[i] = c.particleWetDep
		}
	case "SO2wetdeposition":
		for i, c := range d.Data {
			o.Elements[i] = c.SO2WetDep
		}
	case "Non-SO2gaswetdeposition":
		for i, c := range d.Data {
			o.Elements[i] = c.otherGasWetDep
		}
	case "KxxWest":
		for i, c := range d.Data {
			o.Elements[i] = c.KxxWest
		}
	case "KyySouth":
		for i, c := range d.Data {
			o.Elements[i] = c.KyySouth
		}
	case "Kz":
		for i, c := range d.Data {
			o.Elements[i] = c.Kz
		}
	case "M2u":
		for i, c := range d.Data {
			o.Elements[i] = c.M2u
		}
	case "M2d":
		for i, c := range d.Data {
			o.Elements[i] = c.M2d
		}
	case "kPblTop":
		for i, c := range d.Data {
			o.Elements[i] = c.kPblTop
		}
	default:
		panic(fmt.Sprintf("Unknown variable %v.", pol))
	}
	d.arrayLock.RUnlock()
	return o
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
