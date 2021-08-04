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
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ctessum/cdf"
	"github.com/ctessum/sparse"
	"github.com/spatialmodel/inmap/emissions/aep"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/proj"
	"gonum.org/v1/gonum/floats"
)

// VarGridConfig is a holder for the configuration information for creating a
// variable-resolution grid.
type VarGridConfig struct {
	VariableGridXo float64 // lower left of output grid, x
	VariableGridYo float64 // lower left of output grid, y
	VariableGridDx float64 // m
	VariableGridDy float64 // m
	Xnests         []int   // Nesting multiples in the X direction
	Ynests         []int   // Nesting multiples in the Y direction
	HiResLayers    int     // number of layers to do in high resolution (layers above this will be lowest resolution.

	PopDensityThreshold float64 // limit for people per unit area in the grid cell
	PopThreshold        float64 // limit for total number of people in the grid cell

	// PopConcThreshold is the limit for
	// Σ(|ΔConcentration|)*combinedVolume*|ΔPopulation| / {Σ(|totalMass|)*totalPopulation}.
	// See the documentation for PopConcMutator for more information.
	PopConcThreshold float64

	CensusFile        string   // Path to census shapefile or COARDS-compliant NetCDF file
	CensusPopColumns  []string // Shapefile fields containing populations for multiple demographics
	PopGridColumn     string   // Name of field in shapefile to be used for determining variable grid resolution
	MortalityRateFile string   // Path to the mortality rate shapefile

	// MortalityRateColumns give the columns in the mortality rate
	// shapefile containing mortality rates, and the population groups that
	// should be used for population-weighting each mortality rate.
	MortalityRateColumns map[string]string

	GridProj string // projection info for CTM grid; Proj4 format
}

func (c *VarGridConfig) bounds() *geom.Bounds {
	return &geom.Bounds{
		Min: geom.Point{X: c.VariableGridXo, Y: c.VariableGridYo},
		Max: geom.Point{
			X: c.VariableGridXo + c.VariableGridDx*float64(c.Xnests[0]),
			Y: c.VariableGridYo + c.VariableGridDy*float64(c.Ynests[0]),
		},
	}
}

// CTMData holds processed data from a chemical transport model
type CTMData struct {
	gridTree *rtree.Rtree

	xo float64 // lower left of Chemical Transport Model (CTM) grid, x
	yo float64 // lower left of grid, y
	dx float64 // m
	dy float64 // m
	nx int
	ny int

	// Data is a map of information about processed CTM variables,
	// with the keys being the variable names.
	Data map[string]struct {
		Dims        []string           // netcdf dimensions for this variable
		Description string             // variable description
		Units       string             // variable units
		Data        *sparse.DenseArray // variable data
	}
}

// AddVariable adds data for a new variable to d.
func (d *CTMData) AddVariable(name string, dims []string, description, units string, data *sparse.DenseArray) {
	if d.Data == nil {
		d.Data = make(map[string]struct {
			Dims        []string
			Description string
			Units       string
			Data        *sparse.DenseArray
		})
	}
	d.Data[name] = struct {
		Dims        []string           // netcdf dimensions for this variable
		Description string             // variable description
		Units       string             // variable units
		Data        *sparse.DenseArray // variable data
	}{
		Dims:        dims,
		Description: description,
		Units:       units,
		Data:        data,
	}
}

// LoadCTMData loads CTM data from a netcdf file.
func (config *VarGridConfig) LoadCTMData(rw cdf.ReaderWriterAt) (*CTMData, error) {
	f, err := cdf.Open(rw)
	if err != nil {
		return nil, fmt.Errorf("inmap.LoadCTMData: %v", err)
	}
	o := new(CTMData)
	nz := f.Header.Lengths("UAvg")[0]

	// Get CTM grid attributes
	o.dx = f.Header.GetAttribute("", "dx").([]float64)[0]
	o.dy = f.Header.GetAttribute("", "dy").([]float64)[0]
	o.nx = int(f.Header.GetAttribute("", "nx").([]int32)[0])
	o.ny = int(f.Header.GetAttribute("", "ny").([]int32)[0])
	o.xo = f.Header.GetAttribute("", "x0").([]float64)[0]
	o.yo = f.Header.GetAttribute("", "y0").([]float64)[0]

	dataVersion := f.Header.GetAttribute("", "data_version").(string)

	if dataVersion != InMAPDataVersion {
		return nil, fmt.Errorf("inmap.LoadCTMData: data version %s is incompatible "+
			"with the required version %s", dataVersion, InMAPDataVersion)
	}

	o.makeCTMgrid(nz)

	od := make(map[string]struct {
		Dims        []string
		Description string
		Units       string
		Data        *sparse.DenseArray
	})
	for _, v := range f.Header.Variables() {
		d := struct {
			Dims        []string
			Description string
			Units       string
			Data        *sparse.DenseArray
		}{}
		d.Description = f.Header.GetAttribute(v, "description").(string)
		d.Units = f.Header.GetAttribute(v, "units").(string)
		dims := f.Header.Lengths(v)
		r := f.Reader(v, nil, nil)
		d.Data = sparse.ZerosDense(dims...)
		tmp := make([]float32, len(d.Data.Elements))
		_, err = r.Read(tmp)
		if err != nil {
			return nil, fmt.Errorf("inmap.LoadCTMData: %v", err)
		}
		d.Dims = f.Header.Dimensions(v)

		// Check that data matches dimensions.
		n := 1
		for _, v := range dims {
			n *= v
		}
		if len(tmp) != n {
			return nil, fmt.Errorf("inmap.VarGridConfig.LoadCTMData: dims are %d but "+
				"array length is %d", n, len(tmp))
		}

		for i, v := range tmp {
			d.Data.Elements[i] = float64(v)
		}
		od[v] = d
	}
	o.Data = od
	return o, nil
}

