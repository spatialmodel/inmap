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

// Package sr contains functions for creating a source-receptor (SR) matrix from
// the InMAP air pollution model and interacting with it.
package sr

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"bitbucket.org/ctessum/cdf"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/science/chem/simplechem"
	"golang.org/x/net/context"
)

// SR can be used to create a source-receptor matrix.
type SR struct {
	d           *inmap.InMAP
	c           *Cluster
	numNodes    int     // The number of nodes for performing calculations.
	localWorker *Worker // Worker for local processing where there are 0 nodes.
	m           inmap.Mechanism
}

// NewSR initializes an SR object.
// varGridFile specifies the location of the InMAP variable grid data.
// inmapDataFile specifies the location of the InMAP regular grid data.
// command is the command that should be executed to start slave processes.
// nodes specify unique addresses of the machines that the simulations
// should be carried out on. If len(nodes) == 0, then calculations will be
// carried out locally instead of on a cluster.
func NewSR(varGridFile, inmapDataFile, command, logDir string, config *inmap.VarGridConfig, nodes []string) (*SR, error) {
	r, err := os.Open(varGridFile)
	if err != nil {
		return nil, fmt.Errorf("problem opening file to load VariableGridData: %v", err)
	}

	var m simplechem.Mechanism
	sr := &SR{
		d: &inmap.InMAP{
			InitFuncs: []inmap.DomainManipulator{
				inmap.Load(r, config, nil, m),
			},
		},
		c:        NewCluster(command, logDir, "Worker.Exit", RPCPort),
		numNodes: len(nodes),
		m:        m,
	}
	if err = sr.d.Init(); err != nil {
		return nil, fmt.Errorf("problem initializing variable grid data: %v\n", err)
	}
	// Start up workers
	errChan := make(chan error)
	for _, n := range nodes {
		go func(n string) {
			errChan <- sr.c.NewWorker(n)
		}(n)
	}
	for range nodes {
		if err = <-errChan; err != nil {
			return nil, err
		}
	}
	if sr.numNodes == 0 {
		sr.localWorker = NewWorker(config, inmapDataFile, sr.d.GetGeometry(0, false))
	}
	return sr, nil
}

// layerGridCells returns the number of grid cells in each of the specified
// layers. All of the layers are expected to have the same number of grid cells;
// if they don't, an error is returned.
func (sr *SR) layerGridCells(layers []int) (int, error) {
	layerMap := make(map[int]int)
	for _, c := range sr.d.Cells() {
		layerMap[c.Layer]++
	}
	var nCells int
	for i, l := range layers {
		if i == 0 {
			nCells = layerMap[l]
		} else {
			if nCells != layerMap[l] {
				return -1, fmt.Errorf("sr: number of layer cells don't match: %d!=%d", nCells, layerMap[l])
			}
		}
	}
	return nCells, nil
}

type resulter interface {
	Result() (*IOData, error)
}

// Run runs the simulations necessary to create a source-receptor matrix and writes out the
// results. layers specifies the grid layers that SR relationships
// should be calculated for. begin and end are indices in the static variable
// grid where the computations should begin and end. if end<0, then end will
// be set to the last grid cell in the static grid.
// outfile is the location of the output file. The units of the SR matrix will
// be μg/m3 PM2.5 concentration at each receptor per μg/s emission at each source.
func (sr *SR) Run(outfile string, layers []int, begin, end int) error {

	errChan := make(chan error)
	reqChan := make(chan resulter, 1000*sr.numNodes+1) // make sure there is enough buffering.
	ctx := context.TODO()

	go sr.writeResults(outfile, layers, reqChan, errChan) // Start process to write results to file

	layersMap := make(map[int]Empty)
	for _, l := range layers {
		layersMap[l] = Empty{}
	}
	if l := len(sr.d.Cells()); end < 0 || end > l {
		end = l
	}
	for i := 0; i < len(sr.d.Cells()); i++ {

		// check for errors in writer.
		select {
		case err := <-errChan:
			sr.c.Shutdown()
			return fmt.Errorf("sr.writeOutput: %v", err)
		default:
		}

		cell := sr.d.Cells()[i]
		_, layerok := layersMap[cell.Layer]
		if i < begin || i >= end || !layerok {
			continue
		}

		log.Printf("Sending row=%v, layer=%v\n", i, cell.Layer)
		rp := sr.newRequestPayload(i, cell)
		if sr.numNodes > 0 {
			r := sr.c.NewRequest(ctx, "Worker.Calculate", rp)
			r.Send()
			reqChan <- r
		} else {
			o := new(IOData)
			if err := sr.localWorker.Calculate(rp, o); err != nil {
				sr.c.Shutdown()
				return err
			}
			reqChan <- o
		}
	}
	close(reqChan)
	sr.c.Shutdown()
	return <-errChan // Wait for writer to finish.
}

