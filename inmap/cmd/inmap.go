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

package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/BurntSushi/toml"
	"github.com/ctessum/geom/proj"
	"github.com/spatialmodel/inmap"
)

type configData struct {

	// VarGridConfig provides information for specifying the variable resolution
	// grid.
	VarGrid inmap.VarGridConfig

	// InMAPData is the path to location of baseline meteorology and pollutant data.
	// The path can include environment variables.
	InMAPData string

	// EmissionsShapefiles are the paths to any emissions shapefiles.
	// Can be elevated or ground level; elevated files need to have columns
	// labeled "height", "diam", "temp", and "velocity" containing stack
	// information in units of m, m, K, and m/s, respectively.
	// Emissions will be allocated from the geometries in the shape file
	// to the InMAP computational grid, but the mapping projection of the
	// shapefile must be the same as the projection InMAP uses.
	// Can include environment variables.
	EmissionsShapefiles []string

	// EmissionUnits gives the units that the input emissions are in.
	// Acceptable values are 'tons/year' and 'kg/year'.
	EmissionUnits string

	// Path to desired output file location, where [layer] is a stand-in
	// for the model layer number. Can include environment variables.
	OutputTemplate string

	// If OutputAllLayers is true, output data for all model layers. If false, only output
	// the lowest layer.
	OutputAllLayers bool

	// OutputVariables specifies which model variables should be included in the
	// output file.
	OutputVariables []string

	// NumIterations is the number of iterations to calculate. If < 1, convergence
	// is automatically calculated.
	NumIterations int

	// Port for hosting web page. If HTTPport is `8080`, then the GUI
	// would be viewed by visiting `localhost:8080` in a web browser.
	// If HTTPport is "", then the web server doesn't run.
	HTTPport string

	sr *proj.SR
}

// Run runs the model.
func Run(dynamic bool) error {

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

	log.Println("Reading input data...")

	f, err := os.Open(config.InMAPData)
	if err != nil {
		return fmt.Errorf("Problem loading input data: %v\n", err)
	}
	ctmData, err := config.VarGrid.LoadCTMData(f)
	if err != nil {
		return fmt.Errorf("Problem loading input data: %v\n", err)
	}

	emis, err := inmap.ReadEmissionShapefiles(config.sr, config.EmissionUnits,
		msgLog, config.EmissionsShapefiles...)

	log.Println("Loading population and mortality rate data")

	pop, popIndices, mr, err := config.VarGrid.LoadPopMort()
	if err != nil {
		return err
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
		initFuncs = []inmap.DomainManipulator{
			config.VarGrid.RegularGrid(ctmData, pop, popIndices, mr, emis),
			config.VarGrid.MutateGrid(inmap.PopulationMutator(&config.VarGrid, popIndices),
				ctmData, pop, mr, emis),
			inmap.SetTimestepCFL(),
		}
		runFuncs = []inmap.DomainManipulator{
			inmap.Log(cLog),
			inmap.Calculations(inmap.AddEmissionsFlux()),
			scienceFuncs,
			inmap.SteadyStateConvergenceCheck(config.NumIterations, cConverge),
		}
	} else {
		initFuncs = []inmap.DomainManipulator{
			config.VarGrid.RegularGrid(ctmData, pop, popIndices, mr, emis),
			inmap.SetTimestepCFL(),
		}
		const gridMutateInterval = 3600. // seconds
		runFuncs = []inmap.DomainManipulator{
			inmap.Log(cLog),
			inmap.Calculations(inmap.AddEmissionsFlux()),
			scienceFuncs,
			inmap.RunPeriodically(gridMutateInterval,
				config.VarGrid.MutateGrid(inmap.PopConcMutator(
					config.VarGrid.PopConcThreshold, &config.VarGrid, popIndices),
					ctmData, pop, mr, emis)),
			inmap.RunPeriodically(gridMutateInterval, inmap.SetTimestepCFL()),
			inmap.SteadyStateConvergenceCheck(config.NumIterations, cConverge),
		}
	}

	d := &inmap.InMAP{
		InitFuncs: initFuncs,
		RunFuncs:  runFuncs,
		CleanupFuncs: []inmap.DomainManipulator{
			inmap.Output(config.OutputTemplate, config.OutputAllLayers, config.OutputVariables...),
		},
	}
	if err = d.Init(); err != nil {
		return fmt.Errorf("InMAP: problem initializing model: %v\n", err)
	}

	emisTotals := make([]float64, len(d.Cells[0].Cf))
	for _, c := range d.Cells {
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

// readConfigFile reads and parses a TOML configuration file.
// See below for the required variables.
func readConfigFile(filename string) (config *configData, err error) {
	// Open the configuration file
	var (
		file  *os.File
		bytes []byte
	)
	file, err = os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("the configuration file you have specified, %v, does not "+
			"appear to exist. Please check the file name and location and "+
			"try again.\n", filename)
	}
	reader := bufio.NewReader(file)
	bytes, err = ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("problem reading configuration file: %v", err)
	}

	config = new(configData)
	_, err = toml.Decode(string(bytes), config)
	if err != nil {
		return nil, fmt.Errorf(
			"there has been an error parsing the configuration file: %v\n", err)
	}

	config.InMAPData = os.ExpandEnv(config.InMAPData)
	config.OutputTemplate = os.ExpandEnv(config.OutputTemplate)

	for i := 0; i < len(config.EmissionsShapefiles); i++ {
		config.EmissionsShapefiles[i] =
			os.ExpandEnv(config.EmissionsShapefiles[i])
	}

	if config.OutputTemplate == "" {
		return nil, fmt.Errorf("you need to specify an output template in the " +
			"configuration file(for example: " +
			"\"OutputTemplate\":\"output_[layer].shp\"")
	}

	if config.VarGrid.GridProj == "" {
		return nil, fmt.Errorf("you need to specify the InMAP grid projection in the " +
			"'GridProj' configuration variable.")
	}
	config.sr, err = proj.Parse(config.VarGrid.GridProj)
	if err != nil {
		return nil, fmt.Errorf("the following error occured while parsing the InMAP grid"+
			"projection (the InMAPProj variable): %v", err)
	}

	if len(config.OutputVariables) == 0 {
		return nil, fmt.Errorf("there are no variables specified for output. Please fill in " +
			"the OutputVariables section of the configuration file and try again.")
	}

	if config.EmissionUnits != "tons/year" && config.EmissionUnits != "kg/year" {
		return nil, fmt.Errorf("the EmissionUnits variable in the configuration file "+
			"needs to be set to either tons/year or kg/year, but is currently set to `%s`",
			config.EmissionUnits)
	}

	outdir := filepath.Dir(config.OutputTemplate)
	err = os.MkdirAll(outdir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("problem creating output directory: %v", err)
	}
	return
}
