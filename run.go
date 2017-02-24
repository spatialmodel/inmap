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
	"regexp"
	"runtime"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"bitbucket.org/ctessum/aqhealth"

	"github.com/Knetic/govaluate"
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

var emisLabels = map[string]int{"VOCEmissions": igOrg,
	"NOxEmissions":  igNO,
	"NH3Emissions":  igNH,
	"SOxEmissions":  igS,
	"PM25Emissions": iPM2_5,
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
	"TotalPM25": {[]int{iPM2_5, ipOrg, ipNH, ipS, ipNO},
		[]float64{1, 1, NtoNH4, StoSO4, NtoNO3}},
	"VOC":         {[]int{igOrg}, []float64{1.}},
	"SOA":         {[]int{ipOrg}, []float64{1.}},
	"PrimaryPM25": {[]int{iPM2_5}, []float64{1.}},
	"NH3":         {[]int{igNH}, []float64{1. / NH3ToN}},
	"pNH4":        {[]int{ipNH}, []float64{NtoNH4}},
	"SOx":         {[]int{igS}, []float64{1. / SOxToS}},
	"pSO4":        {[]int{ipS}, []float64{StoSO4}},
	"NOx":         {[]int{igNO}, []float64{1. / NOxToN}},
	"pNO3":        {[]int{ipNO}, []float64{NtoNO3}},
}

