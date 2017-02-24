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
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"

	"bitbucket.org/ctessum/cdf"
	"bitbucket.org/ctessum/sparse"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/proj"
	"github.com/gonum/floats"
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

	ctmGridXo float64 // lower left of Chemical Transport Model (CTM) grid, x
	ctmGridYo float64 // lower left of grid, y
	ctmGridDx float64 // m
	ctmGridDy float64 // m
	ctmGridNx int
	ctmGridNy int

	PopDensityThreshold float64 // limit for people per unit area in the grid cell
	PopThreshold        float64 // limit for total number of people in the grid cell

	// PopConcThreshold is the limit for
	// Σ(|ΔConcentration|)*combinedVolume*|ΔPopulation| / {Σ(|totalMass|)*totalPopulation}.
	// See the documentation for PopConcMutator for more information.
	PopConcThreshold float64

	CensusFile          string   // Path to census shapefile
	CensusPopColumns    []string // Shapefile fields containing populations for multiple demographics
	PopGridColumn       string   // Name of field in shapefile to be used for determining variable grid resolution
	MortalityRateFile   string   // Path to the mortality rate shapefile
	MortalityRateColumn string   // Name of field in mortality rate shapefile containing the mortality rate.

	GridProj string // projection info for CTM grid; Proj4 format
}

// CTMData holds processed data from a chemical transport model
type CTMData struct {
	gridTree *rtree.Rtree

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
	config.ctmGridDx = f.Header.GetAttribute("", "dx").([]float64)[0]
	config.ctmGridDy = f.Header.GetAttribute("", "dy").([]float64)[0]
	config.ctmGridNx = int(f.Header.GetAttribute("", "nx").([]int32)[0])
	config.ctmGridNy = int(f.Header.GetAttribute("", "ny").([]int32)[0])
	config.ctmGridXo = f.Header.GetAttribute("", "x0").([]float64)[0]
	config.ctmGridYo = f.Header.GetAttribute("", "y0").([]float64)[0]

	dataVersion := f.Header.GetAttribute("", "data_version").(string)

	if dataVersion != InMAPDataVersion {
		return nil, fmt.Errorf("inmap.LoadCTMData: data version %s is incompatible "+
			"with the required version %s", dataVersion, InMAPDataVersion)
	}

	o.gridTree = config.makeCTMgrid(nz)

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

// Write writes d to w. x0 and y0 are the left and y coordinates of the
// lower-left corner of the domain, and dx and dy are the x and y edge
// lengths of the grid cells, respectively.
func (d *CTMData) Write(w *os.File, x0, y0, dx, dy float64) error {
	windSpeed := d.Data["WindSpeed"].Data
	uAvg := d.Data["UAvg"].Data
	vAvg := d.Data["VAvg"].Data
	wAvg := d.Data["WAvg"].Data
	h := cdf.NewHeader(
		[]string{"x", "y", "z", "xStagger", "yStagger", "zStagger"},
		[]int{windSpeed.Shape[2], windSpeed.Shape[1], windSpeed.Shape[0],
			uAvg.Shape[2], vAvg.Shape[1], wAvg.Shape[0]})
	h.AddAttribute("", "comment", "InMAP meteorology and baseline chemistry data file")

	h.AddAttribute("", "x0", []float64{x0})
	h.AddAttribute("", "y0", []float64{y0})
	h.AddAttribute("", "dx", []float64{dx})
	h.AddAttribute("", "dy", []float64{dy})
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
	tree *rtree.Rtree
}

// MortalityRates is a holder for information about the average human
// mortality rate (in units of deaths per 100,000 people per year) in the
// model domain.
type MortalityRates struct {
	tree *rtree.Rtree
}

// PopIndices give the array indices of each
// population type.
type PopIndices map[string]int

// LoadPopMort loads the population and mortality rate data from the shapefiles
// specified in config.
func (config *VarGridConfig) LoadPopMort() (*Population, PopIndices, *MortalityRates, error) {
	gridSR, err := proj.Parse(config.GridProj)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("inmap: while parsing GridProj: %v", err)
	}

	pop, popIndex, err := config.loadPopulation(gridSR)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("inmap: while loading population: %v", err)
	}
	mort, err := config.loadMortality(gridSR)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("inmap: while loading mortality rate: %v", err)
	}
	return &Population{tree: pop}, PopIndices(popIndex), &MortalityRates{tree: mort}, nil
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

