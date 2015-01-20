/*
Copyright (C) 2013-2014 Regents of the University of Minnesota.
This file is part of InMAP.

InMAP is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

InMAP is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with InMAP.  If not, see <http://www.gnu.org/licenses/>.
*/

package inmap

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

const tolerance = 0.005 // tolerance for convergence
//const tolerance = 0.5     // tolerance for convergence
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
var OutputVariables = []string{"VOC", "SOA", "PrimaryPM2_5", "NH3", "pNH4",
	"SOx", "pSO4", "NOx", "pNO3", "TotalPM2_5"} //,
//"Total deaths", "White deaths", "Non-white deaths",
//"High income deaths", "Low income deaths",
//"High income white deaths", "Low income non-white deaths"}

// Run air quality model. Emissions are assumed to be in units
// of μg/s, and must only include the pollutants listed in "EmisNames".
// Output is in the form of map[pollutant][layer][row]concentration,
// in units of μg/m3.
func (d *InMAPdata) Run(emissions map[string][]float64) (
	outputConc map[string][][]float64) {

	for _, c := range d.Data {
		c.Ci = make([]float64, len(polNames))
		c.Cf = make([]float64, len(polNames))
		c.emisFlux = make([]float64, len(polNames))
	}

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
	funcChan := make([]chan func(*Cell, *InMAPdata), nprocs)
	var wg sync.WaitGroup

	for procNum := 0; procNum < nprocs; procNum++ {
		funcChan[procNum] = make(chan func(*Cell, *InMAPdata), 1)
		// Start thread for concurrent computations
		go d.doScience(nprocs, procNum, funcChan[procNum], &wg)
	}

	// make list of science functions to run at each timestep
	scienceFuncs := []func(c *Cell, d *InMAPdata){
		func(c *Cell, d *InMAPdata) { c.addEmissionsFlux(d) },
		func(c *Cell, d *InMAPdata) {
			c.UpwindAdvection(d.Dt)
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
		fmt.Printf("Iteration %-4d  walltime=%6.3gh  Δwalltime=%4.2gs  "+
			"timestep=%2.0fs  day=%.3g\n",
			iteration, time.Since(startTime).Hours(),
			time.Since(timeStepTime).Seconds(), d.Dt, nDaysRun)
		timeStepTime = time.Now()
		timeSinceLastCheck += d.Dt

		// Occasionally, check to see if the pollutant concentrations have converged
		if timeSinceLastCheck >= checkPeriod {
			wg.Wait() // Wait for the science to finish, only when we need to check
			// for convergence.
			//timeToQuit := true
			timeSinceLastCheck = 0.
			for ii, pol := range polNames {
				var sum float64
				for _, c := range d.Data {
					sum += c.Cf[ii]
				}
				//if !checkConvergence(sum, oldSum[ii], pol) {
				//timeToQuit = false
				//}
				checkConvergence(sum, oldSum[ii], pol)
				oldSum[ii] = sum
			}
			if iteration > 10000 {
				//if timeToQuit {
				break // leave calculation loop because we're finished
			}
		}
	}
	// Prepare output data
	outputConc = make(map[string][][]float64)
	for _, name := range OutputVariables {
		outputConc[name] = make([][]float64, d.Nlayers)
		for k := 0; k < d.Nlayers; k++ {
			outputConc[name][k] = d.toArray(name, k)
		}
	}
	return
}

// Carry out the atmospheric chemistry and physics calculations
func (d *InMAPdata) doScience(nprocs, procNum int,
	funcChan chan func(*Cell, *InMAPdata), wg *sync.WaitGroup) {
	var c *Cell
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
func (d *InMAPdata) addEmisFlux(arr []float64, scale float64, iPol int) {
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
