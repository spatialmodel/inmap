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
	"io"
	"math"
	"runtime"
	"sync"
	"time"
)

// Molar masses [grams per mole]
const (
	mwNOx = 46.0055
	mwN   = 14.0067
	mwNO3 = 62.00501
	mwNH3 = 17.03056
	mwNH4 = 18.03851
	mwS   = 32.0655
	mwSO2 = 64.0644
	mwSO4 = 96.0632
)

// Chemical mass conversions [ratios]
const (
	NOxToN = mwN / mwNOx
	NtoNO3 = mwNO3 / mwN
	SOxToS = mwSO2 / mwS
	StoSO4 = mwS / mwSO4
	NH3ToN = mwN / mwNH3
	NtoNH4 = mwNH4 / mwN
)

const daysPerSecond = 1. / 3600. / 24.

// EmisNames are the names of pollutants accepted as emissions [μg/s]
var EmisNames = []string{"VOC", "NOx", "NH3", "SOx", "PM2_5"}

var emisLabels = map[string]int{"VOC Emissions": igOrg,
	"NOx emissions":   igNO,
	"NH3 emissions":   igNH,
	"SOx emissions":   igS,
	"PM2.5 emissions": iPM2_5,
}

// These are the names of pollutants within the model
var polNames = []string{"gOrg", "pOrg", // gaseous and particulate organic matter
	"PM2_5",      // PM2.5
	"gNH", "pNH", // gaseous and particulate N in ammonia
	"gS", "pS", // gaseous and particulate S in sulfur
	"gNO", "pNO", // gaseous and particulate N in nitrate
}

// Indicies of individual pollutants in arrays.
const (
	igOrg, ipOrg, iPM2_5, igNH, ipNH, igS, ipS, igNO, ipNO = 0, 1, 2, 3, 4, 5, 6, 7, 8
)

// map relating emissions to the associated PM2.5 concentrations
var gasParticleMap = map[int]int{igOrg: ipOrg,
	igNO: ipNO, igNH: ipNH, igS: ipS, iPM2_5: iPM2_5}

type polConv struct {
	index      []int     // index in concentration array
	conversion []float64 // conversion from N to NH4, S to SO4, etc...
}

// Labels and conversions for pollutants.
var polLabels = map[string]polConv{
	"TotalPM2_5": polConv{[]int{iPM2_5, ipOrg, ipNH, ipS, ipNO},
		[]float64{1, 1, NtoNH4, StoSO4, NtoNO3}},
	"VOC":          polConv{[]int{igOrg}, []float64{1.}},
	"SOA":          polConv{[]int{ipOrg}, []float64{1.}},
	"PrimaryPM2_5": polConv{[]int{iPM2_5}, []float64{1.}},
	"NH3":          polConv{[]int{igNH}, []float64{1. / NH3ToN}},
	"pNH4":         polConv{[]int{ipNH}, []float64{NtoNH4}},
	"SOx":          polConv{[]int{igS}, []float64{1. / SOxToS}},
	"pSO4":         polConv{[]int{ipS}, []float64{StoSO4}},
	"NOx":          polConv{[]int{igNO}, []float64{1. / NOxToN}},
	"pNO3":         polConv{[]int{ipNO}, []float64{NtoNO3}},
}

// ResetCells clears concentration and emissions information from all of the
// grid cells and boundary cells.
func ResetCells() DomainManipulator {
	return func(d *InMAPdata) error {
		for _, g := range [][]*Cell{d.Cells, d.westBoundary, d.eastBoundary,
			d.northBoundary, d.southBoundary, d.topBoundary} {
			for _, c := range g {
				c.Ci = make([]float64, len(polNames))
				c.Cf = make([]float64, len(polNames))
				c.emisFlux = make([]float64, len(polNames))
			}
		}
		return nil
	}
}

