/*
Copyright © 2013 the InMAP authors.
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

// PolNames are the names of pollutants within the model
var PolNames = []string{"gOrg", "pOrg", // gaseous and particulate organic matter
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

// PolLabels are labels and conversions for InMAP pollutants.
var PolLabels = map[string]polConv{
	"Total PM2.5": {[]int{iPM2_5, ipOrg, ipNH, ipS, ipNO},
		[]float64{1, 1, NtoNH4, StoSO4, NtoNO3}},
	"VOC":           {[]int{igOrg}, []float64{1.}},
	"SOA":           {[]int{ipOrg}, []float64{1.}},
	"Primary PM2.5": {[]int{iPM2_5}, []float64{1.}},
	"NH3":           {[]int{igNH}, []float64{1. / NH3ToN}},
	"pNH4":          {[]int{ipNH}, []float64{NtoNH4}},
	"SOx":           {[]int{igS}, []float64{1. / SOxToS}},
	"pSO4":          {[]int{ipS}, []float64{StoSO4}},
	"NOx":           {[]int{igNO}, []float64{1. / NOxToN}},
	"pNO3":          {[]int{ipNO}, []float64{NtoNO3}},
}

// baselinePolLabels specifies labels for the baseline (i.e., background
// concentrations) pollutant species. It is different than polLabels in that
// TotalPM2_5 is its own category and there is no PrimaryPM2_5.
var baselinePolLabels = map[string]polConv{
	"Baseline Total PM2.5": {[]int{iPM2_5}, []float64{1}},
	"Baseline VOC":         {[]int{igOrg}, []float64{1.}},
	"Baseline SOA":         {[]int{ipOrg}, []float64{1.}},
	"Baseline NH3":         {[]int{igNH}, []float64{1. / NH3ToN}},
	"Baseline pNH4":        {[]int{ipNH}, []float64{NtoNH4}},
	"Baseline SOx":         {[]int{igS}, []float64{1. / SOxToS}},
	"Baseline pSO4":        {[]int{ipS}, []float64{StoSO4}},
	"Baseline NOx":         {[]int{igNO}, []float64{1. / NOxToN}},
	"Baseline pNO3":        {[]int{ipNO}, []float64{NtoNO3}},
}

// ResetCells clears concentration and emissions information from all of the
// grid cells and boundary cells.
func ResetCells() DomainManipulator {
	return func(d *InMAP) error {
		for _, g := range [][]*Cell{d.cells, d.westBoundary, d.eastBoundary,
			d.northBoundary, d.southBoundary, d.topBoundary} {
			for _, c := range g {
				c.Ci = make([]float64, len(PolNames))
				c.Cf = make([]float64, len(PolNames))
				c.EmisFlux = make([]float64, len(PolNames))
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

	return func(d *InMAP) error {
		// Concurrently run all of the calculators on all of the cells.
		wg.Add(nprocs)
		for pp := 0; pp < nprocs; pp++ {
			go func(pp int) {
				var c *Cell
				for ii := pp; ii < len(d.cells); ii += nprocs {
					c = d.cells[ii]
					c.mutex.Lock() // Lock the cell to avoid race conditions
					// run functions
					for _, f := range calculators {
						f(c, d.Dt)
					}
					c.mutex.Unlock() // Unlock the cell: we're done editing it
				}
				wg.Done()
			}(pp)
		}
		wg.Wait()
		return nil
	}
}

// RunPeriodically runs f periodically during the simulation, with the time
// in seconds between runs specified by period.
func RunPeriodically(period float64, f DomainManipulator) DomainManipulator {
	timeSinceLastRun := 0.
	return func(d *InMAP) error {
		timeSinceLastRun += d.Dt
		if timeSinceLastRun >= period {
			timeSinceLastRun = 0.
			return f(d)
		}
		if d.Dt == 0 {
			return fmt.Errorf("timestep is zero")
		}
		return nil
	}
}

// ConvergenceStatus holds the percent difference for each pollutant between
// the last convergence check and this one.
type ConvergenceStatus []float64

func (c ConvergenceStatus) String() string {
	s := "Percent change since last convergence check:"
	for i, n := range PolNames {
		s += fmt.Sprintf("\n%s: %.2g%%", n, c[i]*100)
	}
	return s
}

// SteadyStateConvergenceCheck checks whether a steady-state
// simulation is finished and sets the Done
// flag if it is. If numIterations > 0, the simulation is finished after
// that number of iterations have completed. Otherwise, the simulation has
// finished if the change in mass in the domain since the last check is less
// than 0.1%. c is a channel over which the percent change between checks is
// sent. If c is nil, no status updates will be sent.
func SteadyStateConvergenceCheck(numIterations int, c chan ConvergenceStatus) DomainManipulator {

	const tolerance = 0.001   // tolerance for convergence
	const checkPeriod = 3600. // seconds, how often to check for convergence

	// oldSum is the sum of mass in the domain at the last check
	oldSum := make([]float64, len(PolNames))

	timeSinceLastCheck := 0.
	iteration := 0

	return func(d *InMAP) error {

		if d.Dt == 0 {
			return fmt.Errorf("inmap: timestep is zero")
		}

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
			status := make(ConvergenceStatus, len(PolNames))
			for ii := range PolNames {
				var sum float64
				for _, c := range d.cells {
					sum += c.Cf[ii]
				}
				bias, converged := checkConvergence(sum, oldSum[ii], tolerance)
				if !converged {
					timeToQuit = false
				}
				status[ii] = bias
				oldSum[ii] = sum
			}
			if c != nil {
				c <- status
			}
			if timeToQuit {
				d.Done = true
			}
		}
		return nil
	}
}

func checkConvergence(newSum, oldSum, tolerance float64) (float64, bool) {
	bias := (newSum - oldSum) / oldSum
	if math.Abs(bias) > tolerance || math.IsInf(bias, 0) {
		return bias, false
	}
	return bias, true
}

// SimulationStatus holds information about the progress of a simulation.
type SimulationStatus struct {
	// SimulationDays is the number of days in simulation time since the
	// start of the simulation.
	SimulationDays float64

	// Iteration is the current iteration number.
	Iteration int

	// Walltime is the total wall time since the beginning of the simulation.
	Walltime time.Duration

	// StepWalltime is the wall time that elapsed during the most recent time step.
	StepWalltime time.Duration

	// Dt is the timestep in seconds.
	Dt float64
}

func (s SimulationStatus) String() string {
	return fmt.Sprintf("iteration %-4d  walltime=%6.3gh  Δwalltime=%4.2gs  "+
		"timestep=%2.0fs  day=%.3g", s.Iteration, s.Walltime.Hours(),
		s.StepWalltime.Hours(), s.Dt, s.SimulationDays)
}

// Log sends simulation status messages to c.
func Log(c chan *SimulationStatus) DomainManipulator {
	startTime := time.Now()
	timeStepTime := time.Now()

	iteration := 0
	nDaysRun := 0.

	return func(d *InMAP) error {
		iteration++
		nDaysRun += d.Dt * daysPerSecond

		c <- &SimulationStatus{
			Iteration:      iteration,
			Walltime:       time.Since(startTime),
			StepWalltime:   time.Since(timeStepTime),
			Dt:             d.Dt,
			SimulationDays: nDaysRun,
		}
		timeStepTime = time.Now()
		return nil
	}
}

func (d *InMAP) checkOutputNames(n ...string) error {
	tempOutputNames, _, _ := d.OutputOptions()
	outputNames := make(map[string]uint8)
	for _, n := range tempOutputNames {
		outputNames[n] = 0
	}
	for _, v := range n {
		if _, ok := outputNames[v]; !ok {
			return fmt.Errorf("inmap: unsupported output variable name '%s'", v)
		}
	}
	return nil
}

// Results returns the simulation results.
// Output is in the form of map[variable][row]concentration.
// If  allLayers` is true, the function returns data for all of the vertical
// layers, otherwise only the ground-level layer is returned.
// outputVariables is a list of the names of the variables for which data should be
// returned.
func (d *InMAP) Results(allLayers bool, outputVariables ...string) (map[string][]float64, error) {
	if err := d.checkOutputNames(outputVariables...); err != nil {
		return nil, err
	}

	// Prepare output data
	outputConc := make(map[string][]float64)
	for _, name := range outputVariables {
		if allLayers {
			outputConc[name] = d.toArray(name, -1)
		} else {
			outputConc[name] = d.toArray(name, 0)
		}
	}
	return outputConc, nil
}
