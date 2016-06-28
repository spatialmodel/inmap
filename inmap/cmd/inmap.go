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

package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spatialmodel/inmap"
)

func getCTMData() (*inmap.CTMData, error) {
	log.Println("Reading input data...")

	f, err := os.Open(Config.InMAPData)
	if err != nil {
		return nil, fmt.Errorf("Problem loading input data: %v\n", err)
	}
	ctmData, err := Config.VarGrid.LoadCTMData(f)
	if err != nil {
		return nil, fmt.Errorf("Problem loading input data: %v\n", err)
	}
	return ctmData, nil
}

// Run runs the model.
func Run(dynamic, createGrid bool) error {

	// Start a function to receive and print log messages.
	cConverge := make(chan inmap.ConvergenceStatus)
	cLog := make(chan *inmap.SimulationStatus)
	msgLog := make(chan string)
	go func() {
		for {
			select {
			case msg := <-cConverge:
				fmt.Println(msg.String())
			case msg := <-cLog:
				fmt.Println(msg.String())
			case msg := <-msgLog:
				log.Println(msg)
			}
		}
	}()

	emis, err := inmap.ReadEmissionShapefiles(Config.sr, Config.EmissionUnits,
		msgLog, Config.EmissionsShapefiles...)
	if err != nil {
		return err
	}

	// Only load the population if we're creating the grid.
	var pop *inmap.Population
	var mr *inmap.MortalityRates
	var popIndices inmap.PopIndices
	var ctmData *inmap.CTMData
	if dynamic || createGrid {
		log.Println("Loading CTM data")
		ctmData, err = getCTMData()
		if err != nil {
			return err
		}
		log.Println("Loading population and mortality rate data")
		pop, popIndices, mr, err = Config.VarGrid.LoadPopMort()
		if err != nil {
			return err
		}
	}

	scienceFuncs := inmap.Calculations(
		inmap.UpwindAdvection(),
		inmap.Mixing(),
		inmap.MeanderMixing(),
		inmap.DryDeposition(),
		inmap.WetDeposition(),
		inmap.Chemistry(),
	)

	var initFuncs, runFuncs []inmap.DomainManipulator
	if !dynamic {
		if createGrid {
			initFuncs = []inmap.DomainManipulator{
				inmap.HTMLUI(Config.HTTPAddress),
				Config.VarGrid.RegularGrid(ctmData, pop, popIndices, mr, emis),
				Config.VarGrid.MutateGrid(inmap.PopulationMutator(&Config.VarGrid, popIndices),
					ctmData, pop, mr, emis),
				inmap.SetTimestepCFL(),
			}
		} else {
			var r io.Reader
			r, err = os.Open(Config.VariableGridData)
			if err != nil {
				return fmt.Errorf("problem opening file to load VariableGridData: %v", err)
			}
			initFuncs = []inmap.DomainManipulator{
				inmap.Load(r, &Config.VarGrid, emis),
				inmap.SetTimestepCFL(),
			}
		}
		runFuncs = []inmap.DomainManipulator{
			inmap.Log(cLog),
			inmap.Calculations(inmap.AddEmissionsFlux()),
			scienceFuncs,
			inmap.SteadyStateConvergenceCheck(Config.NumIterations, cConverge),
		}
	} else {
		initFuncs = []inmap.DomainManipulator{
			Config.VarGrid.RegularGrid(ctmData, pop, popIndices, mr, emis),
			inmap.SetTimestepCFL(),
		}
		const gridMutateInterval = 3600. // seconds
		runFuncs = []inmap.DomainManipulator{
			inmap.Log(cLog),
			inmap.Calculations(inmap.AddEmissionsFlux()),
			scienceFuncs,
			inmap.RunPeriodically(gridMutateInterval,
				Config.VarGrid.MutateGrid(inmap.PopConcMutator(
					Config.VarGrid.PopConcThreshold, &Config.VarGrid, popIndices),
					ctmData, pop, mr, emis)),
			inmap.RunPeriodically(gridMutateInterval, inmap.SetTimestepCFL()),
			inmap.SteadyStateConvergenceCheck(Config.NumIterations, cConverge),
		}
	}

	d := &inmap.InMAP{
		InitFuncs: initFuncs,
		RunFuncs:  runFuncs,
		CleanupFuncs: []inmap.DomainManipulator{
			inmap.Output(Config.OutputFile, Config.OutputAllLayers, Config.OutputVariables...),
		},
	}
	if err = d.Init(); err != nil {
		return fmt.Errorf("InMAP: problem initializing model: %v\n", err)
	}

	emisTotals := make([]float64, len(d.Cells()[0].Cf))
	for _, c := range d.Cells() {
		for i, val := range c.EmisFlux {
			emisTotals[i] += val
		}
	}
	log.Println("Emission totals:")
	for i, pol := range inmap.PolNames {
		fmt.Printf("%v, %g μg/s\n", pol, emisTotals[i])
	}

	if err = d.Run(); err != nil {
		return fmt.Errorf("InMAP: problem running simulation: %v\n", err)
	}

	if err = d.Cleanup(); err != nil {
		return fmt.Errorf("InMAP: problem shutting down model.: %v\n", err)
	}

	fmt.Println("\nIntake fraction results:")
	breathingRate := 15. // [m³/day]
	iF := d.IntakeFraction(breathingRate)
	// Write iF to stdout
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)
	var popList []string
	for _, m := range iF {
		for p := range m {
			popList = append(popList, p)
		}
		break
	}
	sort.Strings(popList)
	fmt.Fprintln(w, strings.Join(append([]string{"pol"}, popList...), "\t"))
	for pol, m := range iF {
		temp := make([]string, len(popList))
		for i, pop := range popList {
			temp[i] = fmt.Sprintf("%.3g", m[pop])
		}
		fmt.Fprintln(w, strings.Join(append([]string{pol}, temp...), "\t"))
	}
	w.Flush()

	return nil
}
