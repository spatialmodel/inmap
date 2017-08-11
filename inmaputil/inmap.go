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
	"time"

	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/science/chem/simplechem"
)

func getCTMData(inmapData string, VarGrid *inmap.VarGridConfig) (*inmap.CTMData, error) {
	log.Println("Reading input data...")

	f, err := os.Open(inmapData)
	if err != nil {
		return nil, fmt.Errorf("Problem loading input data: %v\n", err)
	}
	ctmData, err := VarGrid.LoadCTMData(f)
	if err != nil {
		return nil, fmt.Errorf("Problem loading input data: %v\n", err)
	}
	return ctmData, nil
}

var m simplechem.Mechanism

func scienceMust(c inmap.CellManipulator, err error) inmap.CellManipulator {
	if err != nil {
		panic(err)
	}
	return c
}

// DefaultScienceFuncs are the science functions that are run in
// typical simulations.
var DefaultScienceFuncs = []inmap.CellManipulator{
	inmap.UpwindAdvection(),
	inmap.Mixing(),
	inmap.MeanderMixing(),
	scienceMust(m.DryDep("simple")),
	scienceMust(m.WetDep("emep")),
	m.Chemistry(),
}

// Run runs the model. dynamic and createGrid specify whether the variable
// resolution grid should be created dynamically and whether the static
// grid should be created or read from a file, respectively.
//
// LogFile is the path to the desired logfile location. It can include
// environment variables. If LogFile is left blank, the logfile will be saved in
// the same location as the OutputFile.
//
// OutputFile is the path to the desired output shapefile location. It can
// include environment variables.
//
// If OutputAllLayers is true, output data for all model layers. If false, only output
// the lowest layer.
//
// OutputVariables specifies which model variables should be included in the
// output file.
//
// EmissionUnits gives the units that the input emissions are in.
// Acceptable values are 'tons/year', 'kg/year', 'ug/s', and 'μg/s'.
//
// EmissionsShapefiles are the paths to any emissions shapefiles.
// Can be elevated or ground level; elevated files need to have columns
// labeled "height", "diam", "temp", and "velocity" containing stack
// information in units of m, m, K, and m/s, respectively.
// Emissions will be allocated from the geometries in the shape file
// to the InMAP computational grid, but the mapping projection of the
// shapefile must be the same as the projection InMAP uses.
//
// VarGrid provides information for specifying the variable resolution grid.
//
// InMAPData is the path to location of baseline meteorology and pollutant data.
//
// VariableGridData is the path to the location of the variable-resolution gridded
// InMAP data, or the location where it should be created if it doesn't already
// exist.
//
// NumIterations is the number of iterations to calculate. If < 1, convergence
// is automatically calculated.
//
// If dynamic is
// true, createGrid is ignored. scienceFuncs specifies the science functions
// to perform in each cell at each time step. addInit, addRun, and addCleanup
// specifies functions beyond the default functions to run at initialization,
// runtime, and cleanup, respectively.
func Run(LogFile string, OutputFile string, OutputAllLayers bool, OutputVariables map[string]string,
	EmissionUnits string, EmissionsShapefiles []string, VarGrid *inmap.VarGridConfig, InMAPData, VariableGridData string,
	NumIterations int,
	dynamic, createGrid bool, scienceFuncs []inmap.CellManipulator, addInit, addRun, addCleanup []inmap.DomainManipulator, m inmap.Mechanism) error {

	startTime := time.Now()

	// Start a function to receive and print log messages.
	logfile, err := os.Create(LogFile)
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

	o, err := inmap.NewOutputter(OutputFile, OutputAllLayers, OutputVariables, nil, m)
	if err != nil {
		return err
	}
	log.Println("Parsing output variable expressions...")

	sr, err := spatialRef(VarGrid)
	if err != nil {
		return err
	}
	emis, err := inmap.ReadEmissionShapefiles(sr, EmissionUnits, msgLog, EmissionsShapefiles...)
	if err != nil {
		return err
	}

	// Only load the population if we're creating the grid.
	var pop *inmap.Population
	var mr *inmap.MortalityRates
	var popIndices inmap.PopIndices
	var mortIndices inmap.MortIndices
	var ctmData *inmap.CTMData
	if dynamic || createGrid {
		log.Println("Loading CTM data...")
		ctmData, err = getCTMData(InMAPData, VarGrid)
		if err != nil {
			return err
		}
		log.Println("Loading population and mortality rate data...")
		pop, popIndices, mr, mortIndices, err = VarGrid.LoadPopMort()
		if err != nil {
			return err
		}
	}

	scienceCalcs := inmap.Calculations(scienceFuncs...)

	var initFuncs, runFuncs []inmap.DomainManipulator
	if !dynamic {
		if createGrid {
			var mutator inmap.GridMutator
			mutator, err = inmap.PopulationMutator(VarGrid, popIndices)
			if err != nil {
				return err
			}
			initFuncs = []inmap.DomainManipulator{
				VarGrid.RegularGrid(ctmData, pop, popIndices, mr, mortIndices, emis, m),
				VarGrid.MutateGrid(mutator, ctmData, pop, mr, emis, m, msgLog),
				inmap.SetTimestepCFL(),
			}
		} else { // pre-created static grid
			var r io.Reader
			r, err = os.Open(VariableGridData)
			if err != nil {
				return fmt.Errorf("problem opening file to load VariableGridData: %v", err)
			}
			initFuncs = []inmap.DomainManipulator{
				inmap.Load(r, VarGrid, emis, m),
				inmap.SetTimestepCFL(),
				o.CheckOutputVars(m),
			}
		}
		runFuncs = []inmap.DomainManipulator{
			inmap.Log(cLog),
			inmap.Calculations(inmap.AddEmissionsFlux()),
			scienceCalcs,
			inmap.SteadyStateConvergenceCheck(NumIterations,
				VarGrid.PopGridColumn, m, cConverge),
		}
	} else { // dynamic grid
		initFuncs = []inmap.DomainManipulator{
			VarGrid.RegularGrid(ctmData, pop, popIndices, mr, mortIndices, emis, m),
			inmap.SetTimestepCFL(),
			o.CheckOutputVars(m),
		}
		popConcMutator := inmap.NewPopConcMutator(VarGrid, popIndices)
		const gridMutateInterval = 3 * 60 * 60 // every 3 hours in seconds
		runFuncs = []inmap.DomainManipulator{
			inmap.Log(cLog),
			inmap.Calculations(inmap.AddEmissionsFlux()),
			scienceCalcs,
			inmap.RunPeriodically(gridMutateInterval,
				VarGrid.MutateGrid(popConcMutator.Mutate(), ctmData, pop, mr, emis, m, msgLog)),
			inmap.RunPeriodically(gridMutateInterval, inmap.SetTimestepCFL()),
			inmap.SteadyStateConvergenceCheck(NumIterations, VarGrid.PopGridColumn, m, cConverge),
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

	elapsedTime := time.Since(startTime)
	log.Printf("Elapsed time: %f hours", elapsedTime.Hours())

	return nil
}