func (config *VarGridConfig) webMapTrans() (proj.Transformer, error) {

	// webMapProj is the spatial reference definition for web mapping.
	const webMapProj = "+proj=merc +a=6378137 +b=6378137 +lat_ts=0.0 +lon_0=0.0 +x_0=0.0 +y_0=0 +k=1.0 +units=m +nadgrids=@null +no_defs"
	// webMapSR is the spatial reference for web mapping.
	webMapSR, err := proj.Parse(webMapProj)
	if err != nil {
		return nil, fmt.Errorf("inmap: while parsing webMapProj: %v", err)
	}

	gridSR, err := proj.Parse(config.GridProj)
	if err != nil {
		return nil, fmt.Errorf("inmap: while parsing GridProj: %v", err)
	}
	webMapTrans, err := gridSR.NewTransform(webMapSR)
	if err != nil {
		return nil, fmt.Errorf("inmap: while creating webMapTrans: %v", err)
	}
	return webMapTrans, nil
}

// RegularGrid returns a function that creates a new regular
// (i.e., not variable resolution) grid
// as specified by the information in c.
func (config *VarGridConfig) RegularGrid(data *CTMData, pop *Population, popIndex PopIndices, mort *MortalityRates, emis *Emissions) DomainManipulator {
	return func(d *InMAP) error {

		webMapTrans, err := config.webMapTrans()
		if err != nil {
			return err
		}

		d.popIndices = (map[string]int)(popIndex)

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
		err = d.addCells(config, indices, layers, nil, data, pop, mort, emis, webMapTrans)
		if err != nil {
			return err
		}
		return nil
	}
}