// baselinePolLabels specifies labels for the baseline (i.e., background
// concentrations) pollutant species. It is different than polLabels in that
// TotalPM2_5 is its own category and there is no PrimaryPM2_5.
var baselinePolLabels = map[string]polConv{
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
type ConvergenceStatus []float64

func (c ConvergenceStatus) String() string {
	b := bytes.NewBufferString("Percent change since last convergence check:")
	w := tabwriter.NewWriter(b, 0, 8, 1, '\t', 0)
	for i, n := range PolNames {
		fmt.Fprintf(w, "\n%s:\t%.2g%%", n, c[i*2]*100)
		fmt.Fprintf(w, "\n%s pop-wtd:\t%.2g%%", n, c[i*2+1]*100)
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
func SteadyStateConvergenceCheck(numIterations int, popGridColumn string, c chan ConvergenceStatus) DomainManipulator {
	const tolerance = 0.001         // tolerance for convergence
	const checkPeriod = 60 * 60 * 3 // seconds, how often to check for convergence

	// oldSum is the sum of mass or population-weighted concentration
	// in the domain at the last check.
	oldSum := make([]float64, len(PolNames)*2)

	timeSinceLastCheck := 0.
	iteration := 0

	return func(d *InMAP) error {
		popIndex := d.popIndices[popGridColumn]

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
			status := make(ConvergenceStatus, len(PolNames)*2)
			for ii := range PolNames {
				var sum, bias float64
				var converged bool
				// calculate total mass.
				for _, c := range *d.cells {
					sum += c.Cf[ii] * c.Volume
				}
				if bias, converged = checkConvergence(sum, oldSum[ii*2], tolerance); !converged {
					timeToQuit = false
				}
				status[ii*2] = bias
				oldSum[ii*2] = sum
				sum = 0
				// Calculate population-weighted concentration.
				for _, c := range *d.cells {
					sum += c.Cf[ii] * c.PopData[popIndex]
				}
				if bias, converged = checkConvergence(sum, oldSum[ii*2+1], tolerance); !converged {
					timeToQuit = false
				}
				status[ii*2+1] = bias
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

// removeDuplicates removes all duplicated strings from a slice, returning a
// slice that contains only unique strings.
func removeDuplicates(s []string) []string {
	result := make([]string, 0, len(s))
	seen := make(map[string]string)
	for _, val := range s {
		if _, ok := seen[val]; !ok {
			result = append(result, val)
			seen[val] = val
		}
	}
	return result
}

// checkForDerivitives identifies the unique input variables that are required
// to calculate the requested output variables.
// Inputs:
// (1) Map of requested output variable names to their corresponding expressions.
// (2) Map of all function names to function definitions that are used in expressions.
// Outputs:
// (1) Map of output variable names to revised expressions where any user-defined
// output variable showing up in a subsequent expression is replaced by its
// corresponding user-defined expression.
// (2) Slice of all unique input variables required to calculate the requested
// output variables.
func checkForDerivitives(m map[string]string, f map[string]govaluate.ExpressionFunction) (map[string]string, []string, error) {
	getVariables := make([]string, 0, len(m))
	for key, val := range m {
		expression, err := govaluate.NewEvaluableExpressionWithFunctions(val, f)
		if err != nil {
			return nil, nil, fmt.Errorf("inmap OutputVariables: %v", err)
		}
		uniqueVars := removeDuplicates(expression.Vars())
		getVariables = append(getVariables, uniqueVars...)
		for _, uniqueVar := range uniqueVars {
			if m[uniqueVar] != "" && m[uniqueVar] != uniqueVar {
				m[key] = strings.Replace(m[key], uniqueVar, "("+m[uniqueVar]+")", -1)
				return checkForDerivitives(m, f)
			}
		}
	}
	return m, removeDuplicates(getVariables), nil
}

// checkGetNames checks whether the unique input variables required to calculate
// the user-requested output variables are available in the model.
func (d *InMAP) checkGetNames(g ...string) error {
	tempOutputNames, _, _ := d.OutputOptions()
	outputNames := make(map[string]uint8)
	for _, n := range tempOutputNames {
		outputNames[n] = 0
	}
	for _, v := range g {
		if _, ok := outputNames[v]; !ok {
			return fmt.Errorf("inmap: unsupported variable name '%s'", v)
		}
	}
	return nil
}

// checkOutputNames checks (1) if any output variable names exceed 10 characters
// and (2) if any output variable names include characters that are unsupported
// in shapefile field names.
func (d *InMAP) checkOutputNames(o map[string]string) error {
	for key := range o {
		long := len(key) > 10
		noCharError, err := regexp.MatchString("^[A-Za-z]\\w*$", key)
		if err != nil {
			panic(err)
		}
		if long && !noCharError {
			return fmt.Errorf("inmap: output variable name '%s' exceeds 10 characters and includes unsupported character(s)", key)
		} else if long {
			return fmt.Errorf("inmap: output variable name '%s' exceeds 10 characters", key)
		} else if !noCharError {
			return fmt.Errorf("inmap: output variable name '%s' includes unsupported characters", key)
		}
	}
	return nil
}

// Results returns the simulation results.
// Output is in the form of map[variable][row]concentration.
// If allLayers is true, the function returns data for all of the vertical
// layers, otherwise only the ground-level layer is returned.
// If checkNames is true, the length and characters of output variable names
// will be checked for compatibility with shapefiles.
// outputVariables maps the names of the variables for which data
// should be returned to expressions that define how the
// requested data should be calculated. These expressions can utilize variables
// built into the model, user-defined variables, and functions. Available
// functions include:
//
// 'exp(x)' which applies the exponetional function e^x.
//
// 'loglogRR(PM 2.5 Concentration)' which calculates relative risk (or risk ratio)
// associated with a given change in PM 2.5 concentration, assumung a log-log
// dose response (almost a linear relationship).
//
// 'coxHazard(Relative Risk, Population, Mortality Rate)' which calculates a
// deaths estimate based on the relative risk associated with PM 2.5 changes
// and the baseline number of deaths.
func (d *InMAP) Results(allLayers bool, checkNames bool, outputVariables map[string]string) (map[string][]float64, error) {

	functions := map[string]govaluate.ExpressionFunction{
		"exp": func(arg ...interface{}) (interface{}, error) {
			if len(arg) != 1 {
				return nil, fmt.Errorf("inmap: got %d arguments for function 'exp', but needs 1", len(arg))
			}
			return (float64)(math.Exp(arg[0].(float64))), nil
		},
		"loglogRR": func(arg ...interface{}) (interface{}, error) {
			if len(arg) != 1 {
				return nil, fmt.Errorf("inmap: got %d arguments for function 'exp', but needs 1", len(arg))
			}
			return (float64)(aqhealth.RRpm25Linear(arg[0].(float64))), nil
		},
		"coxHazard": func(args ...interface{}) (interface{}, error) {
			if len(args) != 3 {
				return nil, fmt.Errorf("inmap: got %d arguments for function 'exp', but needs 3", len(args))
			}
			return (float64)((args[0].(float64) - 1) * args[1].(float64) * args[2].(float64) / 100000), nil
		},
	}

	outputVariables, getVariables, err := checkForDerivitives(outputVariables, functions)
	if err != nil {
		return nil, err
	}

	if err := d.checkGetNames(getVariables...); err != nil {
		return nil, err
	}

	if checkNames {
		if err := d.checkOutputNames(outputVariables); err != nil {
			return nil, err
		}
	}

	// Prepare output data
	getConc := make(map[string][]float64)
	concByRow := make(map[string]interface{})
	outputConc := make(map[string][]float64)
	var nCells int

	for _, name := range getVariables {
		if allLayers {
			data := d.toArray(name, -1)
			getConc[name] = data
			nCells = len(data)
		} else {
			data := d.toArray(name, 0)
			getConc[name] = data
			nCells = len(data)
		}
	}
	for k, v := range outputVariables {
		expression, err := govaluate.NewEvaluableExpressionWithFunctions(v, functions)
		if err != nil {
			return nil, err
		}
		for i := 0; i < nCells; i++ {
			for name := range getConc {
				concByRow[name] = getConc[name][i]
			}
			result, err := expression.Evaluate(concByRow)
			if err != nil {
				return nil, err
			}
			outputConc[k] = append(outputConc[k], result.(float64))
		}
	}
	return outputConc, nil
}