// Write writes d to netcdf file w.
func (d *CTMData) Write(w *os.File) error {
	windSpeed := d.Data["WindSpeed"].Data
	uAvg := d.Data["UAvg"].Data
	vAvg := d.Data["VAvg"].Data
	wAvg := d.Data["WAvg"].Data
	h := cdf.NewHeader(
		[]string{"x", "y", "z", "xStagger", "yStagger", "zStagger"},
		[]int{windSpeed.Shape[2], windSpeed.Shape[1], windSpeed.Shape[0],
			uAvg.Shape[2], vAvg.Shape[1], wAvg.Shape[0]})
	h.AddAttribute("", "comment", "InMAP meteorology and baseline chemistry data file")

	h.AddAttribute("", "x0", []float64{d.xo})
	h.AddAttribute("", "y0", []float64{d.yo})
	h.AddAttribute("", "dx", []float64{d.dx})
	h.AddAttribute("", "dy", []float64{d.dy})
	h.AddAttribute("", "nx", []int32{int32(windSpeed.Shape[2])})
	h.AddAttribute("", "ny", []int32{int32(windSpeed.Shape[1])})

	h.AddAttribute("", "data_version", InMAPDataVersion)

	// Sort the names so they write in the same order every time.
	names := make([]string, 0, len(d.Data))
	for n := range d.Data {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, name := range names {
		dd := d.Data[name]
		h.AddVariable(name, dd.Dims, []float32{0})
		h.AddAttribute(name, "description", dd.Description)
		h.AddAttribute(name, "units", dd.Units)
	}
	h.Define()

	f, err := cdf.Create(w, h) // writes the header to ff
	if err != nil {
		return err
	}

	for _, name := range names {
		dd := d.Data[name]
		if err = writeNCF(f, name, dd.Data); err != nil {
			return fmt.Errorf("inmap: writing variable %s to netcdf file: %v", name, err)
		}
	}
	err = cdf.UpdateNumRecs(w)
	if err != nil {
		return err
	}
	return nil
}

// CombineCTMData returns the combination of the input data nests.
// The output will have the extent of the first nest and the horizontal
// resolution of the highest resolution nest. It is assumed that
// the nests fit neatly inside each other; no interpolation will be
// performed. The input nests will be
// overlayed onto the output in the provided order, so each sequential
// nest will write over any previous nest(s) that it overlaps with.
// Vertical layers are assumed to be the same among all nests;
// no vertical layer interpolation is performed.
// If the nests do not all have the same number of layers, an
// error will be returned.
func CombineCTMData(nests ...*CTMData) (*CTMData, error) {
	if len(nests) == 0 {
		return nil, nil
	}

	o := new(CTMData)

	// Get extent and resolution of resulting grid.
	o.xo, o.yo = nests[0].xo, nests[0].yo
	o.dx, o.dy = math.Inf(1), math.Inf(1)
	var nz int
	for i, nest := range nests {
		if _, ok := nest.Data["Dz"]; !ok {
			return nil, errors.New("inmap: CTM data is missing variable `Dz`")
		}
		nestNz := nest.Data["Dz"].Data.Shape[0]
		if i == 0 {
			nz = nestNz
		} else if nz != nestNz {
			return nil, errors.New("inmap: inconsistent number of layers when combining CTM data files")
		}
		if nest.dx < o.dx {
			o.dx = nest.dx
		}
		if nest.dy < o.dy {
			o.dy = nest.dy
		}
	}
	o.nx = nests[0].nx * round(nests[0].dx/o.dx)
	o.ny = nests[0].ny * round(nests[0].dy/o.dy)

	// Copy data.
	for _, nest := range nests {
		xNestFac := round(nest.dx / o.dx)        // nesting ratio in x-direction
		yNestFac := round(nest.dy / o.dy)        // nesting ratio in y-direction
		nestio := round((nest.xo - o.xo) / o.dx) // x-index in output grid of nest ll corner.
		nestjo := round((nest.yo - o.yo) / o.dy) // y-index in output grid of nest ll corner.

		// Closure for copying one layer
		copyLayer := func(get func(j, i int) float64, set func(v float64, j, i int)) {
			for nj := 0; nj < nest.ny; nj++ {
				for ni := 0; ni < nest.nx; ni++ {
					v := get(nj, ni)
					for oj := nestjo + nj*yNestFac; oj < nestjo+(nj+1)*yNestFac; oj++ {
						for oi := nestio + ni*xNestFac; oi < nestio+(ni+1)*xNestFac; oi++ {
							if oi >= 0 && oj >= 0 && oi < o.nx && oj < o.ny {
								set(v, oj, oi)
							}
						}
					}
				}
			}
		}

		for name, data := range nest.Data {
			switch len(data.Dims) {
			case 3:
				if _, ok := o.Data[name]; !ok {
					o.AddVariable(name, data.Dims, data.Description, data.Units, sparse.ZerosDense(nz, o.ny, o.nx))
				}
				od := o.Data[name]
				for k := 0; k < nz; k++ {
					get := func(j, i int) float64 { return data.Data.Get(k, j, i) }
					set := func(v float64, j, i int) { od.Data.Set(v, k, j, i) }
					copyLayer(get, set)
				}
			case 2:
				if _, ok := o.Data[name]; !ok {
					o.AddVariable(name, data.Dims, data.Description, data.Units, sparse.ZerosDense(o.ny, o.nx))
				}
				od := o.Data[name]
				get := func(j, i int) float64 { return data.Data.Get(j, i) }
				set := func(v float64, j, i int) { od.Data.Set(v, j, i) }
				copyLayer(get, set)
			default:
				return nil, fmt.Errorf("inmap: invalid number of dimensions (%d) when combining CTM data", len(data.Dims))
			}
		}
	}
	return o, nil
}

func round(v float64) int { return int(v + 0.5) }

func writeNCF(f *cdf.File, Var string, data *sparse.DenseArray) error {
	// Check that data matches dimensions.
	n := 1
	for _, v := range data.Shape {
		n *= v
	}
	if len(data.Elements) != n {
		return fmt.Errorf("dims are %d but "+"array length is %d", n, len(data.Elements))
	}

	data32 := make([]float32, len(data.Elements))
	for i, e := range data.Elements {
		data32[i] = float32(e)
	}
	end := f.Header.Lengths(Var)
	start := make([]int, len(end))
	w := f.Writer(Var, start, end)
	_, err := w.Write(data32)
	if err != nil {
		return err
	}
	return nil
}

// Population is a holder for information about the human population in
// the model domain.
type Population struct {
	tree func(*geom.Bounds) func() (*population, error)
}

// MortalityRates is a holder for information about the average human
// mortality rate (in units of deaths per 100,000 people per year) in the
// model domain.
type MortalityRates struct {
	tree *rtree.Rtree
}

// PopIndices gives the array indices of each
// population type.
type PopIndices map[string]int

// MortIndices gives the array indices of each
// mortality rate.
type MortIndices map[string]int

// LoadPopMort loads the population and mortality rate data from the shapefiles
// specified in config.
func (config *VarGridConfig) LoadPopMort() (*Population, PopIndices, *MortalityRates, MortIndices, error) {
	gridSR, err := proj.Parse(config.GridProj)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("inmap: while parsing GridProj: %v", err)
	}

	pop, popIndex, err := config.loadPopulation(gridSR, config.bounds())
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("inmap: while loading population: %v", err)
	}
	mort, mortIndex, err := config.loadMortality(gridSR)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("inmap: while loading mortality rate: %v", err)
	}
	return &Population{tree: pop}, PopIndices(popIndex), &MortalityRates{tree: mort}, MortIndices(mortIndex), nil
}