// totalMassPopulation calculates the total pollution mass in the domain and the
// total population of group popGridColumn.
func (d *InMAP) totalMassPopulation(popGridColumn string) (totalMass, totalPopulation float64, err error) {
	iPop, ok := d.popIndices[popGridColumn]
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
func (config *VarGridConfig) MutateGrid(divideRule GridMutator, data *CTMData, pop *Population, mort *MortalityRates, emis *Emissions, logChan chan string) DomainManipulator {
	return func(d *InMAP) error {

		if logChan != nil {
			logChan <- fmt.Sprint("Adding grid cells...")
		}

		beginCells := d.cells.len()

		totalMass, totalPopulation, err := d.totalMassPopulation(config.PopGridColumn)
		if err != nil {
			return err
		}

		webMapTrans, err := config.webMapTrans()
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
				data, pop, mort, emis, webMapTrans)
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
	mort *MortalityRates, emis *Emissions, webMapTrans proj.Transformer) error {
	type cellErr struct {
		cell *Cell
		err  error
	}
	cellErrChan := make(chan cellErr)
	nprocs := runtime.GOMAXPROCS(-1)

	// Create the new cells.
	for p := 0; p < nprocs; p++ {
		go func(p int) {
			for i := p; i < len(newCellIndices); i += nprocs {
				ii := newCellIndices[i]
				var conci []float64
				if conc != nil {
					conci = conc[i]
				}
				cell, err2 := config.createCell(data, pop, d.popIndices, mort, ii,
					newCellLayers[i], conci, webMapTrans)
				cellErrChan <- cellErr{cell: cell, err: err2}
			}
		}(p)
	}
	// Insert the new cells into d.
	for range newCellIndices {
		cellerr := <-cellErrChan
		if cellerr.err != nil {
			return cellerr.err
		}
		d.InsertCell(cellerr.cell)
	}
	// Add emissions to new cells.
	if emis != nil {
		doneChan := make(chan int)
		for p := 0; p < nprocs; p++ {
			go func(p int) {
				for i := p; i < d.cells.len(); i += nprocs {
					c := (*d.cells)[i]
					if len(c.EmisFlux) == 0 {
						c.setEmissionsFlux(emis) // This needs to be called after setNeighbors.
					}
				}
				doneChan <- 0
			}(p)
		}
		for p := 0; p < nprocs; p++ {
			<-doneChan
		}
	}
	return nil
}

// InsertCell adds a new cell to the grid. The function will take the necessary
// steps to fit the new cell in with existing cells, but it is the caller's
// reponsibility that the new cell doesn't overlap any existing cells.
func (d *InMAP) InsertCell(c *Cell) {
	if d.index == nil {
		d.init()
	}
	if c.Layer > d.nlayers-1 { // Make sure we still have the right number of layers
		d.nlayers = c.Layer + 1
	}
	d.cells.add(c)
	d.index.Insert(c)
	d.setNeighbors(c)
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
	return geom.Polygon([]geom.Path{{{X: l, Y: b}, {X: r, Y: b}, {X: r, Y: u}, {X: l, Y: u}, {X: l, Y: b}}})
}

// createCell creates a new grid cell. If any of the census shapes
// that intersect the cell are above the population density threshold,
// then the grid cell is also set to being above the density threshold.
// If conc != nil, the concentration data for the new cell will be set to conc.
func (config *VarGridConfig) createCell(data *CTMData, pop *Population, popIndices PopIndices,
	mort *MortalityRates, index [][2]int, layer int, conc []float64, webMapTrans proj.Transformer) (*Cell, error) {

	cell := new(Cell)
	cell.PopData = make([]float64, len(popIndices))
	cell.Index = index
	// Polygon must go counter-clockwise
	cell.Polygonal = config.cellGeometry(index)
	if layer == 0 {
		// only ground level grid cells have people
		cell.loadPopulation(config, pop, popIndices)
		cell.loadMortalityRate(mort)
	}
	bounds := cell.Polygonal.Bounds()
	cell.Dx = bounds.Max.X - bounds.Min.X
	cell.Dy = bounds.Max.Y - bounds.Min.Y

	cell.make()
	if err := cell.loadData(data, layer); err != nil {
		return nil, err
	}
	cell.Volume = cell.Dx * cell.Dy * cell.Dz

	gg, err := cell.Polygonal.Transform(webMapTrans)
	if err != nil {
		return nil, err
	}
	cell.WebMapGeom = gg.(geom.Polygonal)

	if conc != nil {
		copy(cell.Cf, conc)
		copy(cell.Ci, conc)
	}

	return cell, nil
}

// loadPopulation calculates the population in this Cell.
func (c *Cell) loadPopulation(config *VarGridConfig, pop *Population, popIndices PopIndices) {
	for _, pInterface := range pop.tree.SearchIntersect(c.Bounds()) {
		p := pInterface.(*population)
		intersection := c.Intersection(p)
		area1 := intersection.Area()
		area2 := p.Area() // we want to conserve the total population
		if area2 == 0. {
			panic("divide by zero")
		}
		areaFrac := area1 / area2
		for popType, pop := range p.PopData {
			c.PopData[popType] += pop * areaFrac
		}

		// Check if this census shape is above the density threshold
		pDensity := p.PopData[popIndices[config.PopGridColumn]] / area2
		if pDensity > config.PopDensityThreshold {
			c.AboveDensityThreshold = true
		}
	}
}

// loadMortalityRate calculates the baseline mortality rate for this cell.
func (c *Cell) loadMortalityRate(mort *MortalityRates) {
	for _, mInterface := range mort.tree.SearchIntersect(c.Bounds()) {
		m := mInterface.(*mortality)
		intersection := c.Intersection(m)
		area1 := intersection.Area()
		area2 := c.Area() // we want to conserve the average rate here, not the total
		if area2 == 0. {
			panic("divide by zero")
		}
		areaFrac := area1 / area2
		c.MortalityRate += m.AllCause * areaFrac
	}
}

type population struct {
	geom.Polygonal

	// PopData holds the number of people in each population category
	PopData []float64
}

type mortality struct {
	geom.Polygonal
	AllCause float64 // Deaths per 100,000 people per year
}

// loadPopulation loads population information from a shapefile, converting it
// to spatial reference sr. The function outputs an index holding the population
// information and a map giving the array index of each population type.
func (config *VarGridConfig) loadPopulation(sr *proj.SR) (*rtree.Rtree, map[string]int, error) {
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
		p := new(population)
		p.PopData = make([]float64, len(config.CensusPopColumns))
		for i, pop := range config.CensusPopColumns {
			p.PopData[i], err = s2f(fields[pop])
			if err != nil {
				return nil, nil, err
			}
			if math.IsNaN(p.PopData[i]) {
				panic("NaN!")
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
		pop.Insert(p)
	}
	if err := popshp.Error(); err != nil {
		return nil, nil, err
	}

	popshp.Close()
	return pop, popIndices, nil
}

func (config *VarGridConfig) loadMortality(sr *proj.SR) (*rtree.Rtree, error) {
	mortshp, err := shp.NewDecoder(config.MortalityRateFile)
	if err != nil {
		return nil, err
	}

	mortshpSR, err := mortshp.SR()
	if err != nil {
		return nil, err
	}
	trans, err := mortshpSR.NewTransform(sr)
	if err != nil {
		return nil, err
	}

	mortalityrate := rtree.NewTree(25, 50)
	for {
		g, fields, more := mortshp.DecodeRowFields(config.MortalityRateColumn)
		if !more {
			break
		}
		m := new(mortality)
		m.AllCause, err = s2f(fields[config.MortalityRateColumn])
		if err != nil {
			return nil, err
		}
		if math.IsNaN(m.AllCause) {
			return nil, fmt.Errorf("NaN mortality rate")
		}
		gg, err := g.Transform(trans)
		if err != nil {
			return nil, err
		}
		switch gg.(type) {
		case geom.Polygonal:
			m.Polygonal = gg.(geom.Polygonal)
		default:
			return nil, fmt.Errorf("inmap: loadMortality: mortality rate shapes need to be polygons")
		}
		mortalityrate.Insert(m)
	}
	if err := mortshp.Error(); err != nil {
		return nil, err
	}
	mortshp.Close()
	return mortalityrate, nil
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
			isect := ccc.Intersection(c)
			if isect != nil {
				fractions = append(fractions, isect.Area()/cellArea)
				ctmcells = append(ctmcells, ccc)
			}
		}
	}
	if sum := floats.Sum(fractions); sum < 0.9 {
		return fmt.Errorf("there is not CTM data overlapping at least 90%% of the InMAP cell at %+v", c.Centroid())
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
func (config *VarGridConfig) makeCTMgrid(nlayers int) *rtree.Rtree {
	tree := rtree.NewTree(25, 50)
	for k := 0; k < nlayers; k++ {
		for ix := 0; ix < config.ctmGridNx; ix++ {
			for iy := 0; iy < config.ctmGridNy; iy++ {
				cell := new(gridCellLight)
				x0 := config.ctmGridXo + config.ctmGridDx*float64(ix)
				x1 := config.ctmGridXo + config.ctmGridDx*float64(ix+1)
				y0 := config.ctmGridYo + config.ctmGridDy*float64(iy)
				y1 := config.ctmGridYo + config.ctmGridDy*float64(iy+1)
				cell.Polygonal = geom.Polygon{[]geom.Point{
					{X: x0, Y: y0},
					{X: x1, Y: y0},
					{X: x1, Y: y1},
					{X: x0, Y: y1},
					{X: x0, Y: y0},
				}}
				cell.Row = iy
				cell.Col = ix
				cell.layer = k
				tree.Insert(cell)
			}
		}
	}
	return tree
}

type gridCellLight struct {
	geom.Polygonal
	Row, Col, layer int
}
