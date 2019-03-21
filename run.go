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
	"bytes"
	"fmt"
	"math"
	"runtime"
	"sync"
	"text/tabwriter"
	"time"
)

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

// baselinePolLabels specifies labels for the baseline (i.e., background
// concentrations) pollutant species. It is different than polLabels in that
// TotalPM2_5 is its own category and there is no PrimaryPM2_5.
var baselinePolLabels = map[string]struct {
	index      []int     // index in concentration array
	conversion []float64 // conversion from N to NH4, S to SO4, etc...
}{
	"BaselineTotalPM25": {[]int{iPM2_5}, []float64{1}},
	"BaselineVOC":       {[]int{igOrg}, []float64{1.}},
	"BaselineSOA":       {[]int{ipOrg}, []float64{1.}},
	"BaselineNH3":       {[]int{igNH}, []float64{1. / NH3ToN}},
	"BaselinePNH4":      {[]int{ipNH}, []float64{NtoNH4}},
	"BaselineSOx":       {[]int{igS}, []float64{1. / SOxToS}},
	"BaselinePSO4":      {[]int{ipS}, []float64{StoSO4}},
	"BaselineNOx":       {[]int{igNO}, []float64{1. / NOxToN}},
	"BaselinePNO3":      {[]int{ipNO}, []float64{NtoNO3}},
}

// ResetCells clears concentration and emissions information from all of the
// grid cells and boundary cells.
func ResetCells() DomainManipulator {
	return func(d *InMAP) error {
		for _, g := range []*cellList{d.cells, d.westBoundary, d.eastBoundary,
			d.northBoundary, d.southBoundary, d.topBoundary} {
			for _, c := range *g {
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
				for i := pp; i < d.cells.len(); i += nprocs {
					c := (*d.cells)[i]
					c.mutex.Lock() // Lock the cell to avoid race conditions
					// run functions
					for _, f := range calculators {
						f(c.Cell, d.Dt)
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
type ConvergenceStatus struct {
	data []float64
	m    Mechanism
}

func (c ConvergenceStatus) String() string {
	b := bytes.NewBufferString("Percent change since last convergence check:")
	w := tabwriter.NewWriter(b, 0, 8, 1, '\t', 0)
	for i, n := range c.m.Species() {
		fmt.Fprintf(w, "\n%s:\t%.2g%%", n, c.data[i*2]*100)
		fmt.Fprintf(w, "\n%s pop-wtd:\t%.2g%%", n, c.data[i*2+1]*100)
	}
	w.Flush()
	return b.String()
}

// SteadyStateConvergenceCheck checks whether a steady-state
// simulation is finished and sets the Done
// flag if it is. If numIterations > 0, the simulation is finished after
// that number of iterations have completed. Otherwise, the simulation has
// finished if the change in mass and population-weighted concentration
// of each pollutant in the domain since the
// last check are both less than 0.1%. Checks occur every 3 hours of
// simulation time.
// popGridColumn is the name of the population type used to determine grid
// cell sizes as in VarGridConfig.PopGridColumn.
// c is a channel over which the percent change between checks is
// sent. If c is nil, no status updates will be sent.
func SteadyStateConvergenceCheck(numIterations int, popGridColumn string, m Mechanism, c chan ConvergenceStatus) DomainManipulator {
	const tolerance = 0.001         // tolerance for convergence
	const checkPeriod = 60 * 60 * 3 // seconds, how often to check for convergence

	// oldSum is the sum of mass or population-weighted concentration
	// in the domain at the last check.
	oldSum := make([]float64, m.Len()*2)

	timeSinceLastCheck := 0.
	iteration := 0

	return func(d *InMAP) error {
		popIndex := d.PopIndices[popGridColumn]

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

			status := ConvergenceStatus{
				data: make([]float64, m.Len()*2),
				m:    m,
			}
			for ii := 0; ii < m.Len(); ii++ {
				var sum, bias float64
				var converged bool
				// calculate total mass.
				for _, c := range *d.cells {
					sum += c.Cf[ii] * c.Volume
				}
				if bias, converged = checkConvergence(sum, oldSum[ii*2], tolerance); !converged {
					timeToQuit = false
				}
				status.data[ii*2] = bias
				oldSum[ii*2] = sum
				sum = 0
				// Calculate population-weighted concentration.
				for _, c := range *d.cells {
					sum += c.Cf[ii] * c.PopData[popIndex]
				}
				if bias, converged = checkConvergence(sum, oldSum[ii*2+1], tolerance); !converged {
					timeToQuit = false
				}
				status.data[ii*2+1] = bias
				oldSum[ii*2+1] = sum
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
		s.StepWalltime.Seconds(), s.Dt, s.SimulationDays)
}

// Log sends simulation status messages to c.
func Log(c chan *SimulationStatus) DomainManipulator {
	startTime := time.Now()
	timeStepTime := time.Now()

	iteration := 0
	nDaysRun := 0.

	const daysPerSecond = 1. / 3600. / 24.

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