// getCells returns all the grid cells in cellTree that are within box
// and at vertical layer layer.
func getCells(cellTree *rtree.Rtree, box *geom.Bounds, layer int) *cellList {
	x := cellTree.SearchIntersect(box)
	cells := new(cellList)
	for _, xx := range x {
		c := xx.(*Cell)
		if c.Layer == layer {
			cells.add(c)
		}
	}
	return cells
}

func (config *VarGridConfig) webMapTrans() (t proj.Transformer, notMeters bool, err error) {

	// webMapProj is the spatial reference definition for web mapping.
	const webMapProj = "+proj=merc +a=6378137 +b=6378137 +lat_ts=0.0 +lon_0=0.0 +x_0=0.0 +y_0=0 +k=1.0 +units=m +nadgrids=@null +no_defs"
	// webMapSR is the spatial reference for web mapping.
	webMapSR, err := proj.Parse(webMapProj)
	if err != nil {
		return nil, false, fmt.Errorf("inmap: while parsing webMapProj: %v", err)
	}

	gridSR, err := proj.Parse(config.GridProj)
	if err != nil {
		return nil, false, fmt.Errorf("inmap: while parsing GridProj: %v", err)
	}
	webMapTrans, err := gridSR.NewTransform(webMapSR)
	if err != nil {
		return nil, false, fmt.Errorf("inmap: while creating webMapTrans: %v", err)
	}
	if gridSR.ToMeter > 1.0000001 || gridSR.ToMeter < 0.999999 || gridSR.Name == "longlat" {
		notMeters = true
	}
	return webMapTrans, notMeters, nil
}

// RegularGrid returns a function that creates a new regular
// (i.e., not variable resolution) grid
// as specified by the information in c.
func (config *VarGridConfig) RegularGrid(data *CTMData, pop *Population, popIndex PopIndices, mortRates *MortalityRates, mortIndex MortIndices, emis *Emissions, m Mechanism) DomainManipulator {
	return func(d *InMAP) error {
		webMapTrans, notMeters, err := config.webMapTrans()
		if err != nil {
			return err
		}

		d.PopIndices = (map[string]int)(popIndex)
		d.mortIndices = (map[string]int)(mortIndex)

		nz := data.Data["UAvg"].Data.Shape[0]
		d.nlayers = nz

		type cellErr struct {
			cell *Cell
			err  error
		}

		nx := config.Xnests[0]
		ny := config.Ynests[0]
		// Iterate through indices and create the cells in the outermost nest.
		indices := make([][][2]int, 0, nz*ny*nx)
		layers := make([]int, 0, nz*ny*nx)
		for k := 0; k < nz; k++ {
			for j := 0; j < ny; j++ {
				for i := 0; i < nx; i++ {
					indices = append(indices, [][2]int{{i, j}})
					layers = append(layers, k)
				}
			}
		}
		err = d.addCells(config, indices, layers, nil, data, pop, mortRates, emis, webMapTrans, m, notMeters)
		if err != nil {
			return err
		}
		return nil
	}
}

// totalMassPopulation calculates the total pollution mass in the domain and the
// total population of group popGridColumn.
func (d *InMAP) totalMassPopulation(popGridColumn string) (totalMass, totalPopulation float64, err error) {
	iPop, ok := d.PopIndices[popGridColumn]
	if !ok {
		return math.Inf(-1), math.Inf(-1), fmt.Errorf("inmap: PopGridColumn '%s' does not exist in census file", popGridColumn)
	}
	for _, c := range *d.cells {
		totalMass += floats.Sum(c.Cf) * c.Volume
		if c.Layer == 0 { // only track population at ground level
			totalPopulation += c.PopData[iPop]
		}
	}
	return
}

