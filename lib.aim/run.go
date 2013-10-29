package aim

import (
	"bitbucket.org/ctessum/sparse"
	"fmt"
	"math"
	"runtime"
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
const nDaysCheckConvergence = 0.005
const tolerance = 0.01
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

	// sums for calculating convergence
	oldFinalMassSum := 0.
	finalMassSum := 0.

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

		iiChan := make(chan int)
		sumChan := make(chan float64)
		for i := 0; i < runtime.GOMAXPROCS(0); i++ {
			go func() {
				var c *AIMcell
				sum := 0.
				for ii := range iiChan {
					c = d.Data[ii]
					//zdiff = m.DiffusiveFlux(c, d)
					c.AdvectiveFluxUpwind(d.Dt)
					c.GravitationalSettling(d)
					c.VOCoxidationFlux(d)
					c.WetDeposition(d.Dt)
					c.ChemicalPartitioning()

					for _, val := range c.finalConc {
						sum += val
					}
				}
				sumChan <- sum * c.Volume
			}()
		}
		for ii := 0; ii < len(d.Data); ii += 1 {
			iiChan <- ii
		}
		close(iiChan)
		for i := 0; i < runtime.GOMAXPROCS(0); i++ {
			finalMassSum += <-sumChan
		}
		if nDaysSinceConvergenceCheck > nDaysCheckConvergence {
			timeToQuit := true
			finalMassSum /= float64(nIterationsSinceConvergenceCheck)
			if !checkConvergence(finalMassSum, oldFinalMassSum) {
				timeToQuit = false
			}
			oldFinalMassSum = finalMassSum
			nDaysSinceConvergenceCheck = 0.
			nIterationsSinceConvergenceCheck = 0
			finalMassSum = 0.
			if timeToQuit {
				break
			}
		}
	}
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
			o.Elements[i] = c.finalConc[igOrg]
		}
	case "SOA":
		for i, c := range d.Data {
			o.Elements[i] = c.finalConc[ipOrg]
		}
	case "PrimaryPM2_5":
		for i, c := range d.Data {
			o.Elements[i] = c.finalConc[iPM2_5]
		}
	case "NH3":
		for i, c := range d.Data {
			o.Elements[i] = c.finalConc[igNH] / NH3ToN
		}
	case "pNH4":
		for i, c := range d.Data {
			o.Elements[i] = c.finalConc[ipNH] * NtoNH4
		}
	case "SOx":
		for i, c := range d.Data {
			o.Elements[i] = c.finalConc[igS] / SOxToS
		}
	case "pSO4":
		for i, c := range d.Data {
			o.Elements[i] = c.finalConc[ipS] * StoSO4
		}
	case "NOx":
		for i, c := range d.Data {
			o.Elements[i] = c.finalConc[igNO] / NOxToN
		}
	case "pNO3":
		for i, c := range d.Data {
			o.Elements[i] = c.finalConc[ipNO] * NtoNO3
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
	case "U":
		for i, c := range d.Data {
			o.Elements[i] = c.Uwest
		}
	case "V":
		for i, c := range d.Data {
			o.Elements[i] = c.Vsouth
		}
	case "W":
		for i, c := range d.Data {
			o.Elements[i] = c.Wbelow
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
			o.Elements[i] = c.wdParticle
		}
	case "SO2wetdeposition":
		for i, c := range d.Data {
			o.Elements[i] = c.wdSO2
		}
	case "Non-SO2gaswetdeposition":
		for i, c := range d.Data {
			o.Elements[i] = c.wdOtherGas
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

func checkConvergence(newSum, oldSum float64) bool {
	bias := (newSum - oldSum) / oldSum
	fmt.Printf("Total mass difference = %3.2g%% from last check.\n", bias*100)
	if math.Abs(bias) > tolerance || math.IsInf(bias, 0) {
		return false
	} else {
		return true
	}
}
