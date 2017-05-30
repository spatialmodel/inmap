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

package inmaputil

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spatialmodel/inmap"
)

func getCTMData(cfg *ConfigData) (*inmap.CTMData, error) {
	log.Println("Reading input data...")

	f, err := os.Open(cfg.InMAPData)
	if err != nil {
		return nil, fmt.Errorf("Problem loading input data: %v\n", err)
	}
	ctmData, err := cfg.VarGrid.LoadCTMData(f)
	if err != nil {
		return nil, fmt.Errorf("Problem loading input data: %v\n", err)
	}
	return ctmData, nil
}

// DefaultScienceFuncs are the science functions that are run in
// typical simulations.
var DefaultScienceFuncs = []inmap.CellManipulator{
	inmap.UpwindAdvection(),
	inmap.Mixing(),
	inmap.MeanderMixing(),
	inmap.DryDeposition(),
	inmap.WetDeposition(),
	inmap.Chemistry(),
}

// Run runs the model. dynamic and createGrid specify whether the variable
// resolution grid should be created dynamically and whether the static
// grid should be created or read from a file, respectively. If dynamic is
// true, createGrid is ignored. scienceFuncs specifies the science functions
// to perform in each cell at each time step. addInit, addRun, and addCleanup
// specifies functions beyond the default functions to run at initialization,
// runtime, and cleanup, respectively.
func Run(cfg *ConfigData, dynamic, createGrid bool, scienceFuncs []inmap.CellManipulator, addInit, addRun, addCleanup []inmap.DomainManipulator) error {

	startTime := time.Now()

	// Start a function to receive and print log messages.
	logfile, err := os.Create(cfg.LogFile)
	if err != nil {
		return fmt.Errorf("inmap: problem creating log file: %v", err)
	}
	defer logfile.Close()
	mw := io.MultiWriter(os.Stdout, logfile)
	log.SetOutput(mw)
	cConverge := make(chan inmap.ConvergenceStatus)
	cLog := make(chan *inmap.SimulationStatus)
	msgLog := make(chan string)
	go func() {
		for {
			select {
			case msg := <-cConverge:
				log.Println(msg.String())
			case msg := <-cLog:
				log.Println(msg.String())
			case msg := <-msgLog:
				log.Println(msg)
			}
		}
	}()

	o, err := inmap.NewOutputter(cfg.OutputFile, cfg.OutputAllLayers, cfg.OutputVariables, nil)
	if err != nil {
		return err
	}
	log.Println("Parsing output variable expressions...")

	emis, err := inmap.ReadEmissionShapefiles(cfg.sr, cfg.EmissionUnits,
		msgLog, cfg.EmissionsShapefiles...)
	if err != nil {
		return err
	}

	// Only load the population if we're creating the grid.
	var pop *inmap.Population
	var mr *inmap.MortalityRates
	var popIndices inmap.PopIndices
	var ctmData *inmap.CTMData
	if dynamic || createGrid {
		log.Println("Loading CTM data...")
		ctmData, err = getCTMData(cfg)
		if err != nil {
			return err
		}
		log.Println("Loading population and mortality rate data...")
		pop, popIndices, mr, err = cfg.VarGrid.LoadPopMort()
		if err != nil {
			return err
		}
	}

	scienceCalcs := inmap.Calculations(scienceFuncs...)

	var initFuncs, runFuncs []inmap.DomainManipulator
	if !dynamic {
		if createGrid {
			var mutator inmap.GridMutator
			mutator, err = inmap.PopulationMutator(&cfg.VarGrid, popIndices)
			if err != nil {
				return err
			}
			initFuncs = []inmap.DomainManipulator{
				cfg.VarGrid.RegularGrid(ctmData, pop, popIndices, mr, emis),
				cfg.VarGrid.MutateGrid(mutator, ctmData, pop, mr, emis, msgLog),
				inmap.SetTimestepCFL(),
			}
		} else { // pre-created static grid
			var r io.Reader
			r, err = os.Open(cfg.VariableGridData)
			if err != nil {
				return fmt.Errorf("problem opening file to load VariableGridData: %v", err)
			}
			initFuncs = []inmap.DomainManipulator{
				inmap.Load(r, &cfg.VarGrid, emis),
				inmap.SetTimestepCFL(),
				o.CheckOutputVars(),
			}
		}
		runFuncs = []inmap.DomainManipulator{
			inmap.Log(cLog),
			inmap.Calculations(inmap.AddEmissionsFlux()),
			scienceCalcs,
			inmap.SteadyStateConvergenceCheck(cfg.NumIterations,
				cfg.VarGrid.PopGridColumn, cConverge),
		}
	} else { // dynamic grid
		initFuncs = []inmap.DomainManipulator{
			cfg.VarGrid.RegularGrid(ctmData, pop, popIndices, mr, emis),
			inmap.SetTimestepCFL(),
			o.CheckOutputVars(),
		}
		popConcMutator := inmap.NewPopConcMutator(&cfg.VarGrid, popIndices)
		const gridMutateInterval = 3 * 60 * 60 // every 3 hours in seconds
		runFuncs = []inmap.DomainManipulator{
			inmap.Log(cLog),
			inmap.Calculations(inmap.AddEmissionsFlux()),
			scienceCalcs,
			inmap.RunPeriodically(gridMutateInterval,
				cfg.VarGrid.MutateGrid(popConcMutator.Mutate(),
					ctmData, pop, mr, emis, msgLog)),
			inmap.RunPeriodically(gridMutateInterval, inmap.SetTimestepCFL()),
			inmap.SteadyStateConvergenceCheck(cfg.NumIterations,
				cfg.VarGrid.PopGridColumn, cConverge),
		}
	}

	d := &inmap.InMAP{
		InitFuncs: append(initFuncs, addInit...),
		RunFuncs:  append(runFuncs, addRun...),
		CleanupFuncs: append([]inmap.DomainManipulator{
			o.Output(),
		}, addCleanup...),
	}

	log.Println("Initializing model...")
	if err = d.Init(); err != nil {
		return fmt.Errorf("InMAP: problem initializing model: %v\n", err)
	}

	emisTotals := make([]float64, len(d.Cells()[0].Cf))
	for _, c := range d.Cells() {
		for i, val := range c.EmisFlux {
			emisTotals[i] += val * c.Volume
		}
	}
	log.Println("Emission totals:")
	for i, pol := range inmap.PolNames {
		log.Printf("%v, %g μg/s\n", pol, emisTotals[i])
	}

	if err = d.Run(); err != nil {
		return fmt.Errorf("InMAP: problem running simulation: %v\n", err)
	}

	if err = d.Cleanup(); err != nil {
		return fmt.Errorf("InMAP: problem shutting down model: %v\n", err)
	}

	log.Println("\nIntake fraction results:")
	breathingRate := 15. // [m³/day]
	iF := d.IntakeFraction(breathingRate)
	// Write iF to stdout
	w1 := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)
	// Write iF to log
	w2 := tabwriter.NewWriter(logfile, 0, 8, 1, '\t', 0)
	var popList []string
	for _, m := range iF {
		for p := range m {
			popList = append(popList, p)
		}
		break
	}
	sort.Strings(popList)
	fmt.Fprintln(w1, strings.Join(append([]string{"pol"}, popList...), "\t"))
	fmt.Fprintln(w2, strings.Join(append([]string{"pol"}, popList...), "\t"))
	for pol, m := range iF {
		temp := make([]string, len(popList))
		for i, pop := range popList {
			temp[i] = fmt.Sprintf("%.3g", m[pop])
		}
		fmt.Fprintln(w1, strings.Join(append([]string{pol}, temp...), "\t"))
		fmt.Fprintln(w2, strings.Join(append([]string{pol}, temp...), "\t"))
	}
	w1.Flush()
	w2.Flush()

	elapsedTime := time.Since(startTime)
	log.Printf("Elapsed time: %f hours", elapsedTime.Hours())

	return nil
}