// MutateGrid returns a function that creates a static variable
// resolution grid (i.e., one that does not change during the simulation)
// by dividing cells as determined by divideRule. Cells where divideRule is
// true are divided to the next nest level (up to the maximum nest level), and
// cells where divideRule is false are combined (down to the baseline nest level).
// Log messages are written to logChan if it is not nil.
func (config *VarGridConfig) MutateGrid(divideRule GridMutator, data *CTMData, pop *Population, mortRates *MortalityRates, emis *Emissions, m Mechanism, logChan chan string) DomainManipulator {
	return func(d *InMAP) error {
		if logChan != nil {
			logChan <- fmt.Sprint("Adding grid cells...")
		}

		beginCells := d.cells.len()

		totalMass, totalPopulation, err := d.totalMassPopulation(config.PopGridColumn)
		if err != nil {
			return err
		}

		webMapTrans, notMeters, err := config.webMapTrans()
		if err != nil {
			return err
		}

		continueMutating := true
		for continueMutating {
			continueMutating = false
			var newCellIndices [][][2]int
			var newCellLayers []int
			var newCellConc [][]float64
			var cellsToDelete []*cellRef
			for _, cell := range *d.cells {
				if len(cell.Index) < len(config.Xnests) {
					if divideRule(cell.Cell, totalMass, totalPopulation) {
						continueMutating = true

						// mark the grid cell for deletion
						cellsToDelete = append(cellsToDelete, cell)

						// Create inner nested cells instead of using this one.
						for ii := 0; ii < config.Xnests[len(cell.Index)]; ii++ {
							for jj := 0; jj < config.Ynests[len(cell.Index)]; jj++ {

								newIndex := make([][2]int, len(cell.Index)+1)
								for k, ij := range cell.Index {
									newIndex[k] = [2]int{ij[0], ij[1]}
								}
								newIndex[len(newIndex)-1] = [2]int{ii, jj}
								newCellIndices = append(newCellIndices, newIndex)
								newCellLayers = append(newCellLayers, cell.Layer)
								newCellConc = append(newCellConc, cell.Cf)
							}
						}
					}
				}
			}

			// Delete the grid cells.
			for _, cell := range cellsToDelete {
				d.cells.delete(cell)
				d.index.Delete(cell.Cell)
				cell.dereferenceNeighbors(d)
			}

			// Add new cells.
			err = d.addCells(config, newCellIndices, newCellLayers, newCellConc,
				data, pop, mortRates, emis, webMapTrans, m, notMeters)
			if err != nil {
				return err
			}
		}

		endCells := d.cells.len()
		if logChan != nil {
			logChan <- fmt.Sprintf("Added %d grid cells; there are now %d cells total",
				endCells-beginCells, endCells)
		}

		return nil
	}
}

func (d *InMAP) addCells(config *VarGridConfig, newCellIndices [][][2]int,
	newCellLayers []int, conc [][]float64, data *CTMData, pop *Population,
	mortRates *MortalityRates, emis *Emissions, webMapTrans proj.Transformer,
	m Mechanism, notMeters bool) error {
	type cellErr struct {
		cell *Cell
		err  error
	}
	cellErrChan := make(chan cellErr, len(newCellIndices))
	cellIndexChan := make(chan int)
	nprocs := runtime.GOMAXPROCS(-1)

	for p := 0; p < nprocs; p++ {
		go func() {
			for i := range cellIndexChan {
				ii := newCellIndices[i]
				var conci []float64
				if conc != nil {
					conci = conc[i]
				}
				cell, err2 := config.createCell(data, pop, d.PopIndices, mortRates, d.mortIndices, ii,
					newCellLayers[i], conci, webMapTrans, m, notMeters)
				cellErrChan <- cellErr{cell: cell, err: err2}
			}
		}()
	}

	// Create the new cells.
	for i := 0; i < len(newCellIndices); i++ {
		cellIndexChan <- i
	}
	close(cellIndexChan)
	// Insert the new cells into d.
	for range newCellIndices {
		cellerr := <-cellErrChan
		if cellerr.err != nil {
			return cellerr.err
		}
		d.InsertCell(cellerr.cell, m)
	}

	// Add emissions to new cells.
	// This needs to be called after setNeighbors.
	if err := d.SetEmissionsFlux(emis, m); err != nil {
		return err
	}
	return nil
}

// SetEmissionsFlux sets the emissions flux for the cells in the receiver
// based on the emissions in e.
func (d *InMAP) SetEmissionsFlux(emis *Emissions, m Mechanism) error {
	nprocs := runtime.GOMAXPROCS(-1)
	if emis != nil {
		cellIndexChan2 := make(chan int)
		errChan := make(chan error)
		for p := 0; p < nprocs; p++ {
			go func() {
				for i := range cellIndexChan2 {
					c := (*d.cells)[i]
					if len(c.EmisFlux) == 0 {
						if err := c.SetEmissionsFlux(emis, m); err != nil { // This needs to be called after setNeighbors.
							errChan <- err
							return
						}
					}
				}
				errChan <- nil
			}()
		}
		for i := 0; i < d.cells.len(); i++ {
			cellIndexChan2 <- i
		}
		close(cellIndexChan2)
		for p := 0; p < nprocs; p++ {
			if err := <-errChan; err != nil {
				return err
			}
		}
	}
	return nil
}

// InsertCell adds a new cell to the grid. The function will take the necessary
// steps to fit the new cell in with existing cells, but it is the caller's
// reponsibility that the new cell doesn't overlap any existing cells.
func (d *InMAP) InsertCell(c *Cell, m Mechanism) {
	if d.index == nil {
		d.init()
	}
	if c.Layer > d.nlayers-1 { // Make sure we still have the right number of layers
		d.nlayers = c.Layer + 1
	}
	d.cells.add(c)
	d.index.Insert(c)
	d.setNeighbors(c, m)
}

// A GridMutator is a function whether a Cell should be mutated (i.e., either
// divided or combined with other cells), where totalMass is absolute value
// of the total mass of pollution in the system and totalPopulation is the
// total population in the system.
type GridMutator func(cell *Cell, totalMass, totalPopulation float64) bool

// PopulationMutator returns a function that determines whether a grid cell
// should be split by determining whether either the cell population or
// maximum poulation density are above the thresholds specified in config.
func PopulationMutator(config *VarGridConfig, popIndices PopIndices) (GridMutator, error) {
	popIndex := popIndices[config.PopGridColumn]
	if config.PopThreshold <= 0 {
		return nil, fmt.Errorf("PopThreshold=%g. It needs to be set to a positive value.",
			config.PopThreshold)
	}
	if config.PopDensityThreshold <= 0 {
		return nil, fmt.Errorf("PopDensityThreshold=%g. It needs to be set to a positive value.",
			config.PopDensityThreshold)
	}
	return func(cell *Cell, _, _ float64) bool {
		population := 0.
		aboveDensityThreshold := false
		for _, g := range *cell.groundLevel {
			population += g.PopData[popIndex]
			if g.AboveDensityThreshold {
				aboveDensityThreshold = true
			}
		}
		return cell.Layer < config.HiResLayers &&
			(aboveDensityThreshold || population > config.PopThreshold)
	}, nil
}