func (sr *SR) newRequestPayload(i int, cell *inmap.Cell) *IOData {
	requestPayload := new(IOData)
	requestPayload.Row = i
	requestPayload.Layer = cell.Layer
	requestPayload.Emis = []*inmap.EmisRecord{
		{
			Height: cell.LayerHeight + cell.Dz/2,
			VOC:    1, // all units = μg/s
			NOx:    1,
			NH3:    1,
			SOx:    1,
			PM25:   1,
			Geom:   cell.Centroid(),
		},
	}
	return requestPayload
}

var outputVars = map[string]string{"SOA": "SOA",
	"PrimaryPM25": "PrimaryPM25",
	"pNH4":        "pNH4",
	"pSO4":        "pSO4",
	"pNO3":        "pNO3",
}

func sortKeys(m map[string]string) []string {
	o := make([]string, len(m))
	var i int
	for k := range m {
		o[i] = k
		i++
	}
	sort.Strings(o)
	return o
}

func (sr *SR) writeResults(outfile string, layers []int, requestChan chan resulter, errChan chan error) {
	nGridCells, err := sr.layerGridCells(layers)
	if err != nil {
		errChan <- err
		return
	}

	var f *cdf.File
	var ff *os.File

	if _, fileErr := os.Stat(outfile); fileErr != nil { // file doesn't exist
		log.Println("creating output file")

		// Get model variable names for inclusion in the SR matrix.
		vars, descriptions, units := sr.d.OutputOptions(sr.m)
		inmapVars := make(map[string]string)
		inmapDescriptions := make(map[string]string)
		inmapUnits := make(map[string]string)
		pols := make(map[string]struct{})
		for _, v := range sr.m.Species() {
			if !strings.Contains(v, "Emissions") {
				pols[v] = struct{}{}
			}
		}

		for i, v := range vars {
			if _, ok := pols[v]; ok {
				continue // ignore modeled pollutants
			}
			inmapVars[v] = v
			inmapDescriptions[v] = descriptions[i]
			inmapUnits[v] = units[i]
		}

		h := cdf.NewHeader([]string{"layer", "source", "receptor", "allcells", "layers"},
			[]int{len(layers), nGridCells, nGridCells, len(sr.d.Cells()), len(layers)})

		h.AddVariable("layers", []string{"layers"}, []int32{0})
		h.AddAttribute("layers", "description", "Layer indices for which the SR calculation was performed")

		for _, k := range sortKeys(outputVars) {
			vs := outputVars[k]
			h.AddVariable(vs, []string{"layer", "source", "receptor"},
				[]float32{0})
			h.AddAttribute(vs, "description", fmt.Sprintf("%s source-receptor relationships", vs))
			h.AddAttribute(vs, "units", "μg m-3 concentration at receptor location per μg s-1 emissions at source location")
		}
		// InMAP data.
		for _, i := range sortKeys(inmapVars) {
			v := inmapVars[i]
			h.AddVariable(v, []string{"allcells"}, []float64{0.})
			h.AddAttribute(v, "description", inmapDescriptions[i])
			h.AddAttribute(v, "units", inmapUnits[i])
		}
		// Grid cell edges.
		for _, v := range []string{"N", "S", "E", "W"} {
			h.AddVariable(v, []string{"allcells"}, []float64{0.})
			h.AddAttribute(v, "description", fmt.Sprintf("%s grid cell edge", v))
		}

		h.Define()

		ff, err = os.Create(outfile)
		if err != nil {
			errChan <- fmt.Errorf("creating SR netcdf file: %v", err)
			return
		}
		f, err = cdf.Create(ff, h)
		if err != nil {
			errChan <- fmt.Errorf("creating new SR netcdf file: %v", err)
			return
		}
		defer ff.Close()

		// Add included layers
		l := make([]int32, len(layers))
		for i, ll := range layers {
			l[i] = int32(ll)
		}
		w := f.Writer("layers", []int{0}, []int{len(l)})
		if _, err = w.Write(l); err != nil {
			errChan <- fmt.Errorf("writing SR netcdf layers: %v", err)
			return
		}

		// Add InMAP data
		o, err := inmap.NewOutputter("", true, inmapVars, nil, sr.m)
		if err != nil {
			errChan <- fmt.Errorf("inmap: preparing output variables: %v", err)
			return
		}
		data, err := sr.d.Results(o)
		if err != nil {
			errChan <- fmt.Errorf("writing InMAP variables to SR netcdf file: %v", err)
			return
		}
		for _, v := range inmapVars {
			w := f.Writer(v, []int{0}, []int{len(data[v])})
			if _, err := w.Write(data[v]); err != nil {
				errChan <- fmt.Errorf("writing variable %s to SR netcdf file: %v", v, err)
				return
			}
		}

		// Add grid geometry
		cells := sr.d.Cells()
		N := make([]float64, len(cells))
		S := make([]float64, len(cells))
		E := make([]float64, len(cells))
		W := make([]float64, len(cells))
		for i, c := range cells {
			b := c.Bounds()
			N[i] = b.Max.Y
			S[i] = b.Min.Y
			E[i] = b.Max.X
			W[i] = b.Min.X
		}
		g := [][]float64{N, S, E, W}
		for i, v := range []string{"N", "S", "E", "W"} {
			w := f.Writer(v, []int{0}, []int{len(N)})
			if _, err := w.Write(g[i]); err != nil {
				errChan <- fmt.Errorf("writing direction %s to SR netcdf file: %v", v, err)
				return
			}
		}

	} else { // file exists.
		log.Println("opening existing output file")
		ff, err = os.OpenFile(outfile, os.O_RDWR, os.ModePerm)
		if err != nil {
			errChan <- fmt.Errorf("opening SR netcdf file: %v", err)
			return
		}
		f, err = cdf.Open(ff)
		if err != nil {
			errChan <- fmt.Errorf("initializing exisiting SR netcdf file: %v", err)
			return
		}
		defer ff.Close()
	}

	// Figure out the starting index for each layer.
	layerStarts := make(map[int]int)
	var il = -1
	for i, c := range sr.d.Cells() {
		l := c.Layer
		if il != l {
			il = l
			layerStarts[l] = i
		}
	}

	// make a map between the model layers and the SR layers
	layerMap := make(map[int]int)
	for i, l := range layers {
		layerMap[l] = i
	}
	for req := range requestChan {
		result, err := req.Result()
		if err != nil {
			errChan <- fmt.Errorf("SR simulation: %v", err)
			return
		}

		for _, v := range outputVars {
			data := result.Output[v]
			data32 := make([]float32, len(data))
			for i, val := range data {
				data32[i] = float32(val)
			}
			l, ok := layerMap[result.Layer]
			if !ok {
				panic(fmt.Errorf("missing layer %d from %v", result.Layer, layerMap))
			}
			row := result.Row - layerStarts[result.Layer]
			begin := []int{l, row, 0}
			end := []int{l, row, len(data32)}
			w := f.Writer(v, begin, end)
			if _, err := w.Write(data32); err != nil {
				errChan <- fmt.Errorf("writing results for for row=%v, layer=%v: %v\n",
					result.Row, result.Layer, err)
				return
			}
		}
		log.Printf("Finished writing results for row=%v, layer=%v", result.Row, result.Layer)
	}
	cdf.UpdateNumRecs(ff)
	errChan <- nil
}
