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

package sr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"

	"github.com/ctessum/geom"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/science/chem/simplechem"
	"github.com/spatialmodel/inmap/science/drydep/simpledrydep"
	"github.com/spatialmodel/inmap/science/wetdep/emepwetdep"
)

// Empty is used for passing content-less messages.
type Empty struct{}

// RPCPort specifies the port for RPC communications. The default is
// 6060.
var RPCPort = "6060"

// Worker is a worker for performing InMAP simulations. It should not be interacted
// with directly, but it is exported to meet RPC requirements.
type Worker struct {
	Config        *inmap.VarGridConfig
	CTMData       *inmap.CTMData
	Pop           *inmap.Population
	PopIndices    inmap.PopIndices
	MR            *inmap.MortalityRates
	MortIndices   inmap.MortIndices
	GridGeom      []geom.Polygonal // Geometry of the output grid.
	InMAPDataFile string           // inmapDataFile is the path to the input data file in .gob format.
}

// IOData holds the input to and output from a simulation request.
type IOData struct {
	Emis       []*inmap.EmisRecord
	Output     map[string][]float64
	Row, Layer int
}

// Result allows a local worker to look like a distributed request.
func (io *IOData) Result() (*IOData, error) {
	return io, nil
}

// Calculate performs an InMAP simulation. It meets the requirements for
// use with rpc.Call.
func (s *Worker) Calculate(input *IOData, output *IOData) error {
	if s.Pop == nil {
		// Initialize the worker if it hasn't already been done.
		if err := s.Init(nil, nil); err != nil {
			return err
		}
	}

	log.Printf("Slave calculating row=%v, layer=%v\n", input.Row, input.Layer)

	var m simplechem.Mechanism
	scienceFuncs := inmap.Calculations(
		inmap.UpwindAdvection(),
		inmap.Mixing(),
		inmap.MeanderMixing(),
		simpledrydep.DryDeposition(simplechem.SimpleDryDepIndices),
		emepwetdep.WetDeposition(simplechem.EMEPWetDepIndices),
		m.Chemistry(),
	)

	emis := inmap.NewEmissions()
	for _, e := range input.Emis {
		emis.Add(e)
	}

	initFuncs := []inmap.DomainManipulator{
		s.Config.RegularGrid(s.CTMData, s.Pop, s.PopIndices, s.MR, s.MortIndices, emis, m),
		inmap.SetTimestepCFL(),
	}
	popConcMutator := inmap.NewPopConcMutator(s.Config, s.PopIndices)
	const gridMutateInterval = 3 * 60 * 60 // every 3 hours in seconds
	runFuncs := []inmap.DomainManipulator{
		inmap.Calculations(inmap.AddEmissionsFlux()),
		scienceFuncs,
		inmap.RunPeriodically(gridMutateInterval,
			s.Config.MutateGrid(popConcMutator.Mutate(),
				s.CTMData, s.Pop, s.MR, emis, m, nil)),
		inmap.RunPeriodically(gridMutateInterval, inmap.SetTimestepCFL()),
		inmap.SteadyStateConvergenceCheck(-1, s.Config.PopGridColumn, nil),
	}

	d := &inmap.InMAP{
		InitFuncs: initFuncs,
		RunFuncs:  runFuncs,
	}

	if err := d.Init(); err != nil {
		return fmt.Errorf("InMAP: problem initializing model: %v\n", err)
	}

	if err := d.Run(); err != nil {
		return fmt.Errorf("InMAP: problem running simulation: %v\n", err)
	}

	output.Output = make(map[string][]float64)
	output.Row = input.Row
	output.Layer = input.Layer

	o, err := inmap.NewOutputter("", false, outputVars, nil, m)
	if err != nil {
		return err
	}
	r, err := d.Results(o)
	if err != nil {
		return err
	}
	g := d.GetGeometry(0, false)
	for pol, data := range r {
		d, err := inmap.Regrid(g, s.GridGeom, data)
		if err != nil {
			return err
		}
		output.Output[pol] = d
	}
	return nil
}

// Exit shuts down the worker. It meets the requirements for
// use with rpc.Call.
func (s *Worker) Exit(in, out *Empty) error {
	os.Exit(0)
	return nil
}

// NewWorker sets up an RPC listener for performing simulations.
// InMAPDataFile specifies
// the location of the inmap regular-gridded data, and GridGeom specifies the
// output grid geometry.
func NewWorker(config *inmap.VarGridConfig, InMAPDataFile string, GridGeom []geom.Polygonal) *Worker {
	s := new(Worker)
	s.Config = config
	s.GridGeom = GridGeom
	s.InMAPDataFile = InMAPDataFile
	return s
}

// Init initializes the worker. It needs to be called after NewWorker and
// before any simulations are performed. It meets the requirements for use
// with rpc.Call.
func (s *Worker) Init(_, _ *Empty) error {
	f, err := os.Open(s.InMAPDataFile)
	if err != nil {
		return fmt.Errorf("problem loading input data: %v\n", err)
	}
	s.CTMData, err = s.Config.LoadCTMData(f)
	if err != nil {
		return fmt.Errorf("problem loading input data: %v\n", err)
	}

	log.Println("Loading population and mortality rate data")
	s.Pop, s.PopIndices, s.MR, s.MortIndices, err = s.Config.LoadPopMort()
	if err != nil {
		return fmt.Errorf("problem loading population or mortality data: %v", err)
	}
	return nil
}

// WorkerListen directs the Worker to start listening for requests over RPCPort.
// It is a top-level function rather than a method of s to avoid problems with
// RPC registration.
func WorkerListen(s *Worker, RPCPort string) error {
	if err := rpc.Register(s); err != nil {
		return err
	}
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", ":"+RPCPort)
	if err != nil {
		return err
	}
	log.Println("Started slave")
	return http.Serve(l, nil)
}