// PopConcMutator is a holds an algorithm for dividing grid cells based on
// gradients in population density and concentration. Refer to the methods
// for additional documentation.
type PopConcMutator struct {
	config     *VarGridConfig
	popIndices PopIndices
}

// NewPopConcMutator initializes a new PopConcMutator object.
func NewPopConcMutator(config *VarGridConfig, popIndices PopIndices) *PopConcMutator {
	return &PopConcMutator{config: config, popIndices: popIndices}
}

// Mutate returns a function that takes a grid cell and returns whether
// Σ(|ΔConcentration|)*combinedVolume*|ΔPopulation| / {Σ(|totalMass|)*totalPopulation}
// > a threshold between the
// grid cell in question and any of its horizontal neighbors, where Σ(|totalMass|)
// is the sum of the absolute values of the mass of all pollutants in
// all grid cells in the system,
// Σ(|ΔConcentration|) is the sum of the absolute value of the difference
// between pollution concentations in the cell in question and the neighbor in
// question, |ΔPopulation| is the absolute value of the difference in population
// between the two grid cells, totalPopulation is the total population in the domain,
// and combinedVolume is the combined volume of the cell in question
// and the neighbor in question.
func (p *PopConcMutator) Mutate() GridMutator {
	iPop := p.popIndices[p.config.PopGridColumn]
	return func(cell *Cell, totalMass, totalPopulation float64) bool {
		if totalMass == 0. || totalPopulation == 0 {
			return false
		}
		var groundCellPop float64
		for _, gc := range *cell.groundLevel {
			groundCellPop += gc.PopData[iPop]
		}
		totalMassPop := totalMass * totalPopulation
		for _, group := range []*cellList{cell.west, cell.east, cell.north, cell.south} {
			for _, neighbor := range *group {
				var groundNeighborPop float64
				for _, gc := range *neighbor.groundLevel {
					groundNeighborPop += gc.PopData[iPop]
				}
				ΣΔC := 0.
				for i, conc := range neighbor.Cf {
					ΣΔC += math.Abs(conc - cell.Cf[i])
				}
				ΔP := math.Abs(groundCellPop - groundNeighborPop)
				if ΣΔC*(cell.Volume+neighbor.Volume)*ΔP/totalMassPop > p.config.PopConcThreshold {
					return true
				}
			}
		}
		return false
	}
}

// cellGeometry returns the geometry of a cell with the give index.
func (config *VarGridConfig) cellGeometry(index [][2]int) geom.Polygonal {
	xResFac, yResFac := 1., 1.
	l := config.VariableGridXo
	b := config.VariableGridYo
	for i, ii := range index {
		if i > 0 {
			xResFac *= float64(config.Xnests[i])
			yResFac *= float64(config.Ynests[i])
		}
		l += float64(ii[0]) * config.VariableGridDx / xResFac
		b += float64(ii[1]) * config.VariableGridDy / yResFac
	}
	r := l + config.VariableGridDx/xResFac
	u := b + config.VariableGridDy/yResFac
	return &geom.Bounds{Min: geom.Point{X: l, Y: b}, Max: geom.Point{X: r, Y: u}}
}

// createCell creates a new grid cell. If any of the census shapes
// that intersect the cell are above the population density threshold,
// then the grid cell is also set to being above the density threshold.
// If conc != nil, the concentration data for the new cell will be set to conc.
// notMeters should be set to true if the units of the grid are not
// in meters.
func (config *VarGridConfig) createCell(data *CTMData, pop *Population, popIndices PopIndices,
	mortRates *MortalityRates, mortIndices MortIndices, index [][2]int, layer int, conc []float64, webMapTrans proj.Transformer, m Mechanism, notMeters bool) (*Cell, error) {

	cell := new(Cell)
	cell.PopData = make([]float64, len(popIndices))
	cell.MortData = make([]float64, len(mortIndices))

	cell.Index = index
	// Polygon must go counter-clockwise
	cell.Polygonal = config.cellGeometry(index)
	if layer == 0 {
		// only ground level grid cells have people
		cell.loadPopMortalityRate(config, mortRates, mortIndices, pop, popIndices)
	}

	gg, err := cell.Polygonal.Transform(webMapTrans)
	if err != nil {
		return nil, err
	}
	cell.WebMapGeom = gg.(geom.Polygonal)

	var bounds *geom.Bounds
	if notMeters {
		bounds = cell.WebMapGeom.Bounds()
	} else {
		bounds = cell.Polygonal.Bounds()
	}
	cell.Dx = bounds.Max.X - bounds.Min.X
	cell.Dy = bounds.Max.Y - bounds.Min.Y

	cell.make(m)
	if err := cell.loadData(data, layer); err != nil {
		return nil, err
	}
	cell.Volume = cell.Dx * cell.Dy * cell.Dz

	if conc != nil {
		copy(cell.Cf, conc)
		copy(cell.Ci, conc)
	}

	return cell, nil
}

