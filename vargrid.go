package inmap

import (
	"fmt"
	"math"
	"os"
	"sort"

	"bitbucket.org/ctessum/cdf"
	"bitbucket.org/ctessum/sparse"

	"github.com/ctessum/geom"
	"github.com/ctessum/geom/encoding/shp"
	"github.com/ctessum/geom/index/rtree"
	"github.com/ctessum/geom/proj"
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

	PopDensityCutoff    float64  // limit for people per unit area in the grid cell
	PopCutoff           float64  // limit for total number of people in the grid cell
	BboxOffset          float64  // A number significantly less than the smallest grid size but not small enough to be confused with zero.
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
	data     map[string]ctmVariable
}

type ctmVariable struct {
	dims        []string           // netcdf dimensions for this variable
	description string             // variable description
	units       string             // variable units
	data        *sparse.DenseArray // variable data
}

// AddVariable adds data for a new variable to d.
func (d *CTMData) AddVariable(name string, dims []string, description, units string, data *sparse.DenseArray) {
	if d.data == nil {
		d.data = make(map[string]ctmVariable)
	}
	d.data[name] = ctmVariable{
		dims:        dims,
		description: description,
		units:       units,
		data:        data,
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

	o.gridTree = config.makeCTMgrid(nz)

	od := make(map[string]ctmVariable)
	for _, v := range f.Header.Variables() {
		d := ctmVariable{}
		d.description = f.Header.GetAttribute(v, "description").(string)
		d.units = f.Header.GetAttribute(v, "units").(string)
		dims := f.Header.Lengths(v)
		r := f.Reader(v, nil, nil)
		d.data = sparse.ZerosDense(dims...)
		tmp := make([]float32, len(d.data.Elements))
		_, err = r.Read(tmp)
		if err != nil {
			return nil, fmt.Errorf("inmap.LoadCTMData: %v", err)
		}
		d.dims = f.Header.Dimensions(v)

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
			d.data.Elements[i] = float64(v)
		}
		od[v] = d
	}
	o.data = od
	return o, nil
}