// Calculations returns a function that concurrently runs a series of calculations
// on all of the model grid cells.
func Calculations(calculators ...CellManipulator) DomainManipulator {

	nprocs := runtime.GOMAXPROCS(0) // number of processors
	var wg sync.WaitGroup

	return func(d *InMAPdata) error {
		// Concurrently run all of the calculators on all of the cells.
		wg.Add(nprocs)
		for pp := 0; pp < nprocs; pp++ {
			go func(pp int) {
				var c *Cell
				for ii := pp; ii < len(d.Cells); ii += nprocs {
					c = d.Cells[ii]
					c.Lock() // Lock the cell to avoid race conditions
					// run functions
					for _, f := range calculators {
						f(c, d.Dt)
					}
					c.Unlock() // Unlock the cell: we're done editing it
				}
				wg.Done()
			}(pp)
		}
		wg.Wait()
		return nil
	}
}

// SteadyStateConvergenceCheck checks whether a steady-state
// simulation is finished and sets the Done
// flag if it is. If numIterations > 0, the simulation is finished after
// that number of iterations have completed. Otherwise, the simulation has
// finished if the change in mass in the domain since the last check is less
// than 5%.
func SteadyStateConvergenceCheck(numIterations int) DomainManipulator {

	const tolerance = 0.005   // tolerance for convergence
	const checkPeriod = 3600. // seconds, how often to check for convergence

	// oldSum is the sum of mass in the domain at the last check
	oldSum := make([]float64, len(polNames))

	timeSinceLastCheck := 0.
	iteration := 0

	return func(d *InMAPdata) error {
		timeSinceLastCheck += d.Dt
		iteration++

		// If NumIterations has been set, used it to determine when to
		// stop the model.
		if numIterations > 0 {
			if iteration >= numIterations {
				d.Done = true
			}
			// Otherwise, occasionally check to see if the pollutant
			// concentrations have converged
		} else if timeSinceLastCheck >= checkPeriod {
			timeToQuit := true
			timeSinceLastCheck = 0.
			for ii, pol := range polNames {
				var sum float64
				for _, c := range d.Cells {
					sum += c.Cf[ii]
				}
				if !checkConvergence(sum, oldSum[ii], tolerance, pol) {
					timeToQuit = false
				}
				oldSum[ii] = sum
			}
			if timeToQuit {
				d.Done = true
			}
		}
		return nil
	}
}

func checkConvergence(newSum, oldSum, tolerance float64, Var string) bool {
	bias := (newSum - oldSum) / oldSum
	fmt.Printf("%v: total mass difference = %3.2g%% from last check.\n",
		Var, bias*100)
	if math.Abs(bias) > tolerance || math.IsInf(bias, 0) {
		return false
	}
	return true
}

// Log writes simulation status messages to w.
func Log(w io.Writer) DomainManipulator {
	startTime := time.Now()
	timeStepTime := time.Now()

	iteration := 0
	nDaysRun := 0.

	return func(d *InMAPdata) error {
		iteration++
		nDaysRun += d.Dt * daysPerSecond
		fmt.Fprintf(w, "Iteration %-4d  walltime=%6.3gh  Δwalltime=%4.2gs  "+
			"timestep=%2.0fs  day=%.3g\n",
			iteration, time.Since(startTime).Hours(),
			time.Since(timeStepTime).Seconds(), d.Dt, nDaysRun)
		timeStepTime = time.Now()
		return nil
	}
}

// Results returns the simulation results.
// Output is in the form of map[pollutant][layer][row]concentration,
// in units of μg/m3.
// If  allLayers` is true, the function returns data for all of the vertical
// layers, otherwise only the ground-level layer is returned.
// outputVariables is a list of the names of the variables for which data should be
// returned.
func (d *InMAPdata) Results(allLayers bool, outputVariables ...string) map[string][][]float64 {

	// Prepare output data
	outputConc := make(map[string][][]float64)
	/*var outputVariables []string
	for pol := range polLabels {
		outputVariables = append(outputVariables, pol)
	}
	for pop := range popNames {
		outputVariables = append(outputVariables, pop, pop+" deaths")
	}
	outputVariables = append(outputVariables, "MortalityRate")*/
	var outputLay int
	if allLayers {
		outputLay = d.nlayers
	} else {
		outputLay = 1
	}
	for _, name := range outputVariables {
		outputConc[name] = make([][]float64, d.nlayers)
		for k := 0; k < outputLay; k++ {
			outputConc[name][k] = d.toArray(name, k)
		}
	}
	return outputConc
}