// loadPopMortalityRate calculates the population and baseline mortality rate for this cell.
// The population in each cell is calculated as an area-weighted average.
// The mortality rate in each cell is calculated as a population-weighted average. If
// multiple mortality rate polygons overlap or lie within a single population
// polygon, the mortality rate in each cell is equal to the population-weighted
// average of: the area-weighted average of mortality rates within each population polygon.
func (c *Cell) loadPopMortalityRate(config *VarGridConfig, mortRates *MortalityRates, mortIndices MortIndices, pop *Population, popIndices PopIndices) {
	// First, prepare mortality rates for later processing.
	cellMortI := mortRates.tree.SearchIntersect(c.Bounds())
	cellMort := make([]*mortality, len(cellMortI))
	for i, mI := range cellMortI {
		m := mI.(*mortality)
		cellMort[i] = &mortality{
			Polygonal: c.Polygonal.Intersection(m.Polygonal),
			MortData:  m.MortData,
		}
	}

	// Second, intersect each grid cell with population polygons
	popGen := pop.tree(c.Bounds())
	for {
		p, err := popGen()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		if p == nil {
			continue
		}
		pIntersection := c.Polygonal.Intersection(p.Polygonal)
		if pIntersection == nil {
			continue
		}
		pAreaIntersect := pIntersection.Area()
		if pAreaIntersect == 0 {
			continue
		}
		pArea := p.Area() // we want to conserve the total population
		if pArea == 0. {
			panic("divide by zero")
		}
		pAreaFrac := pAreaIntersect / pArea
		for popType, pop := range p.PopData {
			c.PopData[popType] += pop * pAreaFrac
		}
		// Check if this census shape is above the density threshold.
		pDensity := p.PopData[popIndices[config.PopGridColumn]] / pArea
		if pDensity > config.PopDensityThreshold {
			c.AboveDensityThreshold = true
		}
		var mAreaTotal float64
		// Third, intersect each intersection from first step with
		// mortality rate polygons.
		for _, m := range cellMort {
			mIntersection := pIntersection.Intersection(m.Polygonal)
			if mIntersection == nil {
				continue
			}
			mAreaIntersect := mIntersection.Area()
			if mAreaIntersect == 0 {
				continue
			}
			// Sum areas of intersecting mortality rate polygons for use in area-weighting.
			mAreaTotal += mAreaIntersect
		}
		for _, mInterface := range mortRates.tree.SearchIntersect(pIntersection.Bounds()) {
			m := mInterface.(*mortality)
			mIntersection := pIntersection.Intersection(m.Polygonal)
			mAreaIntersect := mIntersection.Area()
			if mAreaIntersect == 0 {
				continue
			}
			// Perform population-weighted average of area-weighted average mortality rates.
			for mortType, popType := range config.MortalityRateColumns {
				c.MortData[mortIndices[mortType]] += p.PopData[popIndices[popType]] * pAreaFrac * m.MortData[mortIndices[mortType]] * (mAreaIntersect / mAreaTotal)
			}
		}
	}
	for mortType, popType := range config.MortalityRateColumns {
		if c.PopData[popIndices[popType]] > 0 {
			c.MortData[mortIndices[mortType]] = c.MortData[mortIndices[mortType]] / c.PopData[popIndices[popType]]
		}
	}
}

type population struct {
	geom.Polygonal

	// PopData holds the number of people in each population category
	PopData []float64
}

type mortality struct {
	geom.Polygonal

	// MortData holds the mortality rate for each population category
	MortData []float64 // Deaths per 100,000 people per year
}

// loadPopulation loads population information from a shapefile or
// COARDS-compliant NetCDF file (determined by file extension), converting it
// to spatial reference sr and then discarding any geometries that do not
// overlap with bounds. The function outputs an index holding the population
// information and a map giving the array index of each population type.
func (config *VarGridConfig) loadPopulation(sr *proj.SR, bounds *geom.Bounds) (func(*geom.Bounds) func() (*population, error), map[string]int, error) {
	x := filepath.Ext(config.CensusFile)
	if x == ".shp" {
		return config.loadPopulationShapefile(sr, bounds)
	} else if x == ".ncf" || x == ".nc" {
		return config.loadPopulationCOARDS(sr)
	}
	return nil, nil, fmt.Errorf("inmap: invalid CensusFile type %s; valid types are .shp, .nc and .ncf", x)
}

// loadPopulationShapefile loads population information from a shapefile, converting it
// to spatial reference sr and discarding any geometryies that do not overlap
// with bounds. The function outputs an index holding the population
// information and a map giving the array index of each population type.
func (config *VarGridConfig) loadPopulationShapefile(sr *proj.SR, bounds *geom.Bounds) (func(*geom.Bounds) func() (*population, error), map[string]int, error) {
	var err error
	popshp, err := shp.NewDecoder(config.CensusFile)
	if err != nil {
		return nil, nil, err
	}
	popsr, err := popshp.SR()
	if err != nil {
		return nil, nil, err
	}
	trans, err := popsr.NewTransform(sr)
	if err != nil {
		return nil, nil, err
	}

	// Create a list of array indices for each population type.
	popIndices := make(map[string]int)
	for i, p := range config.CensusPopColumns {
		popIndices[p] = i
	}

	pop := rtree.NewTree(25, 50)
	for {
		g, fields, more := popshp.DecodeRowFields(config.CensusPopColumns...)
		if !more {
			break
		}
		p := &population{PopData: make([]float64, len(config.CensusPopColumns))}
		for i, pop := range config.CensusPopColumns {
			s, ok := fields[pop]
			if !ok {
				return nil, nil, fmt.Errorf("inmap: loading population shapefile: missing attribute column %s", pop)
			}
			p.PopData[i], err = s2f(s)
			if err != nil {
				return nil, nil, err
			}
			if math.IsNaN(p.PopData[i]) {
				return nil, nil, fmt.Errorf("inmap: loadPopulation: NaN population value")
			}
		}
		gg, err := g.Transform(trans)
		if err != nil {
			return nil, nil, err
		}
		switch gg.(type) {
		case geom.Polygonal:
			p.Polygonal = gg.(geom.Polygonal)
		default:
			return nil, nil, fmt.Errorf("inmap: loadPopulation: population shapes need to be polygons")
		}
		if bounds.Overlaps(p.Bounds()) {
			pop.Insert(p)
		}
	}
	if err := popshp.Error(); err != nil {
		return nil, nil, err
	}

	popshp.Close()
	return func(b *geom.Bounds) func() (*population, error) {
		pops := pop.SearchIntersect(b)
		i := -1
		return func() (*population, error) {
			i++
			if i >= len(pops) {
				return nil, io.EOF
			}
			return pops[i].(*population), nil
		}
	}, popIndices, nil
}

