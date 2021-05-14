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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/ctessum/cdf"
	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/lnashier/viper"
	"github.com/spatialmodel/inmap"
	"github.com/spatialmodel/inmap/cloud"
	"github.com/spatialmodel/inmap/cloud/cloudrpc"
	"github.com/spatialmodel/inmap/science/chem/simplechem"
	"github.com/spf13/cobra"
)

// SR can be used to create a source-receptor matrix.
type SR struct {
	d      *inmap.InMAP
	client cloudrpc.CloudRPCClient
	m      inmap.Mechanism
	grid   []geom.Polygonal // grid is the geometry of the SR grid.

	// prj is the grid projection.
	prj string

	// tempDir is a temporary directory for staging input and output files.
	tempDir string
}

// NewSR initializes an SR object.
// varGridData specifies a reader for the variable grid data file,
// varGridConfig specifies the variable-resolution grid, and
// client specifies a client to the service for running the simulations.
func NewSR(varGridData io.Reader, varGridConfig *inmap.VarGridConfig, client cloudrpc.CloudRPCClient) (*SR, error) {
	tempDir, err := ioutil.TempDir("", "inmap_sr")
	if err != nil {
		return nil, err
	}

	var m simplechem.Mechanism
	sr := &SR{
		d: &inmap.InMAP{
			InitFuncs: []inmap.DomainManipulator{
				inmap.Load(varGridData, varGridConfig, nil, m),
			},
		},
		client:  client,
		m:       m,
		tempDir: tempDir,
		prj:     varGridConfig.GridProj,
	}

	if err = sr.d.Init(); err != nil {
		return nil, fmt.Errorf("problem initializing variable grid data: %v", err)
	}
	sr.grid = sr.d.GetGeometry(0, false)
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

// Start starts the simulations necessary to create a source-receptor matrix
// on a Kubernetes cluster.layers specifies the grid layers that SR relationships
// should be calculated for. begin and end are indices in the static variable
// grid where the computations should begin and end. if end<0, then end will
// be set to the last grid cell in the static grid.
// Version is the version of the InMAP docker container to use, e.g. "latest" or "v1.7.2".
func (sr *SR) Start(ctx context.Context, jobName, version string, layers []int, begin, end int, root *cobra.Command, config *viper.Viper, cmdArgs, inputFiles []string, memoryGB int32) error {
	// Set mandatory configuration variables.
	config.Set("OutputVariables", outputVarsStr)
	config.Set("EmissionUnits", "ug/s")

	var maxLayer int
	for _, l := range layers {
		if l > maxLayer {
			maxLayer = l
		}
	}

	layersMap := make(map[int]struct{})
	for _, l := range layers {
		layersMap[l] = struct{}{}
	}
	if l := len(sr.d.Cells()); end < 0 || end > l {
		end = l
	}
	for i := 0; i < len(sr.d.Cells()); i++ {
		cell := sr.d.Cells()[i]
		_, layerok := layersMap[cell.Layer]
		if i >= end || cell.Layer > maxLayer {
			break
		} else if i < begin || !layerok {
			continue
		}
		log.Println("starting", i)

		// Create emissions shapefile for this source location.
		fname, err := sr.writeEmisShapefile(i, cell)
		if err != nil {
			return err
		}
		config.Set("EmissionsShapefiles", []string{fname})

		js, err := cloud.JobSpec(root, config, version, sr.jobName(jobName, i, cell), cmdArgs, inputFiles, memoryGB)
		if err != nil {
			return err
		}

		err = backoff.RetryNotify(
			func() error {
				// Start the simulation.
				_, err = sr.client.RunJob(ctx, js)
				if err != nil {
					if strings.Contains(err.Error(), "already exists") {
						log.Println(err)
					} else {
						return fmt.Errorf("sr: starting index %d layer %d: %v", i, cell.Layer, err)
					}
				}
				return nil
			},
			backoff.NewExponentialBackOff(),
			func(err error, d time.Duration) {
				log.Printf("%v: retrying in %v", err, d)
			},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sr *SR) jobName(jobName string, i int, cell *inmap.Cell) string {
	return fmt.Sprintf("%s-%d-%d", jobName, i, cell.Layer)
}

// writeEmisShapefile writes an emissions input shapefile for SR index i
// and the given source cell. It returns the path to the shapefile.
func (sr *SR) writeEmisShapefile(i int, cell *inmap.Cell) (string, error) {
	type emisRec struct {
		geom.Point
		VOC, NOx, NH3, SOx float64 // emissions [μg/s]
		PM25               float64 `shp:"PM2_5"` // emissions [μg/s]
		Height             float64 // stack height [m]
	}

	fname := filepath.Join(sr.tempDir, fmt.Sprintf("emis_%d_%d.shp", i, cell.Layer))
	prjfname := filepath.Join(sr.tempDir, fmt.Sprintf("emis_%d_%d.prj", i, cell.Layer))
	e, err := shp.NewEncoder(fname, emisRec{})
	if err != nil {
		return "", err
	}
	err = e.Encode(&emisRec{
		Height: cell.LayerHeight + cell.Dz/2,
		VOC:    1, // all units = μg/s
		NOx:    1,
		NH3:    1,
		SOx:    1,
		PM25:   1,
		Point:  cell.Centroid(),
	})
	if err != nil {
		return "", err
	}
	e.Close()
	o, err := os.Create(prjfname)
	if err != nil {
		return "", err
	}
	_, err = o.Write([]byte(sr.prj))
	if err != nil {
		return "", err
	}
	return fname, nil
}

var outputVars = map[string]string{"SOA": "SOA",
	"PrimPM25": "PrimaryPM25",
	"pNH4":     "pNH4",
	"pSO4":     "pSO4",
	"pNO3":     "pNO3",
}

var outputVarsStr = "{\"SOA\":\"SOA\",\"PrimPM25\":\"PrimaryPM25\",\"pNH4\":\"pNH4\"," +
	"\"pSO4\":\"pSO4\",\"pNO3\":\"pNO3\"}"

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

// Save saves the results of the simulations that were run to create the SR
// matrix specified by jobName to outfile.
// layers specifies the grid layers that SR relationships were calculated for.
// begin and end are indices in the static variable grid where the computations
// should began and ended. if end<0, then end will be set to the last grid cell
// in the static grid. outfile is the location of the output file. The units of
// the SR matrix will be μg/m3 PM2.5 concentration at each receptor per μg/s
// emission at each source.
// If outfile already exists, the results will be written to the existing file;
// otherwise a new file will be created.
func (sr *SR) Save(ctx context.Context, outfile, jobName string, layers []int, begin, end int) error {
	ff, f, err := sr.createOrOpenOutputFile(outfile, layers)
	if err != nil {
		return err
	}
	defer ff.Close()
	defer os.RemoveAll(sr.tempDir)

	var maxLayer int
	for _, l := range layers {
		if l > maxLayer {
			maxLayer = l
		}
	}
	cells := sr.d.Cells()

	// Figure out the starting index for each layer.
	layerStarts := make(map[int]int)
	var il = -1
	for i, c := range cells {
		l := c.Layer
		if il != l {
			il = l
			layerStarts[l] = i
		}
	}

	// Make a map between the model layers and the SR layers.
	layerMap := make(map[int]int)
	for i, l := range layers {
		layerMap[l] = i
	}
	if l := len(cells); end < 0 || end > l {
		end = l
	}

	// Create functions to asynchronously retrieve the results.
	numGetters := runtime.GOMAXPROCS(-1) * 3
	var lock sync.Mutex
	jobChan := make(chan int, len(cells))
	errChan := make(chan error)
	for x := 0; x < numGetters; x++ {
		go func() {
			for i := range jobChan {
				cell := cells[i]
				log.Println("saving", i, cell.Layer)
				result, err := sr.results(ctx, jobName, i, cell)
				if err != nil {
					errChan <- err
				}
				for name, species := range outputVars {
					data, ok := result[name]
					if !ok {
						errChan <- fmt.Errorf("sr: missing result variable %v from simulation %d layer %d", name, i, cell.Layer)
					}
					if len(data) != layerStarts[1] {
						errChan <- fmt.Errorf("sr: wrong number of records in variable %v from simulation %d layer %d: %d != %d", name, i, cell.Layer, len(data), layerStarts[1])
					}
					data32 := make([]float32, len(data))
					for j, val := range data {
						data32[j] = float32(val)
					}
					l, ok := layerMap[cell.Layer]
					if !ok {
						panic(fmt.Errorf("sr: missing layer %d from %v", cell.Layer, layerMap))
					}
					row := i - layerStarts[cell.Layer]
					begin := []int{l, row, 0}
					end := []int{l, row, len(data32)}
					lock.Lock()
					w := f.Writer(species, begin, end)
					if _, err := w.Write(data32); err != nil {
						lock.Unlock()
						errChan <- fmt.Errorf("sr: writing results for for row=%v, layer=%v: %v", i, cell.Layer, err)
					}
					lock.Unlock()
				}
			}
			errChan <- nil
		}()
	}

	// Save results asynchronously.
	for i := 0; i < len(cells); i++ {
		cell := cells[i]
		_, layerok := layerMap[cell.Layer]
		if i >= end || cell.Layer > maxLayer {
			break
		} else if i < begin || !layerok {
			continue
		}
		jobChan <- i
	}
	close(jobChan)

	// Check errors.
	for i := 0; i < numGetters; i++ {
		if err := <-errChan; err != nil {
			return err
		}
	}
	if err := cdf.UpdateNumRecs(ff); err != nil {
		return fmt.Errorf("sr: finalizing output NetCDF file: %v", err)
	}
	return nil
}

// results gets the results of the simulation specified by the arguments
// and regrids them to match the SR grid.
func (sr *SR) results(ctx context.Context, jobName string, i int, cell *inmap.Cell) (map[string][]float64, error) {
	var jobOutput *cloudrpc.JobOutput
	err := backoff.RetryNotify(
		func() error {
			var err error
			jobOutput, err = sr.client.Output(ctx, &cloudrpc.JobName{
				Version: inmap.Version,
				Name:    sr.jobName(jobName, i, cell),
			})
			return err
		},
		backoff.NewExponentialBackOff(),
		func(err error, d time.Duration) {
			log.Printf("%v: retrying in %v", err, d)
		},
	)
	if err != nil {
		return nil, err
	}
	if len(jobOutput.Files) == 0 {
		return nil, fmt.Errorf("sr: getting results for i=%d, layer=%d: no results", i, cell.Layer)
	}
	path := filepath.Join(sr.tempDir, fmt.Sprint(i))
	os.Mkdir(path, os.ModePerm)
	defer os.RemoveAll(path)
	for fname, data := range jobOutput.Files {
		w, err := os.Create(filepath.Join(sr.tempDir, fmt.Sprint(i), fname))
		if err != nil {
			return nil, fmt.Errorf("sr: getting results for i=%d, layer=%d: problem staging output shapefile: %v", i, cell.Layer, err)
		}
		_, err = w.Write(data)
		if err != nil {
			return nil, fmt.Errorf("sr: getting results for i=%d, layer=%d: problem staging output shapefile: %v", i, cell.Layer, err)
		}
	}
	d, err := shp.NewDecoder(filepath.Join(sr.tempDir, fmt.Sprint(i), "OutputFile.shp"))
	if err != nil {
		return nil, fmt.Errorf("sr: getting results for i=%d, layer=%d: problem opening output shapefile: %v", i, cell.Layer, err)
	}
	defer d.Close()
	fields := d.Reader.Fields()
	vars := make([]string, len(fields))
	for i, f := range fields {
		vars[i] = f.String()
	}
	results := make(map[string][]float64)
	for _, v := range vars {
		results[v] = make([]float64, d.AttributeCount())
	}
	grid := make([]geom.Polygonal, d.AttributeCount())
	for i := 0; i < d.AttributeCount(); i++ {
		g, fields, more := d.DecodeRowFields(vars...)
		if !more {
			return nil, fmt.Errorf("sr: getting results for i=%d, layer=%d: problem reading output shapefile: ran out of rows", i, cell.Layer)
		}
		grid[i] = g.(geom.Polygonal)
		for n, valStr := range fields {
			v, err := strconv.ParseFloat(valStr, 64)
			if err != nil {
				return nil, fmt.Errorf("sr: getting results for i=%d, layer=%d: problem reading output shapefile: %v", i, cell.Layer, err)
			}
			results[n][i] = v
		}
	}
	if err := d.Error(); err != nil {
		return nil, fmt.Errorf("sr: reading results for i=%d, layer=%d: %v", i, cell.Layer, err)
	}
	for n, oldData := range results { // Regrid to SR grid.
		newData, err := inmap.Regrid(grid, sr.grid, oldData)
		if err != nil {
			return nil, fmt.Errorf("sr: reading results for i=%d, layer=%d: %v", i, cell.Layer, err)
		}
		results[n] = newData
	}
	return results, nil
}

// Clean removes intermediate files created during simulations carried out
// to create a source-receptor matrix.
// layers specifies the grid layers that SR relationships
// should be calculated for. begin and end are indices in the static variable
// grid where the computations should begin and end. if end<0, then end will
// be set to the last grid cell in the static grid.
func (sr *SR) Clean(ctx context.Context, jobName string, layers []int, begin, end int) error {

	var maxLayer int
	for _, l := range layers {
		if l > maxLayer {
			maxLayer = l
		}
	}

	layersMap := make(map[int]struct{})
	for _, l := range layers {
		layersMap[l] = struct{}{}
	}
	if l := len(sr.d.Cells()); end < 0 || end > l {
		end = l
	}
	for i := 0; i < len(sr.d.Cells()); i++ {
		cell := sr.d.Cells()[i]
		_, layerok := layersMap[cell.Layer]
		if i >= end || cell.Layer > maxLayer {
			break
		} else if i < begin || !layerok {
			continue
		}

		// Delete the job.
		_, err := sr.client.Delete(ctx, &cloudrpc.JobName{
			Name:    sr.jobName(jobName, i, cell),
			Version: inmap.Version,
		})
		if err != nil {
			return fmt.Errorf("sr: cleaning up index %d layer %d: %v", i, cell.Layer, err)
		}
	}
	return nil
}

func (sr *SR) createOrOpenOutputFile(outfile string, layers []int) (*os.File, *cdf.File, error) {
	nGridCells, err := sr.layerGridCells(layers)
	if err != nil {
		return nil, nil, err
	}

	var f *cdf.File
	var ff *os.File
	// Create the file if it doesn't exist, otherwise use the pre-existing file.
	if _, fileErr := os.Stat(outfile); fileErr != nil {
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

		for _, err := range h.Check() {
			return nil, nil, fmt.Errorf("creating SR netcdf file: %v", err)
		}

		ff, err = os.Create(outfile)
		if err != nil {
			return nil, nil, fmt.Errorf("creating SR netcdf file: %v", err)
		}
		f, err = cdf.Create(ff, h)
		if err != nil {
			return nil, nil, fmt.Errorf("creating new SR netcdf file: %v", err)
		}

		// Add included layers
		l := make([]int32, len(layers))
		for i, ll := range layers {
			l[i] = int32(ll)
		}
		w := f.Writer("layers", []int{0}, []int{len(l)})
		if _, err = w.Write(l); err != nil {
			return nil, nil, fmt.Errorf("writing SR netcdf layers: %v", err)
		}

		// Add InMAP data
		o, err := inmap.NewOutputter("", true, inmapVars, nil, sr.m)
		if err != nil {
			return nil, nil, fmt.Errorf("inmap: preparing output variables: %v", err)
		}
		data, err := sr.d.Results(o)
		if err != nil {
			return nil, nil, fmt.Errorf("writing InMAP variables to SR netcdf file: %v", err)
		}
		for _, v := range inmapVars {
			w := f.Writer(v, []int{0}, []int{len(data[v])})
			if _, err := w.Write(data[v]); err != nil {
				return nil, nil, fmt.Errorf("writing variable %s to SR netcdf file: %v", v, err)
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
				return nil, nil, fmt.Errorf("writing direction %s to SR netcdf file: %v", v, err)
			}
		}
	} else { // file exists.
		ff, err = os.OpenFile(outfile, os.O_RDWR, os.ModePerm)
		if err != nil {
			return nil, nil, fmt.Errorf("opening SR netcdf file: %v", err)
		}
		f, err = cdf.Open(ff)
		if err != nil {
			return nil, nil, fmt.Errorf("initializing exisiting SR netcdf file: %v", err)
		}
	}
	return ff, f, nil
}
