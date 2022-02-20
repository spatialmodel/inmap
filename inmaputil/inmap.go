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
	"math"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/ctessum/geom"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/emissions/aep"
	"github.com/spatialmodel/inmap/emissions/aep/aeputil"
	"github.com/spatialmodel/inmap/science/chem/simplechem"
	"github.com/spf13/cobra"
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
// CobraCommand is the cobra.Command instance where Run is called from.
// It is needed to print certain outputs to the web interface.
//
// LogFile is the path to the desired logfile location. It can include
// environment variables.
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
// EmissionsMask specifies a polygon boundary to constrain emissions, assumed
// to use the same spatial reference as VarGrid. It will
// be ignored if it is nil.
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
//
// notMeters should be set to true if the units of the grid are not meters
// (e.g., if the grid is in degrees latitude/longitude.)
func Run(CobraCommand *cobra.Command, LogFile string, OutputFile string, OutputAllLayers bool, OutputVariables map[string]string,
	EmissionUnits string, EmissionsShapefiles []string, EmissionsMask geom.Polygon, VarGrid *inmap.VarGridConfig,
	inventoryConfig *aeputil.InventoryConfig, spatialConfig *aeputil.SpatialConfig,
	InMAPData, VariableGridData string, NumIterations int,
	dynamic, createGrid bool, scienceFuncs []inmap.CellManipulator, addInit, addRun, addCleanup []inmap.DomainManipulator,
	m inmap.Mechanism) error {

	startTime := time.Now()

	var upload uploader

	// Start a function to receive and print log messages.
	logfile, err := os.Create(upload.maybeUpload(LogFile))
	if err != nil {
		return fmt.Errorf("inmap: problem creating log file: %v", err)
	}
	mw := io.MultiWriter(CobraCommand.OutOrStdout(), logfile)
	log.SetOutput(mw)
	cConverge := make(chan inmap.ConvergenceStatus)
	cLog := make(chan *inmap.SimulationStatus)
	cLogTick := time.Tick(2 * time.Second)
	msgLog := make(chan string)
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		for msg := range cConverge {
			log.Println(msg.String())
		}
		wg.Done()
	}()
	go func() {
		for msg := range cLog {
			select {
			case <-cLogTick:
				log.Println(msg.String())
			default:
				runtime.Gosched()
			}
		}
		wg.Done()
	}()
	go func() {
		for msg := range msgLog {
			log.Println(msg)
		}
		wg.Done()
	}()

	defer func() { // Wait for the logging to finish.
		close(cConverge)
		close(cLog)
		close(msgLog)
		wg.Wait()
		logfile.Close()
	}()

	o, err := inmap.NewOutputter(upload.maybeUpload(OutputFile), OutputAllLayers, OutputVariables, nil, m)
	if err != nil {
		return err
	}
	log.Println("Parsing output variable expressions...")

	if upload.err != nil {
		return upload.err
	}

	sr, err := spatialRef(VarGrid)
	if err != nil {
		return err
	}
	emis, err := inmap.ReadEmissionShapefiles(sr, EmissionUnits, msgLog, EmissionsMask, EmissionsShapefiles...)
	if err != nil {
		return err
	}

	aepSetEmis := setEmissionsAEP(inventoryConfig, spatialConfig, emis, EmissionsMask)

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
				VarGrid.RegularGrid(ctmData, pop, popIndices, mr, mortIndices, nil, m),
				VarGrid.MutateGrid(mutator, ctmData, pop, mr, nil, m, msgLog),
				aepSetEmis,
				inmap.SetTimestepCFL(),
				o.CheckOutputVars(m),
			}
		} else { // pre-created static grid
			var r io.Reader
			r, err = os.Open(VariableGridData)
			if err != nil {
				return fmt.Errorf("problem opening file to load VariableGridData: %v", err)
			}
			initFuncs = []inmap.DomainManipulator{
				inmap.Load(r, VarGrid, nil, m),
				aepSetEmis,
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
			VarGrid.RegularGrid(ctmData, pop, popIndices, mr, mortIndices, nil, m),
			aepSetEmis,
			inmap.SetTimestepCFL(),
			o.CheckOutputVars(m),
		}

		// Set up a domain manipulator that mutates the grid, sets the emissions,
		// the sets the timestep.
		popConcMutator := inmap.NewPopConcMutator(VarGrid, popIndices)
		const gridMutateInterval = 3 * 60 * 60 // every 3 hours in seconds
		mg := VarGrid.MutateGrid(popConcMutator.Mutate(), ctmData, pop, mr, nil, m, msgLog)
		setTS := inmap.SetTimestepCFL()
		mutateThenAddEmis := func(d *inmap.InMAP) error {
			if err := mg(d); err != nil {
				return err
			}
			if err := aepSetEmis(d); err != nil {
				return err
			}
			return setTS(d)
		}

		runFuncs = []inmap.DomainManipulator{
			inmap.Log(cLog),
			inmap.Calculations(inmap.AddEmissionsFlux()),
			scienceCalcs,
			inmap.RunPeriodically(gridMutateInterval, mutateThenAddEmis),
			inmap.SteadyStateConvergenceCheck(NumIterations, VarGrid.PopGridColumn, m, cConverge),
		}
	}

	d := &inmap.InMAP{
		InitFuncs: append(initFuncs, addInit...),
		RunFuncs:  append(runFuncs, addRun...),
		CleanupFuncs: append([]inmap.DomainManipulator{
			o.Output(sr),
			upload.uploadOutput,
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

// setEmissionsAEP adds AEP-processed emissions flux to an existing grid.
// The returned DomainManipulator must be run after each time the grid changes.
// extraEmis specifies any extra emissions that should be added. It is ignored
// if nil.
func setEmissionsAEP(inventoryConfig *aeputil.InventoryConfig, spatialConfig *aeputil.SpatialConfig, extraEmis *inmap.Emissions, mask geom.Polygon) func(d *inmap.InMAP) error {
	// Read in emissions records and save in memory.
	recs := make(map[string][]aep.Record)
	var err error
	if len(inventoryConfig.NEIFiles) > 0 || len(inventoryConfig.COARDSFiles) > 0 {
		recs, _, err = inventoryConfig.ReadEmissions() // Remember to check error below.
	}

	if mask != nil { // Remove records that do not overlap with mask.
		mb := mask.Bounds()
		for s, srecs := range recs {
			i := 0 // output index
			for _, r := range srecs {
				if r.Location().Bounds().Overlaps(mb) {
					// copy and increment index
					srecs[i] = r
					i++
				}
			}
			// Prevent memory leak by erasing truncated values
			for j := i; j < len(srecs); j++ {
				srecs[j] = nil
			}
			srecs = srecs[:i]
			recs[s] = srecs
		}
	}

	return func(d *inmap.InMAP) error {
		if err != nil { // Check error from ReadEmissions
			return err
		}

		// Specify the grid cells we want to allocate to.
		cells := d.Cells()
		spatialConfig.GridCells = make([]geom.Polygonal, 0, len(cells))
		for _, c := range cells {
			if c.Layer == 0 {
				spatialConfig.GridCells = append(spatialConfig.GridCells, c)
			}
		}

		iter := spatialConfig.Iterator(aeputil.IteratorFromMap(recs), 0)
		var spatialRecs []aep.RecordGridded
		for {
			rec, err := iter.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}
			var totalEmis float64
			for _, v := range rec.Totals() {
				totalEmis += math.Abs(v.Value())
			}
			if totalEmis == 0 {
				continue
			}
			spatialRecs = append(spatialRecs, rec.(aep.RecordGridded))
		}

		var emisRecs []*inmap.EmisRecord
		if len(spatialRecs) > 0 {
			sp, err := spatialConfig.SpatialProcessor()
			if err != nil {
				return err
			}
			emisRecs, err = inmap.FromAEP(spatialRecs, sp.Grids, 0,
				[]aep.Pollutant{{Name: "VOC"}},
				[]aep.Pollutant{{Name: "NOx"}},
				[]aep.Pollutant{{Name: "NH3"}},
				[]aep.Pollutant{{Name: "SOx"}},
				[]aep.Pollutant{{Name: "PM2_5"}},
			)
			if err != nil {
				return err
			}
		}
		emis := inmap.NewEmissions()
		emis.Mask = mask
		for _, e := range emisRecs {
			emis.Add(e)
		}
		if extraEmis != nil { // Add in extra emissions.
			for _, e := range extraEmis.EmisRecords() {
				emis.Add(e)
			}
		}
		return d.SetEmissionsFlux(emis, m)
	}
}