// loadPopulationCOARDS loads population information from a
// COARDS-compliant NetCDF file (NetCDF 4 and greater not supported), converting it
// to spatial reference sr and discarding any geometryies that do not overlap
// with bounds. The function outputs an index holding the population
// information and a map giving the array index of each population type.
// Data in the COARDS file are assumed to be row-major (i.e., latitude-major).
// Information regarding the COARDS NetCDF conventions are
// available here: https://ferret.pmel.noaa.gov/Ferret/documentation/coards-netcdf-conventions.COARDs.
func (config *VarGridConfig) loadPopulationCOARDS(sr *proj.SR) (func(*geom.Bounds) func() (*population, error), map[string]int, error) {
	// Pretend this is an emissions file to avoid rewriting the COARDS reader.
	raster, err := aep.ReadCOARDSFile(config.CensusFile, time.Unix(0, 0), time.Unix(1, 0), aep.Kg, aep.SourceData{})
	if err != nil {
		return nil, nil, fmt.Errorf("inmap: reading NetCDF CensusFile: %w", err)
	}

	inputSR, err := proj.Parse("+proj=longlat")
	if err != nil {
		panic(err)
	}

	ct, err := inputSR.NewTransform(sr)
	if err != nil {
		return nil, nil, fmt.Errorf("inmap: loading population COARDS file: %w", err)
	}
	inverseCT, err := sr.NewTransform(inputSR)
	if err != nil {
		return nil, nil, fmt.Errorf("inmap: loading population COARDS file: %w", err)
	}

	popIndex := make(map[string]int)
	for i, p := range config.CensusPopColumns {
		popIndex[p] = i
	}

	return func(b *geom.Bounds) func() (*population, error) {
		gb, err := densePolygonFromBounds(b).Transform(inverseCT)
		if err != nil {
			panic(err)
		}
		gen := raster.RecordGenerator(gb.Bounds())

		return func() (*population, error) {
			rec, err := gen()
			if err != nil {
				if err == io.EOF {
					return nil, io.EOF
				}
				return nil, fmt.Errorf("inmap: reading NetCDF CensusFile records: %w", err)
			}

			vals := rec.Totals()
			pops := make([]float64, len(config.CensusPopColumns))
			var nonZero bool
			for i, p := range config.CensusPopColumns {
				u, ok := vals[aep.Pollutant{Name: p}]
				if !ok {
					return nil, fmt.Errorf("inmap: missing CensusFile CensusPopColumn %s", p)
				}
				v := u.Value()
				if math.IsNaN(v) {
					continue
				}
				nonZero = true
				pops[i] = v
			}

			if nonZero {
				lg, err := rec.Location().Transform(ct)
				if err != nil {
					panic(err)
				}
				return &population{
					Polygonal: lg.(geom.Polygonal),
					PopData:   pops,
				}, nil
			}
			return nil, nil
		}
	}, popIndex, nil
}

func densePolygonFromBounds(b *geom.Bounds) geom.Polygon {
	dx := b.Max.X - b.Min.X
	dy := b.Max.Y - b.Min.Y
	return geom.Polygon{{
		{X: b.Min.X, Y: b.Min.Y},
		{X: b.Min.X + dx/4, Y: b.Min.Y},
		{X: b.Min.X + dx/2, Y: b.Min.Y},
		{X: b.Min.X + dx*3/4, Y: b.Min.Y},
		{X: b.Max.X, Y: b.Min.Y},
		{X: b.Max.X, Y: b.Min.Y + dy/4},
		{X: b.Max.X, Y: b.Min.Y + dy/2},
		{X: b.Max.X, Y: b.Min.Y + dy*3/4},
		{X: b.Max.X, Y: b.Max.Y},
		{X: b.Max.X - dx/4, Y: b.Max.Y},
		{X: b.Max.X - dx/2, Y: b.Max.Y},
		{X: b.Max.X - dx*3/4, Y: b.Max.Y},
		{X: b.Min.X, Y: b.Max.Y},
		{X: b.Min.X, Y: b.Max.Y - dy/4},
		{X: b.Min.X, Y: b.Max.Y - dy/2},
		{X: b.Min.X, Y: b.Max.Y - dy*3/4},
		{X: b.Min.X, Y: b.Min.Y},
	}}
}