// Write writes d to w. x0 and y0 are the left and y coordinates of the
// lower-left corner of the domain, and dx and dy are the x and y edge
// lengths of the grid cells, respectively.
func (d *CTMData) Write(w *os.File, x0, y0, dx, dy float64) error {
	windSpeed := d.data["WindSpeed"].data
	uAvg := d.data["UAvg"].data
	vAvg := d.data["VAvg"].data
	wAvg := d.data["WAvg"].data
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

	for name, dd := range d.data {
		h.AddVariable(name, dd.dims, []float32{0})
		h.AddAttribute(name, "description", dd.description)
		h.AddAttribute(name, "units", dd.units)
	}
	h.Define()

	f, err := cdf.Create(w, h) // writes the header to ff
	if err != nil {
		return err
	}
	for name, dd := range d.data {
		if err = writeNCF(f, name, dd.data); err != nil {
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
// model domain
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

func (d *InMAP) sort() {
	sortCells(d.Cells)
	sortCells(d.westBoundary)
	sortCells(d.eastBoundary)
	sortCells(d.northBoundary)
	sortCells(d.southBoundary)
	sortCells(d.topBoundary)
}

// sortCells sorts the cells by layer, x centroid, and y centroid.
func sortCells(cells []*Cell) {
	sc := &cellsSorter{
		cells: cells,
	}
	sort.Sort(sc)
}

type cellsSorter struct {
	cells []*Cell
}

// Len is part of sort.Interface.
func (c *cellsSorter) Len() int {
	return len(c.cells)
}

// Swap is part of sort.Interface.
func (c *cellsSorter) Swap(i, j int) {
	c.cells[i], c.cells[j] = c.cells[j], c.cells[i]
}

func (c *cellsSorter) Less(i, j int) bool {
	ci := c.cells[i]
	cj := c.cells[j]
	if ci.Layer != cj.Layer {
		return ci.Layer < cj.Layer
	}

	icent := ci.Polygonal.Centroid()
	jcent := cj.Polygonal.Centroid()

	if icent.X != jcent.X {
		return icent.X < jcent.X
	}
	if icent.Y != jcent.Y {
		return icent.Y < jcent.Y
	}
	// We apparently have concentric or identical cells if we get to here.
	panic(fmt.Errorf("problem sorting: i: %v, i layer: %d, j: %v, j layer: %d",
		ci.Polygonal, ci.Layer, cj.Polygonal, cj.Layer))
}

// getCells returns all the grid cells in cellTree that are within box
// and at vertical layer layer.
func getCells(cellTree *rtree.Rtree, box *geom.Bounds, layer int) []*Cell {
	x := cellTree.SearchIntersect(box)
	cells := make([]*Cell, 0, len(x))
	for _, xx := range x {
		c := xx.(*Cell)
		if c.Layer == layer {
			cells = append(cells, c)
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

		nz := data.data["UAvg"].data.Shape[0]
		d.nlayers = nz
		d.index = rtree.NewTree(25, 50)

		nx := config.Xnests[0]
		ny := config.Ynests[0]
		d.Cells = make([]*Cell, 0, nx*ny*nz)
		// Iterate through indices and create the cells in the outermost nest.
		for k := 0; k < nz; k++ {
			for j := 0; j < ny; j++ {
				for i := 0; i < nx; i++ {
					index := [][2]int{{i, j}}
					// Create the cell
					cell, err := config.createCell(data, pop, d.popIndices, mort, index, k, webMapTrans)
					if err != nil {
						return err
					}
					d.AddCells(cell)
				}
			}
		}
		// Add emissions to new cells.
		for _, c := range d.Cells {
			c.setEmissionsFlux(emis) // This needs to be called after setNeighbors.
		}
		return nil
	}
}

// StaticVariableGrid returns a function that creates a static variable
// resolution grid (i.e., one that does not change during the simulation)
// by dividing cells in the previously created grid
// based on the population and population density cutoffs in config.
func (config *VarGridConfig) StaticVariableGrid(data *CTMData, pop *Population, mort *MortalityRates, emis *Emissions) DomainManipulator {
	return func(d *InMAP) error {

		webMapTrans, err := config.webMapTrans()
		if err != nil {
			return err
		}

		continueSplitting := true
		for continueSplitting {
			continueSplitting = false
			var newCellIndices [][][2]int
			var newCellLayers []int
			var indicesToDelete []int
			for i, cell := range d.Cells {
				if len(cell.index) < len(config.Xnests) {
					// Check if this grid cell is above the population threshold
					// or the population density threshold.
					if cell.Layer < config.HiResLayers &&
						(cell.aboveDensityThreshold ||
							cell.PopData[d.popIndices[config.PopGridColumn]] > config.PopCutoff) {

						continueSplitting = true
						indicesToDelete = append(indicesToDelete, i)

						// If this cell is above a threshold, create inner
						// nested cells instead of using this one.
						for ii := 0; ii < config.Xnests[len(cell.index)]; ii++ {
							for jj := 0; jj < config.Ynests[len(cell.index)]; jj++ {
								newIndex := append(cell.index, [2]int{ii, jj})
								newCellIndices = append(newCellIndices, newIndex)
								newCellLayers = append(newCellLayers, cell.Layer)
							}
						}
					}
				}
			}
			// Delete the cells that were split.
			d.DeleteCells(indicesToDelete...)
			// Add the new cells.
			oldNumCells := len(d.Cells)
			for i, ii := range newCellIndices {
				cell, err := config.createCell(data, pop, d.popIndices, mort, ii, newCellLayers[i], webMapTrans)
				if err != nil {
					return err
				}
				d.AddCells(cell)
			}
			// Add emissions to new cells.
			for i := oldNumCells - 1; i < len(d.Cells); i++ {
				d.Cells[i].setEmissionsFlux(emis) // This needs to be called after setNeighbors.
			}
		}
		d.sort()
		return nil
	}
}

// AddCells adds a new cell to the grid. The function will take the necessary
// steps to fit the new cell in with existing cells, but it is the caller's
// reponsibility that the new cell doesn't overlap any existing cells.
func (d *InMAP) AddCells(cells ...*Cell) {
	for _, c := range cells {
		d.Cells = append(d.Cells, c)
		d.index.Insert(c)
		const bboxOffset = 1.e-10
		d.setNeighbors(c, bboxOffset)
	}
}

// DeleteCells deletes the cell with index i from the grid and removes any
// references to it from other cells.
func (d *InMAP) DeleteCells(indicesToDelete ...int) {
	indexToSubtract := 0
	for _, ii := range indicesToDelete {
		i := ii - indexToSubtract
		c := d.Cells[i]
		copy(d.Cells[i:], d.Cells[i+1:])
		d.Cells[len(d.Cells)-1] = nil
		d.Cells = d.Cells[:len(d.Cells)-1]
		d.index.Delete(c)
		c.dereferenceNeighbors(d)
		indexToSubtract++
	}
}

// createCell creates a new grid cell. If any of the census shapes
// that intersect the cell are above the population density threshold,
// then the grid cell is also set to being above the density threshold.
func (config *VarGridConfig) createCell(data *CTMData, pop *Population, popIndices PopIndices, mort *MortalityRates, index [][2]int, layer int, webMapTrans proj.Transformer) (*Cell, error) {

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

	cell := new(Cell)
	cell.PopData = make([]float64, len(popIndices))
	cell.index = index
	// Polygon must go counter-clockwise
	cell.Polygonal = geom.Polygon([][]geom.Point{{{l, b}, {r, b}, {r, u}, {l, u}, {l, b}}})
	for _, pInterface := range pop.tree.SearchIntersect(cell.Bounds()) {
		p := pInterface.(*population)
		intersection := cell.Intersection(p)
		area1 := intersection.Area()
		area2 := p.Area() // we want to conserve the total population
		if area2 == 0. {
			panic("divide by zero")
		}
		areaFrac := area1 / area2
		for popType, pop := range p.PopData {
			cell.PopData[popType] += pop * areaFrac
		}

		// Check if this census shape is above the density threshold
		pDensity := p.PopData[popIndices[config.PopGridColumn]] / area2
		if pDensity > config.PopDensityCutoff {
			cell.aboveDensityThreshold = true
		}
	}
	for _, mInterface := range mort.tree.SearchIntersect(cell.Bounds()) {
		m := mInterface.(*mortality)
		intersection := cell.Intersection(m)
		area1 := intersection.Area()
		area2 := cell.Area() // we want to conserve the average rate here, not the total
		if area2 == 0. {
			panic("divide by zero")
		}
		areaFrac := area1 / area2
		cell.MortalityRate += m.AllCause * areaFrac
	}
	cell.Dx = r - l
	cell.Dy = u - b

	cell.make()
	cell.loadData(data, layer)
	cell.Volume = cell.Dx * cell.Dy * cell.Dz

	gg, err := cell.Polygonal.Transform(webMapTrans)
	if err != nil {
		return nil, err
	}
	cell.WebMapGeom = gg.(geom.Polygonal)

	return cell, nil
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
		p.Polygonal = gg.(geom.Polygonal)
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
		m.Polygonal = gg.(geom.Polygonal)
		mortalityrate.Insert(m)
	}
	if err := mortshp.Error(); err != nil {
		return nil, err
	}
	mortshp.Close()
	return mortalityrate, nil
}

func (c *Cell) loadData(data *CTMData, k int) {
	c.Layer = k
	ctmcellsAllLayers := data.gridTree.SearchIntersect(c.Bounds())
	var ctmcells []*gridCellLight
	for _, cc := range ctmcellsAllLayers {
		// we only want grid cells that match our layer.
		ccc := cc.(*gridCellLight)
		if ccc.layer == k {
			ctmcells = append(ctmcells, ccc)
		}
	}
	ncells := float64(len(ctmcells))
	if len(ctmcells) == 0. {
		panic("No matching cells!")
	}
	for _, ctmcell := range ctmcells {
		ctmrow := ctmcell.Row
		ctmcol := ctmcell.Col

		// TODO: Average velocity is on a staggered grid, so we should
		// do some sort of interpolation here.
		c.UAvg += data.data["UAvg"].data.Get(k, ctmrow, ctmcol) / ncells
		c.VAvg += data.data["VAvg"].data.Get(k, ctmrow, ctmcol) / ncells
		c.WAvg += data.data["WAvg"].data.Get(k, ctmrow, ctmcol) / ncells

		c.UDeviation += data.data["UDeviation"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.VDeviation += data.data["VDeviation"].data.Get(
			k, ctmrow, ctmcol) / ncells

		c.AOrgPartitioning += data.data["aOrgPartitioning"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.BOrgPartitioning += data.data["bOrgPartitioning"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.NOPartitioning += data.data["NOPartitioning"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.SPartitioning += data.data["SPartitioning"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.NHPartitioning += data.data["NHPartitioning"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.SO2oxidation += data.data["SO2oxidation"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.ParticleDryDep += data.data["ParticleDryDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.SO2DryDep += data.data["SO2DryDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.NOxDryDep += data.data["NOxDryDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.NH3DryDep += data.data["NH3DryDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.VOCDryDep += data.data["VOCDryDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.Kxxyy += data.data["Kxxyy"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.LayerHeight += data.data["LayerHeights"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.Dz += data.data["Dz"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.ParticleWetDep += data.data["ParticleWetDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.SO2WetDep += data.data["SO2WetDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.OtherGasWetDep += data.data["OtherGasWetDep"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.Kzz += data.data["Kzz"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.M2u += data.data["M2u"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.M2d += data.data["M2d"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.WindSpeed += data.data["WindSpeed"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.WindSpeedInverse += data.data["WindSpeedInverse"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.WindSpeedMinusThird += data.data["WindSpeedMinusThird"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.WindSpeedMinusOnePointFour +=
			data.data["WindSpeedMinusOnePointFour"].data.Get(
				k, ctmrow, ctmcol) / ncells
		c.Temperature += data.data["Temperature"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.S1 += data.data["S1"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.SClass += data.data["Sclass"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.CBaseline[iPM2_5] += data.data["TotalPM25"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.CBaseline[igNH] += data.data["gNH"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.CBaseline[ipNH] += data.data["pNH"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.CBaseline[igNO] += data.data["gNO"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.CBaseline[ipNO] += data.data["pNO"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.CBaseline[igS] += data.data["gS"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.CBaseline[ipS] += data.data["pS"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.CBaseline[igOrg] += data.data["aVOC"].data.Get(
			k, ctmrow, ctmcol) / ncells
		c.CBaseline[ipOrg] += data.data["aSOA"].data.Get(
			k, ctmrow, ctmcol) / ncells

	}
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
					geom.Point{X: x0, Y: y0},
					geom.Point{X: x1, Y: y0},
					geom.Point{X: x1, Y: y1},
					geom.Point{X: x0, Y: y1},
					geom.Point{X: x0, Y: y0},
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