func s2f(s string) (float64, error) {
	s = strings.Trim(s, "\x00* ")
	if s == "" {
		// null value
		return 0., nil
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err
}

func (config *VarGridConfig) loadMortality(sr *proj.SR) (*rtree.Rtree, map[string]int, error) {
	mortshp, err := shp.NewDecoder(config.MortalityRateFile)
	if err != nil {
		return nil, nil, err
	}

	mortshpSR, err := mortshp.SR()
	if err != nil {
		return nil, nil, err
	}
	trans, err := mortshpSR.NewTransform(sr)
	if err != nil {
		return nil, nil, err
	}

	// Create a list of array indices for each mortality rate.
	mortIndices := make(map[string]int)

	// Extract mortality rate column names from map of population to mortality rates
	mortRateColumns := make([]string, len(config.MortalityRateColumns))
	i := 0
	for m := range config.MortalityRateColumns {
		mortRateColumns[i] = m
		i++
	}
	sort.Strings(mortRateColumns)
	for i, m := range mortRateColumns {
		mortIndices[m] = i
	}
	mortRates := rtree.NewTree(25, 50)
	for {
		g, fields, more := mortshp.DecodeRowFields(mortRateColumns...)
		if !more {
			break
		}
		m := new(mortality)
		m.MortData = make([]float64, len(mortRateColumns))
		for i, mort := range mortRateColumns {
			s, ok := fields[mort]
			if !ok {
				return nil, nil, fmt.Errorf("inmap: loading mortality rate shapefile: missing attribute column %s", mort)
			}
			m.MortData[i], err = s2f(s)
			if err != nil {
				return nil, nil, err
			}
			if math.IsNaN(m.MortData[i]) {
				panic("NaN mortality rate!")
			}
		}
		gg, err := g.Transform(trans)
		if err != nil {
			return nil, nil, err
		}
		switch gg.(type) {
		case geom.Polygonal:
			m.Polygonal = gg.(geom.Polygonal)
		default:
			return nil, nil, fmt.Errorf("inmap: loadMortality: mortality rate shapes need to be polygons")
		}
		mortRates.Insert(m)
	}
	if err := mortshp.Error(); err != nil {
		return nil, nil, err
	}
	mortshp.Close()
	return mortRates, mortIndices, nil
}

// loadData allocates cell information from the CTM data to the Cell. If the
// cell overlaps more than one CTM cells, weighted averaging is used.
func (c *Cell) loadData(data *CTMData, k int) error {
	c.Layer = k
	cellArea := c.Area()
	ctmcellsAllLayers := data.gridTree.SearchIntersect(c.Bounds())
	var ctmcells []*gridCellLight
	var fractions []float64
	for _, cc := range ctmcellsAllLayers {
		// we only want grid cells that match our layer.
		ccc := cc.(*gridCellLight)
		if ccc.layer == k {
			isect := ccc.Intersection(c.Polygonal)
			if isect != nil {
				fractions = append(fractions, isect.Area()/cellArea)
				ctmcells = append(ctmcells, ccc)
			}
		}
	}
	if sum := floats.Sum(fractions); sum < 0.9 {
		return fmt.Errorf("there is not CTM data overlapping at least 90 percent of the "+
			"InMAP cell at %+v; grid dimensions: X=%g -- %g; Y=%g -- %g",
			c.Polygonal, data.xo, data.xo+data.dx*float64(data.nx),
			data.yo, data.yo+data.dy*float64(data.ny))
	}
	for i, ctmcell := range ctmcells {
		ctmrow := ctmcell.Row
		ctmcol := ctmcell.Col
		frac := fractions[i]

		// TODO: Average velocity is on a staggered grid, so we should
		// do some sort of interpolation here.
		c.UAvg += data.Data["UAvg"].Data.Get(k, ctmrow, ctmcol) * frac
		c.VAvg += data.Data["VAvg"].Data.Get(k, ctmrow, ctmcol) * frac
		c.WAvg += data.Data["WAvg"].Data.Get(k, ctmrow, ctmcol) * frac

		c.UDeviation += data.Data["UDeviation"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.VDeviation += data.Data["VDeviation"].Data.Get(
			k, ctmrow, ctmcol) * frac

		c.AOrgPartitioning += data.Data["aOrgPartitioning"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.BOrgPartitioning += data.Data["bOrgPartitioning"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.NOPartitioning += data.Data["NOPartitioning"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.SPartitioning += data.Data["SPartitioning"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.NHPartitioning += data.Data["NHPartitioning"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.SO2oxidation += data.Data["SO2oxidation"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.ParticleDryDep += data.Data["ParticleDryDep"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.SO2DryDep += data.Data["SO2DryDep"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.NOxDryDep += data.Data["NOxDryDep"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.NH3DryDep += data.Data["NH3DryDep"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.VOCDryDep += data.Data["VOCDryDep"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.Kxxyy += data.Data["Kxxyy"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.LayerHeight += data.Data["LayerHeights"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.Dz += data.Data["Dz"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.ParticleWetDep += data.Data["ParticleWetDep"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.SO2WetDep += data.Data["SO2WetDep"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.OtherGasWetDep += data.Data["OtherGasWetDep"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.Kzz += data.Data["Kzz"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.M2u += data.Data["M2u"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.M2d += data.Data["M2d"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.WindSpeed += data.Data["WindSpeed"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.WindSpeedInverse += data.Data["WindSpeedInverse"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.WindSpeedMinusThird += data.Data["WindSpeedMinusThird"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.WindSpeedMinusOnePointFour +=
			data.Data["WindSpeedMinusOnePointFour"].Data.Get(
				k, ctmrow, ctmcol) * frac
		c.Temperature += data.Data["Temperature"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.S1 += data.Data["S1"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.SClass += data.Data["Sclass"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.CBaseline[iPM2_5] += data.Data["TotalPM25"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.CBaseline[igNH] += data.Data["gNH"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.CBaseline[ipNH] += data.Data["pNH"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.CBaseline[igNO] += data.Data["gNO"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.CBaseline[ipNO] += data.Data["pNO"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.CBaseline[igS] += data.Data["gS"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.CBaseline[ipS] += data.Data["pS"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.CBaseline[igOrg] += data.Data["aVOC"].Data.Get(
			k, ctmrow, ctmcol) * frac
		c.CBaseline[ipOrg] += data.Data["aSOA"].Data.Get(
			k, ctmrow, ctmcol) * frac
	}
	return nil
}

// make a vector representation of the chemical transport model grid
func (data *CTMData) makeCTMgrid(nlayers int) {
	data.gridTree = rtree.NewTree(25, 50)
	for k := 0; k < nlayers; k++ {
		for ix := 0; ix < data.nx; ix++ {
			for iy := 0; iy < data.ny; iy++ {
				cell := new(gridCellLight)
				x0 := data.xo + data.dx*float64(ix)
				x1 := data.xo + data.dx*float64(ix+1)
				y0 := data.yo + data.dy*float64(iy)
				y1 := data.yo + data.dy*float64(iy+1)
				cell.Polygonal = &geom.Bounds{
					Min: geom.Point{X: x0, Y: y0},
					Max: geom.Point{X: x1, Y: y1},
				}
				cell.Row = iy
				cell.Col = ix
				cell.layer = k
				data.gridTree.Insert(cell)
			}
		}
	}
}

type gridCellLight struct {
	geom.Polygonal
	Row, Col, layer int
}
